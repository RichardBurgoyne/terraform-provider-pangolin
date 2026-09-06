package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestRoleDataSource_Schema(t *testing.T) {
	d := NewRoleDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["org_id"].IsRequired() {
		t.Error("expected org_id to be required")
	}
}
