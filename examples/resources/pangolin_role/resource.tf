resource "pangolin_role" "example" {
  org_id      = pangolin_organization.example.org_id
  name        = "editors"
  description = "Can edit but not manage billing"
}
