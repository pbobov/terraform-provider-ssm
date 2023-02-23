package awstools

import (
	"context"
	"time"

	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Resource timeouts
var createTimeout time.Duration = time.Duration(24) * time.Hour
var readTimeout time.Duration = time.Duration(60) * time.Second
var updateTimeout time.Duration = time.Duration(24) * time.Hour
var deleteTimeout time.Duration = time.Duration(60) * time.Second
var defaultTimeout time.Duration = time.Duration(24) * time.Hour

// Attributes of ssm_command resource
const (
	attDocumentName        string = "document_name"
	attParameters          string = "parameters"
	attDestroyDocumentName string = "destroy_document_name"
	attDestroyParameters   string = "destroy_parameters"
	attTargets             string = "targets"
	attExecutionTimeout    string = "execution_timeout"
	attComment             string = "comment"
	attOutputLocation      string = "output_location"
	attS3BucketName        string = "s3_bucket_name"
	attS3KeyPrefix         string = "s3_key_prefix"
	attName                string = "name"
	attKey                 string = "key"
	attValues              string = "values"
	attStatus              string = "status"
	attRequestedTime       string = "requested_time"
)

type OutputLocation struct {
	s3Bucket    *string
	s3KeyPrefix *string
}

func getParameters(d *schema.ResourceData, parametersKey string) map[string][]string {
	ssmParameters := make(map[string][]string)

	parameters := d.Get(parametersKey).([]interface{})

	for _, p := range parameters {
		parameter := p.(map[string]interface{})
		name := parameter[attName].(string)
		var values []string
		for _, value := range parameter[attValues].([]interface{}) {
			if value != nil {
				values = append(values, value.(string))
			}
		}
		ssmParameters[name] = values
	}

	return ssmParameters
}

func getTargets(d *schema.ResourceData) []ssmtypes.Target {
	var ssmTargets []ssmtypes.Target

	targets := d.Get(attTargets).([]interface{})

	for _, t := range targets {
		target := t.(map[string]interface{})
		key := target[attKey].(string)
		var values []string
		for _, value := range target[attValues].([]interface{}) {
			values = append(values, value.(string))
		}
		ssmTargets = append(ssmTargets, ssmtypes.Target{Key: &key, Values: values})
	}

	return ssmTargets
}

func getOutputLocation(d *schema.ResourceData) OutputLocation {
	outputLocation := d.Get(attOutputLocation).([]interface{})

	if len(outputLocation) == 0 {
		return OutputLocation{}
	}

	location := outputLocation[0].(map[string]interface{})

	var s3Bucket *string = nil
	var s3KeyPrefix *string = nil

	val, ok := location[attS3BucketName]
	if ok {
		str := val.(string)
		if str != "" {
			s3Bucket = &str
		}
	}

	val, ok = location[attS3KeyPrefix]
	if ok {
		str := val.(string)
		if str != "" {
			s3KeyPrefix = &str
		}
	}

	return OutputLocation{s3Bucket: s3Bucket, s3KeyPrefix: s3KeyPrefix}
}

func resourceCommandCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	documentName := d.Get(attDocumentName).(string)
	executionTimeout := d.Get(attExecutionTimeout).(int)
	comment := d.Get(attComment).(string)
	ssmParameters := getParameters(d, attParameters)
	ssmTargets := getTargets(d)
	outputLocation := getOutputLocation(d)

	extendedCtx, cancel := context.WithTimeout(ctx, time.Duration(executionTimeout+60)*time.Second)
	defer cancel()

	clients, err := NewAwsClients(extendedCtx)

	if err != nil {
		return diag.FromErr(err)
	}

	command, err := clients.RunCommand(&documentName, ssmParameters, ssmTargets, &executionTimeout, &comment, outputLocation.s3Bucket, outputLocation.s3KeyPrefix)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*command.CommandId)

	if err := d.Set(attStatus, command.Status); err != nil {
		return diag.FromErr(err)
	}

	requestedTime := command.RequestedDateTime.UTC().Format(time.RFC3339)

	if err := d.Set(attRequestedTime, requestedTime); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceCommandRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	commandId := d.Id()

	clients, err := NewAwsClients(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	command, err := clients.GetCommand(commandId)

	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(attStatus, command.Status); err != nil {
		return diag.FromErr(err)
	}

	requestedTime := command.RequestedDateTime.UTC().Format(time.RFC3339)

	if err := d.Set(attRequestedTime, requestedTime); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceCommandUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceCommandCreate(ctx, d, m)
}

func resourceCommandDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	documentName := d.Get(attDestroyDocumentName).(string)

	if documentName != "" {
		executionTimeout := d.Get(attExecutionTimeout).(int)
		comment := d.Get(attComment).(string)
		ssmParameters := getParameters(d, attDestroyParameters)
		ssmTargets := getTargets(d)
		outputLocation := getOutputLocation(d)

		extendedCtx, cancel := context.WithTimeout(ctx, time.Duration(executionTimeout+60)*time.Second)
		defer cancel()

		clients, err := NewAwsClients(extendedCtx)

		if err != nil {
			return diag.FromErr(err)
		}

		_, err = clients.RunCommand(&documentName, ssmParameters, ssmTargets, &executionTimeout, &comment, outputLocation.s3Bucket, outputLocation.s3KeyPrefix)

		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId("")

	return diags
}

func resourceCommand() *schema.Resource {
	return &schema.Resource{
		Timeouts: &schema.ResourceTimeout{
			Create:  &createTimeout,
			Read:    &readTimeout,
			Update:  &updateTimeout,
			Delete:  &deleteTimeout,
			Default: &defaultTimeout,
		},
		CreateContext: resourceCommandCreate,
		ReadContext:   resourceCommandRead,
		UpdateContext: resourceCommandUpdate,
		DeleteContext: resourceCommandDelete,
		Schema: map[string]*schema.Schema{
			attDocumentName: {
				Type:     schema.TypeString,
				Required: true,
			},
			attParameters: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						attName: {
							Type:     schema.TypeString,
							Required: true,
						},
						attValues: {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			attDestroyDocumentName: {
				Type:     schema.TypeString,
				Optional: true,
			},
			attDestroyParameters: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						attName: {
							Type:     schema.TypeString,
							Required: true,
						},
						attValues: {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			attTargets: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						attKey: {
							Type:     schema.TypeString,
							Required: true,
						},
						attValues: {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			attExecutionTimeout: {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3600,
			},
			attComment: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			attOutputLocation: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						attS3BucketName: {
							Type:     schema.TypeString,
							Optional: true,
						},
						attS3KeyPrefix: {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			attStatus: {
				Type:     schema.TypeString,
				Computed: true,
			},
			attRequestedTime: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}
