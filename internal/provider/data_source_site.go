package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &siteDataSource{}

func NewSiteDataSource() datasource.DataSource { return &siteDataSource{} }

type siteDataSource struct {
	client *client.Client
}

type siteDataSourceModel struct {
	SiteID types.Int64  `tfsdk:"site_id"`
	NiceID types.String `tfsdk:"nice_id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
}

func (d *siteDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (d *siteDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Pangolin site by site_id.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.Int64Attribute{Required: true},
			"nice_id": schema.StringAttribute{Computed: true},
			"name":    schema.StringAttribute{Computed: true},
			"type":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *siteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *siteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model siteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := d.client.GetSite(ctx, model.SiteID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading site", err.Error())
		return
	}

	model.NiceID = types.StringValue(site.NiceID)
	model.Name = types.StringValue(site.Name)
	model.Type = types.StringValue(site.Type)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
