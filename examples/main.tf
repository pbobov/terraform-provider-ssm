/**
 * # ssm_command Resource Example
 *
 * Terraform module demonstrated ssm_command resource from pbobov/ssm Terraform provider.
 * 
 * The module: 
 * * Creates S3 bucket for SSM commands stdout and stderr outputs, 
 * * Launches SSM managed EC2 instance from Amazon Linux 2 AMI, 
 * * Creates ssm_command resource that runs simple shell scripts on the EC2 instance 
 *   when the resource gets created and destroyed,
 * * Adds tags to the EC2 instance if the ssm_command resource creation succeeds. 
 */

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    ssm = {
      source  = "pbobov/ssm"
      #source  = "local/providers/ssm"
      version = "~> 0.2"
    }
  }
}

variable "subnet_id" {
  type        = string
  description = "VPC subnet Id"
}

# Amazon Linux 2 AMI Id
data "aws_ssm_parameter" "ami" {
  name = "/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2"
}

# S3 bucket for SSM commands stdout and stderr outputs
resource "aws_s3_bucket" "output" {
  bucket_prefix = "output"
  force_destroy = true
}

# IAM policy that allows PutObject on the output S3 bucket
resource "aws_iam_policy" "ssm_output_policy" {
  name_prefix = "SSMCommandPolicy"
  description = "Allow write access to the output S3 bucket"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = ["s3:PutObject"]
        Effect = "Allow"
        Resource = [
          "${aws_s3_bucket.output.arn}/*"
        ]
      }
    ]
  })
}

# IAM role for SSM managed EC2 instances with access to the output S3 bucket
resource "aws_iam_role" "ssm_managed" {
  name_prefix = "SSMCommandRole"
  description = "Permissions required for managed SSM instances"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = ["ec2.amazonaws.com", "ssm.amazonaws.com"]
        }
      }
    ]
  })
  managed_policy_arns = [
    aws_iam_policy.ssm_output_policy.arn,
    "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
  ]
}

# IAM instance profile for SSM managed EC2 instnces
resource "aws_iam_instance_profile" "ssm_managed_instance" {
  name = "SSMManagedInstance"
  role = aws_iam_role.ssm_managed.name
}

# SSM managed EC2 instance
resource "aws_instance" "world" {
  ami                  = nonsensitive(data.aws_ssm_parameter.ami.value)
  instance_type        = "t3.micro"
  iam_instance_profile = aws_iam_instance_profile.ssm_managed_instance.name
  subnet_id            = var.subnet_id
  tags = {
    Name = "TheWorld"
  }
}

# Run AWS-RunShellScript in aws_instance.world EC2 instance 
# when ssm_command.greeting resource is creaded and destroyed
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

# Add GreetingAck tag to aws_instance.world EC2 instance 
# if ssm_command.greeting resource creation succeeded
resource "aws_ec2_tag" "greeting_ack" {
  resource_id = aws_instance.world.id
  key         = "GreetingAck"
  value       = ssm_command.greeting.requested_time
}
