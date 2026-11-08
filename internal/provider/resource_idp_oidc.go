package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &idpOidcResource{}
	_ resource.ResourceWithImportState = &idpOidcResource{}
)

func NewIdpOidcResource() resource.Resource { return &idpOidcResource{} }

type idpOidcResource struct {
	resourceClient
}

type idpOidcModel struct {
	IdpID               types.Int64  `tfsdk:"idp_id"`
	Name                types.String `tfsdk:"name"`
	Variant             types.String `tfsdk:"variant"`
	ClientID            types.String `tfsdk:"client_id"`
	ClientSecret        types.String `tfsdk:"client_secret"`
	AuthURL             types.String `tfsdk:"auth_url"`
	TokenURL            types.String `tfsdk:"token_url"`
	Scopes              types.String `tfsdk:"scopes"`
	IdentifierPath      types.String `tfsdk:"identifier_path"`
	EmailPath           types.String `tfsdk:"email_path"`
	NamePath            types.String `tfsdk:"name_path"`
	AutoProvision       types.Bool   `tfsdk:"auto_provision"`
	SessionRevalidation types.Bool   `tfsdk:"session_revalidation"`
	DefaultRoleMapping  types.String `tfsdk:"default_role_mapping"`
	DefaultOrgMapping   types.String `tfsdk:"default_org_mapping"`
	Tags                types.String `tfsdk:"tags"`
	RedirectURL         types.String `tfsdk:"redirect_url"`
}

func (r *idpOidcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_idp_oidc"
}

func (r *idpOidcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A global OIDC identity provider users can sign in with. Needs a root API key, and isn't available when the server runs with `IDENTITY_PROVIDER_MODE=org`. Import using the IdP ID, e.g. `terraform import pangolin_idp_oidc.example 4`. With session_revalidation on, an imported IdP's scopes include `offline_access`; leave it in `scopes` or expect one in-place update that settles the difference.",
		Attributes: map[string]schema.Attribute{
			"idp_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Pangolin-assigned IdP ID.",
			},
			"name": schema.StringAttribute{Required: true, Description: "Display name shown on the login page."},
			"variant": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("oidc"),
				Description: "Provider flavour: oidc, google or azure. Defaults to oidc.",
				Validators:  []validator.String{stringvalidator.OneOf("oidc", "google", "azure")},
			},
			"client_id":     schema.StringAttribute{Required: true, Sensitive: true, Description: "OAuth client ID."},
			"client_secret": schema.StringAttribute{Required: true, Sensitive: true, Description: "OAuth client secret."},
			"auth_url":      schema.StringAttribute{Required: true, Description: "Authorization endpoint URL."},
			"token_url":     schema.StringAttribute{Required: true, Description: "Token endpoint URL."},
			"scopes": schema.StringAttribute{
				Required:    true,
				Description: "Space-separated scopes to request, e.g. `openid profile email`. With session_revalidation on, Pangolin adds `offline_access` itself (or, for Google, removes it), so there is no need to list it.",
			},
			"identifier_path": schema.StringAttribute{Required: true, Description: "JMESPath to the user's unique identifier in the ID token, e.g. `sub`."},
			"email_path":      schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, Description: "JMESPath to the user's email in the ID token."},
			"name_path":       schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, Description: "JMESPath to the user's display name in the ID token."},
			"auto_provision": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Create users and assign their orgs and roles automatically on first sign-in. Defaults to false.",
			},
			"session_revalidation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Keep checking a signed-in user's session with the identity provider using refresh tokens, and end it once the provider no longer accepts it. Requires Pangolin 1.26 or later with an Enterprise license. Defaults to false.",
			},
			"default_role_mapping": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "JMESPath expression that picks the role for auto-provisioned users. Pangolin starts it as `'Member'`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"default_org_mapping": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "JMESPath expression that picks the orgs for auto-provisioned users. Pangolin starts it as `'{{orgId}}'`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tags": schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, Description: "Free-form tags."},
			"redirect_url": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Callback URL to register with the identity provider. Only known for IdPs created by Terraform, not imported ones.",
			},
		},
	}
}

func (m *idpOidcModel) request() client.OidcIdpRequest {
	return client.OidcIdpRequest{
		Name:               m.Name.ValueString(),
		ClientID:           m.ClientID.ValueString(),
		ClientSecret:       m.ClientSecret.ValueString(),
		AuthURL:            m.AuthURL.ValueString(),
		TokenURL:           m.TokenURL.ValueString(),
		IdentifierPath:     m.IdentifierPath.ValueString(),
		EmailPath:          strPtr(m.EmailPath),
		NamePath:           strPtr(m.NamePath),
		Scopes:             m.Scopes.ValueString(),
		AutoProvision:      boolPtr(m.AutoProvision),
		Tags:               strPtr(m.Tags),
		Variant:            strPtr(m.Variant),
		DefaultRoleMapping: strPtr(m.DefaultRoleMapping),
		DefaultOrgMapping:  strPtr(m.DefaultOrgMapping),
	}
}

