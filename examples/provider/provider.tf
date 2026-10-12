terraform {
  required_providers {
    pangolin = {
      source = "RichardBurgoyne/pangolin"
    }
  }
}

# Credentials can also come from PANGOLIN_ENDPOINT and PANGOLIN_API_KEY.
provider "pangolin" {
  endpoint = "https://api.pangolin.example.com/v1"
  api_key  = var.pangolin_api_key
}

variable "pangolin_api_key" {
  type      = string
  sensitive = true
}
