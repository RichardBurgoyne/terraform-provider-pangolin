package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &pangolinResourceResource{}
	_ resource.ResourceWithImportState = &pangolinResourceResource{}
)

func NewPangolinResourceResource() resource.Resource { return &pangolinResourceResource{} }

type pangolinResourceResource struct {
	client *client.Client
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
}

func (r *pangolinResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *pangolinResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin HTTP/SSH/RDP/VNC resource (a proxied endpoint). Raw TCP/UDP resources and inference-mode (AI gateway) resources are not yet supported by this provider. mode and domain_id changes replace the resource.",
		Attributes: map[string]schema.Attribute{
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this resource belongs to."},
			"resource_id": schema.Int64Attribute{Computed: true, Description: "Server-generated resource ID."},
			"nice_id":     schema.StringAttribute{Computed: true, Description: "Human-readable ID, unique per org."},
			"name":        schema.StringAttribute{Required: true, Description: "Display name of the resource."},
			"mode": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "One of http, ssh, rdp, vnc. inference and raw tcp/udp modes are not supported by this provider.",
			},
			"domain_id":        schema.StringAttribute{Required: true, Description: "Domain ID this resource is served from."},
			"subdomain":        schema.StringAttribute{Optional: true, Computed: true, Description: "Subdomain under domain_id."},
			"full_domain":      schema.StringAttribute{Computed: true, Description: "Fully-qualified domain this resource is reachable at."},
			"sticky_session":   schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether to enable sticky sessions."},
			"post_auth_path":   schema.StringAttribute{Optional: true, Description: "Path to redirect to after authentication."},
			"pam_mode":         schema.StringAttribute{Optional: true, Description: "SSH PAM mode: passthrough or push. Only meaningful for mode = ssh."},
			"auth_daemon_mode": schema.StringAttribute{Optional: true, Description: "One of site, remote, native. Only meaningful for mode = ssh."},
			"auth_daemon_port": schema.Int64Attribute{Optional: true, Description: "Auth daemon port. Only meaningful for mode = ssh."},
			"enabled":          schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the resource is enabled."},
			"ssl":              schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether SSL is enabled."},
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

func (r *pangolinResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pangolinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateResourceRequest{
		Name:     plan.Name.ValueString(),
		Mode:     plan.Mode.ValueString(),
		DomainID: plan.DomainID.ValueString(),
	}
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

	created, err := r.client.CreateResource(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	setPangolinResourceModelFromAPI(&plan, created)
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pangolinResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pangolinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateResourceRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
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
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.StickySession.IsUnknown() {
		v := plan.StickySession.ValueBool()
		in.StickySession = &v
	}
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

	updated, err := r.client.UpdateResource(ctx, state.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	setPangolinResourceModelFromAPI(&plan, updated)
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
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "resource_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), id)...)
}

func setPangolinResourceModelFromAPI(model *pangolinResourceModel, res *client.PangolinResource) {
	model.ResourceID = types.Int64Value(res.ResourceID)
	model.NiceID = types.StringValue(res.NiceID)
	model.Name = types.StringValue(res.Name)
	model.Mode = types.StringValue(res.Mode)
	model.DomainID = types.StringValue(res.DomainID)
	model.Subdomain = types.StringValue(res.Subdomain)
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
}
