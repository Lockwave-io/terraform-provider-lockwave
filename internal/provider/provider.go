// Package provider implements the Terraform Plugin Framework provider for Lockwave.
package provider

import (
	"context"
	"os"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/fwartner/terraform-provider-lockwave/internal/datasources"
	"github.com/fwartner/terraform-provider-lockwave/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure LockwaveProvider satisfies provider.Provider.
var _ provider.Provider = &LockwaveProvider{}
var _ provider.ProviderWithFunctions = &LockwaveProvider{}

// LockwaveProvider is the root provider struct.
type LockwaveProvider struct {
	version string
}

// LockwaveProviderModel is the configuration model for the provider block.
type LockwaveProviderModel struct {
	APIURL   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
	TeamID   types.String `tfsdk:"team_id"`
}

// New returns a factory function for the Lockwave provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LockwaveProvider{version: version}
	}
}

// Metadata returns provider metadata.
func (p *LockwaveProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lockwave"
	resp.Version = p.version
}

// Schema returns the provider configuration schema.
func (p *LockwaveProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Lockwave provider manages SSH key lifecycle via the Lockwave SaaS API.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Lockwave API. Defaults to https://lockwave.io. Can also be set via the LOCKWAVE_API_URL environment variable.",
			},
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Sanctum Bearer token for authenticating with the Lockwave API. Can also be set via the LOCKWAVE_API_TOKEN environment variable.",
			},
			"team_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the Lockwave team that all resources belong to. Can also be set via the LOCKWAVE_TEAM_ID environment variable.",
			},
		},
	}
}

// Configure creates the shared API client from provider configuration.
func (p *LockwaveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config LockwaveProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiURL := os.Getenv("LOCKWAVE_API_URL")
	if !config.APIURL.IsNull() && !config.APIURL.IsUnknown() {
		apiURL = config.APIURL.ValueString()
	}

	apiToken := os.Getenv("LOCKWAVE_API_TOKEN")
	if !config.APIToken.IsNull() && !config.APIToken.IsUnknown() {
		apiToken = config.APIToken.ValueString()
	}

	teamID := os.Getenv("LOCKWAVE_TEAM_ID")
	if !config.TeamID.IsNull() && !config.TeamID.IsUnknown() {
		teamID = config.TeamID.ValueString()
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The Lockwave provider requires an api_token. Set it in the provider configuration or via the LOCKWAVE_API_TOKEN environment variable.",
		)
	}

	if teamID == "" {
		resp.Diagnostics.AddError(
			"Missing Team ID",
			"The Lockwave provider requires a team_id. Set it in the provider configuration or via the LOCKWAVE_TEAM_ID environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewClient(apiURL, apiToken, teamID)
	resp.DataSourceData = c
	resp.ResourceData = c
}

// Resources returns all managed resource types.
func (p *LockwaveProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewHostResource,
		resources.NewHostUserResource,
		resources.NewSshKeyResource,
		resources.NewAssignmentResource,
		resources.NewWebhookEndpointResource,
		resources.NewNotificationChannelResource,
		resources.NewAuditLogStreamResource,
	}
}

// DataSources returns all data source types.
func (p *LockwaveProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewHostDataSource,
		datasources.NewHostsDataSource,
		datasources.NewSshKeyDataSource,
		datasources.NewSshKeysDataSource,
		datasources.NewTeamDataSource,
	}
}

// Functions returns provider-level functions (none currently).
func (p *LockwaveProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}
