package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                   = &redirectResource{}
	_ resource.ResourceWithImportState    = &redirectResource{}
	_ resource.ResourceWithValidateConfig = &redirectResource{}
)

func NewRedirectResource() resource.Resource { return &redirectResource{} }

type redirectResource struct {
	resourceClient
}

type redirectModel struct {
	OrgID           types.String `tfsdk:"org_id"`
	RedirectID      types.Int64  `tfsdk:"redirect_id"`
	NiceID          types.String `tfsdk:"nice_id"`
	Name            types.String `tfsdk:"name"`
	ResourceID      types.Int64  `tfsdk:"resource_id"`
	DomainID        types.String `tfsdk:"domain_id"`
	Subdomain       types.String `tfsdk:"subdomain"`
	DestinationHost types.String `tfsdk:"destination_host"`
	PathMatchType   types.String `tfsdk:"path_match_type"`
	MatchPath       types.String `tfsdk:"match_path"`
	RewritePath     types.String `tfsdk:"rewrite_path"`
	RewritePathType types.String `tfsdk:"rewrite_path_type"`
	Priority        types.Int64  `tfsdk:"priority"`
	Permanent       types.Bool   `tfsdk:"permanent"`
	SSL             types.Bool   `tfsdk:"ssl"`
	Enabled         types.Bool   `tfsdk:"enabled"`
}

func (r *redirectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect"
}

func (r *redirectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Redirects requests to another host, either for paths on an existing resource (`resource_id`) or for a domain of its own (`domain_id`). Requires Pangolin 1.25 or later. Permanent redirects also need Badger 1.8.0 or later. Import using the format `<org_id>:<redirect_id>`, e.g. `terraform import pangolin_redirect.example acme:5`.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Organization the redirect belongs to.",
			},
			"redirect_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Pangolin-assigned redirect ID.",
			},
			"nice_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Human-readable ID, unique per org.",
			},
			"name": schema.StringAttribute{Required: true, Description: "Display name."},
			"resource_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Resource whose domain the redirect applies to. Exactly one of resource_id and domain_id must be set.",
				Validators:  []validator.Int64{int64validator.ExactlyOneOf(path.MatchRoot("domain_id"))},
			},
			"domain_id": schema.StringAttribute{
				Optional:    true,
				Description: "Domain the redirect listens on. Exactly one of resource_id and domain_id must be set.",
			},
			"subdomain": schema.StringAttribute{
				Optional:    true,
				Description: "Subdomain under domain_id. Omit to redirect the domain itself. Only valid with domain_id.",
			},
			"destination_host": schema.StringAttribute{
				Required:    true,
				Description: "Where requests go, as `scheme://host[:port]` with no path, e.g. `https://example.com`. The request path, after any rewrite, is appended to it.",
			},
			"path_match_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("regex"),
				Description: "How match_path is compared to the request path: exact, prefix or regex. Defaults to regex.",
				Validators:  []validator.String{stringvalidator.OneOf("exact", "prefix", "regex")},
			},
			"match_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to match. Omit to match every path.",
			},
			"rewrite_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to send to the destination. Required when rewrite_path_type is exact, prefix or regex. Omit to keep the original path.",
			},
			"rewrite_path_type": schema.StringAttribute{
				Optional:    true,
				Description: "How rewrite_path is applied: exact, prefix, regex or stripPrefix. stripPrefix removes the matched prefix and takes no rewrite_path.",
				Validators:  []validator.String{stringvalidator.OneOf("exact", "prefix", "regex", "stripPrefix")},
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(100),
				Description: "Evaluation order, from 1 to 1000. Defaults to 100, which lets Pangolin order the redirect.",
				Validators:  []validator.Int64{int64validator.Between(1, 1000)},
			},
			"permanent": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Send a permanent redirect instead of a temporary one. Requires Badger 1.8.0 or later. Defaults to false.",
			},
			"ssl": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Serve the redirect's domain over TLS. Can only be turned off with domain_id; redirects on a resource follow the resource's ssl setting. Defaults to true.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the redirect is active. Defaults to true.",
			},
		},
	}
}

func (r *redirectResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config redirectModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.DomainID.IsNull() {
		if !config.Subdomain.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("subdomain"), "Invalid subdomain", "subdomain is only valid with domain_id.")
		}
		if known(config.SSL) && !config.SSL.ValueBool() {
			resp.Diagnostics.AddAttributeError(path.Root("ssl"), "Invalid ssl", "ssl can only be turned off with domain_id; redirects on a resource follow the resource's ssl setting.")
		}
	}

	if known(config.RewritePathType) && config.RewritePathType.ValueString() != "stripPrefix" && config.RewritePath.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("rewrite_path"), "Missing rewrite_path", "rewrite_path is required unless rewrite_path_type is stripPrefix.")
	}
}

func (m *redirectModel) request() client.RedirectRequest {
	return client.RedirectRequest{
		Name:            m.Name.ValueString(),
		ResourceID:      int64Ptr(m.ResourceID),
		DomainID:        strPtr(m.DomainID),
		Subdomain:       strPtr(m.Subdomain),
		DestinationHost: m.DestinationHost.ValueString(),
		PathMatchType:   strPtr(m.PathMatchType),
		MatchPath:       strPtr(m.MatchPath),
		RewritePath:     strPtr(m.RewritePath),
		RewritePathType: strPtr(m.RewritePathType),
		Priority:        int64Ptr(m.Priority),
		Permanent:       boolPtr(m.Permanent),
		SSL:             boolPtr(m.SSL),
		Enabled:         boolPtr(m.Enabled),
	}
}

func (m *redirectModel) fromAPI(redirect *client.Redirect) {
	m.RedirectID = types.Int64Value(redirect.RedirectID)
	m.NiceID = types.StringValue(redirect.NiceID)
	m.Name = types.StringValue(redirect.Name)
	m.ResourceID = nullableInt64(redirect.ResourceID)
	m.DomainID = nullableString(redirect.DomainID)
	m.Subdomain = nullableString(redirect.Subdomain)
	m.DestinationHost = types.StringValue(redirect.DestinationHost)
	m.PathMatchType = types.StringValue(redirect.PathMatchType)
	m.MatchPath = nullableString(redirect.MatchPath)
	m.RewritePath = nullableString(redirect.RewritePath)
	m.RewritePathType = nullableString(redirect.RewritePathType)
	m.Priority = nullableInt64(redirect.Priority)
	m.Permanent = types.BoolValue(redirect.Permanent)
	m.SSL = types.BoolValue(redirect.SSL)
	m.Enabled = types.BoolValue(redirect.Enabled)
}

func (r *redirectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan redirectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	redirect, err := r.client.CreateRedirect(ctx, plan.OrgID.ValueString(), plan.request())
	if err != nil {
		resp.Diagnostics.AddError("Error creating redirect", err.Error())
		return
	}
	plan.fromAPI(redirect)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redirectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state redirectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	redirect, err := r.client.GetRedirect(ctx, state.OrgID.ValueString(), state.RedirectID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading redirect", err.Error())
		return
	}
	state.fromAPI(redirect)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *redirectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state redirectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	redirect, err := r.client.UpdateRedirect(ctx, state.OrgID.ValueString(), state.RedirectID.ValueInt64(), plan.request())
	if err != nil {
		resp.Diagnostics.AddError("Error updating redirect", err.Error())
		return
	}
	plan.fromAPI(redirect)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redirectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state redirectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteRedirect(ctx, state.OrgID.ValueString(), state.RedirectID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error deleting redirect", err.Error())
	}
}

func (r *redirectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStringAndInt64(ctx, req.ID, resp, "org_id", "redirect_id")
}
