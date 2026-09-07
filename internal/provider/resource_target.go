package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

	HCEnabled            types.Bool   `tfsdk:"hc_enabled"`
	HCPath               types.String `tfsdk:"hc_path"`
	HCScheme             types.String `tfsdk:"hc_scheme"`
	HCMode               types.String `tfsdk:"hc_mode"`
	HCHostname           types.String `tfsdk:"hc_hostname"`
	HCPort               types.Int64  `tfsdk:"hc_port"`
	HCInterval           types.Int64  `tfsdk:"hc_interval"`
	HCUnhealthyInterval  types.Int64  `tfsdk:"hc_unhealthy_interval"`
	HCTimeout            types.Int64  `tfsdk:"hc_timeout"`
	HCHeadersJSON        types.String `tfsdk:"hc_headers_json"`
	HCFollowRedirects    types.Bool   `tfsdk:"hc_follow_redirects"`
	HCMethod             types.String `tfsdk:"hc_method"`
	HCStatus             types.Int64  `tfsdk:"hc_status"`
	HCTlsServerName      types.String `tfsdk:"hc_tls_server_name"`
	HCHealthyThreshold   types.Int64  `tfsdk:"hc_healthy_threshold"`
	HCUnhealthyThreshold types.Int64  `tfsdk:"hc_unhealthy_threshold"`
	HCHealth             types.String `tfsdk:"hc_health"`
}

func (r *targetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (r *targetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a backend target on a Pangolin resource, including its health check configuration. Import using the format `<resource_id>:<target_id>`, e.g. `terraform import pangolin_target.example 5:9`.",
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

			"hc_enabled":  schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether health checks are enabled for this target. Requires hc_hostname to be set."},
			"hc_hostname": schema.StringAttribute{Optional: true, Computed: true, Description: "Hostname or IP the health check connects to. Required when hc_enabled is true."},
			"hc_path":     schema.StringAttribute{Optional: true, Computed: true, Description: "HTTP path to request for the health check."},
			"hc_scheme":   schema.StringAttribute{Optional: true, Computed: true, Description: "Scheme used for the health check request, e.g. http or https."},
			"hc_mode":     schema.StringAttribute{Optional: true, Computed: true, Description: "Health check protocol, e.g. http or tcp."},
			"hc_port":     schema.Int64Attribute{Optional: true, Computed: true, Description: "Port the health check connects to, if different from port."},
			"hc_interval": schema.Int64Attribute{Optional: true, Computed: true, Description: "Seconds between health checks while the target is healthy."},
			"hc_unhealthy_interval": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Seconds between health checks while the target is unhealthy.",
			},
			"hc_timeout":           schema.Int64Attribute{Optional: true, Computed: true, Description: "Seconds to wait for a health check response before failing it."},
			"hc_follow_redirects":  schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the HTTP health check follows redirects."},
			"hc_method":            schema.StringAttribute{Optional: true, Computed: true, Description: "HTTP method used for the health check request."},
			"hc_status":            schema.Int64Attribute{Optional: true, Computed: true, Description: "Expected HTTP status code for a healthy response."},
			"hc_tls_server_name":   schema.StringAttribute{Optional: true, Computed: true, Description: "TLS server name (SNI) to use for the health check request."},
			"hc_healthy_threshold": schema.Int64Attribute{Optional: true, Computed: true, Description: "Consecutive successful checks required to mark the target healthy."},
			"hc_unhealthy_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Consecutive failed checks required to mark the target unhealthy.",
			},
			"hc_headers_json": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Extra headers sent with the health check request, as a JSON array of {name, value} objects, e.g. `[{\"name\":\"X-Check\",\"value\":\"1\"}]`. Modeled as a JSON string rather than a nested list because the Pangolin API itself is inconsistent about this field's wire format across endpoints.",
			},
			"hc_health": schema.StringAttribute{
				Computed:    true,
				Description: "Server-computed health status: unknown, healthy, or unhealthy.",
			},
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

