package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &domainDataSource{}

func NewDomainDataSource() datasource.DataSource { return &domainDataSource{} }

type domainDataSource struct {
	client *client.Client
}

func (d *domainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing domain by org_id and domain_id.",
		Attributes: map[string]schema.Attribute{
			"org_id":                schema.StringAttribute{Required: true},
			"domain_id":             schema.StringAttribute{Required: true},
			"type":                  schema.StringAttribute{Computed: true},
			"base_domain":           schema.StringAttribute{Computed: true},
			"cert_resolver":         schema.StringAttribute{Computed: true},
			"prefer_wildcard_cert":  schema.BoolAttribute{Computed: true},
			"verified":              schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *domainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model domainResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.GetDomain(ctx, model.OrgID.ValueString(), model.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	model.Type = types.StringValue(domain.Type)
	model.BaseDomain = types.StringValue(domain.BaseDomain)
	model.CertResolver = types.StringValue(domain.CertResolver)
	model.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	model.Verified = types.BoolValue(domain.Verified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
