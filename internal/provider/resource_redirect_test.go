package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

func TestRedirectResource_ValidateConfig(t *testing.T) {
	for name, tc := range map[string]struct {
		model   redirectModel
		wantErr bool
	}{
		"domain with subdomain": {redirectModel{DomainID: types.StringValue("d1"), Subdomain: types.StringValue("go"), SSL: types.BoolValue(false)}, false},
		"resource":              {redirectModel{ResourceID: types.Int64Value(5)}, false},
		"resource with subdomain": {
			redirectModel{ResourceID: types.Int64Value(5), Subdomain: types.StringValue("go")}, true,
		},
		"resource with ssl off": {redirectModel{ResourceID: types.Int64Value(5), SSL: types.BoolValue(false)}, true},
		"rewrite without path":  {redirectModel{ResourceID: types.Int64Value(5), RewritePathType: types.StringValue("prefix")}, true},
		"strip prefix":          {redirectModel{ResourceID: types.Int64Value(5), RewritePathType: types.StringValue("stripPrefix")}, false},
	} {
		t.Run(name, func(t *testing.T) {
			diags := validateConfig(t, &redirectResource{}, tc.model)
			if diags.HasError() != tc.wantErr {
				t.Errorf("wantErr %v, got %v", tc.wantErr, diags)
			}
		})
	}
}

func TestRedirectModel_FromAPIKeepsNullsNull(t *testing.T) {
	var model redirectModel
	resourceID := int64(5)
	model.fromAPI(&client.Redirect{RedirectID: 7, ResourceID: &resourceID})
	if model.ResourceID.ValueInt64() != 5 {
		t.Errorf("unexpected resource_id %s", model.ResourceID)
	}
	for name, value := range map[string]types.String{
		"domain_id": model.DomainID, "subdomain": model.Subdomain, "match_path": model.MatchPath,
		"rewrite_path": model.RewritePath, "rewrite_path_type": model.RewritePathType,
	} {
		if !value.IsNull() {
			t.Errorf("expected %s to be null, got %s", name, value)
		}
	}
}
