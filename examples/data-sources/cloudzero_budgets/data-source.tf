data "cloudzero_budgets" "all" {}

output "budget_names" {
  value = [for b in data.cloudzero_budgets.all.budgets : b.name]
}
