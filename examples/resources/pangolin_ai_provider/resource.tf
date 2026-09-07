resource "pangolin_ai_provider" "example" {
  org_id  = pangolin_organization.example.org_id
  name    = "OpenAI"
  type    = "openai"
  api_key = var.openai_api_key
}
