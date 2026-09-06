resource "pangolin_user_resource_grant" "example" {
  resource_id = pangolin_resource.example.resource_id
  user_id     = pangolin_user.example.user_id
}
