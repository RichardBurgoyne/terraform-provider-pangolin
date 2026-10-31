package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource                = &labelAttachmentResource{}
	_ resource.ResourceWithImportState = &labelAttachmentResource{}
)

func NewLabelAttachmentResource() resource.Resource { return &labelAttachmentResource{} }

type labelAttachmentResource struct {
	resourceClient
}

type labelAttachmentModel struct {
	OrgID             types.String `tfsdk:"org_id"`
	LabelID           types.Int64  `tfsdk:"label_id"`
	SiteID            types.Int64  `tfsdk:"site_id"`
	ResourceID        types.Int64  `tfsdk:"resource_id"`
	PrivateResourceID types.Int64  `tfsdk:"private_resource_id"`
	ClientID          types.Int64  `tfsdk:"client_id"`
}

func (r *labelAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_label_attachment"
}

func (r *labelAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	target := func(description string) schema.Int64Attribute {
		return schema.Int64Attribute{
			Optional:      true,
			PlanModifiers: replace,
			Description:   description + " Exactly one of site_id, resource_id, private_resource_id and client_id must be set.",
		}
	}
	resourceAttr := target("Resource to attach the label to.")
	resourceAttr.Validators = []validator.Int64{int64validator.ExactlyOneOf(
		path.MatchRoot("site_id"), path.MatchRoot("private_resource_id"), path.MatchRoot("client_id"),
	)}

	resp.Schema = schema.Schema{
		Description: "Attaches a label to a site, resource, private resource or machine client. There is no update: changing any attribute replaces the attachment. Requires Pangolin 1.25 or later. Import using the format `<org_id>:<label_id>:<site|resource|private_resource|client>:<id>`, e.g. `terraform import pangolin_label_attachment.example acme:3:site:12`.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Organization the label belongs to.",
			},
			"label_id": schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Label to attach."},

			"site_id":             target("Site to attach the label to."),
			"resource_id":         resourceAttr,
			"private_resource_id": target("Private resource to attach the label to."),
			"client_id":           target("Machine client to attach the label to."),
		},
	}
}

func (m *labelAttachmentModel) target() client.LabelTarget {
	return client.LabelTarget{
		SiteID:         int64Ptr(m.SiteID),
		ResourceID:     int64Ptr(m.ResourceID),
		SiteResourceID: int64Ptr(m.PrivateResourceID),
		ClientID:       int64Ptr(m.ClientID),
	}
}

func (r *labelAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan labelAttachmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AttachLabel(ctx, plan.OrgID.ValueString(), plan.LabelID.ValueInt64(), plan.target()); err != nil {
		resp.Diagnostics.AddError("Error attaching label", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state labelAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attached, err := r.client.IsLabelAttached(ctx, state.OrgID.ValueString(), state.LabelID.ValueInt64(), state.target())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading label attachment", err.Error())
		return
	}
	if !attached {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *labelAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan labelAttachmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state labelAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DetachLabel(ctx, state.OrgID.ValueString(), state.LabelID.ValueInt64(), state.target())); err != nil {
		resp.Diagnostics.AddError("Error detaching label", err.Error())
	}
}

func (r *labelAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const format = `expected format: <org_id>:<label_id>:<site|resource|private_resource|client>:<id>, e.g. "acme:3:site:12"`
	parts := strings.Split(req.ID, ":")
	if len(parts) != 4 || parts[0] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", format)
		return
	}
	attrs := map[string]string{"site": "site_id", "resource": "resource_id", "private_resource": "private_resource_id", "client": "client_id"}
	attr, ok := attrs[parts[2]]
	if !ok {
		resp.Diagnostics.AddError("Invalid Import ID", format)
		return
	}
	labelID, ok := parseImportInt64("label_id", parts[1], &resp.Diagnostics)
	if !ok {
		return
	}
	targetID, ok := parseImportInt64(attr, parts[3], &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("label_id"), labelID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), targetID)...)
}
