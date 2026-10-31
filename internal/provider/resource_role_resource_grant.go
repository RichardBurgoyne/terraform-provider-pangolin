package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &roleResourceGrantResource{}
	_ resource.ResourceWithImportState = &roleResourceGrantResource{}
)

func NewRoleResourceGrantResource() resource.Resource { return &roleResourceGrantResource{} }

type roleResourceGrantResource struct {
	resourceClient
}

type roleResourceGrantModel struct {
	ResourceID types.Int64 `tfsdk:"resource_id"`
	RoleID     types.Int64 `tfsdk:"role_id"`
}

func (r *roleResourceGrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_resource_grant"
}

func (r *roleResourceGrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Grants a role access to a Pangolin resource. There is no update: changing either ID replaces the grant. Import using the format `<resource_id>:<role_id>`, e.g. `terraform import pangolin_role_resource_grant.example 5:2`.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Resource ID to grant access to."},
			"role_id":     schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Role ID being granted access."},
		},
	}
}

func (r *roleResourceGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AddRoleToResource(ctx, plan.ResourceID.ValueInt64(), plan.RoleID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error granting role access to resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResourceGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIDs, err := r.client.ListResourceRoles(ctx, state.ResourceID.ValueInt64())
	if err != nil {
		// A 404 here means the parent resource is gone, which means the grant
		// is gone too. Treat it as a removal rather than a hard refresh error.
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource roles", err.Error())
		return
	}

	found := false
	for _, id := range roleIDs {
		if id == state.RoleID.ValueInt64() {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResourceGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResourceGrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Pair(ctx, req.ID, resp, "resource_id", "role_id")
}

func (r *roleResourceGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.RemoveRoleFromResource(ctx, state.ResourceID.ValueInt64(), state.RoleID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error revoking role access from resource", err.Error())
	}
}
