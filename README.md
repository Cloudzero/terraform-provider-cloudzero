# Terraform Provider for CloudZero

Manage [CloudZero](https://www.cloudzero.com) resources as code. Create and manage Views, Budgets, Insights, and AWS account connections using Terraform.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.3
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider)

## Authentication

The provider authenticates using a CloudZero API key. Generate one at **Organization > API Keys** in the [CloudZero app](https://app.cloudzero.com/organization/api-keys).

Set it via environment variable (recommended):

```bash
export CLOUDZERO_API_KEY="your-api-key"
```

Or in the provider configuration:

```hcl
provider "cloudzero" {
  api_key = "your-api-key"
}
```

## Usage

```hcl
terraform {
  required_providers {
    cloudzero = {
      source  = "cloudzero/cloudzero"
      version = "~> 0.1"
    }
  }
}

provider "cloudzero" {}
```

### Create a View

```hcl
resource "cloudzero_view" "engineering" {
  name                = "Engineering"
  principal_dimension = "CZ:Account"

  filter = {
    "CZ:Account" = ["123456789012", "987654321098"]
  }

  connections {
    email {
      addresses              = ["team@example.com"]
      include_all_organizers = false
    }
  }

  anomalies {
    enabled        = true
    threshold_type = "Percent"
    threshold_value = 20
  }
}
```

### Create a Budget

```hcl
resource "cloudzero_budget" "monthly" {
  name    = "Engineering Monthly Budget"
  view_id = cloudzero_view.engineering.id

  planned_limits = {
    "2025-01-01T00:00:00+00:00" = "50000"
    "2025-02-01T00:00:00+00:00" = "50000"
  }

  alerts = {
    "80"  = "N"
    "100" = "Y"
  }
}
```

### Create an Insight

```hcl
resource "cloudzero_insight" "rds_optimization" {
  title       = "RDS Optimization"
  description = "Switch from Optimized to General Purpose volumes for underutilized RDS instances."
  category    = "Cost Optimization"
  status      = "new"
  effort      = "medium"
  cost_impact = "2000"
}
```

### Connect an AWS Account

```hcl
module "cloudzero" {
  source = "github.com/Cloudzero/provision-account//terraform/cloudzero-aws"

  external_id = var.cloudzero_external_id
}

resource "cloudzero_aws_account" "production" {
  cloud_account_id = module.cloudzero.account_id
  external_id      = var.cloudzero_external_id
  role_arn         = module.cloudzero.role_arn
  account_name     = "production"
}
```

The `cloudzero_aws_account` resource registers an AWS account with CloudZero after the cross-account IAM role has been created. Use it with the [`cloudzero-aws` module](https://github.com/Cloudzero/provision-account/tree/master/terraform/cloudzero-aws) for the complete onboarding experience — the module creates the IAM role, policies, and optionally the CUR, while this resource handles the registration.

## Resources

| Resource | Description |
|----------|-------------|
| `cloudzero_view` | Manage Views with filters, connections, and anomaly detection |
| `cloudzero_budget` | Manage Budgets with monthly limits and alert thresholds |
| `cloudzero_insight` | Manage Insights with status, effort, and cost impact |
| `cloudzero_aws_account` | Register AWS accounts with CloudZero |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `cloudzero_views` | List all Views |
| `cloudzero_budgets` | List all Budgets |

## Development

### Build

```bash
go build ./...
```

### Test

```bash
# Unit tests
go test ./...

# Acceptance tests (requires CLOUDZERO_API_KEY)
CLOUDZERO_API_KEY="..." CLOUDZERO_TEST_KEY="test" TF_ACC=1 go test -v ./internal/resources/...
```

### Local Development

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "cloudzero/cloudzero" = "/path/to/go/bin"
  }
  direct {}
}
```

Then `go install .` and run `terraform plan` against your configs (skip `terraform init` for the CloudZero provider when using dev overrides).

## License

This project is licensed under the Apache License, Version 2.0 — see the
[LICENSE](./LICENSE) file for details.

## Trademarks

"CloudZero" and the CloudZero logo are trademarks of CloudZero, Inc. Use of
these trademarks is limited to identification and attribution as required by
the Apache License. You may not use CloudZero trademarks in a way that
suggests endorsement or affiliation without written permission.
