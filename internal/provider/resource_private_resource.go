package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &privateResourceResource{}
	_ resource.ResourceWithImportState = &privateResourceResource{}
)

func NewPrivateResourceResource() resource.Resource { return &privateResourceResource{} }

type privateResourceResource struct {
	client *client.Client
}

type privateResourceModel struct {
	OrgID             types.String `tfsdk:"org_id"`
	PrivateResourceID types.Int64  `tfsdk:"private_resource_id"`
	NiceID            types.String `tfsdk:"nice_id"`
	Name              types.String `tfsdk:"name"`
	Mode              types.String `tfsdk:"mode"`
	SiteIDs           types.Set    `tfsdk:"site_ids"`
	Destination       types.String `tfsdk:"destination"`
	DestinationPort   types.Int64  `tfsdk:"destination_port"`
	Scheme            types.String `tfsdk:"scheme"`
	SSL               types.Bool   `tfsdk:"ssl"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Alias             types.String `tfsdk:"alias"`
	AliasAddress      types.String `tfsdk:"alias_address"`
	TCPPortRange      types.String `tfsdk:"tcp_port_range"`
	UDPPortRange      types.String `tfsdk:"udp_port_range"`
	DisableICMP       types.Bool   `tfsdk:"disable_icmp"`
	DomainID          types.String `tfsdk:"domain_id"`
	Subdomain         types.String `tfsdk:"subdomain"`
	FullDomain        types.String `tfsdk:"full_domain"`
	AuthDaemonPort    types.Int64  `tfsdk:"auth_daemon_port"`
	AuthDaemonMode    types.String `tfsdk:"auth_daemon_mode"`
	PamMode           types.String `tfsdk:"pam_mode"`
	UserIDs           types.Set    `tfsdk:"user_ids"`
	RoleIDs           types.Set    `tfsdk:"role_ids"`
	ClientIDs         types.Set    `tfsdk:"client_ids"`
}

func (r *privateResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_resource"
}

func (r *privateResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	replaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	emptyStringSet := setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{}))
	emptyInt64Set := setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{}))

	resp.Schema = schema.Schema{
		Description: "Manages a private resource (the API calls it a site resource): a host, network range, web service, SSH target or gateway that Pangolin clients reach through one or more sites. Changing org_id, mode or nice_id replaces the resource. Import using the format `<org_id>:<private_resource_id>`, e.g. `terraform import pangolin_private_resource.example acme:12`.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{Required: true, PlanModifiers: replaceStr, Description: "Organization the resource belongs to."},
			"private_resource_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Pangolin-assigned numeric ID.",
			},
			"nice_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Description:   "Human-readable ID, unique per org. Generated when omitted.",
			},
			"name": schema.StringAttribute{Required: true, Description: "Display name."},
			"mode": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replaceStr,
				Description:   "What kind of private resource this is: `host` (a single IP or hostname), `cidr` (a network range), `http` (a web service published on a private domain), `ssh` (an SSH target), or `gateway` (route all traffic through the site).",
				Validators:    []validator.String{stringvalidator.OneOf("host", "cidr", "http", "ssh", "gateway")},
			},
			"site_ids": schema.SetAttribute{
				ElementType: types.Int64Type,
				Required:    true,
				Description: "IDs of the sites the resource is reachable through. SSH resources in native or push mode accept a single site.",
			},
			"destination": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Where traffic is sent: an IPv4 address or hostname (`host`, `ssh`; hostnames also need `alias`), a CIDR (`cidr`), or the backend host for `http`. Not required for `gateway` or native-auth `ssh`.",
			},
			"destination_port": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Backend port. Required for `http` mode, optional for `ssh`.",
			},
			"scheme": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Backend scheme for `http` mode: `http` or `https`.",
				Validators:    []validator.String{stringvalidator.OneOf("http", "https")},
			},
			"ssl": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Serve the private domain over TLS. Only used by `http` mode. Defaults to false.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the resource is active. Defaults to true.",
			},
			"alias": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Fully qualified private DNS name clients can use to reach the resource, e.g. `db.internal`. Wildcards (`*`, `?`) are allowed. Required when destination is a hostname.",
			},
			"alias_address": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Virtual IP Pangolin assigned to the alias.",
			},
			"tcp_port_range": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "TCP ports clients may reach, e.g. `80,443,8000-9000`, or `*` for all (the default).",
			},
			"udp_port_range": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "UDP ports clients may reach, in the same format as tcp_port_range. Defaults to `*`.",
			},
			"disable_icmp": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Block ICMP (ping) to the resource. Defaults to false.",
			},
			"domain_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Domain to publish the resource on. Required for `http` mode.",
			},
			"subdomain": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Subdomain under domain_id for `http` mode.",
			},
			"full_domain": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Full private domain of an `http` resource.",
			},
			"auth_daemon_port": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Port of the SSH auth daemon. Only meaningful for `ssh` mode.",
			},
			"auth_daemon_mode": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "Where the SSH auth daemon runs: `site`, `remote` or `native`. Only meaningful for `ssh` mode.",
				Validators:    []validator.String{stringvalidator.OneOf("site", "remote", "native")},
			},
			"pam_mode": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: keep,
				Description:   "SSH PAM behavior: `passthrough` or `push`. Only meaningful for `ssh` mode.",
				Validators:    []validator.String{stringvalidator.OneOf("passthrough", "push")},
			},
			"user_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyStringSet,
				Description: "IDs of users granted access. Replaces the full set on every apply.",
			},
			"role_ids": schema.SetAttribute{
				ElementType: types.Int64Type,
				Optional:    true,
				Computed:    true,
				Default:     emptyInt64Set,
				Description: "IDs of roles granted access, in addition to the admin role Pangolin always keeps attached. Replaces the full set on every apply.",
			},
			"client_ids": schema.SetAttribute{
				ElementType: types.Int64Type,
				Optional:    true,
				Computed:    true,
				Default:     emptyInt64Set,
				Description: "IDs of machine clients granted access. Replaces the full set on every apply.",
			},
		},
	}
}

func (r *privateResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func known[T interface {
	IsNull() bool
	IsUnknown() bool
}](v T) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func strPtr(v types.String) *string {
	if !known(v) {
		return nil
	}
	s := v.ValueString()
	return &s
}

func int64Ptr(v types.Int64) *int64 {
	if !known(v) {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

func boolPtr(v types.Bool) *bool {
	if !known(v) {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func setToInt64s(ctx context.Context, s types.Set) ([]int64, diag.Diagnostics) {
	out := []int64{}
	if !known(s) {
		return out, nil
	}
	d := s.ElementsAs(ctx, &out, false)
	return out, d
}

func setToStrings(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	out := []string{}
	if !known(s) {
		return out, nil
	}
	d := s.ElementsAs(ctx, &out, false)
	return out, d
}

func nullableString(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

// refresh copies the API representation, plus the separately-listed sites,
// roles, users and clients, into the model.
func (r *privateResourceResource) refresh(ctx context.Context, m *privateResourceModel, sr *client.SiteResource) diag.Diagnostics {
	var diags diag.Diagnostics
	m.PrivateResourceID = types.Int64Value(sr.SiteResourceID)
	m.NiceID = types.StringValue(sr.NiceID)
	m.Name = types.StringValue(sr.Name)
	m.Mode = types.StringValue(sr.Mode)
	m.Destination = nullableString(sr.Destination)
	m.DestinationPort = types.Int64Null()
	if sr.DestinationPort != nil {
		m.DestinationPort = types.Int64Value(*sr.DestinationPort)
	}
	m.Scheme = nullableString(sr.Scheme)
	m.SSL = types.BoolValue(sr.SSL)
	m.Enabled = types.BoolValue(sr.Enabled)
	m.Alias = nullableString(sr.Alias)
	m.AliasAddress = nullableString(sr.AliasAddress)
	m.TCPPortRange = types.StringValue(sr.TCPPortRangeString)
	m.UDPPortRange = types.StringValue(sr.UDPPortRangeString)
	m.DisableICMP = types.BoolValue(sr.DisableICMP)
	m.DomainID = nullableString(sr.DomainID)
	m.Subdomain = nullableString(sr.Subdomain)
	m.FullDomain = nullableString(sr.FullDomain)
	m.AuthDaemonPort = types.Int64Null()
	if sr.AuthDaemonPort != nil {
		m.AuthDaemonPort = types.Int64Value(*sr.AuthDaemonPort)
	}
	m.AuthDaemonMode = nullableString(sr.AuthDaemonMode)
	m.PamMode = nullableString(sr.PamMode)

	siteIDs, err := r.client.ListSiteResourceSiteIDs(ctx, m.OrgID.ValueString(), sr.SiteResourceID)
	if err != nil {
		diags.AddError("Error reading site resource sites", err.Error())
		return diags
	}
	roleIDs, err := r.client.ListSiteResourceRoleIDs(ctx, sr.SiteResourceID)
	if err != nil {
		diags.AddError("Error reading site resource roles", err.Error())
		return diags
	}
	userIDs, err := r.client.ListSiteResourceUserIDs(ctx, sr.SiteResourceID)
	if err != nil {
		diags.AddError("Error reading site resource users", err.Error())
		return diags
	}
	clientIDs, err := r.client.ListSiteResourceClientIDs(ctx, sr.SiteResourceID)
	if err != nil {
		diags.AddError("Error reading site resource clients", err.Error())
		return diags
	}
	var d diag.Diagnostics
	m.SiteIDs, d = types.SetValueFrom(ctx, types.Int64Type, siteIDs)
	diags.Append(d...)
	m.RoleIDs, d = types.SetValueFrom(ctx, types.Int64Type, roleIDs)
	diags.Append(d...)
	m.UserIDs, d = types.SetValueFrom(ctx, types.StringType, userIDs)
	diags.Append(d...)
	m.ClientIDs, d = types.SetValueFrom(ctx, types.Int64Type, clientIDs)
	diags.Append(d...)
	return diags
}

func (r *privateResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	siteIDs, d := setToInt64s(ctx, plan.SiteIDs)
	resp.Diagnostics.Append(d...)
	roleIDs, d := setToInt64s(ctx, plan.RoleIDs)
	resp.Diagnostics.Append(d...)
	userIDs, d := setToStrings(ctx, plan.UserIDs)
	resp.Diagnostics.Append(d...)
	clientIDs, d := setToInt64s(ctx, plan.ClientIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateSiteResourceRequest{
		Name:               plan.Name.ValueString(),
		NiceID:             plan.NiceID.ValueString(),
		Mode:               plan.Mode.ValueString(),
		SiteIDs:            siteIDs,
		Destination:        strPtr(plan.Destination),
		DestinationPort:    int64Ptr(plan.DestinationPort),
		Scheme:             strPtr(plan.Scheme),
		Alias:              strPtr(plan.Alias),
		TCPPortRangeString: strPtr(plan.TCPPortRange),
		UDPPortRangeString: strPtr(plan.UDPPortRange),
		DisableICMP:        boolPtr(plan.DisableICMP),
		DomainID:           strPtr(plan.DomainID),
		Subdomain:          strPtr(plan.Subdomain),
		UserIDs:            userIDs,
		RoleIDs:            roleIDs,
		ClientIDs:          clientIDs,
	}
	// Only http mode accepts ssl, and only ssh mode the auth daemon fields;
	// the API's strict schemas make sending a default for the wrong mode
	// harmless, but leaving them out keeps requests minimal.
	if plan.Mode.ValueString() == "http" {
		in.SSL = boolPtr(plan.SSL)
	}
	if plan.Mode.ValueString() == "ssh" {
		in.AuthDaemonPort = int64Ptr(plan.AuthDaemonPort)
		in.AuthDaemonMode = strPtr(plan.AuthDaemonMode)
		in.PamMode = strPtr(plan.PamMode)
	}

	created, err := r.client.CreateSiteResource(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site resource", err.Error())
		return
	}

	// enabled can't be set at creation.
	if known(plan.Enabled) && !plan.Enabled.ValueBool() {
		created, err = r.client.UpdateSiteResource(ctx, created.SiteResourceID, client.UpdateSiteResourceRequest{Enabled: boolPtr(plan.Enabled)})
		if err != nil {
			resp.Diagnostics.AddError("Error disabling new site resource", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.refresh(ctx, &plan, created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sr, err := r.client.GetSiteResource(ctx, state.PrivateResourceID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site resource", err.Error())
		return
	}
	resp.Diagnostics.Append(r.refresh(ctx, &state, sr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state privateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	siteIDs, d := setToInt64s(ctx, plan.SiteIDs)
	resp.Diagnostics.Append(d...)
	roleIDs, d := setToInt64s(ctx, plan.RoleIDs)
	resp.Diagnostics.Append(d...)
	userIDs, d := setToStrings(ctx, plan.UserIDs)
	resp.Diagnostics.Append(d...)
	clientIDs, d := setToInt64s(ctx, plan.ClientIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	mode := plan.Mode.ValueString()
	in := client.UpdateSiteResourceRequest{
		Name:               strPtr(plan.Name),
		SiteIDs:            siteIDs,
		Destination:        strPtr(plan.Destination),
		DestinationPort:    int64Ptr(plan.DestinationPort),
		Scheme:             strPtr(plan.Scheme),
		Enabled:            boolPtr(plan.Enabled),
		Alias:              strPtr(plan.Alias),
		TCPPortRangeString: strPtr(plan.TCPPortRange),
		UDPPortRangeString: strPtr(plan.UDPPortRange),
		DisableICMP:        boolPtr(plan.DisableICMP),
		DomainID:           strPtr(plan.DomainID),
		Subdomain:          strPtr(plan.Subdomain),
		UserIDs:            &userIDs,
		RoleIDs:            &roleIDs,
		ClientIDs:          &clientIDs,
	}
	if mode == "http" {
		in.SSL = boolPtr(plan.SSL)
	}
	if mode == "ssh" {
		in.AuthDaemonPort = int64Ptr(plan.AuthDaemonPort)
		in.AuthDaemonMode = strPtr(plan.AuthDaemonMode)
		in.PamMode = strPtr(plan.PamMode)
	}
	// The server re-validates destination/siteIds as a unit whenever mode is
	// present, so send it so those checks run against the full picture.
	in.Mode = &mode

	sr, err := r.client.UpdateSiteResource(ctx, state.PrivateResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site resource", err.Error())
		return
	}
	plan.PrivateResourceID = state.PrivateResourceID
	resp.Diagnostics.Append(r.refresh(ctx, &plan, sr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteSiteResource(ctx, state.PrivateResourceID.ValueInt64())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting site resource", err.Error())
	}
}

func (r *privateResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", `expected format: <org_id>:<private_resource_id>, e.g. "acme:12"`)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "private_resource_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_resource_id"), id)...)
}
