resource "pangolin_resource" "example" {
  org_id    = pangolin_organization.example.org_id
  name      = "example-app"
  mode      = "http"
  domain_id = pangolin_domain.example.domain_id
  subdomain = "app"

  # Distribute requests by client IP hash (Pangolin 1.25+, Traefik 3.6+).
  load_balancing_strategy = "hrw"

  # Evaluate pangolin_resource_rule rules for this resource.
  apply_rules = true

  # Headers added to the request sent to the backend...
  headers_json = jsonencode([
    { name = "X-Forwarded-Team", value = "platform" },
  ])

  # ...and to the response sent back to the client (Pangolin 1.24+).
  response_headers_json = jsonencode([
    { name = "X-Frame-Options", value = "DENY" },
    { name = "Referrer-Policy", value = "same-origin" },
  ])
}
