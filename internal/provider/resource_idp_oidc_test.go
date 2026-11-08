package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

func TestStoredScopes(t *testing.T) {
	for _, tc := range []struct {
		scopes, variant string
		revalidation    bool
		want            string
	}{
		{"openid profile", "oidc", false, "openid profile"},
		{"openid profile", "oidc", true, "openid profile offline_access"},
		{"openid  offline_access", "azure", true, "openid offline_access"},
		{"openid offline_access email", "google", true, "openid email"},
	} {
		if got := storedScopes(tc.scopes, tc.revalidation, tc.variant); got != tc.want {
			t.Errorf("storedScopes(%q, %v, %q) = %q, want %q", tc.scopes, tc.revalidation, tc.variant, got, tc.want)
		}
	}
}

func TestIdpOidcModel_FromAPI(t *testing.T) {
	empty := ""
	model := idpOidcModel{Scopes: types.StringValue("openid email")}
	model.fromAPI(&client.OidcIdp{IdpID: 4, Variant: "oidc", Scopes: "openid email offline_access", SessionRevalidation: true, EmailPath: &empty})
	// The offline_access Pangolin adds must not show up as drift.
	if model.Scopes.ValueString() != "openid email" {
		t.Errorf("scopes = %s", model.Scopes)
	}
	// A cleared optional field is stored as "", which must read as null.
	if !model.EmailPath.IsNull() || !model.NamePath.IsNull() || !model.Tags.IsNull() {
		t.Errorf("expected cleared fields to be null: %s %s %s", model.EmailPath, model.NamePath, model.Tags)
	}

	model.fromAPI(&client.OidcIdp{IdpID: 4, Variant: "oidc", Scopes: "openid", SessionRevalidation: false})
	if model.Scopes.ValueString() != "openid" {
		t.Errorf("server change not picked up: %s", model.Scopes)
	}
}
