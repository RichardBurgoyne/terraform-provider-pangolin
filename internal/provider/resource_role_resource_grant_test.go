package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestRoleResourceGrantResource_Schema(t *testing.T) {
	r := NewRoleResourceGrantResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"resource_id", "role_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok || !attr.IsRequired() {
			t.Errorf("expected required attribute %q", name)
		}
	}
}
