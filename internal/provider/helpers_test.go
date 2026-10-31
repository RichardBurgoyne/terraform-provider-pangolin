package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// validateConfig runs r's ValidateConfig against model, with any attribute not
// set on model as null, and returns the resulting diagnostics.
func validateConfig(t *testing.T, r resource.ResourceWithValidateConfig, model any) diag.Diagnostics {
	t.Helper()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	var value attr.Value
	diags := tfsdk.ValueFrom(ctx, model, schemaResp.Schema.Type(), &value)
	if diags.HasError() {
		t.Fatalf("building test config: %v", diags)
	}
	tfValue, err := value.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("converting test config to terraform value: %v", err)
	}

	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Raw: tfValue, Schema: schemaResp.Schema}}, resp)
	return resp.Diagnostics
}

func TestSplitImportID(t *testing.T) {
	var diags diag.Diagnostics
	first, second, ok := splitImportID("acme:5", "org_id", "site_id", "acme:5", &diags)
	if !ok || first != "acme" || second != "5" {
		t.Errorf("got %q, %q, %v", first, second, ok)
	}

	for _, id := range []string{"acme", ":5", "acme:", ""} {
		var diags diag.Diagnostics
		if _, _, ok := splitImportID(id, "org_id", "site_id", "acme:5", &diags); ok || !diags.HasError() {
			t.Errorf("expected %q to be rejected", id)
		}
	}
}

// Regression test: a Computed attribute that forces replacement must also keep
// its state value. Otherwise an update to any other attribute marks it
// unknown, and Terraform destroys the object to change, for example, its name.
func TestComputedReplaceAttributesKeepState(t *testing.T) {
	ctx := context.Background()
	for _, newResource := range New("test")().Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pangolin"}, &meta)
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)

		for name, attribute := range resp.Schema.Attributes {
			if !attribute.IsComputed() {
				continue
			}
			var descriptions []string
			switch a := attribute.(type) {
			case schema.StringAttribute:
				for _, m := range a.PlanModifiers {
					descriptions = append(descriptions, m.Description(ctx))
				}
			case schema.Int64Attribute:
				for _, m := range a.PlanModifiers {
					descriptions = append(descriptions, m.Description(ctx))
				}
			}
			joined := strings.Join(descriptions, "\n")
			if strings.Contains(joined, "destroy and recreate") && !strings.Contains(joined, "will not change") {
				t.Errorf("%s.%s forces replacement but doesn't keep its state value: %v", meta.TypeName, name, descriptions)
			}
		}
	}
}
