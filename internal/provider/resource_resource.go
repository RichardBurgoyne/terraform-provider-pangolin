package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                   = &pangolinResourceResource{}
	_ resource.ResourceWithImportState    = &pangolinResourceResource{}
	_ resource.ResourceWithValidateConfig = &pangolinResourceResource{}
)

func NewPangolinResourceResource() resource.Resource { return &pangolinResourceResource{} }

type pangolinResourceResource struct {
	resourceClient
}

type resourceAIProviderModel struct {
	ProviderID types.Int64  `tfsdk:"provider_id"`
	AccessMode types.String `tfsdk:"access_mode"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func aiProviderObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"provider_id": types.Int64Type,
			"access_mode": types.StringType,
			"enabled":     types.BoolType,
		},
	}
}

type pangolinResourceModel struct {
	OrgID          types.String `tfsdk:"org_id"`
	ResourceID     types.Int64  `tfsdk:"resource_id"`
	NiceID         types.String `tfsdk:"nice_id"`
	Name           types.String `tfsdk:"name"`
	Mode           types.String `tfsdk:"mode"`
	DomainID       types.String `tfsdk:"domain_id"`
	Subdomain      types.String `tfsdk:"subdomain"`
	FullDomain     types.String `tfsdk:"full_domain"`
	StickySession  types.Bool   `tfsdk:"sticky_session"`
	PostAuthPath   types.String `tfsdk:"post_auth_path"`
	PamMode        types.String `tfsdk:"pam_mode"`
	AuthDaemonMode types.String `tfsdk:"auth_daemon_mode"`
	AuthDaemonPort types.Int64  `tfsdk:"auth_daemon_port"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	SSL            types.Bool   `tfsdk:"ssl"`

	// Raw tcp/udp fields. Only valid when mode is tcp or udp; must be left
	// unset for every other mode. Raw resources also require the
	// `allow_raw_resources` flag to be enabled in the Pangolin server config.
	ProxyPort            types.Int64 `tfsdk:"proxy_port"`
	ProxyProtocol        types.Bool  `tfsdk:"proxy_protocol"`
	ProxyProtocolVersion types.Int64 `tfsdk:"proxy_protocol_version"`

	// Inference mode fields. Only valid when mode is inference.
	AIProviders types.List `tfsdk:"ai_providers"`

	// Update-only http-family fields. See the comment on
	// client.PangolinResource for the default-policy indirection affecting
	// SSO/EmailWhitelistEnabled/ApplyRules/SkipToIdpID. Not valid for raw
	// tcp/udp resources.
	SSO                   types.Bool   `tfsdk:"sso"`
	EmailWhitelistEnabled types.Bool   `tfsdk:"email_whitelist_enabled"`
	ApplyRules            types.Bool   `tfsdk:"apply_rules"`
	SkipToIdpID           types.Int64  `tfsdk:"skip_to_idp_id"`
	TLSServerName         types.String `tfsdk:"tls_server_name"`
	SetHostHeader         types.String `tfsdk:"set_host_header"`
	HeadersJSON           types.String `tfsdk:"headers_json"`
	ResponseHeadersJSON   types.String `tfsdk:"response_headers_json"`

	// Update-only http-family fields added in Pangolin 1.25.
	LoadBalancingStrategy  types.String `tfsdk:"load_balancing_strategy"`
	ForwardIdentityHeaders types.Bool   `tfsdk:"forward_identity_headers"`
}

