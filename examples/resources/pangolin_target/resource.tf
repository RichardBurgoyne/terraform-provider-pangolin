resource "pangolin_target" "example" {
  resource_id = pangolin_resource.example.resource_id
  site_id     = pangolin_site.example.site_id
  ip          = "10.0.0.5"
  port        = 8080
}
