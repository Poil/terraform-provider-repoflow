package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fe80/go-repoflow/pkg/repoflow"
)

// Ensure RepoflowProvider satisfies various provider interfaces.
var _ provider.Provider = &RepoflowProvider{}
var _ provider.ProviderWithFunctions = &RepoflowProvider{}
var _ provider.ProviderWithEphemeralResources = &RepoflowProvider{}
var _ provider.ProviderWithActions = &RepoflowProvider{}

// RepoflowProvider defines the provider implementation.
type RepoflowProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// RepoflowProviderModel describes the provider data model.
type RepoflowProviderModel struct {
	BaseURL types.String `tfsdk:"base_url"`
	ApiKey  types.String `tfsdk:"api_key"`
}

func (p *RepoflowProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "repoflow"
	resp.Version = p.version
}

func (p *RepoflowProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Repoflow",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(os.Getenv("REPOFLOW_BASE_URL")),
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Personnal Repoflow API key",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(os.Getenv("REPOFLOW_API_KEY")),
				Sensitive:           true,
			},
		},
	}
}

func (p *RepoflowProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RepoflowProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }

	// Example client configuration for data sources and resources
	baseURL := data.BaseURL.ValueString()
	apiKey := data.ApiKey.ValueString()

	rfClient := repoflow.NewClient(baseURL, apiKey)
	// rfClient := client.NewClient(&RepoflowProviderModel{})
	resp.DataSourceData = rfClient
	resp.ResourceData = rfClient
}

func (p *RepoflowProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource, NewRepositoryResource,
	}
}

func (p *RepoflowProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *RepoflowProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *RepoflowProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func (p *RepoflowProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RepoflowProvider{
			version: version,
		}
	}
}