func (r *pangolinResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *pangolinResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin resource (a proxied endpoint). mode and domain_id changes replace the resource. Import using the format `<org_id>:<resource_id>`, e.g. `terraform import pangolin_resource.example acme:5`.",
		Attributes: map[string]schema.Attribute{
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this resource belongs to."},
			"resource_id": schema.Int64Attribute{Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Description: "Server-generated resource ID."},
			"nice_id":     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Description: "Human-readable ID, unique per org."},
			"name":        schema.StringAttribute{Required: true, Description: "Display name of the resource."},
			"mode": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "One of http, ssh, rdp, vnc, inference, tcp, udp. tcp and udp are raw (non-HTTP) resources: they require proxy_port, must not set domain_id/subdomain, and require the allow_raw_resources flag to be enabled in the Pangolin server config. inference is an AI gateway resource: it is domain-routed like http but may attach existing AI providers via ai_providers.",
				Validators:    []validator.String{stringvalidator.OneOf("http", "ssh", "rdp", "vnc", "inference", "tcp", "udp")},
			},
			"domain_id": schema.StringAttribute{
				Optional:    true,
				Description: "Domain ID this resource is served from. Required for every mode except tcp/udp, and must be omitted for tcp/udp.",
			},
			"subdomain":        schema.StringAttribute{Optional: true, Computed: true, Description: "Subdomain under domain_id. Not valid for tcp/udp resources."},
			"full_domain":      schema.StringAttribute{Computed: true, Description: "Fully-qualified domain this resource is reachable at. Empty for tcp/udp resources."},
			"sticky_session":   schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether to enable sticky sessions."},
			"post_auth_path":   schema.StringAttribute{Optional: true, Computed: true, Description: "Path to redirect to after authentication. Not valid for tcp/udp resources."},
			"pam_mode":         schema.StringAttribute{Optional: true, Computed: true, Description: "SSH PAM mode: passthrough or push. Only meaningful for mode = ssh.", Validators: []validator.String{stringvalidator.OneOf("passthrough", "push")}},
			"auth_daemon_mode": schema.StringAttribute{Optional: true, Computed: true, Description: "One of site, remote, native. Only meaningful for mode = ssh.", Validators: []validator.String{stringvalidator.OneOf("site", "remote", "native")}},
			"auth_daemon_port": schema.Int64Attribute{Optional: true, Description: "Auth daemon port. Only meaningful for mode = ssh."},
			"enabled":          schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the resource is enabled."},
			"ssl":              schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether SSL is enabled. Update-only: not settable at creation, so a newly created resource keeps the server's default (true) until first set. Not valid for tcp/udp resources."},

			"proxy_port": schema.Int64Attribute{
				Optional:    true,
				Description: "Public port for a raw tcp/udp resource. Required when mode is tcp or udp; must be omitted otherwise.",
			},
			"proxy_protocol": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to send the PROXY protocol header to targets. Only valid for tcp/udp resources.",
			},
			"proxy_protocol_version": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "PROXY protocol version (1 or 2) to send. Only valid for tcp/udp resources.",
			},

			"sso": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether SSO is required to access this resource. Update-only: stored on the resource's default policy, so it retains the server default (true) until first set. Not valid for tcp/udp resources.",
			},
			"email_whitelist_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether access is restricted to whitelisted emails. Update-only (see sso). Not valid for tcp/udp resources.",
			},
			"apply_rules": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether resource rules (IP/path-based access rules) are applied to this resource. Update-only (see sso). Not valid for tcp/udp resources.",
			},
			"skip_to_idp_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "IdP ID to skip directly to for authentication, bypassing the login page's IdP picker. Update-only (see sso). Not valid for tcp/udp resources.",
			},
			"tls_server_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "TLS server name (SNI) to present to the target. Update-only: not settable at creation. Not valid for tcp/udp resources.",
			},
			"set_host_header": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Custom Host header to send to the target. Update-only: not settable at creation. Not valid for tcp/udp resources.",
			},
			"headers_json": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Extra headers sent to the target, as a JSON array of {name, value} objects, e.g. `[{\"name\":\"X-Foo\",\"value\":\"bar\"}]`. Modeled as a JSON string rather than a nested list because the Pangolin API itself is inconsistent about this field's wire format across endpoints (same reasoning as pangolin_target's hc_headers_json). Update-only: not settable at creation. Not valid for tcp/udp resources.",
			},
			"response_headers_json": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Extra headers added to responses sent back to the client, as a JSON array of {name, value} objects, e.g. `[{\"name\":\"X-Frame-Options\",\"value\":\"DENY\"}]`. Requires Pangolin 1.24 or later. Update-only: not settable at creation. Not valid for tcp/udp resources.",
			},

			"load_balancing_strategy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "How requests are distributed across targets: wrr (weighted round robin, the server default), p2c (power of two choices), hrw (highest random weight, hashing the client IP) or leasttime (lowest response time). Requires Pangolin 1.25 or later and Traefik 3.6 or later. Update-only: not settable at creation. Not valid for tcp/udp resources.",
				Validators:  []validator.String{stringvalidator.OneOf("wrr", "p2c", "hrw", "leasttime")},
			},
			"forward_identity_headers": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to send the authenticated user's identity to the backend in the Remote-User, Remote-Email, Remote-Name, Remote-Role and Remote-User-Id headers. Requires Pangolin 1.25 or later. Update-only: not settable at creation. Not valid for tcp/udp resources.",
			},

			"ai_providers": schema.ListNestedAttribute{
				Optional:    true,
				Description: "AI providers to attach to an inference-mode resource. Providers must already exist in Pangolin (this provider does not manage them). Only valid when mode is inference.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"provider_id": schema.Int64Attribute{Required: true, Description: "ID of an existing AI provider in this organization."},
						"access_mode": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "inherit (default, uses the provider's own allow/block lists) or select (uses this resource's selected subset of the provider's catalog).",
						},
						"enabled": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this attachment is enabled."},
					},
				},
			},
		},
	}
}

