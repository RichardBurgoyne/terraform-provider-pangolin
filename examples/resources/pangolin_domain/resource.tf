resource "pangolin_domain" "example" {
  org_id      = pangolin_organization.example.org_id
  type        = "wildcard"
  base_domain = "apps.example.com"
}
