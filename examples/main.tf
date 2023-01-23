terraform {
  required_providers {
    ssm = {
      source  = "local/providers/ssm"
      version = "0.1.1"
    }
  }
}

resource "ssm_command" "test" {
  document_name = "AWS-InstallMissingWindowsUpdates"
  parameters {
    name   = "UpdateLevel"
    values = ["Important"]
  }
  targets {
    key    = "InstanceIds"
    values = ["i-xxxxxxxxxxxxxxxxx"]
  }
  comment           = "Install Microsoft Windows updates."
  execution_timeout = 600
  output_location {
    s3_bucket_name = "mybycket"
    s3_key_prefix  = "updates"
  }
}
