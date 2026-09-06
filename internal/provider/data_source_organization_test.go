package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestOrganizationDataSource_Schema(t *testing.T) {
	d := NewOrganizationDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if _, ok := resp.Schema.Attributes["org_id"]; !ok {
		t.Error("expected attribute \"org_id\"")
	}
	if !resp.Schema.Attributes["name"].IsComputed() {
		t.Error("expected \"name\" to be computed")
	}
}
