package provider

import (
	"context"
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
			"domain_id":        isSet(config.DomainID),
			"subdomain":        isSet(config.Subdomain),
			"post_auth_path":   isSet(config.PostAuthPath),
			"pam_mode":         isSet(config.PamMode),
			"auth_daemon_mode": isSet(config.AuthDaemonMode),
			"auth_daemon_port": isSet(config.AuthDaemonPort),
			"ssl":              isSet(config.SSL),
		}
		for _, name := range []string{"domain_id", "subdomain", "post_auth_path", "pam_mode", "auth_daemon_mode", "auth_daemon_port", "ssl"} {
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

	setPangolinResourceModelFromAPI(&plan, created)

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

	setPangolinResourceModelFromAPI(&state, res)

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
	}

	updated, err := r.client.UpdateResource(ctx, state.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	setPangolinResourceModelFromAPI(&plan, updated)

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

func setPangolinResourceModelFromAPI(model *pangolinResourceModel, res *client.PangolinResource) {
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
	} else {
		model.ProxyPort = types.Int64Null()
		model.ProxyProtocol = types.BoolNull()
		model.ProxyProtocolVersion = types.Int64Null()
	}
}
