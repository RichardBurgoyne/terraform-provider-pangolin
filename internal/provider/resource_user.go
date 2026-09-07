package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ resource.Resource = &userResource{}

func NewUserResource() resource.Resource { return &userResource{} }

type userResource struct {
	client *client.Client
}

type userResourceModel struct {
	OrgID           types.String `tfsdk:"org_id"`
	UserID          types.String `tfsdk:"user_id"`
	Username        types.String `tfsdk:"username"`
	Email           types.String `tfsdk:"email"`
	Name            types.String `tfsdk:"name"`
	IdpID           types.Int64  `tfsdk:"idp_id"`
	RoleIDs         types.List   `tfsdk:"role_ids"`
	AutoProvisioned types.Bool   `tfsdk:"auto_provisioned"`
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages an OIDC-backed org user in Pangolin. Internal (password-based) users are not yet supported: the Pangolin API explicitly rejects user creation with type internal (\"Internal users are not supported yet\"), so this resource only supports OIDC-backed users. Removing an entry from role_ids calls the role-removal route (DELETE /user/:userId/remove-role/:roleId), which is part of Pangolin's commercial integration API; against a plain self-hosted community instance it fails with a clear error rather than silently doing nothing or being retried. Import is not supported: GetOrgUser cannot recover email, idp_id, or role_ids, so an imported user would show a forced replacement on the very next plan.",
		Attributes: map[string]schema.Attribute{
			"org_id":           schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this user belongs to."},
			"user_id":          schema.StringAttribute{Computed: true, Description: "Server-generated user ID."},
			"username":         schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Username, lowercased server-side."},
			"email":            schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Email address."},
			"name":             schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Display name."},
			"idp_id":           schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, Description: "Numeric ID of the OIDC identity provider this user authenticates through."},
			"role_ids":         schema.ListAttribute{Required: true, ElementType: types.Int64Type, Description: "Role IDs to grant. Removing an entry requires a commercial Pangolin integration API route (see resource description); on a plain community instance role_ids can only grow after creation."},
			"auto_provisioned": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}, Description: "Whether this user was auto-provisioned."},
		},
	}
}

func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roleIDs []int64
	resp.Diagnostics.Append(plan.RoleIDs.ElementsAs(ctx, &roleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateOrgUserRequest{
		Username: plan.Username.ValueString(),
		Type:     "oidc",
		IdpID:    plan.IdpID.ValueInt64(),
		RoleIDs:  roleIDs,
	}
	in.Email = plan.Email.ValueString()
	in.Name = plan.Name.ValueString()

	orgID := plan.OrgID.ValueString()
	if err := r.client.CreateOrgUser(ctx, orgID, in); err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	// createOrgUser returns no data at all, so look the new user up by
	// username to learn its server-generated ID.
	user, err := r.client.GetUserByUsername(ctx, orgID, plan.Username.ValueString())
	if err != nil {
		// CreateOrgUser succeeded, so a user very likely exists server-side
		// under this username. Unlike the other resources there is no ID to
		// write to state here: discovering the ID is precisely what this
		// lookup does, so nothing can be persisted and the object is left
		// untracked. Say so explicitly rather than reporting a bare error.
		resp.Diagnostics.AddError(
			"Error looking up newly created user",
			fmt.Sprintf(
				"%s\n\nThe user may have been created successfully server-side under username %q but could not be looked up to record in state. "+
					"Check the Pangolin dashboard before retrying: if the user exists, import it with `terraform import` once you have its ID, "+
					"since retrying the apply may attempt to create a duplicate.",
				err.Error(), plan.Username.ValueString(),
			),
		)
		return
	}

	plan.UserID = types.StringValue(user.UserID)
	plan.AutoProvisioned = types.BoolValue(user.AutoProvisioned)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetOrgUser(ctx, state.OrgID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	state.Username = types.StringValue(user.Username)
	state.AutoProvisioned = types.BoolValue(user.AutoProvisioned)
	// email, name, idp_id, and role_ids are not re-derived from GetOrgUser
	// (its exact response shape wasn't confirmed from source for these
	// fields) - they are left as whatever is already in state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planRoleIDs, stateRoleIDs []int64
	resp.Diagnostics.Append(plan.RoleIDs.ElementsAs(ctx, &planRoleIDs, false)...)
	resp.Diagnostics.Append(state.RoleIDs.ElementsAs(ctx, &stateRoleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing := make(map[int64]bool, len(stateRoleIDs))
	for _, id := range stateRoleIDs {
		existing[id] = true
	}
	planned := make(map[int64]bool, len(planRoleIDs))
	for _, id := range planRoleIDs {
		planned[id] = true
	}

	if !plan.AutoProvisioned.IsUnknown() {
		v := plan.AutoProvisioned.ValueBool()
		if err := r.client.UpdateOrgUser(ctx, state.OrgID.ValueString(), state.UserID.ValueString(), client.UpdateOrgUserRequest{AutoProvisioned: &v}); err != nil {
			resp.Diagnostics.AddError("Error updating user", err.Error())
			return
		}
	}

	for _, id := range stateRoleIDs {
		if !planned[id] {
			if err := r.client.RemoveUserRole(ctx, state.UserID.ValueString(), id); err != nil {
				resp.Diagnostics.AddError(
					"Error removing role from user",
					fmt.Sprintf(
						"Failed to remove role %d: %s\n\nThis route (DELETE /user/:userId/remove-role/:roleId) is part of Pangolin's commercial integration API. Against a plain self-hosted community instance it does not exist and this call will always fail; in that case, destroy and recreate the pangolin_user resource, or remove the role via the Pangolin dashboard and then run terraform apply -refresh-only.",
						id, err.Error(),
					),
				)
				return
			}
		}
	}

	for _, id := range planRoleIDs {
		if !existing[id] {
			if err := r.client.AddUserRole(ctx, id, state.UserID.ValueString()); err != nil {
				resp.Diagnostics.AddError("Error adding role to user", err.Error())
				return
			}
		}
	}

	plan.UserID = state.UserID
	if plan.AutoProvisioned.IsUnknown() {
		plan.AutoProvisioned = state.AutoProvisioned
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrgUser(ctx, state.OrgID.ValueString(), state.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}
