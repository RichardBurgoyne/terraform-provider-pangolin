package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestSiteDataSource_Schema(t *testing.T) {
	d := NewSiteDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["site_id"].IsRequired() {
		t.Error("expected site_id to be required")
	}
}
