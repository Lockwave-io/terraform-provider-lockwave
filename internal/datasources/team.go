package datasources

import (
	"context"
	"fmt"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure TeamDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &TeamDataSource{}

// TeamDataSource reads the current Lockwave team.
type TeamDataSource struct {
	client *client.Client
}

// TeamDataSourceModel is the Terraform model for the lockwave_team data source.
type TeamDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	PersonalTeam types.Bool   `tfsdk:"personal_team"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

// NewTeamDataSource is the factory function.
func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

func (d *TeamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the Lockwave team identified by the provider's team_id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the team.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Team name.",
			},
			"personal_team": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a personal (single-user) team.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 creation timestamp.",
			},
		},
	}
}

func (d *TeamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *TeamDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	team, err := d.client.GetCurrentTeam(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}

	state := TeamDataSourceModel{
		ID:           types.StringValue(team.ID),
		Name:         types.StringValue(team.Name),
		PersonalTeam: types.BoolValue(team.PersonalTeam),
		CreatedAt:    types.StringValue(team.CreatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
