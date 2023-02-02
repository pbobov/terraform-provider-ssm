---
page_title: "ssm_command Resource - terraform-provider-ssm"
subcategory: ""
description: |-
Sends SSM command to managed EC2 instances  
---

# ssm_command (Resource)

The resource sends SSM command to managed EC2 instances and waits for the command invocations to be completed on all the target instances.

Before sending the command, the resource waits for all the target EC2 instances to be online as SSM managed instances.

After the command invocations are completed, the resource retrieves the command outputs from the output S3 bucket and logs them to the terraform log as an INFO level message.

## Example Usage

```terraform
resource "ssm_command" "greeting" {
  document_name = "AWS-RunShellScript"
  parameters {
    name   = "commands"
    values = ["echo 'Hello World!'"]
  }
  destroy_document_name = "AWS-RunShellScript"
  destroy_parameters {
    name   = "commands"
    values = ["echo 'Goodbye World.'"]
  }
  targets {
    key    = "InstanceIds"
    values = [aws_instance.world.id]
  }
  comment           = "Greetings from SSM!"
  execution_timeout = 600
  output_location {
    s3_bucket_name = aws_s3_bucket.output.bucket
    s3_key_prefix  = "greetings"
  }
}
```

## Schema

### Required

- `document_name` (String) - Name of SSM command document to run on the resource creation.
- `parameters` (Block List) - Block of arbitrary string parameters to pass to the SSM document.
- `targets` (Block List) - Block containing the targets of the SSM command invocations. Targets are documented below.

### Optional

- `comment` (String) - User-specified information about the command, such as a brief description of what the command should do.
- `execution_timeout` (Number) - Command invocation timeout in seconds. Default timeout is 3600 seconds.
- `destroy_document_name` (String) - Name of SSM command document to run on the resource destruction. If not set, no SSM command is executed on the resource destruction.
- `destroy_parameters` (Block List) - Block of arbitrary string parameters to pass to the SSM document on the resource destruction.
- `output_location` (Block) - SSM command output location settings. If not specified, the SSM commands use default output location. Output_location is documented below.

### Read-Only

- `id` (String) The SSM command Id.
- `requested_time` (String) - Date and time the command was requested.
- `status` (String) - Status of the SSM command invocations.

### Nested Schema for `parameters`

Parameters blocks specify names and values of SSM command parameters:

- `name` (String) - SSM document parameter name.
- `values` (List of String) - List of parameter values.

### Nested Schema for `targets`

Targets blocks specify what instance IDs or tags to apply the document to and has these keys:

- `key` (String) - Either `InstanceIds` or `tag:Tag Name` to specify an EC2 tag.
- `values` (List of String) - List of instance IDs or tag values.

### Nested Schema for `output_location`

Optional:

- `s3_bucket_name` (String) - Output S3 bucket name. If not specified, the SSM commands use default output location.
- `s3_key_prefix` (String) - S3 objects key prefix.
