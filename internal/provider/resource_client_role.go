package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &clientRoleResource{}
	_ resource.ResourceWithImportState = &clientRoleResource{}
)

func NewClientRoleResource() resource.Resource { return &clientRoleResource{} }

type clientRoleResource struct {
	resourceClient
}

type clientRoleModel struct {
	ClientID types.Int64 `tfsdk:"client_id"`
	RoleID   types.Int64 `tfsdk:"role_id"`
}

func (r *clientRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_role"
}

func (r *clientRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Assigns a role to a machine client, so it gets the private resource access granted to that role. Clients tied to a user can't be assigned roles; they inherit their user's. Requires Pangolin 1.25 or later, Enterprise Edition. There is no update: changing either ID replaces the assignment. Import using the format `<client_id>:<role_id>`, e.g. `terraform import pangolin_client_role.example 5:2`.",
		Attributes: map[string]schema.Attribute{
			"client_id": schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Machine client ID."},
			"role_id":   schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Role ID to assign."},
		},
	}
}

func (r *clientRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AddClientRole(ctx, plan.ClientID.ValueInt64(), plan.RoleID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error assigning role to client", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roleIDs, err := r.client.ListClientRoleIDs(ctx, state.ClientID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client roles", err.Error())
		return
	}
	if !slices.Contains(roleIDs, state.RoleID.ValueInt64()) {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.RemoveClientRole(ctx, state.ClientID.ValueInt64(), state.RoleID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error removing role from client", err.Error())
	}
}

func (r *clientRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Pair(ctx, req.ID, resp, "client_id", "role_id")
}
