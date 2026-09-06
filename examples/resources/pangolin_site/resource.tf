resource "pangolin_site" "example" {
  org_id = pangolin_organization.example.org_id
  name   = "example-site"
  type   = "newt"
}

output "example_site_newt_credentials" {
  value     = { newt_id = pangolin_site.example.newt_id, secret = pangolin_site.example.secret }
  sensitive = true
}
