resource "pangolin_resource" "example" {
  org_id    = pangolin_organization.example.org_id
  name      = "example-app"
  mode      = "http"
  domain_id = pangolin_domain.example.domain_id
  subdomain = "app"
}
