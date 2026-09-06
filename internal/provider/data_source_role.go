package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &roleDataSource{}

func NewRoleDataSource() datasource.DataSource { return &roleDataSource{} }

type roleDataSource struct {
	client *client.Client
}

type roleDataSourceModel struct {
	OrgID  types.String `tfsdk:"org_id"`
	RoleID types.Int64  `tfsdk:"role_id"`
	Name   types.String `tfsdk:"name"`
}

func (d *roleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing role by org_id and either role_id or name. Exactly one of role_id or name must be set.",
		Attributes: map[string]schema.Attribute{
			"org_id":  schema.StringAttribute{Required: true},
			"role_id": schema.Int64Attribute{Optional: true, Computed: true},
			"name":    schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *roleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.ListRoles(ctx, model.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing roles", err.Error())
		return
	}

	var found *client.Role
	for i := range roles {
		if !model.RoleID.IsNull() && roles[i].RoleID == model.RoleID.ValueInt64() {
			found = &roles[i]
			break
		}
		if !model.Name.IsNull() && roles[i].Name == model.Name.ValueString() {
			found = &roles[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Role not found", "No role matched the given role_id or name in this org.")
		return
	}

	model.RoleID = types.Int64Value(found.RoleID)
	model.Name = types.StringValue(found.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
