package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ provider.Provider = &pangolinProvider{}

type pangolinProvider struct {
	version string
}

type pangolinProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
	OrgID    types.String `tfsdk:"org_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pangolinProvider{version: version}
	}
}

func (p *pangolinProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pangolin"
	resp.Version = p.version
}

func (p *pangolinProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with a self-hosted or cloud Pangolin instance.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Pangolin integration API, including any version prefix your instance uses (e.g. https://pangolin.example.com/v1). Defaults to the PANGOLIN_ENDPOINT environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer API key (organization or root scoped). Defaults to the PANGOLIN_API_KEY environment variable.",
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Description: "Default organization ID used by resources that don't set org_id explicitly. Defaults to the PANGOLIN_ORG_ID environment variable.",
			},
		},
	}
}

func (p *pangolinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config pangolinProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("PANGOLIN_ENDPOINT")
	}
	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("PANGOLIN_API_KEY")
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("endpoint"),
			"Missing Pangolin API Endpoint",
			"Set the endpoint attribute or the PANGOLIN_ENDPOINT environment variable.",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("api_key"),
			"Missing Pangolin API Key",
			"Set the api_key attribute or the PANGOLIN_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(endpoint, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *pangolinProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrganizationResource,
		NewDomainResource,
		NewSiteResource,
	}
}

func (p *pangolinProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewDomainDataSource,
		NewSiteDataSource,
	}
}
