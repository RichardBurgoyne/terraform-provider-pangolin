package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                   = &customDnsRecordResource{}
	_ resource.ResourceWithImportState    = &customDnsRecordResource{}
	_ resource.ResourceWithValidateConfig = &customDnsRecordResource{}
)

func NewCustomDnsRecordResource() resource.Resource { return &customDnsRecordResource{} }

type customDnsRecordResource struct {
	resourceClient
}

type customDnsRecordModel struct {
	OrgID             types.String `tfsdk:"org_id"`
	DomainID          types.String `tfsdk:"domain_id"`
	CustomDnsRecordID types.Int64  `tfsdk:"custom_dns_record_id"`
	Name              types.String `tfsdk:"name"`
	FullDomain        types.String `tfsdk:"full_domain"`
	Type              types.String `tfsdk:"type"`
	Values            types.List   `tfsdk:"values"`
	TTL               types.Int64  `tfsdk:"ttl"`
	Priority          types.Int64  `tfsdk:"priority"`
	Weight            types.Int64  `tfsdk:"weight"`
	Port              types.Int64  `tfsdk:"port"`
	Enabled           types.Bool   `tfsdk:"enabled"`
}

func (r *customDnsRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_dns_record"
}

func (r *customDnsRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	uint16Range := []validator.Int64{int64validator.Between(0, 65535)}
	resp.Schema = schema.Schema{
		Description: "An extra DNS record served by Pangolin's nameservers on a domain delegated to Pangolin DNS (an `ns` domain used by this organization alone). Requires Pangolin 1.26 or later with an Enterprise license. Import using the format `<org_id>:<domain_id>:<custom_dns_record_id>`, e.g. `terraform import pangolin_custom_dns_record.example acme:d1:3`. After an import, write `name` and `values` the way Pangolin stores them (lowercase, relative name, no trailing dots) or expect one in-place update that settles the difference.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "Organization the domain belongs to.",
			},
			"domain_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "Delegated domain the record is served under.",
			},
			"custom_dns_record_id": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Pangolin-assigned record ID.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Record name relative to the domain, e.g. `www`, `_dmarc` or `*.apps`. Use `@` for the domain itself. A fully qualified name within the domain is also accepted.",
			},
			"full_domain": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name the record is served at.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "One of A, AAAA, CNAME, MX, NS, SRV, TXT.",
				Validators:  []validator.String{stringvalidator.OneOf("A", "AAAA", "CNAME", "MX", "NS", "SRV", "TXT")},
			},
			"values": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Record data, one entry per resource record (e.g. both nameservers of an NS record). At most 10. CNAME records take exactly one. Hostnames for CNAME, NS, MX and SRV must be fully qualified.",
				Validators:  []validator.List{listvalidator.SizeBetween(1, 10)},
			},
			"ttl": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(300),
				Description: "Time to live in seconds, from 60 to 86400. Defaults to 300.",
				Validators:  []validator.Int64{int64validator.Between(60, 86400)},
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Description: "Priority. Required for MX and SRV records, and not valid for any other type.",
				Validators:  uint16Range,
			},
			"weight": schema.Int64Attribute{
				Optional:    true,
				Description: "Weight. Required for SRV records, and not valid for any other type.",
				Validators:  uint16Range,
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "Port. Required for SRV records, and not valid for any other type.",
				Validators:  uint16Range,
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the record is served. Defaults to true.",
			},
		},
	}
}

func (r *customDnsRecordResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config customDnsRecordModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !known(config.Type) {
		return
	}
	recordType := config.Type.ValueString()

	// The server drops these for types that don't use them, which would show
	// up as an inconsistent result after apply rather than a clear error.
	for _, field := range []struct {
		name  string
		value types.Int64
		types []string
	}{
		{"priority", config.Priority, []string{"MX", "SRV"}},
		{"weight", config.Weight, []string{"SRV"}},
		{"port", config.Port, []string{"SRV"}},
	} {
		if field.value.IsUnknown() {
			continue
		}
		needed := false
		for _, t := range field.types {
			needed = needed || t == recordType
		}
		switch {
		case needed && field.value.IsNull():
			resp.Diagnostics.AddAttributeError(path.Root(field.name), "Missing "+field.name, fmt.Sprintf("%s records require %s.", recordType, field.name))
		case !needed && !field.value.IsNull():
			resp.Diagnostics.AddAttributeError(path.Root(field.name), "Invalid "+field.name, fmt.Sprintf("%s is only valid for %s records.", field.name, strings.Join(field.types, " and ")))
		}
	}

	if recordType == "CNAME" && known(config.Values) && len(config.Values.Elements()) > 1 {
		resp.Diagnostics.AddAttributeError(path.Root("values"), "Too many values", "A CNAME record can only have one value.")
	}
}

