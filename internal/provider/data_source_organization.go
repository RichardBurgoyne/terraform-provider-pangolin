package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &organizationDataSource{}

func NewOrganizationDataSource() datasource.DataSource { return &organizationDataSource{} }

type organizationDataSource struct {
	client *client.Client
}

type organizationDataSourceModel struct {
	OrgID         types.String `tfsdk:"org_id"`
	Name          types.String `tfsdk:"name"`
	Subnet        types.String `tfsdk:"subnet"`
	UtilitySubnet types.String `tfsdk:"utility_subnet"`
}

func (d *organizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Pangolin organization by ID.",
		Attributes: map[string]schema.Attribute{
			"org_id":         schema.StringAttribute{Required: true, Description: "Organization ID to look up."},
			"name":           schema.StringAttribute{Computed: true, Description: "Display name of the organization."},
			"subnet":         schema.StringAttribute{Computed: true, Description: "IPv4 CIDR block for the client subnet."},
			"utility_subnet": schema.StringAttribute{Computed: true, Description: "IPv4 CIDR block for the utility subnet."},
		},
	}
}

func (d *organizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model organizationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := d.client.GetOrganization(ctx, model.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	model.Name = types.StringValue(org.Name)
	model.Subnet = types.StringValue(org.Subnet)
	model.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
