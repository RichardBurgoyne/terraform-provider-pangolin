package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAPIKeyResource_Schema(t *testing.T) {
	r := NewAPIKeyResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "api_key_id", "name", "api_key", "action_ids"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["api_key"].IsSensitive() {
		t.Error("expected api_key to be sensitive")
	}
}
