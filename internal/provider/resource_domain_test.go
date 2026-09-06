package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

func TestDomainResource_Schema(t *testing.T) {
	r := NewDomainResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "domain_id", "type", "base_domain", "cert_resolver", "prefer_wildcard_cert", "verified"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}

// TestDomainResource_Update_UsesStateDomainID is a regression test for a real
// bug found via end-to-end testing: domain_id is Computed-only with no plan
// modifier, so the framework marks it Unknown in the proposed plan whenever
// any other attribute changes. Update() must read the ID from state (always
// known), never from plan (frequently Unknown), or it sends an empty ID in
// the request path.
func TestDomainResource_Update_UsesStateDomainID(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"domainId": "dom-1", "certResolver": "letsencrypt", "preferWildcardCert": false},
			"success": true,
		})
	}))
	defer server.Close()

	r := &domainResource{client: client.New(server.URL, "test-token")}

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(context.Background())

	stateValue := tftypes.NewValue(objType, map[string]tftypes.Value{
		"org_id":               tftypes.NewValue(tftypes.String, "acme"),
		"domain_id":            tftypes.NewValue(tftypes.String, "dom-1"),
		"type":                 tftypes.NewValue(tftypes.String, "wildcard"),
		"base_domain":          tftypes.NewValue(tftypes.String, "apps.example.com"),
		"cert_resolver":        tftypes.NewValue(tftypes.String, ""),
		"prefer_wildcard_cert": tftypes.NewValue(tftypes.Bool, false),
		"verified":             tftypes.NewValue(tftypes.Bool, true),
	})

	// The plan mirrors real Terraform core behavior: domain_id and verified
	// (both Computed-only, no plan modifier) are Unknown; cert_resolver is
	// the field actually being changed by this Update call.
	planValue := tftypes.NewValue(objType, map[string]tftypes.Value{
		"org_id":               tftypes.NewValue(tftypes.String, "acme"),
		"domain_id":            tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"type":                 tftypes.NewValue(tftypes.String, "wildcard"),
		"base_domain":          tftypes.NewValue(tftypes.String, "apps.example.com"),
		"cert_resolver":        tftypes.NewValue(tftypes.String, "letsencrypt"),
		"prefer_wildcard_cert": tftypes.NewValue(tftypes.Bool, false),
		"verified":             tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
	})

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Raw: planValue, Schema: schemaResp.Schema},
		State: tfsdk.State{Raw: stateValue, Schema: schemaResp.Schema},
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Raw: stateValue, Schema: schemaResp.Schema}}

	r.Update(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	if gotPath != "/org/acme/domain/dom-1" {
		t.Errorf("expected request to /org/acme/domain/dom-1, got %q (domain_id was likely read from an Unknown plan value instead of state)", gotPath)
	}

	var finalModel domainResourceModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &finalModel)...)
	if finalModel.DomainID.IsUnknown() || finalModel.DomainID.ValueString() != "dom-1" {
		t.Errorf("expected final state domain_id to be the known value \"dom-1\", got %v", finalModel.DomainID)
	}
	if finalModel.Verified.IsUnknown() {
		t.Error("expected final state verified to be carried forward from prior state, not left Unknown")
	}
}
