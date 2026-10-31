# Send a path on an existing resource to another host.
resource "pangolin_redirect" "docs" {
  org_id            = pangolin_organization.example.org_id
  name              = "docs"
  resource_id       = pangolin_resource.example.resource_id
  destination_host  = "https://docs.example.org"
  path_match_type   = "prefix"
  match_path        = "/docs"
  rewrite_path_type = "stripPrefix"
}

# Redirect a whole domain.
resource "pangolin_redirect" "old_site" {
  org_id           = pangolin_organization.example.org_id
  name             = "old-site"
  domain_id        = pangolin_domain.example.domain_id
  subdomain        = "old"
  destination_host = "https://www.example.org"
}