// fromAPI copies idp into m. Turning on session revalidation makes Pangolin
// rewrite the scopes, so the configured scopes are kept when they match what
// the server stored once that rewrite is applied.
func (m *idpOidcModel) fromAPI(idp *client.OidcIdp) {
	m.IdpID = types.Int64Value(idp.IdpID)
	m.Name = types.StringValue(idp.Name)
	m.Variant = types.StringValue(idp.Variant)
	m.ClientID = types.StringValue(idp.ClientID)
	m.ClientSecret = types.StringValue(idp.ClientSecret)
	m.AuthURL = types.StringValue(idp.AuthURL)
	m.TokenURL = types.StringValue(idp.TokenURL)
	if !known(m.Scopes) || storedScopes(m.Scopes.ValueString(), idp.SessionRevalidation, idp.Variant) != idp.Scopes {
		m.Scopes = types.StringValue(idp.Scopes)
	}
	m.IdentifierPath = types.StringValue(idp.IdentifierPath)
	m.EmailPath = nonEmptyStringPtr(idp.EmailPath)
	m.NamePath = nonEmptyStringPtr(idp.NamePath)
	m.AutoProvision = types.BoolValue(idp.AutoProvision)
	m.SessionRevalidation = types.BoolValue(idp.SessionRevalidation)
	m.DefaultRoleMapping = nullableString(idp.DefaultRoleMapping)
	m.DefaultOrgMapping = nullableString(idp.DefaultOrgMapping)
	m.Tags = nonEmptyStringPtr(idp.Tags)
}

// storedScopes mirrors the scopes Pangolin stores for the configured ones:
// with session revalidation it adds offline_access, except for Google, which
// asks for offline access another way and has the scope removed.
func storedScopes(scopes string, sessionRevalidation bool, variant string) string {
	if !sessionRevalidation {
		return scopes
	}
	parts := []string{}
	hasOfflineAccess := false
	for _, scope := range strings.Fields(scopes) {
		if scope == "offline_access" {
			hasOfflineAccess = true
			if variant == "google" {
				continue
			}
		}
		parts = append(parts, scope)
	}
	if variant != "google" && !hasOfflineAccess {
		parts = append(parts, "offline_access")
	}
	return strings.Join(parts, " ")
}

// nonEmptyStringPtr maps both null and the empty string, which is how a
// cleared optional field is stored, to null.
func nonEmptyStringPtr(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return nonEmptyString(*p)
}

func (r *idpOidcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan idpOidcModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := plan.request()
	// Leave the field out unless it's on, so IdPs without it still work on
	// servers older than 1.26, whose request schema rejects unknown fields.
	if plan.SessionRevalidation.ValueBool() {
		in.SessionRevalidation = boolPtr(plan.SessionRevalidation)
	}
	idpID, redirectURL, err := r.client.CreateOidcIdp(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating identity provider", err.Error())
		return
	}
	plan.IdpID = types.Int64Value(idpID)
	plan.RedirectURL = types.StringValue(redirectURL)
	// Save the ID straight away so a failure below doesn't orphan the IdP.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("idp_id"), idpID)...)

	// Create doesn't take the mappings, so apply any that are configured.
	if in.DefaultRoleMapping != nil || in.DefaultOrgMapping != nil {
		if err := r.client.UpdateOidcIdp(ctx, idpID, in); err != nil {
			resp.Diagnostics.AddError("Error setting identity provider mappings", err.Error())
			return
		}
	}

	idp, err := r.client.GetOidcIdp(ctx, idpID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading identity provider", err.Error())
		return
	}
	plan.fromAPI(idp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *idpOidcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state idpOidcModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idp, err := r.client.GetOidcIdp(ctx, state.IdpID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading identity provider", err.Error())
		return
	}
	state.fromAPI(idp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *idpOidcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state idpOidcModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := plan.request()
	if plan.SessionRevalidation.ValueBool() || !plan.SessionRevalidation.Equal(state.SessionRevalidation) {
		in.SessionRevalidation = boolPtr(plan.SessionRevalidation)
	}
	// Omitted fields are left as they are, so clear removed ones explicitly.
	empty := ""
	for _, field := range []struct {
		value types.String
		dest  **string
	}{
		{plan.EmailPath, &in.EmailPath},
		{plan.NamePath, &in.NamePath},
		{plan.Tags, &in.Tags},
	} {
		if field.value.IsNull() {
			*field.dest = &empty
		}
	}

	if err := r.client.UpdateOidcIdp(ctx, state.IdpID.ValueInt64(), in); err != nil {
		resp.Diagnostics.AddError("Error updating identity provider", err.Error())
		return
	}
	idp, err := r.client.GetOidcIdp(ctx, state.IdpID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading identity provider", err.Error())
		return
	}
	plan.fromAPI(idp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *idpOidcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state idpOidcModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.IgnoreNotFound(r.client.DeleteIdp(ctx, state.IdpID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error deleting identity provider", err.Error())
	}
}

func (r *idpOidcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idpID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "expected the numeric IdP ID, e.g. \"4\"")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("idp_id"), idpID)...)
}
