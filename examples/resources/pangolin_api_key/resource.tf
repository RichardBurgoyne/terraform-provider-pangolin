resource "pangolin_api_key" "ci" {
  org_id     = pangolin_organization.example.org_id
  name       = "ci-deploys"
  action_ids = ["listSites", "createResource"]
}

output "ci_api_key" {
  value     = pangolin_api_key.ci.api_key
  sensitive = true
}
