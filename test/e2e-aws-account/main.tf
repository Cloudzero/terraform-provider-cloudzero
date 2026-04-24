terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
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

# Create IAM role + policies using the cloudzero-aws module
module "cloudzero" {
  source = "github.com/Cloudzero/provision-account//terraform/cloudzero-aws"

  external_id = var.external_id

  tags = {
    ManagedBy = "terraform"
    Purpose   = "cloudzero-provider-e2e-test"
  }
}

# Register the account with CloudZero
resource "cloudzero_aws_account" "this" {
  cloud_account_id = module.cloudzero.account_id
  external_id      = var.external_id
  role_arn         = module.cloudzero.role_arn
  account_name     = "terraform-provider-e2e-test"
}

output "role_arn" {
  value = module.cloudzero.role_arn
}

output "account_id" {
  value = module.cloudzero.account_id
}

output "transaction_id" {
  value = cloudzero_aws_account.this.transaction_id
}
