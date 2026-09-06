package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestNew(t *testing.T) {
	p := New("test")()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestSchema_HasRequiredAttributes(t *testing.T) {
	p := New("test")()
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	for _, name := range []string{"endpoint", "api_key", "org_id"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected schema to have attribute %q", name)
		}
	}
}
