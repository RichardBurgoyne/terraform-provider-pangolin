package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &organizationResource{}
	_ resource.ResourceWithImportState = &organizationResource{}
)

func NewOrganizationResource() resource.Resource { return &organizationResource{} }

type organizationResource struct {
	resourceClient
}

type organizationResourceModel struct {
	OrgID         types.String `tfsdk:"org_id"`
	Name          types.String `tfsdk:"name"`
	Subnet        types.String `tfsdk:"subnet"`
	UtilitySubnet types.String `tfsdk:"utility_subnet"`

	RequireTwoFactor           types.Bool  `tfsdk:"require_two_factor"`
	MaxSessionLengthHours      types.Int64 `tfsdk:"max_session_length_hours"`
	PasswordExpiryDays         types.Int64 `tfsdk:"password_expiry_days"`
	LogRetentionDaysRequest    types.Int64 `tfsdk:"log_retention_days_request"`
	LogRetentionDaysAccess     types.Int64 `tfsdk:"log_retention_days_access"`
	LogRetentionDaysAction     types.Int64 `tfsdk:"log_retention_days_action"`
	LogRetentionDaysConnection types.Int64 `tfsdk:"log_retention_days_connection"`
	LogRetentionDaysAISessions types.Int64 `tfsdk:"log_retention_days_ai_sessions"`
	LogRetentionDaysSSH        types.Int64 `tfsdk:"log_retention_days_ssh"`
	NewtAutoUpdate             types.Bool  `tfsdk:"newt_auto_update"`
}

// orgSetting ties an optional org setting's attribute to its field in the
// model, the update request and the API response, so create, update and
// read handle every setting the same way.
type orgSetting struct {
	name        string
	description string
	isBool      bool
	model       func(*organizationResourceModel) attr.Value
	set         func(m *organizationResourceModel, org *client.Organization)
	request     func(m *organizationResourceModel, in *client.UpdateOrganizationRequest)
}

func retentionSetting(name, logs string, model func(*organizationResourceModel) *types.Int64, field func(*client.Organization) int64, request func(*client.UpdateOrganizationRequest) **int64, extra string) orgSetting {
	return orgSetting{
		name:        name,
		description: fmt.Sprintf("Days to keep %s. 0 keeps none, -1 keeps them forever, and 9001 keeps them until the end of the following year.%s", logs, extra),
		model:       func(m *organizationResourceModel) attr.Value { return *model(m) },
		set:         func(m *organizationResourceModel, org *client.Organization) { *model(m) = types.Int64Value(field(org)) },
		request: func(m *organizationResourceModel, in *client.UpdateOrganizationRequest) {
			*request(in) = int64Ptr(*model(m))
		},
	}
}

const enterpriseNote = " Needs an Enterprise license; without one Pangolin ignores it."

