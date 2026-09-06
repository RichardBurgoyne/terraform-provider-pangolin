package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestPangolinResourceDataSource_Schema(t *testing.T) {
	d := NewPangolinResourceDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["resource_id"].IsRequired() {
		t.Error("expected resource_id to be required")
	}
}
