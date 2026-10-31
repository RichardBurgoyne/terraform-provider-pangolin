resource "pangolin_client_role" "ci_runner" {
  client_id = 5
  role_id   = pangolin_role.example.role_id
}