// hcHeadersFromPlan parses a JSON-string hc_headers_json plan value into the
// wire format the API expects for create/update requests.
func hcHeadersFromPlan(v types.String) ([]client.HCHeader, error) {
	if v.IsUnknown() || v.IsNull() || v.ValueString() == "" {
		return nil, nil
	}
	var headers []client.HCHeader
	if err := json.Unmarshal([]byte(v.ValueString()), &headers); err != nil {
		return nil, fmt.Errorf("hc_headers_json must be a JSON array of {name, value} objects: %w", err)
	}
	return headers, nil
}

// hcHeadersToState re-encodes a Target's hcHeaders (which may have come back
// from the API as a real array or as a JSON-encoded string, depending on the
// endpoint) into a canonical JSON string for state.
func hcHeadersToState(raw any) (types.String, error) {
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

func (r *targetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hcHeaders, err := hcHeadersFromPlan(plan.HCHeadersJSON)
	if err != nil {
		resp.Diagnostics.AddError("Invalid hc_headers_json", err.Error())
		return
	}

	in := client.CreateTargetRequest{
		SiteID:    plan.SiteID.ValueInt64(),
		IP:        plan.IP.ValueString(),
		Port:      plan.Port.ValueInt64(),
		HCHeaders: hcHeaders,
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
	if !plan.HCEnabled.IsUnknown() {
		v := plan.HCEnabled.ValueBool()
		in.HCEnabled = &v
	}
	if !plan.HCHostname.IsUnknown() {
		v := plan.HCHostname.ValueString()
		in.HCHostname = &v
	}
	if !plan.HCPath.IsUnknown() {
		v := plan.HCPath.ValueString()
		in.HCPath = &v
	}
	if !plan.HCScheme.IsUnknown() {
		v := plan.HCScheme.ValueString()
		in.HCScheme = &v
	}
	if !plan.HCMode.IsUnknown() {
		v := plan.HCMode.ValueString()
		in.HCMode = &v
	}
	if !plan.HCPort.IsUnknown() {
		v := plan.HCPort.ValueInt64()
		in.HCPort = &v
	}
	if !plan.HCInterval.IsUnknown() {
		v := plan.HCInterval.ValueInt64()
		in.HCInterval = &v
	}
	if !plan.HCUnhealthyInterval.IsUnknown() {
		v := plan.HCUnhealthyInterval.ValueInt64()
		in.HCUnhealthyInterval = &v
	}
	if !plan.HCTimeout.IsUnknown() {
		v := plan.HCTimeout.ValueInt64()
		in.HCTimeout = &v
	}
	if !plan.HCFollowRedirects.IsUnknown() {
		v := plan.HCFollowRedirects.ValueBool()
		in.HCFollowRedirects = &v
	}
	if !plan.HCMethod.IsUnknown() {
		v := plan.HCMethod.ValueString()
		in.HCMethod = &v
	}
	if !plan.HCStatus.IsUnknown() {
		v := plan.HCStatus.ValueInt64()
		in.HCStatus = &v
	}
	if !plan.HCTlsServerName.IsUnknown() {
		v := plan.HCTlsServerName.ValueString()
		in.HCTlsServerName = &v
	}
	if !plan.HCHealthyThreshold.IsUnknown() {
		v := plan.HCHealthyThreshold.ValueInt64()
		in.HCHealthyThreshold = &v
	}
	if !plan.HCUnhealthyThreshold.IsUnknown() {
		v := plan.HCUnhealthyThreshold.ValueInt64()
		in.HCUnhealthyThreshold = &v
	}

	created, err := r.client.CreateTarget(ctx, plan.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating target", err.Error())
		return
	}

	if diags := setTargetModelFromAPI(&plan, created); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
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
	if diags := setTargetModelFromAPI(&state, target); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
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

	hcHeaders, err := hcHeadersFromPlan(plan.HCHeadersJSON)
	if err != nil {
		resp.Diagnostics.AddError("Invalid hc_headers_json", err.Error())
		return
	}

	// siteId and ip are required on every update call per updateTargetBodySchema.
	in := client.UpdateTargetRequest{
		SiteID:    plan.SiteID.ValueInt64(),
		IP:        plan.IP.ValueString(),
		HCHeaders: hcHeaders,
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
	if !plan.HCEnabled.IsUnknown() {
		v := plan.HCEnabled.ValueBool()
		in.HCEnabled = &v
	}
	if !plan.HCHostname.IsUnknown() {
		v := plan.HCHostname.ValueString()
		in.HCHostname = &v
	}
	if !plan.HCPath.IsUnknown() {
		v := plan.HCPath.ValueString()
		in.HCPath = &v
	}
	if !plan.HCScheme.IsUnknown() {
		v := plan.HCScheme.ValueString()
		in.HCScheme = &v
	}
	if !plan.HCMode.IsUnknown() {
		v := plan.HCMode.ValueString()
		in.HCMode = &v
	}
	if !plan.HCPort.IsUnknown() {
		v := plan.HCPort.ValueInt64()
		in.HCPort = &v
	}
	if !plan.HCInterval.IsUnknown() {
		v := plan.HCInterval.ValueInt64()
		in.HCInterval = &v
	}
	if !plan.HCUnhealthyInterval.IsUnknown() {
		v := plan.HCUnhealthyInterval.ValueInt64()
		in.HCUnhealthyInterval = &v
	}
	if !plan.HCTimeout.IsUnknown() {
		v := plan.HCTimeout.ValueInt64()
		in.HCTimeout = &v
	}
	if !plan.HCFollowRedirects.IsUnknown() {
		v := plan.HCFollowRedirects.ValueBool()
		in.HCFollowRedirects = &v
	}
	if !plan.HCMethod.IsUnknown() {
		v := plan.HCMethod.ValueString()
		in.HCMethod = &v
	}
	if !plan.HCStatus.IsUnknown() {
		v := plan.HCStatus.ValueInt64()
		in.HCStatus = &v
	}
	if !plan.HCTlsServerName.IsUnknown() {
		v := plan.HCTlsServerName.ValueString()
		in.HCTlsServerName = &v
	}
	if !plan.HCHealthyThreshold.IsUnknown() {
		v := plan.HCHealthyThreshold.ValueInt64()
		in.HCHealthyThreshold = &v
	}
	if !plan.HCUnhealthyThreshold.IsUnknown() {
		v := plan.HCUnhealthyThreshold.ValueInt64()
		in.HCUnhealthyThreshold = &v
	}

	updated, err := r.client.UpdateTarget(ctx, state.TargetID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating target", err.Error())
		return
	}

	resourceID := plan.ResourceID
	if diags := setTargetModelFromAPI(&plan, updated); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
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

func setTargetModelFromAPI(model *targetResourceModel, target *client.Target) diag.Diagnostics {
	var diags diag.Diagnostics
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

	model.HCEnabled = types.BoolValue(target.HCEnabled)
	model.HCPath = types.StringValue(target.HCPath)
	model.HCScheme = types.StringValue(target.HCScheme)
	model.HCMode = types.StringValue(target.HCMode)
	model.HCHostname = types.StringValue(target.HCHostname)
	if target.HCPort != 0 {
		model.HCPort = types.Int64Value(target.HCPort)
	} else {
		model.HCPort = types.Int64Null()
	}
	model.HCInterval = types.Int64Value(target.HCInterval)
	model.HCUnhealthyInterval = types.Int64Value(target.HCUnhealthyInterval)
	model.HCTimeout = types.Int64Value(target.HCTimeout)
	model.HCFollowRedirects = types.BoolValue(target.HCFollowRedirects)
	model.HCMethod = types.StringValue(target.HCMethod)
	if target.HCStatus != 0 {
		model.HCStatus = types.Int64Value(target.HCStatus)
	} else {
		model.HCStatus = types.Int64Null()
	}
	model.HCTlsServerName = types.StringValue(target.HCTlsServerName)
	model.HCHealthyThreshold = types.Int64Value(target.HCHealthyThreshold)
	model.HCUnhealthyThreshold = types.Int64Value(target.HCUnhealthyThreshold)
	model.HCHealth = types.StringValue(target.HCHealth)

	headersJSON, err := hcHeadersToState(target.HCHeaders)
	if err != nil {
		diags.AddError("Error decoding hc_headers_json", err.Error())
		return diags
	}
	model.HCHeadersJSON = headersJSON

	return diags
}
