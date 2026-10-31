resource "pangolin_label" "production" {
  org_id = pangolin_organization.example.org_id
  name   = "production"
  color  = "#3b82f6"
}