var orgSettings = []orgSetting{
	{
		name:        "require_two_factor",
		description: "Require members to use two-factor authentication.",
		isBool:      true,
		model:       func(m *organizationResourceModel) attr.Value { return m.RequireTwoFactor },
		set: func(m *organizationResourceModel, org *client.Organization) {
			m.RequireTwoFactor = types.BoolPointerValue(org.RequireTwoFactor)
		},
		request: func(m *organizationResourceModel, in *client.UpdateOrganizationRequest) {
			in.RequireTwoFactor = boolPtr(m.RequireTwoFactor)
		},
	},
	{
		name:        "max_session_length_hours",
		description: "Longest a member's session may last, in hours." + enterpriseNote,
		model:       func(m *organizationResourceModel) attr.Value { return m.MaxSessionLengthHours },
		set: func(m *organizationResourceModel, org *client.Organization) {
			m.MaxSessionLengthHours = nullableInt64(org.MaxSessionLengthHours)
		},
		request: func(m *organizationResourceModel, in *client.UpdateOrganizationRequest) {
			in.MaxSessionLengthHours = int64Ptr(m.MaxSessionLengthHours)
		},
	},
	{
		name:        "password_expiry_days",
		description: "Days before members must change their password." + enterpriseNote,
		model:       func(m *organizationResourceModel) attr.Value { return m.PasswordExpiryDays },
		set: func(m *organizationResourceModel, org *client.Organization) {
			m.PasswordExpiryDays = nullableInt64(org.PasswordExpiryDays)
		},
		request: func(m *organizationResourceModel, in *client.UpdateOrganizationRequest) {
			in.PasswordExpiryDays = int64Ptr(m.PasswordExpiryDays)
		},
	},
	retentionSetting("log_retention_days_request", "request logs",
		func(m *organizationResourceModel) *types.Int64 { return &m.LogRetentionDaysRequest },
		func(o *client.Organization) int64 { return o.LogRetentionDaysRequest },
		func(in *client.UpdateOrganizationRequest) **int64 { return &in.LogRetentionDaysRequest }, ""),
	retentionSetting("log_retention_days_access", "access logs",
		func(m *organizationResourceModel) *types.Int64 { return &m.LogRetentionDaysAccess },
		func(o *client.Organization) int64 { return o.LogRetentionDaysAccess },
		func(in *client.UpdateOrganizationRequest) **int64 { return &in.LogRetentionDaysAccess }, enterpriseNote),
	retentionSetting("log_retention_days_action", "action logs",
		func(m *organizationResourceModel) *types.Int64 { return &m.LogRetentionDaysAction },
		func(o *client.Organization) int64 { return o.LogRetentionDaysAction },
		func(in *client.UpdateOrganizationRequest) **int64 { return &in.LogRetentionDaysAction }, enterpriseNote),
	retentionSetting("log_retention_days_connection", "connection logs",
		func(m *organizationResourceModel) *types.Int64 { return &m.LogRetentionDaysConnection },
		func(o *client.Organization) int64 { return o.LogRetentionDaysConnection },
		func(in *client.UpdateOrganizationRequest) **int64 { return &in.LogRetentionDaysConnection }, enterpriseNote),
	retentionSetting("log_retention_days_ai_sessions", "AI gateway session logs",
		func(m *organizationResourceModel) *types.Int64 { return &m.LogRetentionDaysAISessions },
		func(o *client.Organization) int64 { return o.LogRetentionDaysAISessions },
		func(in *client.UpdateOrganizationRequest) **int64 { return &in.LogRetentionDaysAISessions }, enterpriseNote),
	retentionSetting("log_retention_days_ssh", "SSH session logs",
		func(m *organizationResourceModel) *types.Int64 { return &m.LogRetentionDaysSSH },
		func(o *client.Organization) int64 { return o.LogRetentionDaysSSH },
		func(in *client.UpdateOrganizationRequest) **int64 { return &in.LogRetentionDaysSSH }, " Requires Pangolin 1.26 or later."+enterpriseNote),
	{
		name:        "newt_auto_update",
		description: "Let Pangolin update the Newt agent on the organization's sites automatically." + enterpriseNote,
		isBool:      true,
		model:       func(m *organizationResourceModel) attr.Value { return m.NewtAutoUpdate },
		set: func(m *organizationResourceModel, org *client.Organization) {
			m.NewtAutoUpdate = types.BoolValue(org.EnableGlobalNewtAutoUpdate)
		},
		request: func(m *organizationResourceModel, in *client.UpdateOrganizationRequest) {
			in.EnableGlobalNewtAutoUpdate = boolPtr(m.NewtAutoUpdate)
		},
	},
}

func (r *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	attributes := map[string]schema.Attribute{
		"org_id": schema.StringAttribute{
			Required:      true,
			Description:   "Unique organization identifier: lowercase letters, numbers, underscores, and single hyphens only.",
			PlanModifiers: replace,
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Display name of the organization.",
		},
		"subnet": schema.StringAttribute{
			Required:      true,
			Description:   "IPv4 CIDR block for this organization's client subnet.",
			PlanModifiers: replace,
		},
		"utility_subnet": schema.StringAttribute{
			Required:      true,
			Description:   "IPv4 CIDR block for this organization's utility subnet. Must not overlap with subnet.",
			PlanModifiers: replace,
		},
	}
	// Settings left out of the configuration keep whatever value they have in
	// Pangolin, so the provider can be adopted without resetting them.
	for _, s := range orgSettings {
		description := s.description + " Left unchanged when omitted."
		if s.isBool {
			attributes[s.name] = schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   description,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			}
			continue
		}
		attributes[s.name] = schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			Description:   description,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Validators:    []validator.Int64{int64validator.AtLeast(-1)},
		}
	}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin organization and its settings. Changing org_id, subnet or utility_subnet replaces the organization; the name and settings are updated in place.",
		Attributes:  attributes,
	}
}

