---
page_title: "ssm Provider"
subcategory: ""
description: |-
  Terraform provider for AWS Systems Manager service
---

# SSM Provider

The provider provides resources for AWS Systems Manager service not supported by "hashicorp/aws" provider.

## Example Usage

Do not keep your authentication password in HCL for production environments, use Terraform environment variables.

```terraform
 terraform {
  required_providers {
    ssm = {
      source  = "pbobov/ssm"
      version = "~> 0.2"
    }
  }
```

## Authentication and Configuration

When using the provider you'll generally need your AWS credentials to authenticate
with AWS services. The provider supports multiple methods of supporting these
credentials. By default the provider will source credentials automatically from
its default credential chain. The common items in the credential
chain are the following:

* Environment Credentials - Set of environment variables that are useful
  when sub processes are created for specific roles.

* Shared Credentials file (~/.aws/credentials) - This file stores your
  credentials based on a profile name and is useful for local development.

* EC2 Instance Role Credentials - Use EC2 Instance Role to assign credentials
  to application running on an EC2 instance. This removes the need to manage
  credential files in production.

This order matches the precedence used by "hashicorp/aws" provider,
by the AWS CLI, and the AWS SDKs.  

In addition to the credentials you'll need to specify the region the provider
will use to make AWS API requests to. To set the region via the environment
variable set the "AWS_REGION" to the region you want to the provider to use.

```bash
AWS_REGION=us-west-2
```
