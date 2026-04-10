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