func isRawMode(mode string) bool {
	return mode == "tcp" || mode == "udp"
}

// modeScopedAttribute pairs an attribute with its config value for the
// per-mode validity checks in ValidateConfig.
type modeScopedAttribute struct {
	name  string
	value attr.Value
}

func (r *pangolinResourceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config pangolinResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Mode.IsUnknown() || config.Mode.IsNull() {
		return
	}
	mode := config.Mode.ValueString()

	httpOnly := []modeScopedAttribute{
		{"domain_id", config.DomainID},
		{"subdomain", config.Subdomain},
		{"post_auth_path", config.PostAuthPath},
		{"pam_mode", config.PamMode},
		{"auth_daemon_mode", config.AuthDaemonMode},
		{"auth_daemon_port", config.AuthDaemonPort},
		{"ssl", config.SSL},
		{"sso", config.SSO},
		{"email_whitelist_enabled", config.EmailWhitelistEnabled},
		{"apply_rules", config.ApplyRules},
		{"skip_to_idp_id", config.SkipToIdpID},
		{"tls_server_name", config.TLSServerName},
		{"set_host_header", config.SetHostHeader},
		{"headers_json", config.HeadersJSON},
		{"response_headers_json", config.ResponseHeadersJSON},
		{"load_balancing_strategy", config.LoadBalancingStrategy},
		{"forward_identity_headers", config.ForwardIdentityHeaders},
	}
	rawOnly := []modeScopedAttribute{
		{"proxy_port", config.ProxyPort},
		{"proxy_protocol", config.ProxyProtocol},
		{"proxy_protocol_version", config.ProxyProtocolVersion},
	}
	hasAIProviders := known(config.AIProviders) && len(config.AIProviders.Elements()) > 0

	if isRawMode(mode) {
		if config.ProxyPort.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("proxy_port"), "Missing proxy_port", fmt.Sprintf("proxy_port is required when mode is %q.", mode))
		}
		for _, a := range httpOnly {
			if known(a.value) {
				resp.Diagnostics.AddAttributeError(path.Root(a.name), "Invalid attribute for mode "+mode, fmt.Sprintf("%s is not valid when mode is %q.", a.name, mode))
			}
		}
		if hasAIProviders {
			resp.Diagnostics.AddAttributeError(path.Root("ai_providers"), "Invalid attribute for mode "+mode, fmt.Sprintf("ai_providers is not valid when mode is %q.", mode))
		}
		return
	}

	if config.DomainID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Missing domain_id", fmt.Sprintf("domain_id is required when mode is %q.", mode))
	}
	for _, a := range rawOnly {
		if known(a.value) {
			resp.Diagnostics.AddAttributeError(path.Root(a.name), "Invalid attribute for mode "+mode, fmt.Sprintf("%s is only valid for raw tcp/udp resources, not mode %q.", a.name, mode))
		}
	}
	if mode != "inference" && hasAIProviders {
		resp.Diagnostics.AddAttributeError(path.Root("ai_providers"), "Invalid attribute for mode "+mode, fmt.Sprintf("ai_providers is only valid when mode is \"inference\", not %q.", mode))
	}
}

