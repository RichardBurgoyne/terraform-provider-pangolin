resource "pangolin_idp_oidc" "example" {
  name            = "Company SSO"
  client_id       = var.oidc_client_id
  client_secret   = var.oidc_client_secret
  auth_url        = "https://login.example.com/oauth2/authorize"
  token_url       = "https://login.example.com/oauth2/token"
  scopes          = "openid profile email"
  identifier_path = "sub"
  email_path      = "email"
  name_path       = "name"

  # Sign users out once the identity provider stops accepting their session
  # (Pangolin 1.26+, Enterprise).
  session_revalidation = true

  auto_provision       = true
  default_role_mapping = "'Member'"
}

# Register this as the callback URL with the identity provider.
output "oidc_redirect_url" {
  value = pangolin_idp_oidc.example.redirect_url
}
