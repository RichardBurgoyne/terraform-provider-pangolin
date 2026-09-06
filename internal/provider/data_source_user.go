package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &userDataSource{}

func NewUserDataSource() datasource.DataSource { return &userDataSource{} }

type userDataSource struct {
	client *client.Client
}

type userDataSourceModel struct {
	OrgID    types.String `tfsdk:"org_id"`
	UserID   types.String `tfsdk:"user_id"`
	Username types.String `tfsdk:"username"`
}

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing org user by org_id and either user_id or username. Exactly one of user_id or username must be set.",
		Attributes: map[string]schema.Attribute{
			"org_id":   schema.StringAttribute{Required: true},
			"user_id":  schema.StringAttribute{Optional: true, Computed: true},
			"username": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *client.User
	var err error
	if !model.Username.IsNull() {
		user, err = d.client.GetUserByUsername(ctx, model.OrgID.ValueString(), model.Username.ValueString())
	} else {
		user, err = d.client.GetOrgUser(ctx, model.OrgID.ValueString(), model.UserID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	model.UserID = types.StringValue(user.UserID)
	model.Username = types.StringValue(user.Username)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
