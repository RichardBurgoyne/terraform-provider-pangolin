package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

func TestSiteResource_Schema(t *testing.T) {
	r := NewSiteResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "site_id", "name", "type", "newt_id", "secret"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["secret"].IsSensitive() {
		t.Error("expected secret to be sensitive")
	}
}

// Regression test: pub_key and subnet are written unconditionally from the API
// response, which returns "" when the field is absent (e.g. type = newt). If
// they were Optional without Computed, that "" would conflict with a null
// config value and apply would fail with "Provider produced inconsistent
// result after apply".
func TestSiteResource_OptionalComputedAttributes(t *testing.T) {
	r := NewSiteResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"pub_key", "subnet"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected attribute %q", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q to be optional", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q to be computed (it is set unconditionally from the API response)", name)
		}
	}
}

// Regression test: for a site with no Newt agent (type = local or
// wireguard), the API returns newtId/secret as empty strings. Create starts
// from a plan where these Computed-only attributes are Unknown; if
// setSiteModelFromAPI left them untouched in that case, Terraform errored
// with "Provider returned invalid result object after apply" because an
// unknown value is never valid in a post-apply result.
func TestSetSiteModelFromAPI_NoAgentLeavesKnownNull(t *testing.T) {
	model := siteResourceModel{
		NewtID: types.StringUnknown(),
		Secret: types.StringUnknown(),
	}
	site := &client.Site{
		SiteID: 1,
		Name:   "Local",
		Type:   "local",
		// NewtID and Secret intentionally left as zero-value "" (no agent).
	}

	setSiteModelFromAPI(&model, site)

	if model.NewtID.IsUnknown() {
		t.Error("expected newt_id to be known (null) after setSiteModelFromAPI, got unknown")
	}
	if !model.NewtID.IsNull() {
		t.Errorf("expected newt_id to be null when the API returns no agent, got %q", model.NewtID.ValueString())
	}
	if model.Secret.IsUnknown() {
		t.Error("expected secret to be known (null) after setSiteModelFromAPI, got unknown")
	}
	if !model.Secret.IsNull() {
		t.Errorf("expected secret to be null when the API returns no agent, got %q", model.Secret.ValueString())
	}
}

// Regression test: for a newt-type site, the API does return newtId/secret,
// and those real values must be preserved.
func TestSetSiteModelFromAPI_WithAgentSetsValues(t *testing.T) {
	model := siteResourceModel{
		NewtID: types.StringUnknown(),
		Secret: types.StringUnknown(),
	}
	site := &client.Site{
		SiteID: 2,
		Name:   "Media-SVR",
		Type:   "newt",
		NewtID: "newt-123",
		Secret: "super-secret",
	}

	setSiteModelFromAPI(&model, site)

	if model.NewtID.ValueString() != "newt-123" {
		t.Errorf("expected newt_id %q, got %q", "newt-123", model.NewtID.ValueString())
	}
	if model.Secret.ValueString() != "super-secret" {
		t.Errorf("expected secret %q, got %q", "super-secret", model.Secret.ValueString())
	}
}
