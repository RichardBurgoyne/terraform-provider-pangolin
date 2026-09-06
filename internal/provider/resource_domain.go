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
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

func NewDomainResource() resource.Resource { return &domainResource{} }

type domainResource struct {
	client *client.Client
}

type domainResourceModel struct {
	OrgID              types.String `tfsdk:"org_id"`
	DomainID           types.String `tfsdk:"domain_id"`
	Type               types.String `tfsdk:"type"`
	BaseDomain         types.String `tfsdk:"base_domain"`
	CertResolver       types.String `tfsdk:"cert_resolver"`
	PreferWildcardCert types.Bool   `tfsdk:"prefer_wildcard_cert"`
	Verified           types.Bool   `tfsdk:"verified"`
}

func (r *domainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a domain attached to a Pangolin organization. type and base_domain are immutable after creation.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				Description:   "Organization ID this domain belongs to.",
				PlanModifiers: replace,
			},
			"domain_id": schema.StringAttribute{
				Computed:    true,
				Description: "Server-generated domain ID.",
			},
			"type": schema.StringAttribute{
				Required:      true,
				Description:   "One of ns, cname, or wildcard. Self-hosted (OSS) instances only support wildcard.",
				PlanModifiers: replace,
			},
			"base_domain": schema.StringAttribute{
				Required:      true,
				Description:   "The base domain to attach.",
				PlanModifiers: replace,
			},
			"cert_resolver": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Certificate resolver to use for this domain.",
			},
			"prefer_wildcard_cert": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to prefer a wildcard certificate for this domain.",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the domain has completed verification.",
			},
		},
	}
}

func (r *domainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createIn := client.CreateDomainRequest{
		Type:       plan.Type.ValueString(),
		BaseDomain: plan.BaseDomain.ValueString(),
	}
	if !plan.CertResolver.IsUnknown() {
		createIn.CertResolver = plan.CertResolver.ValueString()
	}
	if !plan.PreferWildcardCert.IsUnknown() && !plan.PreferWildcardCert.IsNull() {
		v := plan.PreferWildcardCert.ValueBool()
		createIn.PreferWildcardCert = &v
	}

	created, err := r.client.CreateDomain(ctx, plan.OrgID.ValueString(), createIn)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	domain, err := r.client.GetDomain(ctx, plan.OrgID.ValueString(), created.DomainID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading newly created domain", err.Error())
		return
	}

	plan.DomainID = types.StringValue(domain.DomainID)
	plan.Type = types.StringValue(domain.Type)
	plan.BaseDomain = types.StringValue(domain.BaseDomain)
	plan.CertResolver = types.StringValue(domain.CertResolver)
	plan.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	plan.Verified = types.BoolValue(domain.Verified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.OrgID.ValueString(), state.DomainID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	state.Type = types.StringValue(domain.Type)
	state.BaseDomain = types.StringValue(domain.BaseDomain)
	state.CertResolver = types.StringValue(domain.CertResolver)
	state.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	state.Verified = types.BoolValue(domain.Verified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateIn := client.UpdateDomainRequest{CertResolver: plan.CertResolver.ValueString()}
	if !plan.PreferWildcardCert.IsNull() {
		v := plan.PreferWildcardCert.ValueBool()
		updateIn.PreferWildcardCert = &v
	}

	domain, err := r.client.UpdateDomain(ctx, plan.OrgID.ValueString(), plan.DomainID.ValueString(), updateIn)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	plan.CertResolver = types.StringValue(domain.CertResolver)
	plan.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDomain(ctx, state.OrgID.ValueString(), state.DomainID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_id"), req, resp)
}
