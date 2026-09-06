resource "pangolin_organization" "example" {
  org_id         = "acme"
  name           = "Acme Corp"
  subnet         = "10.10.0.0/24"
  utility_subnet = "10.10.1.0/24"
}
