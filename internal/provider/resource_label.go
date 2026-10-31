package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &labelResource{}
	_ resource.ResourceWithImportState = &labelResource{}
)

func NewLabelResource() resource.Resource { return &labelResource{} }

type labelResource struct {
	resourceClient
}

type labelModel struct {
	OrgID   types.String `tfsdk:"org_id"`
	LabelID types.Int64  `tfsdk:"label_id"`
	Name    types.String `tfsdk:"name"`
	Color   types.String `tfsdk:"color"`
}

func (r *labelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_label"
}

func (r *labelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A label for organizing sites, resources, private resources and clients. Attach it with `pangolin_label_attachment`. Requires Pangolin 1.25 or later. Import using the format `<org_id>:<label_id>`, e.g. `terraform import pangolin_label.example acme:5`.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Organization the label belongs to.",
			},
			"label_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Pangolin-assigned label ID.",
			},
			"name": schema.StringAttribute{Required: true, Description: "Label name, unique per org regardless of case."},
			"color": schema.StringAttribute{
				Required:    true,
				Description: "Hex color, e.g. `#3b82f6`.",
				Validators: []validator.String{stringvalidator.RegexMatches(
					regexp.MustCompile(`^#?([0-9a-fA-F]{6}|[0-9a-fA-F]{3})$`), "must be a 3 or 6 digit hex color",
				)},
			},
		},
	}
}

func (m *labelModel) fromAPI(label *client.Label) {
	m.LabelID = types.Int64Value(label.LabelID)
	m.Name = types.StringValue(label.Name)
	m.Color = types.StringValue(label.Color)
}

func (r *labelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan labelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	label, err := r.client.CreateLabel(ctx, plan.OrgID.ValueString(), client.LabelRequest{Name: plan.Name.ValueString(), Color: plan.Color.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error creating label", err.Error())
		return
	}
	plan.fromAPI(label)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state labelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	label, err := r.client.GetLabel(ctx, state.OrgID.ValueString(), state.LabelID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading label", err.Error())
		return
	}
	state.fromAPI(label)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *labelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state labelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	label, err := r.client.UpdateLabel(ctx, state.OrgID.ValueString(), state.LabelID.ValueInt64(), client.LabelRequest{Name: plan.Name.ValueString(), Color: plan.Color.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error updating label", err.Error())
		return
	}
	plan.fromAPI(label)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state labelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteLabel(ctx, state.OrgID.ValueString(), state.LabelID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error deleting label", err.Error())
	}
}

func (r *labelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStringAndInt64(ctx, req.ID, resp, "org_id", "label_id")
}
