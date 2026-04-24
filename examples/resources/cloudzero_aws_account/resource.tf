# Use the cloudzero-aws module to create the IAM role and policies,
# then register the account with CloudZero.

module "cloudzero" {
  source = "github.com/Cloudzero/provision-account//terraform/cloudzero-aws"

  external_id = var.cloudzero_external_id

  # For management/payer accounts, uncomment one of:
  # create_cur      = true
  # cur_bucket_name = "my-existing-cur-bucket"
}

resource "cloudzero_aws_account" "this" {
  cloud_account_id = module.cloudzero.account_id
  external_id      = var.cloudzero_external_id
  role_arn         = module.cloudzero.role_arn
  account_name     = "production"

  # For management accounts with CUR:
  # is_master_payer = true
  # bucket_name     = module.cloudzero.cur_bucket_name
}
