terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    cloudzero = {
      source = "cloudzero/cloudzero"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

provider "cloudzero" {}

variable "external_id" {
  type      = string
  sensitive = true
}

data "aws_caller_identity" "current" {}

# IAM role for CloudZero cross-account access
resource "aws_iam_role" "cloudzero" {
  name = "cloudzero-access"
  path = "/"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Action    = "sts:AssumeRole"
        Principal = { AWS = "arn:aws:iam::061190967865:root" }
        Condition = {
          StringEquals = { "sts:ExternalId" = var.external_id }
        }
      }
    ]
  })

  tags = {
    ManagedBy = "terraform"
    Purpose   = "cloudzero-provider-test"
  }
}

# Register with CloudZero
resource "cloudzero_aws_account" "this" {
  cloud_account_id = data.aws_caller_identity.current.account_id
  external_id      = var.external_id
  role_arn         = aws_iam_role.cloudzero.arn
  account_name     = "terraform-provider-e2e-test"
}

output "role_arn" {
  value = aws_iam_role.cloudzero.arn
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}

output "transaction_id" {
  value = cloudzero_aws_account.this.transaction_id
}