func aiProvidersFromPlan(ctx context.Context, list types.List) ([]client.ResourceAIProviderAttachment, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var items []resourceAIProviderModel
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if diags.HasError() {
		return nil, diags
	}
	attachments := make([]client.ResourceAIProviderAttachment, 0, len(items))
	for _, item := range items {
		a := client.ResourceAIProviderAttachment{ProviderID: item.ProviderID.ValueInt64()}
		if !item.AccessMode.IsUnknown() && !item.AccessMode.IsNull() {
			a.AccessMode = item.AccessMode.ValueString()
		}
		if !item.Enabled.IsUnknown() && !item.Enabled.IsNull() {
			v := item.Enabled.ValueBool()
			a.Enabled = &v
		}
		attachments = append(attachments, a)
	}
	return attachments, diags
}

func aiProvidersToState(ctx context.Context, providers []client.ResourceAIProvider) (types.List, diag.Diagnostics) {
	objType := aiProviderObjectType()
	if len(providers) == 0 {
		return types.ListNull(objType), nil
	}
	items := make([]resourceAIProviderModel, 0, len(providers))
	for _, p := range providers {
		items = append(items, resourceAIProviderModel{
			ProviderID: types.Int64Value(p.ProviderID),
			AccessMode: types.StringValue(p.AccessMode),
			Enabled:    types.BoolValue(p.Enabled),
		})
	}
	return types.ListValueFrom(ctx, objType, items)
}

