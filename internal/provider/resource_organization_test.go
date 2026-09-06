package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestOrganizationResource_Schema(t *testing.T) {
	r := NewOrganizationResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "name", "subnet", "utility_subnet"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("expected attribute %q to be required", name)
		}
	}
}
