package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackryze/terraform-provider-stackryze/internal/client"
)

// Ensure the implementation satisfies the provider interface.
var _ provider.Provider = &stackryzeProvider{}

type stackryzeProvider struct {
	version string
}

type providerModel struct {
	APIURL   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &stackryzeProvider{version: version}
	}
}

func (p *stackryzeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "stackryze"
	resp.Version = p.version
}

func (p *stackryzeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage DNS records on Stackryze DNS (dns.stackryze.com).",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "API base URL. Defaults to `https://api-dns.stackryze.com/api` or `STACKRYZE_API_URL`.",
			},
			"api_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API token with write scope. Falls back to `STACKRYZE_API_TOKEN`.",
			},
		},
	}
}

func (p *stackryzeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiURL := os.Getenv("STACKRYZE_API_URL")
	if !cfg.APIURL.IsNull() {
		apiURL = cfg.APIURL.ValueString()
	}
	token := os.Getenv("STACKRYZE_API_TOKEN")
	if !cfg.APIToken.IsNull() {
		token = cfg.APIToken.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"Set the api_token attribute or the STACKRYZE_API_TOKEN environment variable.",
		)
		return
	}

	c := client.New(apiURL, token)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *stackryzeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRecordResource,
	}
}

func (p *stackryzeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZoneDataSource,
	}
}
