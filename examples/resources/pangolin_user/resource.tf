resource "pangolin_user" "example" {
  org_id   = pangolin_organization.example.org_id
  username = "jane"
  email    = "jane@example.com"
  idp_id   = 1
  role_ids = [pangolin_role.example.role_id]
}
