package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestRoleResource_Schema(t *testing.T) {
	r := NewRoleResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "role_id", "name", "allow_ssh"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}
