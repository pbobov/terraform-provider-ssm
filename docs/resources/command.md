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

> Note: The SSM commands invoked by the resource must be idempotent to conform with terraform's declarative design.

## Example Usage

```terraform
resource "ssm_command" "test" {
  document_name = "AWS-InstallMissingWindowsUpdates"
  parameters {
    name   = "UpdateLevel"
    values = ["Important"]
  }
  targets {
    key    = "InstanceIds"
    values = ["i-xxxxxxxxxxxx"]
  }
  comment           = "Install Microsoft Windows updates."
  execution_timeout = 600
  output_location {
    s3_bucket_name = "testbucket"
    s3_key_prefix  = "updates"
  }
}
```

## Schema

### Required

- `document_name` (String) - The name of the SSM document to apply.
- `output_location` (Block) - An output location block. Output Location is documented below.
- `parameters` (Block List) - A block of arbitrary string parameters to pass to the SSM document.
- `targets` (Block List) - A block containing the targets of the SSM command invocations. Targets are documented below.

### Optional

- `comment` (String) - The user-specified information about the command, such as a brief description of what the command should do.
- `execution_timeout` (Number) - The command invocation timeout in seconds. Default timeout is 3600 seconds.

### Read-Only

- `id` (String) The SSM command Id.
- `requested_time` (String) - The date and time the command was requested.
- `status` (String) - The status of the SSM command.

### Nested Schema for `parameters`

Required:

- `name` (String) - The SSM document parameter name.
- `values` (List of String) - A list of parameter values.

### Nested Schema for `targets`

Targets specify what instance IDs or tags to apply the document to and has these keys:

- `key` (String) - Either InstanceIds or tag:Tag Name to specify an EC2 tag.
- `values` (List of String) - A list of instance IDs or tag values.

### Nested Schema for `output_location`

Required:

- `s3_bucket_name` (String) - The S3 bucket name.

Optional:

- `s3_key_prefix` (String) - The S3 bucket prefix. Results stored in the root if not configured.
