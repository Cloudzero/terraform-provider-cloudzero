# Changelog

## 0.1.1 (Unreleased)

FEATURES:

* **New Resource:** `cloudzero_monitor`: manage CloudZero anomaly monitors with filters, grouping, cost type, significance threshold, and email/Slack notifications. System monitors created by CloudZero are not supported.

BUG FIXES:

* provider: stop resending POST and PATCH requests after a network error. A request that timed out after the server applied it could otherwise create a duplicate resource.

* resource/cloudzero_aws_account: fix `Update` not setting `transaction_id` from the registration response, which caused "Provider produced inconsistent result after apply" on every in-place update even though the change had already taken effect ([#6](https://github.com/Cloudzero/terraform-provider-cloudzero/issues/6))

## 0.1.0 (Unreleased)

FEATURES:

* **New Resource:** `cloudzero_view` — manage CloudZero Views with filters, connections (email/Slack), and anomaly detection configuration
* **New Resource:** `cloudzero_budget` — manage CloudZero Budgets with monthly planned limits and alert thresholds
* **New Resource:** `cloudzero_insight` — manage CloudZero Insights with status, effort, cost impact tracking
* **New Resource:** `cloudzero_aws_account` — register AWS accounts with CloudZero via cross-account IAM role
* **New Data Source:** `cloudzero_views` — list all Views
* **New Data Source:** `cloudzero_budgets` — list all Budgets
