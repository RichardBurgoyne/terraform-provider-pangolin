package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestUserDataSource_Schema(t *testing.T) {
	d := NewUserDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["org_id"].IsRequired() {
		t.Error("expected org_id to be required")
	}
}
