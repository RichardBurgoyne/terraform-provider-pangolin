package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

func NewAPIKeyResource() resource.Resource { return &apiKeyResource{} }

type apiKeyResource struct {
	client *client.Client
}

type apiKeyResourceModel struct {
	OrgID     types.String `tfsdk:"org_id"`
	APIKeyID  types.String `tfsdk:"api_key_id"`
	Name      types.String `tfsdk:"name"`
	APIKey    types.String `tfsdk:"api_key"`
	LastChars types.String `tfsdk:"last_chars"`
	ActionIDs types.List   `tfsdk:"action_ids"`
}

func (r *apiKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an organization-scoped Pangolin API key. The secret key value is only ever available at creation time.",
		Attributes: map[string]schema.Attribute{
			"org_id":     schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Organization ID this key belongs to."},
			"api_key_id": schema.StringAttribute{Computed: true, Description: "Server-generated key ID."},
			"name":       schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Display name for the key."},
			"api_key":    schema.StringAttribute{Computed: true, Sensitive: true, Description: "The secret key value. Only ever populated from the create response; not recoverable afterward."},
			"last_chars": schema.StringAttribute{Computed: true, Description: "Last 4 characters of the key, for identification."},
			"action_ids": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Actions this key is permitted to perform. Replaces the full list on every change."},
		},
	}
}

func (r *apiKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrgID.ValueString()
	key, err := r.client.CreateAPIKey(ctx, orgID, client.CreateAPIKeyRequest{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error creating API key", err.Error())
		return
	}

	plan.APIKeyID = types.StringValue(key.APIKeyID)
	plan.APIKey = types.StringValue(key.APIKey)
	plan.LastChars = types.StringValue(key.LastChars)

	if !plan.ActionIDs.IsUnknown() && !plan.ActionIDs.IsNull() {
		var actionIDs []string
		resp.Diagnostics.Append(plan.ActionIDs.ElementsAs(ctx, &actionIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetAPIKeyActions(ctx, orgID, key.APIKeyID, actionIDs); err != nil {
			resp.Diagnostics.AddError("Error setting API key actions", err.Error())
			return
		}
	} else {
		plan.ActionIDs = types.ListValueMust(types.StringType, nil)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	key, err := r.client.GetAPIKey(ctx, orgID, state.APIKeyID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API key", err.Error())
		return
	}

	apiKeyValue := state.APIKey // write-once secret, not returned by GetAPIKey
	state.Name = types.StringValue(key.Name)
	state.LastChars = types.StringValue(key.LastChars)
	state.APIKey = apiKeyValue

	actionIDs, err := r.client.ListAPIKeyActions(ctx, orgID, state.APIKeyID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading API key actions", err.Error())
		return
	}
	listValue, diags := types.ListValueFrom(ctx, types.StringType, actionIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ActionIDs = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actionIDs []string
	resp.Diagnostics.Append(plan.ActionIDs.ElementsAs(ctx, &actionIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	if err := r.client.SetAPIKeyActions(ctx, orgID, state.APIKeyID.ValueString(), actionIDs); err != nil {
		resp.Diagnostics.AddError("Error updating API key actions", err.Error())
		return
	}

	plan.APIKeyID = state.APIKeyID
	plan.APIKey = state.APIKey
	plan.LastChars = state.LastChars
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAPIKey(ctx, state.OrgID.ValueString(), state.APIKeyID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting API key", err.Error())
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("api_key_id"), req, resp)
}
