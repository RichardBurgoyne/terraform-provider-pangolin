package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAIProviderResource_Schema(t *testing.T) {
	r := NewAIProviderResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{
		"org_id", "provider_id", "nice_id", "name", "type", "upstream_url",
		"api_key", "auth_type", "routing_mode", "capabilities", "headers",
		"skip_tls_verification", "enabled",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}

	if !resp.Schema.Attributes["api_key"].IsSensitive() {
		t.Error("expected api_key to be sensitive")
	}
	for _, name := range []string{"org_id", "name", "type"} {
		if !resp.Schema.Attributes[name].IsRequired() {
			t.Errorf("expected %q to be required", name)
		}
	}
}
