# Block destructive HTTP methods from outside the office network.
resource "pangolin_resource_rule" "no_writes" {
  resource_id = pangolin_resource.example.resource_id
  action      = "DROP"
  match       = "METHOD"
  value       = "POST,PUT,PATCH,DELETE"
  priority    = 10
}

# Let the office network through first.
resource "pangolin_resource_rule" "office" {
  resource_id = pangolin_resource.example.resource_id
  action      = "ACCEPT"
  match       = "CIDR"
  value       = "203.0.113.0/24"
  priority    = 1
}
