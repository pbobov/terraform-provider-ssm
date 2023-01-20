package awstools

import (
	"context"
	"time"

	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Attributes of awstools_command resource
const (
	attDocumentName     string = "document_name"
	attParameters       string = "parameters"
	attTargets          string = "targets"
	attExecutionTimeout string = "execution_timeout"
	attComment          string = "comment"
	attOutputLocation   string = "output_location"
	attS3BucketName     string = "s3_bucket_name"
	attS3KeyPrefix      string = "s3_key_prefix"
	attName             string = "name"
	attKey              string = "key"
	attValues           string = "values"
	attStatus           string = "status"
	attRequestedTime    string = "requested_time"
)

func resourceCommandCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	documentName := d.Get(attDocumentName).(string)
	parameters := d.Get(attParameters).([]interface{})
	targets := d.Get(attTargets).([]interface{})
	executionTimeout := d.Get(attExecutionTimeout).(int)
	comment := d.Get(attComment).(string)

	outputLocation := d.Get(attOutputLocation).([]interface{})
	location := outputLocation[0].(map[string]interface{})
	s3Bucket := location[attS3BucketName].(string)
	s3KeyPrefix := location[attS3KeyPrefix].(string)

	ssmParameters := make(map[string][]string)

	for _, p := range parameters {
		parameter := p.(map[string]interface{})
		name := parameter[attName].(string)
		var values []string
		for _, value := range parameter[attValues].([]interface{}) {
			values = append(values, value.(string))
		}
		ssmParameters[name] = values
	}

	var ssmTargets []ssmtypes.Target

	for _, t := range targets {
		target := t.(map[string]interface{})
		key := target[attKey].(string)
		var values []string
		for _, value := range target[attValues].([]interface{}) {
			values = append(values, value.(string))
		}
		ssmTargets = append(ssmTargets, ssmtypes.Target{Key: &key, Values: values})
	}

	aws, err := NewAwsClients(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	command, err := aws.RunCommand(documentName, ssmParameters, ssmTargets, executionTimeout, comment, s3Bucket, s3KeyPrefix)

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

	aws, err := NewAwsClients(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	command, err := aws.GetCommand(commandId)

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

	d.SetId("")

	return diags
}

func resourceCommand() *schema.Resource {
	return &schema.Resource{
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
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						attS3BucketName: {
							Type:     schema.TypeString,
							Required: true,
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
