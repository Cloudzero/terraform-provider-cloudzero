terraform {
  required_providers {
    cloudzero = {
      source  = "cloudzero/cloudzero"
      version = "~> 0.1"
    }
  }
}

provider "cloudzero" {
  # api_key = "..."  # Or set CLOUDZERO_API_KEY env var
}
