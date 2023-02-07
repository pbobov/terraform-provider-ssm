[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/pbobov/terraform-provider-ssm?display_name=tag&sort=semver)](https://github.com/pbobov/terraform-provider-ssm/releases/latest)
[![CodeQL](https://github.com/pbobov/terraform-provider-ssm/actions/workflows/codeql.yml/badge.svg)](https://github.com/pbobov/terraform-provider-ssm/actions/workflows/codeql.yml)

# SSM Terraform Provider

SSM terraform provider resources support AWS Systems Manager service functionality not supported by "hashicorp/aws" provider. So far it provides single ssm_command resource.

## Using ssm_command resource

> ssm_command resource is an alternative to using remote-exec provisioner. The resource addresses various problems related to using provisioners in general and remote-exec provisioner in particular.

ssm_command resource sends command to SSM managed EC2 instances and waits for the command invocations to be completed on all the target instances. Before sending the command, the resource waits for all the target EC2 instances to be online as SSM managed instances. After all the command invocations are completed, the resource retrieves the invocations outputs from the output S3 bucket and logs them to the terraform log as an INFO level message.

If destroy_document_name is specified, the SSM document is executed when the resource is destroyed with parameters specified by destroy_parameters attributes.

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

See [ssm_command example](examples/ssm_command/main.tf) for the complete Terraform module.

Unlike most of terraform resources that declare pieces of infrastructure or their relationships, ssm_command resource declares operations. The operations are applied to EC2 instances to change their states. To work well with declarative Terraform, the operations executed by ssm_command resource must meet certain requirements:

*	To maintain integrity of the system state, the operations must be atomic - either the entire operation is executed, or none of it is executed.
*	To prevent state drift caused by repeated executions, the operations must be idempotent - no matter how many times operation executed; it produces the same result.

In addition, to annotate the new state of the target EC2 instances after commands are completed, it's a good practice to tags the target instances with the new state declaration. For example, if the SSM command installs the latest patches, the operation is not strictly idempotent, but assuming that time is an implicit parameter of the command and the target EC2 instances should be tagged with the timestamp of the latest execution.  

A typical use-case for ssm_command resource is installing software on EC2 instances. The software is installed when ssm_command resource is created, updated/upgraded when the resource is updated, and uninstalled when the resource is destroyed (though typically the EC2 instances are deleted immediately after that).

## Build Provider

To build the provider locally for windows platform run:

```shell
$ go build -o .\examples\terraform.d\plugins\local\providers\ssm\0.2.0\windows_amd64\terraform-provider-ssm.exe
```

## Test the Locally Built Provider

To use the locally built provider set attribute "source" of ssm provider to "local/providers/ssm":

```terraform
terraform {
  required_providers {
    ssm = {
      source  = "local/providers/ssm"
      version = "0.2.0"
    }
  }
}
```
