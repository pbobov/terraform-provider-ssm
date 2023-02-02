package awstools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	log "github.com/hashicorp/terraform-plugin-log/tflog"
)

// EC2 filter names
var ec2FilterInstanceId = "instance-id"
var ec2FilterInstanceStateName = "instance-state-name"

// SSM target keys
var ssmTargetInstanceIds = "InstanceIds"

var sendTimeout int32 = 600

const waitTimeout = 600
const sleepTime = 10

type AwsClients struct {
	ctx       context.Context
	ec2Client *ec2.Client
	ssmClient *ssm.Client
	s3Client  *s3.Client
}

func NewAwsClients(ctx context.Context) (*AwsClients, error) {
	cfg, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		return nil, err
	}

	clients := AwsClients{ctx: ctx, ec2Client: ec2.NewFromConfig(cfg), ssmClient: ssm.NewFromConfig(cfg), s3Client: s3.NewFromConfig(cfg)}

	return &clients, nil
}

// Wait until the target EC2 instances status is online
func (clients AwsClients) waitForTargetInstances(ec2Filters []ec2types.Filter, ssmFilters []ssmtypes.InstanceInformationStringFilter, waitTimeout int) error {
	for i := 0; i < waitTimeout/sleepTime; i++ {
		ec2Instances, err := clients.ec2Client.DescribeInstances(clients.ctx, &ec2.DescribeInstancesInput{
			Filters: ec2Filters,
		})

		if err != nil {
			log.Error(clients.ctx, err.Error())
			return err
		}

		ssmInstances, err := clients.ssmClient.DescribeInstanceInformation(clients.ctx, &ssm.DescribeInstanceInformationInput{
			Filters: ssmFilters,
		})

		if err != nil {
			log.Error(clients.ctx, err.Error())
			return err
		}

		if len(ssmInstances.InstanceInformationList) > 0 {
			ec2InstanceCount := 0

			for _, reservation := range ec2Instances.Reservations {
				ec2InstanceCount += len(reservation.Instances)
			}

			onlineInstanceCount := 0

			for _, instance := range ssmInstances.InstanceInformationList {
				if instance.PingStatus == ssmtypes.PingStatusOnline {
					onlineInstanceCount += 1
				}
			}

			log.Info(clients.ctx, fmt.Sprintf("%d of %d target instances are online.", onlineInstanceCount, ec2InstanceCount))

			if onlineInstanceCount == ec2InstanceCount {
				return nil
			}
		}

		time.Sleep(sleepTime * time.Second)
	}

	log.Error(clients.ctx, "Target instances are not online.")

	return errors.New("target instances are not online")
}

// Wait for the command invocations to complete
func (clients AwsClients) waitForCommandInvocations(commandId string, timeout *int) error {
	for i := 0; i < *timeout/sleepTime; i++ {
		output, err := clients.ssmClient.ListCommandInvocations(clients.ctx, &ssm.ListCommandInvocationsInput{
			CommandId: &commandId,
		})

		if err != nil {
			log.Error(clients.ctx, err.Error())
			return err
		}

		if len(output.CommandInvocations) == 0 {
			time.Sleep(sleepTime * time.Second)
			continue
		}

		pendingExecutionsCount := 0

		for _, invocation := range output.CommandInvocations {
			if invocation.Status == "Pending" || invocation.Status == "InProgress" {
				pendingExecutionsCount += 1
			} else if invocation.Status == "Cancelled" || invocation.Status == "TimedOut" || invocation.Status == "Failed" {
				log.Info(clients.ctx, fmt.Sprintf("Command %s invocation %s on instance %s.",
					commandId, invocation.Status, *invocation.InstanceId))

				return fmt.Errorf("command invocation %s on %s instance", strings.ToLower(string(invocation.Status)), *invocation.InstanceId)
			}
		}

		if pendingExecutionsCount == 0 {
			return nil
		}

		time.Sleep(sleepTime * time.Second)
	}

	log.Error(clients.ctx, "Command invocations timed out.")

	return errors.New("command invocations timed out")
}

