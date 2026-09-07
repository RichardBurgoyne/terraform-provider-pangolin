package provider

import (
	"context"
	"fmt"
	"strconv"

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
	_ resource.Resource                = &aiProviderResource{}
	_ resource.ResourceWithImportState = &aiProviderResource{}
)

func NewAIProviderResource() resource.Resource { return &aiProviderResource{} }

type aiProviderResource struct {
	client *client.Client
}

type aiProviderHeaderModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func aiProviderHeaderObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":  types.StringType,
			"value": types.StringType,
		},
	}
}

type aiProviderResourceModel struct {
	OrgID               types.String `tfsdk:"org_id"`
	ProviderID          types.Int64  `tfsdk:"provider_id"`
	NiceID              types.String `tfsdk:"nice_id"`
	Name                types.String `tfsdk:"name"`
	Type                types.String `tfsdk:"type"`
	UpstreamURL         types.String `tfsdk:"upstream_url"`
	APIKey              types.String `tfsdk:"api_key"`
	AuthType            types.String `tfsdk:"auth_type"`
	RoutingMode         types.String `tfsdk:"routing_mode"`
	Capabilities        types.List   `tfsdk:"capabilities"`
	Headers             types.List   `tfsdk:"headers"`
	SkipTLSVerification types.Bool   `tfsdk:"skip_tls_verification"`
	Enabled             types.Bool   `tfsdk:"enabled"`
}

func (r *aiProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_provider"
}

func (r *aiProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages an AI provider (an upstream AI gateway backend) in Pangolin. Attach an existing provider to an inference-mode pangolin_resource via that resource's ai_providers attribute. Import using the numeric provider ID, e.g. `terraform import pangolin_ai_provider.example 5`.",
		Attributes: map[string]schema.Attribute{
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this provider belongs to."},
			"provider_id": schema.Int64Attribute{Computed: true, Description: "Server-generated provider ID."},
			"nice_id":     schema.StringAttribute{Computed: true, Description: "Human-readable ID, unique per org."},
			"name":        schema.StringAttribute{Required: true, Description: "Display name of the provider."},
			"type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "One of openai, anthropic, googleGemini, vertexAi, bedrock, microsoftFoundry, openRouter, vercelAiGateway, custom.",
			},
			"upstream_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Upstream API base URL. Required for most provider types; not used for type = custom with routing_mode = target.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API key used to authenticate to the upstream provider. Returned back from the API in plaintext on every read, so Terraform can keep state in sync; still marked sensitive to keep it out of plan/apply output.",
			},
			"auth_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "How the API key is sent upstream: bearer, x-api-key, x-goog-api-key, hec, cf-aig-authorization, none, or passthrough. Defaults based on type.",
			},
			"routing_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "url (default; proxies to upstream_url) or target (routes through a pangolin_target, custom type only).",
			},
			"capabilities": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "API surfaces this provider exposes, e.g. openai_chat, anthropic_messages. Required (at least one) for type = custom; defaulted for other types.",
			},
			"headers": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Extra headers sent with every request to the upstream provider.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":  schema.StringAttribute{Required: true},
						"value": schema.StringAttribute{Required: true},
					},
				},
			},
			"skip_tls_verification": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether to skip TLS certificate verification when connecting to the upstream provider."},
			"enabled":               schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this provider is enabled."},
		},
	}
}

