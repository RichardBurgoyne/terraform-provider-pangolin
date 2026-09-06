package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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