func (m *customDnsRecordModel) request(ctx context.Context) (client.CustomDnsRecordRequest, diag.Diagnostics) {
	var values []string
	diags := m.Values.ElementsAs(ctx, &values, false)
	return client.CustomDnsRecordRequest{
		Name:     m.Name.ValueString(),
		Type:     m.Type.ValueString(),
		Values:   values,
		TTL:      int64Ptr(m.TTL),
		Priority: int64Ptr(m.Priority),
		Weight:   int64Ptr(m.Weight),
		Port:     int64Ptr(m.Port),
		Enabled:  boolPtr(m.Enabled),
	}, diags
}

// fromAPI copies record into m. Pangolin normalizes the name and values
// (lowercasing hostnames, dropping trailing dots and TXT quotes, and turning
// a fully qualified name into a relative one), so the configured spelling is
// kept wherever it normalizes to what the server stored.
func (m *customDnsRecordModel) fromAPI(ctx context.Context, record *client.CustomDnsRecord) diag.Diagnostics {
	baseDomain := record.FullDomain
	if record.Name != "@" {
		baseDomain = strings.TrimPrefix(record.FullDomain, record.Name+".")
	}

	m.DomainID = types.StringValue(record.DomainID)
	m.CustomDnsRecordID = types.Int64Value(record.CustomDnsRecordID)
	if !known(m.Name) || normalizeDnsName(m.Name.ValueString(), baseDomain) != record.Name {
		m.Name = types.StringValue(record.Name)
	}
	m.FullDomain = types.StringValue(record.FullDomain)
	m.Type = types.StringValue(record.Type)
	m.TTL = types.Int64Value(record.TTL)
	m.Priority = nullableInt64(record.Priority)
	m.Weight = nullableInt64(record.Weight)
	m.Port = nullableInt64(record.Port)
	m.Enabled = types.BoolValue(record.Enabled)

	var prior []string
	if known(m.Values) {
		if d := m.Values.ElementsAs(ctx, &prior, false); d.HasError() {
			prior = nil
		}
	}
	if !sameDnsValues(record.Type, prior, record.Values) {
		values, diags := types.ListValueFrom(ctx, types.StringType, record.Values)
		if diags.HasError() {
			return diags
		}
		m.Values = values
	}
	return nil
}

// normalizeDnsName mirrors how Pangolin stores a record name: lowercased,
// without a trailing dot, relative to the domain, and "@" for the domain
// itself.
func normalizeDnsName(name, baseDomain string) string {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	baseDomain = strings.ToLower(baseDomain)
	if name == "" || name == "@" || name == baseDomain {
		return "@"
	}
	return strings.TrimSuffix(name, "."+baseDomain)
}

// normalizeDnsValue mirrors how Pangolin stores one value of a record.
func normalizeDnsValue(recordType, value string) string {
	switch recordType {
	case "A":
		return strings.TrimSpace(value)
	case "AAAA":
		return strings.ToLower(strings.TrimSpace(value))
	case "TXT":
		if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value[1 : len(value)-1]
		}
		return value
	default:
		return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	}
}

func sameDnsValues(recordType string, configured, stored []string) bool {
	if len(configured) != len(stored) {
		return false
	}
	for i := range configured {
		if normalizeDnsValue(recordType, configured[i]) != stored[i] {
			return false
		}
	}
	return true
}

func (r *customDnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customDnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in, diags := plan.request(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	record, err := r.client.CreateCustomDnsRecord(ctx, plan.OrgID.ValueString(), plan.DomainID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom DNS record", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromAPI(ctx, record)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customDnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customDnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	record, err := r.client.GetCustomDnsRecord(ctx, state.OrgID.ValueString(), state.DomainID.ValueString(), state.CustomDnsRecordID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom DNS record", err.Error())
		return
	}
	resp.Diagnostics.Append(state.fromAPI(ctx, record)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customDnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customDnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in, diags := plan.request(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	record, err := r.client.UpdateCustomDnsRecord(ctx, state.OrgID.ValueString(), state.DomainID.ValueString(), state.CustomDnsRecordID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom DNS record", err.Error())
		return
	}
	resp.Diagnostics.Append(plan.fromAPI(ctx, record)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customDnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customDnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteCustomDnsRecord(ctx, state.OrgID.ValueString(), state.DomainID.ValueString(), state.CustomDnsRecordID.ValueInt64())
	if err := client.IgnoreNotFound(err); err != nil {
		resp.Diagnostics.AddError("Error deleting custom DNS record", err.Error())
	}
}

func (r *customDnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid Import ID", `expected format: <org_id>:<domain_id>:<custom_dns_record_id>, e.g. "acme:d1:3"`)
		return
	}
	recordID, ok := parseImportInt64("custom_dns_record_id", parts[2], &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("custom_dns_record_id"), recordID)...)
}
