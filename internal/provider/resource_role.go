package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource { return &roleResource{} }

type roleResource struct {
	client *client.Client
}

type roleResourceModel struct {
	OrgID                 types.String `tfsdk:"org_id"`
	RoleID                types.Int64  `tfsdk:"role_id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	RequireDeviceApproval types.Bool   `tfsdk:"require_device_approval"`
	AllowSSH              types.Bool   `tfsdk:"allow_ssh"`
	SSHSudoMode           types.String `tfsdk:"ssh_sudo_mode"`
}

func (r *roleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin role. Fine-grained SSH sudo command/group lists are not yet supported by this provider. Import using the format `<org_id>:<role_id>`, e.g. `terraform import pangolin_role.example acme:5`.",
		Attributes: map[string]schema.Attribute{
			"org_id":                  schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Organization ID this role belongs to."},
			"role_id":                 schema.Int64Attribute{Computed: true, Description: "Server-generated role ID."},
			"name":                    schema.StringAttribute{Required: true, Description: "Role name, unique per org."},
			"description":             schema.StringAttribute{Optional: true, Computed: true, Description: "Role description."},
			"require_device_approval": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether devices used by members of this role require approval."},
			"allow_ssh":               schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether members of this role can sign SSH keys."},
			"ssh_sudo_mode":           schema.StringAttribute{Optional: true, Computed: true, Description: "One of none, full, commands."},
		},
	}
}

func (r *roleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateRoleRequest{Name: plan.Name.ValueString()}
	if !plan.Description.IsUnknown() {
		in.Description = plan.Description.ValueString()
	}
	if !plan.RequireDeviceApproval.IsUnknown() {
		v := plan.RequireDeviceApproval.ValueBool()
		in.RequireDeviceApproval = &v
	}
	if !plan.AllowSSH.IsUnknown() {
		v := plan.AllowSSH.ValueBool()
		in.AllowSSH = &v
	}
	if !plan.SSHSudoMode.IsUnknown() {
		in.SSHSudoMode = plan.SSHSudoMode.ValueString()
	}

	role, err := r.client.CreateRole(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating role", err.Error())
		return
	}

	setRoleModelFromAPI(&plan, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRoleByID(ctx, state.OrgID.ValueString(), state.RoleID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}

	setRoleModelFromAPI(&state, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateRoleRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	if !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		in.Description = &v
	}
	if !plan.RequireDeviceApproval.IsUnknown() {
		v := plan.RequireDeviceApproval.ValueBool()
		in.RequireDeviceApproval = &v
	}
	if !plan.AllowSSH.IsUnknown() {
		v := plan.AllowSSH.ValueBool()
		in.AllowSSH = &v
	}
	if !plan.SSHSudoMode.IsUnknown() {
		v := plan.SSHSudoMode.ValueString()
		in.SSHSudoMode = &v
	}

	role, err := r.client.UpdateRole(ctx, state.RoleID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating role", err.Error())
		return
	}

	// org_id is RequiresReplace, so plan and state always agree here. The
	// update response may not echo orgId back, and setRoleModelFromAPI writes
	// it unconditionally, so preserve the known-good value across the call.
	orgID := plan.OrgID
	setRoleModelFromAPI(&plan, role)
	plan.OrgID = orgID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRole(ctx, state.RoleID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting role", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", `expected format: <org_id>:<role_id>, e.g. "acme:5"`)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "role_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), id)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func setRoleModelFromAPI(model *roleResourceModel, role *client.Role) {
	model.RoleID = types.Int64Value(role.RoleID)
	model.OrgID = types.StringValue(role.OrgID)
	model.Name = types.StringValue(role.Name)
	model.Description = types.StringValue(role.Description)
	model.RequireDeviceApproval = types.BoolValue(role.RequireDeviceApproval)
	model.AllowSSH = types.BoolValue(role.AllowSSH)
	model.SSHSudoMode = types.StringValue(role.SSHSudoMode)
}
