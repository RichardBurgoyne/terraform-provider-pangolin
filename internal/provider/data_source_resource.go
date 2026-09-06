package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &pangolinResourceDataSource{}

func NewPangolinResourceDataSource() datasource.DataSource { return &pangolinResourceDataSource{} }

type pangolinResourceDataSource struct {
	client *client.Client
}

type pangolinResourceDataSourceModel struct {
	ResourceID types.Int64  `tfsdk:"resource_id"`
	Name       types.String `tfsdk:"name"`
	Mode       types.String `tfsdk:"mode"`
	FullDomain types.String `tfsdk:"full_domain"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (d *pangolinResourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (d *pangolinResourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Pangolin resource by resource_id.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{Required: true},
			"name":        schema.StringAttribute{Computed: true},
			"mode":        schema.StringAttribute{Computed: true},
			"full_domain": schema.StringAttribute{Computed: true},
			"enabled":     schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *pangolinResourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *pangolinResourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model pangolinResourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := d.client.GetResource(ctx, model.ResourceID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	model.Name = types.StringValue(res.Name)
	model.Mode = types.StringValue(res.Mode)
	model.FullDomain = types.StringValue(res.FullDomain)
	model.Enabled = types.BoolValue(res.Enabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
