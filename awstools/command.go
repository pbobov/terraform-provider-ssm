package awstools

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	log "github.com/hashicorp/terraform-plugin-log/tflog"
)

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

	aws := AwsClients{ctx: ctx, ec2Client: ec2.NewFromConfig(cfg), ssmClient: ssm.NewFromConfig(cfg), s3Client: s3.NewFromConfig(cfg)}

	return &aws, nil
}

// Wait until the target EC2 instances status is online
func (aws AwsClients) waitForTargetInstances(ec2Filters []ec2types.Filter, ssmFilters []ssmtypes.InstanceInformationStringFilter, waitTimeout int) bool {
	for i := 0; i < waitTimeout/sleepTime; i++ {
		ec2Instances, err := aws.ec2Client.DescribeInstances(aws.ctx, &ec2.DescribeInstancesInput{
			Filters: ec2Filters,
		})

		if err != nil {
			log.Error(aws.ctx, err.Error())
			return false
		}

		ssmInstances, err := aws.ssmClient.DescribeInstanceInformation(aws.ctx, &ssm.DescribeInstanceInformationInput{
			Filters: ssmFilters,
		})

		if err != nil {
			log.Error(aws.ctx, err.Error())
			return false
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

			log.Info(aws.ctx, fmt.Sprintf("%d of %d target instances are online.", onlineInstanceCount, ec2InstanceCount))

			if onlineInstanceCount == ec2InstanceCount {
				return true
			}
		}

		time.Sleep(sleepTime * time.Second)
	}

	log.Info(aws.ctx, "Target instances are not online.")

	return false
}

// Wait for the command invocations to complete
func (aws AwsClients) waitForCommandInvocations(commandId string, timeout int) ssmtypes.CommandInvocationStatus {
	for i := 0; i < timeout/sleepTime; i++ {
		output, err := aws.ssmClient.ListCommandInvocations(aws.ctx, &ssm.ListCommandInvocationsInput{
			CommandId: &commandId,
		})

		if err != nil {
			log.Error(aws.ctx, err.Error())
			return ssmtypes.CommandInvocationStatusFailed
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
				log.Info(aws.ctx, fmt.Sprintf("Command %s invocation %s on instance %s.",
					commandId, invocation.Status, *invocation.InstanceId))
				return invocation.Status
			}
		}

		if pendingExecutionsCount == 0 {
			return ssmtypes.CommandInvocationStatusSuccess
		}

		time.Sleep(sleepTime * time.Second)
	}

	log.Error(aws.ctx, "Command invocations timed out.")

	return ssmtypes.CommandInvocationStatusTimedOut
}

// Retrieves from S3 and prints outputs of the command invocations.
func (aws AwsClients) printCommandOutput(prefix string, commandId string, s3Bucket string) {
	keyPrefix := prefix + "/" + commandId

	objects, err := aws.s3Client.ListObjectsV2(aws.ctx, &s3.ListObjectsV2Input{
		Bucket:  &s3Bucket,
		MaxKeys: 1000,
		Prefix:  &keyPrefix,
	})

	if err != nil {
		log.Error(aws.ctx, err.Error())
		return
	}

	if objects.Contents != nil {
		for _, key := range objects.Contents {
			object, err := aws.s3Client.GetObject(aws.ctx, &s3.GetObjectInput{
				Bucket: &s3Bucket,
				Key:    key.Key,
			})

			if err != nil {
				log.Error(aws.ctx, err.Error())
			} else {
				bytes, err := io.ReadAll(object.Body)
				if err == nil {
					log.Info(aws.ctx, fmt.Sprintf("\n*** %s ***\n%s", *key.Key, string(bytes)))
				}
			}
		}
	}
}

// Wait until the target EC2 instances status is online.
// Send SSM command.
// Wait for the command invocations to complete.
// Retrieves from S3 and prints outputs of the command invocations.
func (aws AwsClients) RunCommand(documentName string, parameters map[string][]string, ssmTargets []ssmtypes.Target, executionTimeout int, comment string, s3Bucket string, s3KeyPrefix string) (ssmtypes.Command, error) {
	var ec2Filters []ec2types.Filter
	var ssmFilters []ssmtypes.InstanceInformationStringFilter

	for _, target := range ssmTargets {
		ec2Filters = append(ec2Filters, ec2types.Filter{Name: target.Key, Values: target.Values})
		ssmFilters = append(ssmFilters, ssmtypes.InstanceInformationStringFilter{Key: target.Key, Values: target.Values})
	}

	instanceStateName := "instance-state-name"
	ec2Filters = append(ec2Filters, ec2types.Filter{Name: &instanceStateName, Values: []string{"pending", "running"}})

	aws.waitForTargetInstances(ec2Filters, ssmFilters, waitTimeout)

	output, err := aws.ssmClient.SendCommand(aws.ctx, &ssm.SendCommandInput{
		Targets:            ssmTargets,
		DocumentName:       &documentName,
		Parameters:         parameters,
		Comment:            &comment,
		TimeoutSeconds:     &sendTimeout,
		OutputS3BucketName: &s3Bucket,
		OutputS3KeyPrefix:  &s3KeyPrefix,
	})

	if err != nil {
		log.Error(aws.ctx, err.Error())
		return ssmtypes.Command{}, err
	}

	aws.waitForCommandInvocations(*output.Command.CommandId, executionTimeout)

	aws.printCommandOutput(s3KeyPrefix, *output.Command.CommandId, s3Bucket)

	commandId := *output.Command.CommandId

	return aws.GetCommand(commandId)
}

// Retrieves SSM command info by Id.
func (aws AwsClients) GetCommand(commandId string) (ssmtypes.Command, error) {
	commands, err := aws.ssmClient.ListCommands(aws.ctx, &ssm.ListCommandsInput{
		CommandId: &commandId,
	})

	if err != nil {
		return ssmtypes.Command{}, err
	}

	return commands.Commands[0], nil
}
