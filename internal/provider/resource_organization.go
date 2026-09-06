package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &organizationResource{}
	_ resource.ResourceWithImportState = &organizationResource{}
)

func NewOrganizationResource() resource.Resource { return &organizationResource{} }

type organizationResource struct {
	client *client.Client
}

type organizationResourceModel struct {
	OrgID         types.String `tfsdk:"org_id"`
	Name          types.String `tfsdk:"name"`
	Subnet        types.String `tfsdk:"subnet"`
	UtilitySubnet types.String `tfsdk:"utility_subnet"`
}

func (r *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin organization. All attributes are immutable after creation (update semantics were not confirmed against source); changing any of them replaces the organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				Description:   "Unique organization identifier: lowercase letters, numbers, underscores, and single hyphens only.",
				PlanModifiers: replace,
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Display name of the organization.",
				PlanModifiers: replace,
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
		},
	}
}

func (r *organizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.CreateOrganization(ctx, client.CreateOrganizationRequest{
		OrgID:         plan.OrgID.ValueString(),
		Name:          plan.Name.ValueString(),
		Subnet:        plan.Subnet.ValueString(),
		UtilitySubnet: plan.UtilitySubnet.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization", err.Error())
		return
	}

	plan.OrgID = types.StringValue(org.OrgID)
	plan.Name = types.StringValue(org.Name)
	plan.Subnet = types.StringValue(org.Subnet)
	plan.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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

	state.Name = types.StringValue(org.Name)
	state.Subnet = types.StringValue(org.Subnet)
	state.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every attribute is RequiresReplace, so Terraform core never actually
	// calls this. The interface still requires an implementation.
	var plan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrganization(ctx, state.OrgID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting organization", err.Error())
	}
}

func (r *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("org_id"), req, resp)
}
