package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
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

func TestOrganizationResource_ApplySettings(t *testing.T) {
	var body map[string]any
	stored := map[string]any{"orgId": "acme", "name": "Acme Ltd", "settingsLogRetentionDaysRequest": float64(7), "settingsLogRetentionDaysSsh": float64(0)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body = nil
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"org": stored}, "success": true})
		}
	}))
	defer server.Close()
	r := &organizationResource{resourceClient{client: client.New(server.URL, "token")}}
	ctx := context.Background()

	prior := organizationResourceModel{OrgID: types.StringValue("acme"), Name: types.StringValue("Acme"), LogRetentionDaysRequest: types.Int64Value(7), LogRetentionDaysSSH: types.Int64Value(0)}
	plan := prior
	plan.Name = types.StringValue("Acme Ltd")
	if _, unapplied, diags, err := r.applySettings(ctx, &plan, &prior); diags.HasError() || err != nil || len(unapplied) != 0 {
		t.Fatal(unapplied, err, diags)
	}
	// Unchanged settings stay out of the request, so a setting an older
	// server doesn't know can't break a rename.
	if len(body) != 1 || body["name"] != "Acme Ltd" {
		t.Errorf("unexpected update body: %v", body)
	}

	// A setting Pangolin ignores (e.g. unlicensed) is reported, not hidden.
	plan.LogRetentionDaysSSH = types.Int64Value(30)
	prior.Name = plan.Name
	org, unapplied, diags, err := r.applySettings(ctx, &plan, &prior)
	if diags.HasError() || err != nil || org == nil || len(unapplied) != 1 || unapplied[0].name != "log_retention_days_ssh" {
		t.Errorf("expected log_retention_days_ssh to be reported as not applied, got %v %v %v", unapplied, err, diags)
	}
	if len(body) != 1 || body["settingsLogRetentionDaysSsh"] != float64(30) {
		t.Errorf("unexpected update body: %v", body)
	}
}
