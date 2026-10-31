package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &resourceRuleResource{}
	_ resource.ResourceWithImportState = &resourceRuleResource{}
)

func NewResourceRuleResource() resource.Resource { return &resourceRuleResource{} }

type resourceRuleResource struct {
	resourceClient
}

type resourceRuleModel struct {
	ResourceID types.Int64  `tfsdk:"resource_id"`
	RuleID     types.Int64  `tfsdk:"rule_id"`
	Action     types.String `tfsdk:"action"`
	Match      types.String `tfsdk:"match"`
	Value      types.String `tfsdk:"value"`
	Priority   types.Int64  `tfsdk:"priority"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (r *resourceRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_rule"
}

func (r *resourceRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An access rule on a Pangolin resource: allow, block or pass matching requests by CIDR, IP, path, country, ASN, region or HTTP method. Rules are only evaluated when the resource has `apply_rules` enabled. Import using the format `<resource_id>:<rule_id>`, e.g. `terraform import pangolin_resource_rule.example 5:12`.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Description:   "ID of the resource the rule belongs to.",
			},
			"rule_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Pangolin-assigned rule ID.",
			},
			"action": schema.StringAttribute{
				Required:    true,
				Description: "What to do with matching requests: ACCEPT, DROP or PASS (fall through to the resource's normal authentication).",
				Validators:  []validator.String{stringvalidator.OneOf("ACCEPT", "DROP", "PASS")},
			},
			"match": schema.StringAttribute{
				Required:    true,
				Description: "What the rule matches on: CIDR, IP, PATH, COUNTRY, COUNTRY_IS_NOT, ASN, REGION or METHOD. METHOD requires Pangolin 1.24 or later.",
				Validators: []validator.String{stringvalidator.OneOf(
					"CIDR", "IP", "PATH", "COUNTRY", "COUNTRY_IS_NOT", "ASN", "REGION", "METHOD",
				)},
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "Value to match, interpreted according to `match`: a CIDR (`10.0.0.0/8`), an IP, a URL glob (`/admin/*`), a two-letter country code, an ASN (`AS13335`, or `ALL`), a region ID, or a comma-separated list of HTTP methods (`POST,PUT,DELETE`).",
			},
			"priority": schema.Int64Attribute{
				Required:    true,
				Description: "Evaluation order; rules with a lower number are evaluated first.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is active. Defaults to true.",
			},
		},
	}
}

func (m *resourceRuleModel) fromAPI(rule *client.ResourceRule) {
	m.RuleID = types.Int64Value(rule.RuleID)
	m.Action = types.StringValue(rule.Action)
	m.Match = types.StringValue(rule.Match)
	m.Value = types.StringValue(rule.Value)
	m.Priority = types.Int64Value(rule.Priority)
	m.Enabled = types.BoolValue(rule.Enabled)
}

func (r *resourceRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	enabled := plan.Enabled.ValueBool()
	rule, err := r.client.CreateResourceRule(ctx, plan.ResourceID.ValueInt64(), client.CreateResourceRuleRequest{
		Action:   plan.Action.ValueString(),
		Match:    plan.Match.ValueString(),
		Value:    plan.Value.ValueString(),
		Priority: plan.Priority.ValueInt64(),
		Enabled:  &enabled,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource rule", err.Error())
		return
	}
	plan.fromAPI(rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.GetResourceRule(ctx, state.ResourceID.ValueInt64(), state.RuleID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource rule", err.Error())
		return
	}
	state.fromAPI(rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourceRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	enabled := plan.Enabled.ValueBool()
	rule, err := r.client.UpdateResourceRule(ctx, state.ResourceID.ValueInt64(), state.RuleID.ValueInt64(), client.UpdateResourceRuleRequest{
		Action:   plan.Action.ValueString(),
		Match:    plan.Match.ValueString(),
		Value:    plan.Value.ValueString(),
		Priority: plan.Priority.ValueInt64(),
		Enabled:  &enabled,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource rule", err.Error())
		return
	}
	plan.RuleID = state.RuleID
	plan.fromAPI(rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteResourceRule(ctx, state.ResourceID.ValueInt64(), state.RuleID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error deleting resource rule", err.Error())
	}
}

func (r *resourceRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64Pair(ctx, req.ID, resp, "resource_id", "rule_id")
}
