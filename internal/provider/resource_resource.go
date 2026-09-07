package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	client *client.Client
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
			"resource_id": schema.Int64Attribute{Computed: true, Description: "Server-generated resource ID."},
			"nice_id":     schema.StringAttribute{Computed: true, Description: "Human-readable ID, unique per org."},
			"name":        schema.StringAttribute{Required: true, Description: "Display name of the resource."},
			"mode": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "One of http, ssh, rdp, vnc, inference, tcp, udp. tcp and udp are raw (non-HTTP) resources: they require proxy_port, must not set domain_id/subdomain, and require the allow_raw_resources flag to be enabled in the Pangolin server config. inference is an AI gateway resource: it is domain-routed like http but may attach existing AI providers via ai_providers.",
			},
			"domain_id": schema.StringAttribute{
				Optional:    true,
				Description: "Domain ID this resource is served from. Required for every mode except tcp/udp, and must be omitted for tcp/udp.",
			},
			"subdomain":        schema.StringAttribute{Optional: true, Computed: true, Description: "Subdomain under domain_id. Not valid for tcp/udp resources."},
			"full_domain":      schema.StringAttribute{Computed: true, Description: "Fully-qualified domain this resource is reachable at. Empty for tcp/udp resources."},
			"sticky_session":   schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether to enable sticky sessions."},
			"post_auth_path":   schema.StringAttribute{Optional: true, Computed: true, Description: "Path to redirect to after authentication. Not valid for tcp/udp resources."},
			"pam_mode":         schema.StringAttribute{Optional: true, Computed: true, Description: "SSH PAM mode: passthrough or push. Only meaningful for mode = ssh."},
			"auth_daemon_mode": schema.StringAttribute{Optional: true, Computed: true, Description: "One of site, remote, native. Only meaningful for mode = ssh."},
			"auth_daemon_port": schema.Int64Attribute{Optional: true, Description: "Auth daemon port. Only meaningful for mode = ssh."},
			"enabled":          schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the resource is enabled."},
			"ssl":              schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether SSL is enabled. Not valid for tcp/udp resources."},

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

func (r *pangolinResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func isRawMode(mode string) bool {
	return mode == "tcp" || mode == "udp"
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

	isSet := func(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }

	if isRawMode(mode) {
		if config.ProxyPort.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("proxy_port"), "Missing proxy_port", fmt.Sprintf("proxy_port is required when mode is %q.", mode))
		}
		httpOnly := map[string]bool{
			"domain_id":               isSet(config.DomainID),
			"subdomain":               isSet(config.Subdomain),
			"post_auth_path":          isSet(config.PostAuthPath),
			"pam_mode":                isSet(config.PamMode),
			"auth_daemon_mode":        isSet(config.AuthDaemonMode),
			"auth_daemon_port":        isSet(config.AuthDaemonPort),
			"ssl":                     isSet(config.SSL),
			"sso":                     isSet(config.SSO),
			"email_whitelist_enabled": isSet(config.EmailWhitelistEnabled),
			"apply_rules":             isSet(config.ApplyRules),
			"skip_to_idp_id":          isSet(config.SkipToIdpID),
			"tls_server_name":         isSet(config.TLSServerName),
			"set_host_header":         isSet(config.SetHostHeader),
			"headers_json":            isSet(config.HeadersJSON),
		}
		for _, name := range []string{
			"domain_id", "subdomain", "post_auth_path", "pam_mode", "auth_daemon_mode", "auth_daemon_port", "ssl",
			"sso", "email_whitelist_enabled", "apply_rules", "skip_to_idp_id", "tls_server_name", "set_host_header", "headers_json",
		} {
			if httpOnly[name] {
				resp.Diagnostics.AddAttributeError(path.Root(name), "Invalid attribute for mode "+mode, fmt.Sprintf("%s is not valid when mode is %q.", name, mode))
			}
		}
		if isSet(config.AIProviders) && len(config.AIProviders.Elements()) > 0 {
			resp.Diagnostics.AddAttributeError(path.Root("ai_providers"), "Invalid attribute for mode "+mode, fmt.Sprintf("ai_providers is not valid when mode is %q.", mode))
		}
		return
	}

	if config.DomainID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Missing domain_id", fmt.Sprintf("domain_id is required when mode is %q.", mode))
	}
	rawOnly := map[string]bool{
		"proxy_port":             isSet(config.ProxyPort),
		"proxy_protocol":         isSet(config.ProxyProtocol),
		"proxy_protocol_version": isSet(config.ProxyProtocolVersion),
	}
	for _, name := range []string{"proxy_port", "proxy_protocol", "proxy_protocol_version"} {
		if rawOnly[name] {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Invalid attribute for mode "+mode, fmt.Sprintf("%s is only valid for raw tcp/udp resources, not mode %q.", name, mode))
		}
	}
	if mode != "inference" && isSet(config.AIProviders) && len(config.AIProviders.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(path.Root("ai_providers"), "Invalid attribute for mode "+mode, fmt.Sprintf("ai_providers is only valid when mode is \"inference\", not %q.", mode))
	}
}

