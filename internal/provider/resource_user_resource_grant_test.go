package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestUserResourceGrantResource_Schema(t *testing.T) {
	r := NewUserResourceGrantResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"resource_id", "user_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok || !attr.IsRequired() {
			t.Errorf("expected required attribute %q", name)
		}
	}
}
