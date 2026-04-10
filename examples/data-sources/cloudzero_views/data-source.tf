data "cloudzero_views" "all" {}

output "view_names" {
  value = [for v in data.cloudzero_views.all.views : v.name]
}
