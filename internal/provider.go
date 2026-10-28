package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type subprocessProvider struct {
	version string
}

var _ provider.Provider = (*subprocessProvider)(nil)
var _ provider.ProviderWithEphemeralResources = (*subprocessProvider)(nil)

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider { return &subprocessProvider{version: version} }
}

func (p *subprocessProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "subprocess"
	resp.Version = p.version
}

func (p *subprocessProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Run a local process during an OpenTofu or Terraform operation."}
}

func (p *subprocessProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *subprocessProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *subprocessProvider) Resources(_ context.Context) []func() resource.Resource { return nil }

func (p *subprocessProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{func() ephemeral.EphemeralResource { return &processResource{} }}
}
