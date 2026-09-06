package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestPangolinResourceResource_Schema(t *testing.T) {
	r := NewPangolinResourceResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "resource_id", "name", "mode", "domain_id", "full_domain"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}

// Regression test: post_auth_path, pam_mode and auth_daemon_mode are written
// unconditionally from the API response, which returns "" when the field is
// absent (e.g. mode = http). If they were Optional without Computed, that ""
// would conflict with a null config value and apply would fail with "Provider
// produced inconsistent result after apply".
func TestPangolinResourceResource_OptionalComputedAttributes(t *testing.T) {
	r := NewPangolinResourceResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"post_auth_path", "pam_mode", "auth_daemon_mode"} {
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
