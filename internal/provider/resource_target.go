package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &targetResource{}
	_ resource.ResourceWithImportState = &targetResource{}
)

func NewTargetResource() resource.Resource { return &targetResource{} }

type targetResource struct {
	client *client.Client
}

type targetResourceModel struct {
	ResourceID      types.Int64  `tfsdk:"resource_id"`
	TargetID        types.Int64  `tfsdk:"target_id"`
	SiteID          types.Int64  `tfsdk:"site_id"`
	IP              types.String `tfsdk:"ip"`
	Mode            types.String `tfsdk:"mode"`
	Method          types.String `tfsdk:"method"`
	Port            types.Int64  `tfsdk:"port"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Path            types.String `tfsdk:"path"`
	PathMatchType   types.String `tfsdk:"path_match_type"`
	RewritePath     types.String `tfsdk:"rewrite_path"`
	RewritePathType types.String `tfsdk:"rewrite_path_type"`
	Priority        types.Int64  `tfsdk:"priority"`
}

func (r *targetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (r *targetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a backend target on a Pangolin resource. Health checks are not yet supported by this provider. Import using the format `<resource_id>:<target_id>`, e.g. `terraform import pangolin_target.example 5:9`.",
		Attributes: map[string]schema.Attribute{
			"resource_id":       schema.Int64Attribute{Required: true, PlanModifiers: replaceInt, Description: "Resource ID this target belongs to."},
			"target_id":         schema.Int64Attribute{Computed: true, Description: "Server-generated target ID."},
			"site_id":           schema.Int64Attribute{Required: true, Description: "Site ID this target routes through."},
			"ip":                schema.StringAttribute{Required: true, Description: "Target IP address or hostname."},
			"mode":              schema.StringAttribute{Optional: true, Computed: true, Description: "One of http, tcp, udp, ssh, rdp, vnc. Defaults to the resource's mode."},
			"method":            schema.StringAttribute{Optional: true, Computed: true, Description: "HTTP method restriction, if any."},
			"port":              schema.Int64Attribute{Required: true, Description: "Target port."},
			"enabled":           schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this target is enabled."},
			"path":              schema.StringAttribute{Optional: true, Computed: true, Description: "Path match for HTTP-mode routing."},
			"path_match_type":   schema.StringAttribute{Optional: true, Computed: true, Description: "One of exact, prefix, regex."},
			"rewrite_path":      schema.StringAttribute{Optional: true, Computed: true, Description: "Path to rewrite to."},
			"rewrite_path_type": schema.StringAttribute{Optional: true, Computed: true, Description: "One of exact, prefix, regex, stripPrefix."},
			"priority":          schema.Int64Attribute{Optional: true, Computed: true, Description: "Routing priority (1-1000)."},
		},
	}
}

func (r *targetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *targetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateTargetRequest{
		SiteID: plan.SiteID.ValueInt64(),
		IP:     plan.IP.ValueString(),
		Port:   plan.Port.ValueInt64(),
	}
	if !plan.Mode.IsUnknown() {
		in.Mode = plan.Mode.ValueString()
	}
	if !plan.Method.IsUnknown() {
		v := plan.Method.ValueString()
		in.Method = &v
	}
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.Path.IsUnknown() {
		v := plan.Path.ValueString()
		in.Path = &v
	}
	if !plan.PathMatchType.IsUnknown() {
		v := plan.PathMatchType.ValueString()
		in.PathMatchType = &v
	}
	if !plan.RewritePath.IsUnknown() {
		v := plan.RewritePath.ValueString()
		in.RewritePath = &v
	}
	if !plan.RewritePathType.IsUnknown() {
		v := plan.RewritePathType.ValueString()
		in.RewritePathType = &v
	}
	if !plan.Priority.IsUnknown() {
		v := plan.Priority.ValueInt64()
		in.Priority = &v
	}

	created, err := r.client.CreateTarget(ctx, plan.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating target", err.Error())
		return
	}

	setTargetModelFromAPI(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *targetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state targetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := r.client.GetTarget(ctx, state.TargetID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading target", err.Error())
		return
	}

	resourceID := state.ResourceID
	setTargetModelFromAPI(&state, target)
	state.ResourceID = resourceID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *targetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// siteId and ip are required on every update call per updateTargetBodySchema.
	in := client.UpdateTargetRequest{
		SiteID: plan.SiteID.ValueInt64(),
		IP:     plan.IP.ValueString(),
	}
	if !plan.Mode.IsUnknown() {
		v := plan.Mode.ValueString()
		in.Mode = &v
	}
	if !plan.Method.IsUnknown() {
		v := plan.Method.ValueString()
		in.Method = &v
	}
	port := plan.Port.ValueInt64()
	in.Port = &port
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.Path.IsUnknown() {
		v := plan.Path.ValueString()
		in.Path = &v
	}
	if !plan.PathMatchType.IsUnknown() {
		v := plan.PathMatchType.ValueString()
		in.PathMatchType = &v
	}
	if !plan.RewritePath.IsUnknown() {
		v := plan.RewritePath.ValueString()
		in.RewritePath = &v
	}
	if !plan.RewritePathType.IsUnknown() {
		v := plan.RewritePathType.ValueString()
		in.RewritePathType = &v
	}
	if !plan.Priority.IsUnknown() {
		v := plan.Priority.ValueInt64()
		in.Priority = &v
	}

	updated, err := r.client.UpdateTarget(ctx, state.TargetID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating target", err.Error())
		return
	}

	resourceID := plan.ResourceID
	setTargetModelFromAPI(&plan, updated)
	plan.ResourceID = resourceID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *targetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state targetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTarget(ctx, state.TargetID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting target", err.Error())
	}
}

func (r *targetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", `expected format: <resource_id>:<target_id>, e.g. "5:9"`)
		return
	}
	resourceID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "resource_id must be a numeric ID: "+err.Error())
		return
	}
	targetID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "target_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), resourceID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_id"), targetID)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func setTargetModelFromAPI(model *targetResourceModel, target *client.Target) {
	model.TargetID = types.Int64Value(target.TargetID)
	model.ResourceID = types.Int64Value(target.ResourceID)
	model.SiteID = types.Int64Value(target.SiteID)
	model.IP = types.StringValue(target.IP)
	model.Mode = types.StringValue(target.Mode)
	model.Method = types.StringValue(target.Method)
	model.Port = types.Int64Value(target.Port)
	model.Enabled = types.BoolValue(target.Enabled)
	model.Path = types.StringValue(target.Path)
	model.PathMatchType = types.StringValue(target.PathMatchType)
	model.RewritePath = types.StringValue(target.RewritePath)
	model.RewritePathType = types.StringValue(target.RewritePathType)
	model.Priority = types.Int64Value(target.Priority)
}
