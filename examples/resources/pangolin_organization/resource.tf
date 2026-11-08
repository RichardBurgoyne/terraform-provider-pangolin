resource "pangolin_organization" "example" {
  org_id         = "acme"
  name           = "Acme Corp"
  subnet         = "10.10.0.0/24"
  utility_subnet = "10.10.1.0/24"

  # Optional settings. Any left out keep their current value in Pangolin.
  require_two_factor         = true
  log_retention_days_request = 30
  log_retention_days_ssh     = 90 # Pangolin 1.26+, Enterprise
}
