resource "cloudzero_monitor" "s3_by_account" {
  name        = "S3 spend by account"
  description = "Alerts the platform team when S3 spend in any account jumps."

  filter = {
    "services" = ["AmazonS3"]
  }
  group_by = ["accounts"]

  threshold {
    min_cost_impact = 100
    percentage      = 20
    operator        = "AND"
  }

  notifications {
    email            = ["platform-team@example.com"]
    slack_channel_id = "C0123ABCD9"
  }
}
