# ssm\_command Resource Example

Terraform module demonstrated ssm\_command resource from pbobov/ssm Terraform provider.

The module:
* Creates S3 bucket for SSM commands stdout and stderr outputs,
* Launches SSM managed EC2 instance from Amazon Linux 2 AMI,
* Creates ssm\_command resource that runs simple shell scripts on the EC2 instance
  when the resource gets created and destroyed,
* Adds tags to the EC2 instance if the ssm\_command resource creation succeeds.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.52.0 |
| <a name="provider_ssm"></a> [ssm](#provider\_ssm) | 0.2.0 |

## Resources

| Name | Type |
|------|------|
| [aws_ec2_tag.greeting_ack](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ec2_tag) | resource |
| [aws_iam_instance_profile.ssm_managed_instance](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_instance_profile) | resource |
| [aws_iam_policy.ssm_output_policy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.ssm_managed](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_instance.world](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/instance) | resource |
| [aws_s3_bucket.output](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
| [ssm_command.greeting](https://registry.terraform.io/providers/pbobov/ssm/latest/docs/resources/command) | resource |
| [aws_ssm_parameter.ami](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/ssm_parameter) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_subnet_id"></a> [subnet\_id](#input\_subnet\_id) | VPC subnet Id | `string` | n/a | yes |

## Outputs

No outputs.
