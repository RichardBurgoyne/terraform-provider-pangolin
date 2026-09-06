package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ resource.Resource = &userResourceGrantResource{}

func NewUserResourceGrantResource() resource.Resource { return &userResourceGrantResource{} }

type userResourceGrantResource struct {
	client *client.Client
}

type userResourceGrantModel struct {
	ResourceID types.Int64  `tfsdk:"resource_id"`
	UserID     types.String `tfsdk:"user_id"`
}

func (r *userResourceGrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_resource_grant"
}

func (r *userResourceGrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants a user access to a Pangolin resource. There is no update: changing either ID replaces the grant.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, Description: "Resource ID to grant access to."},
			"user_id":     schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "User ID being granted access."},
		},
	}
}

func (r *userResourceGrantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResourceGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AddUserToResource(ctx, plan.ResourceID.ValueInt64(), plan.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error granting user access to resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResourceGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userIDs, err := r.client.ListResourceUsers(ctx, state.ResourceID.ValueInt64())
	if err != nil {
		// A 404 here means the parent resource is gone, which means the grant
		// is gone too. Treat it as a removal rather than a hard refresh error.
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource users", err.Error())
		return
	}

	found := false
	for _, id := range userIDs {
		if id == state.UserID.ValueString() {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResourceGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResourceGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveUserFromResource(ctx, state.ResourceID.ValueInt64(), state.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error revoking user access from resource", err.Error())
	}
}
