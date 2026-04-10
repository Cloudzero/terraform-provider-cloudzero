resource "cloudzero_insight" "rds_optimization" {
  title       = "RDS Optimization"
  description = "Consider switching from Optimized to General Purpose volumes for underutilized RDS instances."
  category    = "Cost Optimization"
  status      = "new"
  effort      = "medium"
  cost_impact = "2000"
  link        = "https://app.cloudzero.com/explorer?services=AmazonRDS"
}
