package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestTargetResource_Schema(t *testing.T) {
	r := NewTargetResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"resource_id", "target_id", "site_id", "ip", "port"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected attribute %q", name)
			continue
		}
		if name != "target_id" && !attr.IsRequired() {
			t.Errorf("expected attribute %q to be required", name)
		}
	}
}
