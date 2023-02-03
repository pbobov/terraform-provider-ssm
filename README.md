[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/pbobov/terraform-provider-ssm?display_name=tag&sort=semver)](https://github.com/pbobov/terraform-provider-ssm/releases/latest)
[![CodeQL](https://github.com/pbobov/terraform-provider-ssm/actions/workflows/codeql.yml/badge.svg)](https://github.com/pbobov/terraform-provider-ssm/actions/workflows/codeql.yml)

# SSM Terraform Provider

The provider provides resources for AWS Systems Manager service not supported by "hashicorp/aws" provider.

## Build provider

Run the following command to build the provider for windows platform:

```shell
$ go build -o .\examples\terraform.d\plugins\local\providers\ssm\0.1.0\windows_amd64\terraform-provider-ssm.exe
```

## Test the Locally Built Provider

To use the locally built provider, create a test.tf file in ".\examples" directory and set attribute "source" of ssm provider to "local/providers/ssm":

```teraform
terraform {
  required_providers {
    ssm = {
      source  = "local/providers/ssm"
      version = "0.1.0"
    }
  }
}

...
```
