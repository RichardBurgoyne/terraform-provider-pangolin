resource "pangolin_role_resource_grant" "example" {
  resource_id = pangolin_resource.example.resource_id
  role_id     = pangolin_role.example.role_id
}
