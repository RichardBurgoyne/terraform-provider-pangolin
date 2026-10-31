package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                   = &siteResource{}
	_ resource.ResourceWithImportState    = &siteResource{}
	_ resource.ResourceWithValidateConfig = &siteResource{}
)

func NewSiteResource() resource.Resource { return &siteResource{} }

type siteResource struct {
	resourceClient
}

type siteResourceModel struct {
	OrgID                 types.String `tfsdk:"org_id"`
	SiteID                types.Int64  `tfsdk:"site_id"`
	NiceID                types.String `tfsdk:"nice_id"`
	Name                  types.String `tfsdk:"name"`
	Type                  types.String `tfsdk:"type"`
	ExitNodeID            types.Int64  `tfsdk:"exit_node_id"`
	PubKey                types.String `tfsdk:"pub_key"`
	Subnet                types.String `tfsdk:"subnet"`
	Address               types.String `tfsdk:"address"`
	NewtID                types.String `tfsdk:"newt_id"`
	Secret                types.String `tfsdk:"secret"`
	DockerSocketEnabled   types.Bool   `tfsdk:"docker_socket_enabled"`
	AutoUpdateEnabled     types.Bool   `tfsdk:"auto_update_enabled"`
	AutoUpdateOverrideOrg types.Bool   `tfsdk:"auto_update_override_org"`
}

func (r *siteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin site. Only name, docker_socket_enabled, auto_update_enabled, and auto_update_override_org can be updated after creation; every other attribute replaces the site if changed. Import using the format `<org_id>:<site_id>`, e.g. `terraform import pangolin_site.example acme:5`.",
		Attributes: map[string]schema.Attribute{
			"org_id":  schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this site belongs to."},
			"site_id": schema.Int64Attribute{Computed: true, Description: "Server-generated site ID."},
			"nice_id": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace, Description: "Human-readable ID, unique per org. Server-generated if omitted."},
			"name":    schema.StringAttribute{Required: true, Description: "Display name of the site."},
			"type":    schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "One of newt, wireguard, or local."},
			"exit_node_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Exit node ID. Required for type = wireguard; server-assigned for type = newt.",
			},
			"pub_key":                  schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace, Description: "WireGuard public key. Required for type = wireguard."},
			"subnet":                   schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace, Description: "WireGuard tunnel subnet. Required for type = wireguard."},
			"address":                  schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace, Description: "Client subnet address. Server-assigned if omitted."},
			"newt_id":                  schema.StringAttribute{Computed: true, Sensitive: true, Description: "Newt agent ID (type = newt only). Only ever populated from the create response."},
			"secret":                   schema.StringAttribute{Computed: true, Sensitive: true, Description: "Newt agent secret (type = newt only). Only ever populated from the create response."},
			"docker_socket_enabled":    schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the site can access the Docker socket."},
			"auto_update_enabled":      schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the site's Newt agent auto-updates."},
			"auto_update_override_org": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this site overrides the org's auto-update setting."},
		},
	}
}

func (r *siteResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config siteResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}

	if config.Type.ValueString() != "wireguard" {
		return
	}

	if config.PubKey.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("pub_key"), "Missing pub_key", `pub_key is required when type is "wireguard".`)
	}
	if config.Subnet.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("subnet"), "Missing subnet", `subnet is required when type is "wireguard".`)
	}
	if config.ExitNodeID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("exit_node_id"), "Missing exit_node_id", `exit_node_id is required when type is "wireguard".`)
	}
}

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateSiteRequest{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
	}
	in.ExitNodeID = int64Ptr(plan.ExitNodeID)
	if !plan.NiceID.IsUnknown() {
		in.NiceID = plan.NiceID.ValueString()
	}
	in.PubKey = plan.PubKey.ValueString()
	in.Subnet = plan.Subnet.ValueString()
	if !plan.Address.IsUnknown() {
		in.Address = plan.Address.ValueString()
	}

	site, err := r.client.CreateSite(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site", err.Error())
		return
	}

	setSiteModelFromAPI(&plan, site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := r.client.GetSite(ctx, state.SiteID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site", err.Error())
		return
	}

	// newtId/secret are write-once: GetSite never returns them, so preserve
	// whatever is already in state instead of overwriting with empty values.
	newtID, secret := state.NewtID, state.Secret
	setSiteModelFromAPI(&state, site)
	state.NewtID, state.Secret = newtID, secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateSiteRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	in.DockerSocketEnabled = boolPtr(plan.DockerSocketEnabled)
	in.AutoUpdateEnabled = boolPtr(plan.AutoUpdateEnabled)
	in.AutoUpdateOverrideOrg = boolPtr(plan.AutoUpdateOverrideOrg)

	site, err := r.client.UpdateSite(ctx, state.SiteID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site", err.Error())
		return
	}

	newtID, secret := state.NewtID, state.Secret
	setSiteModelFromAPI(&plan, site)
	plan.NewtID, plan.Secret = newtID, secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteSite(ctx, state.SiteID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error deleting site", err.Error())
	}
}

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStringAndInt64(ctx, req.ID, resp, "org_id", "site_id")
}

func setSiteModelFromAPI(model *siteResourceModel, site *client.Site) {
	model.SiteID = types.Int64Value(site.SiteID)
	model.NiceID = types.StringValue(site.NiceID)
	model.Name = types.StringValue(site.Name)
	model.Type = types.StringValue(site.Type)
	model.ExitNodeID = nullableInt64(site.ExitNodeID)
	model.PubKey = types.StringValue(site.PubKey)
	model.Subnet = types.StringValue(site.Subnet)
	model.Address = types.StringValue(site.Address)
	model.DockerSocketEnabled = types.BoolValue(site.DockerSocketEnabled)
	model.AutoUpdateEnabled = types.BoolValue(site.AutoUpdateEnabled)
	model.AutoUpdateOverrideOrg = types.BoolValue(site.AutoUpdateOverrideOrg)
	model.NewtID = nonEmptyString(site.NewtID)
	model.Secret = nonEmptyString(site.Secret)
}