// resourceHeadersFromPlan parses a JSON-string headers_json plan value into
// the wire format the API expects for update requests.
func resourceHeadersFromPlan(v types.String) ([]client.HCHeader, error) {
	if v.IsUnknown() || v.IsNull() || v.ValueString() == "" {
		return nil, nil
	}
	var headers []client.HCHeader
	if err := json.Unmarshal([]byte(v.ValueString()), &headers); err != nil {
		return nil, fmt.Errorf("headers_json must be a JSON array of {name, value} objects: %w", err)
	}
	return headers, nil
}

// resourceHeadersToState re-encodes a PangolinResource's Headers (which may
// have come back from the API as a real array or as a JSON-encoded string,
// depending on the endpoint) into a canonical JSON string for state.
func resourceHeadersToState(raw any) (types.String, error) {
	headers, err := client.DecodeHCHeaders(raw)
	if err != nil {
		return types.StringNull(), err
	}
	if len(headers) == 0 {
		return types.StringValue("[]"), nil
	}
	b, err := json.Marshal(headers)
	if err != nil {
		return types.StringNull(), err
	}
	return types.StringValue(string(b)), nil
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
		if !plan.ProxyPort.IsNull() {
			v := plan.ProxyPort.ValueInt64()
			in.ProxyPort = &v
		}
	} else {
		in.DomainID = plan.DomainID.ValueString()
		if !plan.Subdomain.IsUnknown() {
			in.Subdomain = plan.Subdomain.ValueString()
		}
		if !plan.StickySession.IsUnknown() && !plan.StickySession.IsNull() {
			v := plan.StickySession.ValueBool()
			in.StickySession = &v
		}
		in.PostAuthPath = plan.PostAuthPath.ValueString()
		in.PamMode = plan.PamMode.ValueString()
		in.AuthDaemonMode = plan.AuthDaemonMode.ValueString()
		if !plan.AuthDaemonPort.IsNull() {
			v := plan.AuthDaemonPort.ValueInt64()
			in.AuthDaemonPort = &v
		}
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
		// sso, email_whitelist_enabled, apply_rules, skip_to_idp_id,
		// tls_server_name, set_host_header and headers_json aren't part of
		// the create request (the API doesn't accept them there), and
		// CreateResource's response reports their zero value regardless of
		// the resource's actual (default-policy-derived) state. A follow-up
		// call is required either way: to apply any of them the plan
		// configured, or - if none were configured - to read back their true
		// values instead of the misleading zeroes from the create response.
		update, hasUpdate, diags := followUpUpdateForNewResource(&plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if hasUpdate {
			updated, err := r.client.UpdateResource(ctx, created.ResourceID, update)
			if err != nil {
				resp.Diagnostics.AddError("Error applying initial settings to resource", err.Error())
				return
			}
			final = updated
		} else {
			refreshed, err := r.client.GetResource(ctx, created.ResourceID)
			if err != nil {
				resp.Diagnostics.AddError("Error reading newly created resource", err.Error())
				return
			}
			final = refreshed
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

// followUpUpdateForNewResource builds an UpdateResourceRequest containing
// only the update-only fields (sso, email_whitelist_enabled, apply_rules,
// skip_to_idp_id, tls_server_name, set_host_header, headers_json) the plan
// explicitly configures. hasUpdate is false when none were configured.
func followUpUpdateForNewResource(plan *pangolinResourceModel) (client.UpdateResourceRequest, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	var update client.UpdateResourceRequest
	hasUpdate := false

	if !plan.SSO.IsUnknown() && !plan.SSO.IsNull() {
		v := plan.SSO.ValueBool()
		update.SSO = &v
		hasUpdate = true
	}
	if !plan.EmailWhitelistEnabled.IsUnknown() && !plan.EmailWhitelistEnabled.IsNull() {
		v := plan.EmailWhitelistEnabled.ValueBool()
		update.EmailWhitelistEnabled = &v
		hasUpdate = true
	}
	if !plan.ApplyRules.IsUnknown() && !plan.ApplyRules.IsNull() {
		v := plan.ApplyRules.ValueBool()
		update.ApplyRules = &v
		hasUpdate = true
	}
	if !plan.SkipToIdpID.IsUnknown() && !plan.SkipToIdpID.IsNull() {
		v := plan.SkipToIdpID.ValueInt64()
		update.SkipToIdpID = &v
		hasUpdate = true
	}
	if !plan.TLSServerName.IsUnknown() && !plan.TLSServerName.IsNull() {
		v := plan.TLSServerName.ValueString()
		update.TLSServerName = &v
		hasUpdate = true
	}
	if !plan.SetHostHeader.IsUnknown() && !plan.SetHostHeader.IsNull() {
		v := plan.SetHostHeader.ValueString()
		update.SetHostHeader = &v
		hasUpdate = true
	}
	if !plan.HeadersJSON.IsUnknown() && !plan.HeadersJSON.IsNull() {
		headers, err := resourceHeadersFromPlan(plan.HeadersJSON)
		if err != nil {
			diags.AddError("Invalid headers_json", err.Error())
			return update, false, diags
		}
		update.Headers = headers
		hasUpdate = true
	}

	return update, hasUpdate, diags
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
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.StickySession.IsUnknown() {
		v := plan.StickySession.ValueBool()
		in.StickySession = &v
	}

	if isRawMode(mode) {
		if !plan.ProxyPort.IsNull() {
			v := plan.ProxyPort.ValueInt64()
			in.ProxyPort = &v
		}
		if !plan.ProxyProtocol.IsUnknown() {
			v := plan.ProxyProtocol.ValueBool()
			in.ProxyProtocol = &v
		}
		if !plan.ProxyProtocolVersion.IsUnknown() {
			v := plan.ProxyProtocolVersion.ValueInt64()
			in.ProxyProtocolVersion = &v
		}
	} else {
		if !plan.Subdomain.IsUnknown() {
			v := plan.Subdomain.ValueString()
			in.Subdomain = &v
		}
		if !plan.SSL.IsUnknown() {
			v := plan.SSL.ValueBool()
			in.SSL = &v
		}
		domainID := plan.DomainID.ValueString()
		in.DomainID = &domainID
		postAuthPath := plan.PostAuthPath.ValueString()
		in.PostAuthPath = &postAuthPath
		pamMode := plan.PamMode.ValueString()
		in.PamMode = &pamMode
		authDaemonMode := plan.AuthDaemonMode.ValueString()
		in.AuthDaemonMode = &authDaemonMode
		if !plan.AuthDaemonPort.IsNull() {
			v := plan.AuthDaemonPort.ValueInt64()
			in.AuthDaemonPort = &v
		}
		if !plan.SSO.IsUnknown() {
			v := plan.SSO.ValueBool()
			in.SSO = &v
		}
		if !plan.EmailWhitelistEnabled.IsUnknown() {
			v := plan.EmailWhitelistEnabled.ValueBool()
			in.EmailWhitelistEnabled = &v
		}
		if !plan.ApplyRules.IsUnknown() {
			v := plan.ApplyRules.ValueBool()
			in.ApplyRules = &v
		}
		if !plan.SkipToIdpID.IsUnknown() && !plan.SkipToIdpID.IsNull() {
			v := plan.SkipToIdpID.ValueInt64()
			in.SkipToIdpID = &v
		}
		if !plan.TLSServerName.IsUnknown() {
			v := plan.TLSServerName.ValueString()
			in.TLSServerName = &v
		}
		if !plan.SetHostHeader.IsUnknown() {
			v := plan.SetHostHeader.ValueString()
			in.SetHostHeader = &v
		}
		if !plan.HeadersJSON.IsUnknown() {
			headers, err := resourceHeadersFromPlan(plan.HeadersJSON)
			if err != nil {
				resp.Diagnostics.AddError("Invalid headers_json", err.Error())
				return
			}
			in.Headers = headers
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
	if err := r.client.DeleteResource(ctx, state.ResourceID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
	}
}

func (r *pangolinResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", `expected format: <org_id>:<resource_id>, e.g. "acme:5"`)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "resource_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), id)...)
	if resp.Diagnostics.HasError() {
		return
	}
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
	if res.AuthDaemonPort != 0 {
		model.AuthDaemonPort = types.Int64Value(res.AuthDaemonPort)
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
	} else {
		model.ProxyPort = types.Int64Null()
		model.ProxyProtocol = types.BoolNull()
		model.ProxyProtocolVersion = types.Int64Null()

		model.SSO = types.BoolValue(res.SSO)
		model.EmailWhitelistEnabled = types.BoolValue(res.EmailWhitelistEnabled)
		model.ApplyRules = types.BoolValue(res.ApplyRules)
		if res.SkipToIdpID != 0 {
			model.SkipToIdpID = types.Int64Value(res.SkipToIdpID)
		} else {
			model.SkipToIdpID = types.Int64Null()
		}
		model.TLSServerName = types.StringValue(res.TLSServerName)
		model.SetHostHeader = types.StringValue(res.SetHostHeader)

		headersJSON, err := resourceHeadersToState(res.Headers)
		if err != nil {
			diags.AddError("Error decoding headers_json", err.Error())
			return diags
		}
		model.HeadersJSON = headersJSON
	}

	return diags
}
