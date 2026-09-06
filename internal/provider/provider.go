package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &pangolinProvider{}

type pangolinProvider struct {
	version string
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

func (p *pangolinProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *pangolinProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
