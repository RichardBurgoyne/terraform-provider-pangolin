# Mail for a domain delegated to Pangolin DNS (Pangolin 1.26+, Enterprise).
resource "pangolin_custom_dns_record" "mx" {
  org_id    = pangolin_organization.example.org_id
  domain_id = pangolin_domain.example.domain_id
  name      = "@"
  type      = "MX"
  values    = ["mx1.mail.example.net", "mx2.mail.example.net"]
  priority  = 10
}

resource "pangolin_custom_dns_record" "spf" {
  org_id    = pangolin_organization.example.org_id
  domain_id = pangolin_domain.example.domain_id
  name      = "@"
  type      = "TXT"
  values    = ["v=spf1 include:mail.example.net -all"]
  ttl       = 3600
}