func (r *pangolinResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pangolinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mode := plan.Mode.ValueString()
	in := client.CreateResourceRequest{
		Name: plan.Name.ValueString(),
		Mode: mode,
	}

	if isRawMode(mode) {
		in.ProxyPort = int64Ptr(plan.ProxyPort)
	} else {
		in.DomainID = plan.DomainID.ValueString()
		if !plan.Subdomain.IsUnknown() {
			in.Subdomain = plan.Subdomain.ValueString()
		}
		in.StickySession = boolPtr(plan.StickySession)
		in.PostAuthPath = plan.PostAuthPath.ValueString()
		in.PamMode = plan.PamMode.ValueString()
		in.AuthDaemonMode = plan.AuthDaemonMode.ValueString()
		in.AuthDaemonPort = int64Ptr(plan.AuthDaemonPort)
		if mode == "inference" {
			attachments, diags := aiProvidersFromPlan(ctx, plan.AIProviders)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			in.AIProviders = attachments
		}
	}

	created, err := r.client.CreateResource(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	final := created
	if !isRawMode(mode) {
		// The attributes set by setUpdateOnlyFields aren't part of the create
		// request (the API doesn't accept them there), and
		// CreateResource's response reports their zero value regardless of
		// the resource's actual (default-policy-derived) state. A follow-up
		// call is required either way: to apply any of them the plan
		// configured, or - if none were configured - to read back their true
		// values instead of the misleading zeroes from the create response.
		var update client.UpdateResourceRequest
		resp.Diagnostics.Append(setUpdateOnlyFields(&update, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var err error
		if update == (client.UpdateResourceRequest{}) {
			final, err = r.client.GetResource(ctx, created.ResourceID)
		} else {
			final, err = r.client.UpdateResource(ctx, created.ResourceID, update)
		}
		if err != nil {
			resp.Diagnostics.AddError("Error applying initial settings to resource", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(setPangolinResourceModelFromAPI(&plan, final)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if mode == "inference" {
		resp.Diagnostics.Append(r.refreshAIProviders(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		plan.AIProviders = types.ListNull(aiProviderObjectType())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// setUpdateOnlyFields copies the http-family fields that the API only accepts
// on update (not create) from the plan into in. Null and unknown values are
// left out, so on a new resource only what the configuration sets is sent.
func setUpdateOnlyFields(in *client.UpdateResourceRequest, plan *pangolinResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	in.SSL = boolPtr(plan.SSL)
	in.SSO = boolPtr(plan.SSO)
	in.EmailWhitelistEnabled = boolPtr(plan.EmailWhitelistEnabled)
	in.ApplyRules = boolPtr(plan.ApplyRules)
	in.SkipToIdpID = int64Ptr(plan.SkipToIdpID)
	in.TLSServerName = strPtr(plan.TLSServerName)
	in.SetHostHeader = strPtr(plan.SetHostHeader)
	in.LoadBalancingStrategy = strPtr(plan.LoadBalancingStrategy)
	in.ForwardIdentityHeaders = boolPtr(plan.ForwardIdentityHeaders)
	if known(plan.HeadersJSON) {
		headers, err := parseHeadersJSON(plan.HeadersJSON)
		if err != nil {
			diags.AddError("Invalid headers_json", err.Error())
			return diags
		}
		in.RequestHeaders = &headers
	}
	if known(plan.ResponseHeadersJSON) {
		headers, err := parseHeadersJSON(plan.ResponseHeadersJSON)
		if err != nil {
			diags.AddError("Invalid response_headers_json", err.Error())
			return diags
		}
		in.ResponseHeaders = &headers
	}
	return diags
}

func (r *pangolinResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pangolinResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetResource(ctx, state.ResourceID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	resp.Diagnostics.Append(setPangolinResourceModelFromAPI(&state, res)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if res.Mode == "inference" {
		resp.Diagnostics.Append(r.refreshAIProviders(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		state.AIProviders = types.ListNull(aiProviderObjectType())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pangolinResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pangolinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mode := state.Mode.ValueString() // mode is RequiresReplace, so plan and state agree.

	in := client.UpdateResourceRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	in.Enabled = boolPtr(plan.Enabled)
	in.StickySession = boolPtr(plan.StickySession)

	if isRawMode(mode) {
		in.ProxyPort = int64Ptr(plan.ProxyPort)
		in.ProxyProtocol = boolPtr(plan.ProxyProtocol)
		in.ProxyProtocolVersion = int64Ptr(plan.ProxyProtocolVersion)
	} else {
		in.Subdomain = strPtr(plan.Subdomain)
		domainID := plan.DomainID.ValueString()
		in.DomainID = &domainID
		in.PostAuthPath = strPtr(plan.PostAuthPath)
		if !plan.PamMode.IsUnknown() && plan.PamMode.ValueString() != "" {
			v := plan.PamMode.ValueString()
			in.PamMode = &v
		}
		if !plan.AuthDaemonMode.IsUnknown() && plan.AuthDaemonMode.ValueString() != "" {
			v := plan.AuthDaemonMode.ValueString()
			in.AuthDaemonMode = &v
		}
		in.AuthDaemonPort = int64Ptr(plan.AuthDaemonPort)
		resp.Diagnostics.Append(setUpdateOnlyFields(&in, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updated, err := r.client.UpdateResource(ctx, state.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	resp.Diagnostics.Append(setPangolinResourceModelFromAPI(&plan, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if mode == "inference" {
		attachments, diags := aiProvidersFromPlan(ctx, plan.AIProviders)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetResourceAIProviders(ctx, state.ResourceID.ValueInt64(), attachments); err != nil {
			resp.Diagnostics.AddError("Error setting AI providers for resource", err.Error())
			return
		}
		resp.Diagnostics.Append(r.refreshAIProviders(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		plan.AIProviders = types.ListNull(aiProviderObjectType())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pangolinResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pangolinResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteResource(ctx, state.ResourceID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
	}
}

func (r *pangolinResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStringAndInt64(ctx, req.ID, resp, "org_id", "resource_id")
}

// refreshAIProviders fetches the current AI provider attachments for an
// inference-mode resource and writes them into model.AIProviders.
func (r *pangolinResourceResource) refreshAIProviders(ctx context.Context, model *pangolinResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	providers, err := r.client.ListResourceAIProviders(ctx, model.ResourceID.ValueInt64())
	if err != nil {
		diags.AddError("Error reading AI providers for resource", err.Error())
		return diags
	}
	list, d := aiProvidersToState(ctx, providers)
	diags.Append(d...)
	model.AIProviders = list
	return diags
}

func setPangolinResourceModelFromAPI(model *pangolinResourceModel, res *client.PangolinResource) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ResourceID = types.Int64Value(res.ResourceID)
	model.NiceID = types.StringValue(res.NiceID)
	model.Name = types.StringValue(res.Name)
	model.Mode = types.StringValue(res.Mode)
	if isRawMode(res.Mode) {
		model.DomainID = types.StringNull()
		model.Subdomain = types.StringNull()
	} else {
		model.DomainID = types.StringValue(res.DomainID)
		model.Subdomain = types.StringValue(res.Subdomain)
	}
	model.FullDomain = types.StringValue(res.FullDomain)
	model.StickySession = types.BoolValue(res.StickySession)
	model.PostAuthPath = types.StringValue(res.PostAuthPath)
	model.PamMode = types.StringValue(res.PamMode)
	model.AuthDaemonMode = types.StringValue(res.AuthDaemonMode)
	// The API always returns a nonzero authDaemonPort, even for non-ssh
	// resources where it's meaningless internal plumbing. auth_daemon_port
	// isn't Computed in the schema, so reflecting that value for a mode
	// where the user never configured it would make Terraform see the
	// provider producing a value the plan said would stay null.
	if res.Mode == "ssh" && res.AuthDaemonPort != 0 {
		model.AuthDaemonPort = types.Int64Value(res.AuthDaemonPort)
	} else {
		model.AuthDaemonPort = types.Int64Null()
	}
	model.Enabled = types.BoolValue(res.Enabled)
	model.SSL = types.BoolValue(res.SSL)

	if isRawMode(res.Mode) {
		if res.ProxyPort != 0 {
			model.ProxyPort = types.Int64Value(res.ProxyPort)
		}
		model.ProxyProtocol = types.BoolValue(res.ProxyProtocol)
		model.ProxyProtocolVersion = types.Int64Value(res.ProxyProtocolVersion)

		model.SSO = types.BoolNull()
		model.EmailWhitelistEnabled = types.BoolNull()
		model.ApplyRules = types.BoolNull()
		model.SkipToIdpID = types.Int64Null()
		model.TLSServerName = types.StringNull()
		model.SetHostHeader = types.StringNull()
		model.HeadersJSON = types.StringNull()
		model.ResponseHeadersJSON = types.StringNull()
		model.LoadBalancingStrategy = types.StringNull()
		model.ForwardIdentityHeaders = types.BoolNull()
	} else {
		model.ProxyPort = types.Int64Null()
		model.ProxyProtocol = types.BoolNull()
		model.ProxyProtocolVersion = types.Int64Null()

		model.SSO = types.BoolValue(res.SSO)
		model.EmailWhitelistEnabled = types.BoolValue(res.EmailWhitelistEnabled)
		model.ApplyRules = types.BoolValue(res.ApplyRules)
		model.SkipToIdpID = nonZeroInt64(res.SkipToIdpID)
		model.TLSServerName = types.StringValue(res.TLSServerName)
		model.SetHostHeader = types.StringValue(res.SetHostHeader)
		model.LoadBalancingStrategy = types.StringValue(res.LoadBalancingStrategy)
		model.ForwardIdentityHeaders = types.BoolValue(res.ForwardIdentityHeaders)

		// Pangolin 1.24 renamed headers to requestHeaders; fall back to the
		// legacy field for older servers.
		rawRequest := res.RequestHeaders
		if rawRequest == nil {
			rawRequest = res.Headers
		}
		headersJSON, err := headersJSONToState(rawRequest)
		if err != nil {
			diags.AddError("Error decoding headers_json", err.Error())
			return diags
		}
		model.HeadersJSON = headersJSON

		responseJSON, err := headersJSONToState(res.ResponseHeaders)
		if err != nil {
			diags.AddError("Error decoding response_headers_json", err.Error())
			return diags
		}
		model.ResponseHeadersJSON = responseJSON
	}

	return diags
}
