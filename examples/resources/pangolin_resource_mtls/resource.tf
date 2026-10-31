resource "pangolin_resource_mtls" "example" {
  resource_id        = pangolin_resource.example.resource_id
  ca_certificate_pem = file("${path.module}/client-ca.pem")
}
