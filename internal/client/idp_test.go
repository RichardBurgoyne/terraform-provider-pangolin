package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOidcIdpLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/idp/oidc":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			// The create schema is strict and has no mapping fields.
			if _, sent := body["defaultRoleMapping"]; sent {
				t.Error("defaultRoleMapping must not be sent on create")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"idpId": float64(4), "redirectUrl": "https://pangolin.example.com/auth/idp/4/oidc/callback"}, "success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/idp/4":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"idp":           map[string]any{"idpId": float64(4), "name": "Corp", "type": "oidc", "defaultRoleMapping": "'Member'", "defaultOrgMapping": nil, "autoProvision": true, "sessionRevalidation": true, "tags": nil},
				"idpOidcConfig": map[string]any{"variant": "oidc", "clientId": "id", "clientSecret": "secret", "authUrl": "https://idp/auth", "tokenUrl": "https://idp/token", "identifierPath": "sub", "emailPath": "email", "namePath": nil, "scopes": "openid offline_access"},
			}, "success": true})
		case r.Method == http.MethodPost && r.URL.Path == "/idp/4/oidc":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"idpId": float64(4)}, "success": true})
		case r.Method == http.MethodDelete && r.URL.Path == "/idp/4":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	role := "'Admin'"
	in := OidcIdpRequest{Name: "Corp", ClientID: "id", ClientSecret: "secret", AuthURL: "https://idp/auth", TokenURL: "https://idp/token", IdentifierPath: "sub", Scopes: "openid", DefaultRoleMapping: &role}
	id, redirectURL, err := c.CreateOidcIdp(ctx, in)
	if err != nil || id != 4 || redirectURL == "" {
		t.Fatalf("create: %d %q %v", id, redirectURL, err)
	}
	got, err := c.GetOidcIdp(ctx, 4)
	if err != nil || got.ClientSecret != "secret" || !got.SessionRevalidation || got.DefaultOrgMapping != nil || got.NamePath != nil || *got.EmailPath != "email" {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if err := c.UpdateOidcIdp(ctx, 4, in); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteIdp(ctx, 4); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
