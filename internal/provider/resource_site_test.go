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
