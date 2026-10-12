# A database reachable by name through a site.
resource "pangolin_private_resource" "db" {
  org_id         = pangolin_organization.example.org_id
  name           = "postgres"
  mode           = "host"
  site_ids       = [pangolin_site.example.site_id]
  destination    = "10.0.0.20"
  alias          = "db.internal"
  tcp_port_range = "5432"
  role_ids       = [pangolin_role.example.role_id]
}

# A whole network range.
resource "pangolin_private_resource" "lan" {
  org_id      = pangolin_organization.example.org_id
  name        = "office-lan"
  mode        = "cidr"
  site_ids    = [pangolin_site.example.site_id]
  destination = "192.168.10.0/24"
}