func (m *organizationResourceModel) fromAPI(org *client.Organization) {
	m.OrgID = types.StringValue(org.OrgID)
	m.Name = types.StringValue(org.Name)
	m.Subnet = types.StringValue(org.Subnet)
	m.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	for _, s := range orgSettings {
		s.set(m, org)
	}
}

// applySettings sends the settings in plan that differ from prior (or, on
// create, that are configured at all) and returns the organization as stored,
// along with the settings that didn't take effect: Pangolin quietly ignores
// settings its license doesn't cover. A failed update is returned as
// updateErr, with every setting it carried counted as not applied.
func (r *organizationResource) applySettings(ctx context.Context, plan, prior *organizationResourceModel) (org *client.Organization, unapplied []orgSetting, diags diag.Diagnostics, updateErr error) {
	orgID := plan.OrgID.ValueString()

	var in client.UpdateOrganizationRequest
	var sent []orgSetting
	if prior != nil && !plan.Name.Equal(prior.Name) {
		in.Name = strPtr(plan.Name)
	}
	for _, s := range orgSettings {
		value := s.model(plan)
		if value.IsNull() || value.IsUnknown() || (prior != nil && value.Equal(s.model(prior))) {
			continue
		}
		s.request(plan, &in)
		sent = append(sent, s)
	}

	if in != (client.UpdateOrganizationRequest{}) {
		updateErr = r.client.UpdateOrganization(ctx, orgID, in)
	}

	org, err := r.client.GetOrganization(ctx, orgID)
	if err != nil {
		diags.AddError("Error reading organization", err.Error())
		return nil, nil, diags, updateErr
	}
	var stored organizationResourceModel
	stored.fromAPI(org)
	for _, s := range sent {
		if !s.model(&stored).Equal(s.model(plan)) {
			unapplied = append(unapplied, s)
		}
	}
	return org, unapplied, diags, updateErr
}

func notAppliedDetail(s orgSetting, stored *organizationResourceModel) string {
	return fmt.Sprintf("Pangolin did not save %s (it reads %s). The setting may need an Enterprise license or a newer version of Pangolin.", s.name, s.model(stored))
}

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CreateOrganization(ctx, client.CreateOrganizationRequest{
		OrgID:         plan.OrgID.ValueString(),
		Name:          plan.Name.ValueString(),
		Subnet:        plan.Subnet.ValueString(),
		UtilitySubnet: plan.UtilitySubnet.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization", err.Error())
		return
	}
	// Save the ID straight away so a failure below doesn't orphan the org.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), plan.OrgID)...)

	org, unapplied, diags, updateErr := r.applySettings(ctx, &plan, nil)
	resp.Diagnostics.Append(diags...)
	if org == nil {
		return
	}

	// An error here would taint the new organization, and the next apply
	// would replace it along with everything in it. Settings that didn't take
	// are warned about instead and recorded as planned, so the next plan
	// shows them as still to do and retries them as an ordinary update.
	planned := plan
	plan.fromAPI(org)
	if updateErr != nil {
		resp.Diagnostics.AddWarning("Organization settings not applied", "The organization was created, but saving its settings failed: "+updateErr.Error())
	}
	for _, s := range unapplied {
		resp.Diagnostics.AddAttributeWarning(path.Root(s.name), "Setting not applied", notAppliedDetail(s, &plan))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	for _, s := range unapplied {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(s.name), s.model(&planned))...)
	}
}

func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.GetOrganization(ctx, state.OrgID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	state.fromAPI(org)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, unapplied, diags, updateErr := r.applySettings(ctx, &plan, &state)
	resp.Diagnostics.Append(diags...)
	if updateErr != nil {
		resp.Diagnostics.AddError("Error updating organization", updateErr.Error())
		return
	}
	if org == nil {
		return
	}
	// Even when a setting wasn't applied, record what Pangolin stored.
	plan.fromAPI(org)
	for _, s := range unapplied {
		resp.Diagnostics.AddAttributeError(path.Root(s.name), "Setting not applied", notAppliedDetail(s, &plan))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteOrganization(ctx, state.OrgID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting organization", err.Error())
	}
}

func (r *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("org_id"), req, resp)
}
