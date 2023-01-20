terraform {
  required_providers {
    ssm = {
      source  = "local/providers/ssm"
      version = "1.0.0"
      # Other parameters...
    }
  }
}

resource "ssm_command" "test" {
  document_name = "arcgis-enterprise-bootstrap"
  parameters {
    name   = "SiteId"
    values = ["arcgis-enterprise"]
  }
  targets {
    key    = "tag:ArcGISSiteId"
    values = ["arcgis-enterprise"]
  }
  targets {
    key    = "tag:ArcGISDeploymentId"
    values = ["arcgis-enterprise-base"]
  }
  targets {
    key    = "tag:ArcGISMachineRole"
    values = ["primary", "standby"]
  }

  comment           = "Runs Chef client with a specific role JSON file. edited"
  execution_timeout = 600

  output_location {
    s3_bucket_name = "arcgis-enterprise-685115441969-us-west-2"
    s3_key_prefix  = "arcgis-enterprise-base"
  }
}

