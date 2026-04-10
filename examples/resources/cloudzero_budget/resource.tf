resource "cloudzero_budget" "monthly" {
  name    = "Engineering Monthly Budget"
  view_id = cloudzero_view.engineering.id

  planned_limits = {
    "2025-01-01T00:00:00+00:00" = "50000"
    "2025-02-01T00:00:00+00:00" = "50000"
    "2025-03-01T00:00:00+00:00" = "55000"
  }

  alerts = {
    "80"  = "N"
    "100" = "Y"
  }
}
