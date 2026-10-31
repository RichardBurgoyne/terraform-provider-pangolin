package provider

import (
	"context"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                   = &resourceMtlsResource{}
	_ resource.ResourceWithImportState    = &resourceMtlsResource{}
	_ resource.ResourceWithValidateConfig = &resourceMtlsResource{}
)

func NewResourceMtlsResource() resource.Resource { return &resourceMtlsResource{} }

type resourceMtlsResource struct {
	resourceClient
}

type resourceMtlsModel struct {
	ResourceID       types.Int64  `tfsdk:"resource_id"`
	CACertificatePEM types.String `tfsdk:"ca_certificate_pem"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Fingerprints     types.Set    `tfsdk:"fingerprints"`
}

func (r *resourceMtlsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_mtls"
}

func (r *resourceMtlsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Requires clients of a resource to present a certificate signed by a trusted CA. Certificates and the on/off switch live together here because Pangolin won't enable mTLS on a resource with no CA. The resource must have SSL enabled and can't be tcp or udp. A resource uses its shared resource policy's mTLS settings if the policy enables mTLS. Requires Pangolin 1.25 or later. Import using the resource ID, e.g. `terraform import pangolin_resource_mtls.example 5`; the API doesn't return PEM data, so the first apply after import uploads `ca_certificate_pem` and removes any other CA.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Description:   "ID of the resource to protect.",
			},
			"ca_certificate_pem": schema.StringAttribute{
				Required:    true,
				Description: "Trusted CA certificates in PEM format. Holds one certificate or a bundle of up to 20; Pangolin stores each separately. This is the complete list: other CAs on the resource are removed.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether clients must present a certificate. Defaults to true.",
			},
			"fingerprints": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "SHA-256 fingerprints of the CA certificates on the resource.",
			},
		},
	}
}

func (r *resourceMtlsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config resourceMtlsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !known(config.CACertificatePEM) {
		return
	}
	if len(pemCertificates(config.CACertificatePEM.ValueString())) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("ca_certificate_pem"), "Invalid ca_certificate_pem", "ca_certificate_pem must contain at least one PEM block starting with -----BEGIN CERTIFICATE-----.")
	}
}

// pemCertificates maps the fingerprint of each certificate in a PEM bundle to
// its re-encoded block, in the format the API reports fingerprints.
func pemCertificates(bundle string) map[string]string {
	certs := map[string]string{}
	rest := []byte(bundle)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return certs
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		sum := sha256.Sum256(block.Bytes)
		hexBytes := make([]string, len(sum))
		for i, b := range sum {
			hexBytes[i] = fmt.Sprintf("%02X", b)
		}
		certs[strings.Join(hexBytes, ":")] = string(pem.EncodeToMemory(block))
	}
}

// sync makes the certificates and enabled flag on the resource match the
// plan. New CAs go on before old ones come off, so that a rotation never
// leaves the resource without one: removing the last CA turns mTLS off.
func (r *resourceMtlsResource) sync(ctx context.Context, plan *resourceMtlsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	resourceID := plan.ResourceID.ValueInt64()
	desired := pemCertificates(plan.CACertificatePEM.ValueString())

	current, err := r.client.ListResourceMtlsCertificates(ctx, resourceID)
	if err != nil {
		diags.AddError("Error listing mTLS certificates", err.Error())
		return diags
	}

	missing := make(map[string]string, len(desired))
	for fingerprint, block := range desired {
		missing[fingerprint] = block
	}
	for _, cert := range current {
		delete(missing, cert.Fingerprint)
	}
	if len(missing) > 0 {
		var bundle strings.Builder
		for _, block := range missing {
			bundle.WriteString(block)
		}
		if err := r.client.AddResourceMtlsCertificates(ctx, resourceID, bundle.String()); err != nil {
			diags.AddError("Error adding mTLS certificates", err.Error())
			return diags
		}
	}

	for _, cert := range current {
		if _, keep := desired[cert.Fingerprint]; keep {
			continue
		}
		if err := client.IgnoreNotFound(r.client.DeleteResourceMtlsCertificate(ctx, resourceID, cert.MtlsCertificateID)); err != nil {
			diags.AddError("Error removing mTLS certificate", err.Error())
			return diags
		}
	}

	enabled := plan.Enabled.ValueBool()
	if _, err := r.client.UpdateResource(ctx, resourceID, client.UpdateResourceRequest{MtlsEnabled: &enabled}); err != nil {
		diags.AddError("Error setting mTLS on resource", err.Error())
		return diags
	}

	fingerprints := make([]string, 0, len(desired))
	for fingerprint := range desired {
		fingerprints = append(fingerprints, fingerprint)
	}
	var d diag.Diagnostics
	plan.Fingerprints, d = types.SetValueFrom(ctx, types.StringType, fingerprints)
	diags.Append(d...)
	return diags
}

func (r *resourceMtlsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceMtlsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.sync(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceMtlsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceMtlsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceID := state.ResourceID.ValueInt64()

	res, err := r.client.GetResource(ctx, resourceID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}
	certs, err := r.client.ListResourceMtlsCertificates(ctx, resourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing mTLS certificates", err.Error())
		return
	}

	fingerprints := make([]string, 0, len(certs))
	for _, cert := range certs {
		fingerprints = append(fingerprints, cert.Fingerprint)
	}
	state.Enabled = types.BoolValue(res.MtlsEnabled)
	var d diag.Diagnostics
	state.Fingerprints, d = types.SetValueFrom(ctx, types.StringType, fingerprints)
	resp.Diagnostics.Append(d...)

	// The API never returns PEM data, so drift can only be detected by
	// fingerprint. Clearing the bundle makes the next plan re-apply it.
	want := pemCertificates(state.CACertificatePEM.ValueString())
	slices.Sort(fingerprints)
	wantFingerprints := make([]string, 0, len(want))
	for fingerprint := range want {
		wantFingerprints = append(wantFingerprints, fingerprint)
	}
	slices.Sort(wantFingerprints)
	if !slices.Equal(fingerprints, wantFingerprints) {
		state.CACertificatePEM = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceMtlsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceMtlsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.sync(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceMtlsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceMtlsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resourceID := state.ResourceID.ValueInt64()
	certs, err := r.client.ListResourceMtlsCertificates(ctx, resourceID)
	if err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error listing mTLS certificates", err.Error())
		}
		return
	}
	for _, cert := range certs {
		if err := client.IgnoreNotFound(r.client.DeleteResourceMtlsCertificate(ctx, resourceID, cert.MtlsCertificateID)); err != nil {
			resp.Diagnostics.AddError("Error removing mTLS certificate", err.Error())
			return
		}
	}
}

func (r *resourceMtlsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, ok := parseImportInt64("resource_id", req.ID, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), id)...)
}