// Retrieves from S3 and prints outputs of the command invocations.
func (clients AwsClients) printCommandOutput(prefix *string, commandId string, s3Bucket *string) error {
	if s3Bucket == nil || *s3Bucket == "" {
		log.Info(clients.ctx, "The output S3 bucket is not specified for ssm_command resource.")
		return nil
	}

	location, err := clients.s3Client.GetBucketLocation(clients.ctx, &s3.GetBucketLocationInput{
		Bucket: s3Bucket,
	})

	if err != nil {
		log.Error(clients.ctx, err.Error())
		return err
	}

	// Create S3 service client with a specific Region.
	cfg, err := config.LoadDefaultConfig(clients.ctx)

	if err != nil {
		log.Error(clients.ctx, err.Error())
		return err
	}

	cfg.Region = string(location.LocationConstraint)
	s3BucketClient := s3.NewFromConfig(cfg)

	keyPrefix := commandId
	if prefix != nil {
		keyPrefix = *prefix + "/" + commandId
	}

	objects, err := s3BucketClient.ListObjectsV2(clients.ctx, &s3.ListObjectsV2Input{
		Bucket:  s3Bucket,
		MaxKeys: 1000,
		Prefix:  &keyPrefix,
	})

	if err != nil {
		log.Error(clients.ctx, err.Error())
		return err
	}

	if objects.Contents != nil {
		for _, key := range objects.Contents {
			object, err := s3BucketClient.GetObject(clients.ctx, &s3.GetObjectInput{
				Bucket: s3Bucket,
				Key:    key.Key,
			})

			if err != nil {
				log.Error(clients.ctx, err.Error())
			} else {
				bytes, err := io.ReadAll(object.Body)
				if err == nil {
					log.Info(clients.ctx, fmt.Sprintf("\n*** %s ***\n%s", *key.Key, string(bytes)))
				}
			}
		}
	}

	return nil
}

// Waits until the target EC2 instances status is online.
// Sends SSM command.
// Waits for the command invocations to complete.
// Retrieves from S3 and prints outputs of the command invocations.
func (clients AwsClients) RunCommand(documentName *string, parameters map[string][]string, ssmTargets []ssmtypes.Target, executionTimeout *int, comment *string, s3Bucket *string, s3KeyPrefix *string) (ssmtypes.Command, error) {
	var ec2Filters []ec2types.Filter
	var ssmFilters []ssmtypes.InstanceInformationStringFilter

	for _, target := range ssmTargets {
		ec2FilterName := target.Key
		if strings.EqualFold(*target.Key, ssmTargetInstanceIds) {
			ec2FilterName = &ec2FilterInstanceId
		}

		ec2Filters = append(ec2Filters, ec2types.Filter{Name: ec2FilterName, Values: target.Values})
		ssmFilters = append(ssmFilters, ssmtypes.InstanceInformationStringFilter{Key: target.Key, Values: target.Values})
	}

	ec2Filters = append(ec2Filters, ec2types.Filter{Name: &ec2FilterInstanceStateName, Values: []string{"pending", "running"}})

	err := clients.waitForTargetInstances(ec2Filters, ssmFilters, waitTimeout)

	if err != nil {
		log.Error(clients.ctx, err.Error())
		return ssmtypes.Command{}, err
	}

	output, err := clients.ssmClient.SendCommand(clients.ctx, &ssm.SendCommandInput{
		Targets:            ssmTargets,
		DocumentName:       documentName,
		Parameters:         parameters,
		Comment:            comment,
		TimeoutSeconds:     &sendTimeout,
		OutputS3BucketName: s3Bucket,
		OutputS3KeyPrefix:  s3KeyPrefix,
	})

	if err != nil {
		log.Error(clients.ctx, err.Error())
		return ssmtypes.Command{}, err
	}

	commandId := *output.Command.CommandId

	err = clients.waitForCommandInvocations(commandId, executionTimeout)

	clients.printCommandOutput(s3KeyPrefix, commandId, s3Bucket)

	if err != nil {
		log.Error(clients.ctx, err.Error())
		return ssmtypes.Command{}, err
	}

	return clients.GetCommand(commandId)
}

// Retrieves SSM command info by Id.
func (clients AwsClients) GetCommand(commandId string) (ssmtypes.Command, error) {
	commands, err := clients.ssmClient.ListCommands(clients.ctx, &ssm.ListCommandsInput{
		CommandId: &commandId,
	})

	if err != nil {
		return ssmtypes.Command{}, err
	}

	return commands.Commands[0], nil
}