func (r *aiProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func aiProviderHeadersFromPlan(ctx context.Context, list types.List) ([]client.HCHeader, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var items []aiProviderHeaderModel
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if diags.HasError() {
		return nil, diags
	}
	headers := make([]client.HCHeader, 0, len(items))
	for _, item := range items {
		headers = append(headers, client.HCHeader{Name: item.Name.ValueString(), Value: item.Value.ValueString()})
	}
	return headers, diags
}

func aiProviderHeadersToState(ctx context.Context, headers []client.HCHeader) (types.List, diag.Diagnostics) {
	objType := aiProviderHeaderObjectType()
	if len(headers) == 0 {
		return types.ListNull(objType), nil
	}
	items := make([]aiProviderHeaderModel, 0, len(headers))
	for _, h := range headers {
		items = append(items, aiProviderHeaderModel{Name: types.StringValue(h.Name), Value: types.StringValue(h.Value)})
	}
	return types.ListValueFrom(ctx, objType, items)
}

func (r *aiProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aiProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateAIProviderRequest{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
	}
	if !plan.UpstreamURL.IsUnknown() && !plan.UpstreamURL.IsNull() {
		v := plan.UpstreamURL.ValueString()
		in.UpstreamURL = &v
	}
	if !plan.APIKey.IsUnknown() && !plan.APIKey.IsNull() {
		v := plan.APIKey.ValueString()
		in.APIKey = &v
	}
	if !plan.AuthType.IsUnknown() {
		in.AuthType = plan.AuthType.ValueString()
	}
	if !plan.RoutingMode.IsUnknown() {
		in.RoutingMode = plan.RoutingMode.ValueString()
	}
	if !plan.Capabilities.IsUnknown() && !plan.Capabilities.IsNull() {
		resp.Diagnostics.Append(plan.Capabilities.ElementsAs(ctx, &in.Capabilities, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	headers, diags := aiProviderHeadersFromPlan(ctx, plan.Headers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	in.Headers = headers
	if !plan.SkipTLSVerification.IsUnknown() && !plan.SkipTLSVerification.IsNull() {
		v := plan.SkipTLSVerification.ValueBool()
		in.SkipTLSVerification = &v
	}
	if !plan.Enabled.IsUnknown() && !plan.Enabled.IsNull() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}

	created, err := r.client.CreateAIProvider(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AI provider", err.Error())
		return
	}

	resp.Diagnostics.Append(setAIProviderModelFromAPI(ctx, &plan, created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aiProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aiProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	provider, err := r.client.GetAIProvider(ctx, state.ProviderID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AI provider", err.Error())
		return
	}

	resp.Diagnostics.Append(setAIProviderModelFromAPI(ctx, &state, provider)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aiProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state aiProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateAIProviderRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	if !plan.UpstreamURL.IsUnknown() {
		v := plan.UpstreamURL.ValueString()
		in.UpstreamURL = &v
	}
	if !plan.APIKey.IsUnknown() && !plan.APIKey.IsNull() {
		v := plan.APIKey.ValueString()
		in.APIKey = &v
	}
	if !plan.AuthType.IsUnknown() {
		in.AuthType = plan.AuthType.ValueString()
	}
	if !plan.RoutingMode.IsUnknown() {
		in.RoutingMode = plan.RoutingMode.ValueString()
	}
	if !plan.Capabilities.IsUnknown() {
		resp.Diagnostics.Append(plan.Capabilities.ElementsAs(ctx, &in.Capabilities, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !plan.Headers.IsUnknown() {
		headers, diags := aiProviderHeadersFromPlan(ctx, plan.Headers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		in.Headers = headers
	}
	if !plan.SkipTLSVerification.IsUnknown() {
		v := plan.SkipTLSVerification.ValueBool()
		in.SkipTLSVerification = &v
	}
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}

	updated, err := r.client.UpdateAIProvider(ctx, state.ProviderID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AI provider", err.Error())
		return
	}

	resp.Diagnostics.Append(setAIProviderModelFromAPI(ctx, &plan, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aiProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aiProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAIProvider(ctx, state.ProviderID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting AI provider", err.Error())
	}
}

func (r *aiProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", `expected a numeric provider ID, e.g. "5": `+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_id"), id)...)
}

func setAIProviderModelFromAPI(ctx context.Context, model *aiProviderResourceModel, provider *client.AIProvider) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ProviderID = types.Int64Value(provider.ProviderID)
	model.OrgID = types.StringValue(provider.OrgID)
	model.NiceID = types.StringValue(provider.NiceID)
	model.Name = types.StringValue(provider.Name)
	model.Type = types.StringValue(provider.Type)
	model.UpstreamURL = types.StringValue(provider.UpstreamURL)
	if provider.APIKey != "" {
		model.APIKey = types.StringValue(provider.APIKey)
	} else {
		model.APIKey = types.StringNull()
	}
	model.AuthType = types.StringValue(provider.AuthType)
	model.RoutingMode = types.StringValue(provider.RoutingMode)
	model.SkipTLSVerification = types.BoolValue(provider.SkipTLSVerification)
	model.Enabled = types.BoolValue(provider.Enabled)

	capabilities, d := types.ListValueFrom(ctx, types.StringType, provider.Capabilities)
	diags.Append(d...)
	model.Capabilities = capabilities

	headers, d := aiProviderHeadersToState(ctx, provider.Headers)
	diags.Append(d...)
	model.Headers = headers

	return diags
}
