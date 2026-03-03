package datasources

import (
	"context"
	"fmt"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectsDataSource{}

// ProjectsDataSource fetches all Lockwave projects for the current team.
type ProjectsDataSource struct {
	client *client.Client
}

// ProjectsDataSourceModel is the Terraform model for the lockwave_projects data source.
type ProjectsDataSourceModel struct {
	Projects []ProjectItemModel `tfsdk:"projects"`
}

// ProjectItemModel represents a single project in the list.
type ProjectItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewProjectsDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

func (d *ProjectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *ProjectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all Lockwave projects for the current team.",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of projects in the team.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "UUID of the project.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the project.",
						},
						"slug": schema.StringAttribute{
							Computed:    true,
							Description: "URL-friendly slug of the project.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Description of the project.",
						},
						"color": schema.StringAttribute{
							Computed:    true,
							Description: "Hex color of the project badge.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "ISO 8601 creation timestamp.",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							Description: "ISO 8601 last-updated timestamp.",
						},
					},
				},
			},
		},
	}
}

func (d *ProjectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	projects, err := d.client.ListProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing projects", err.Error())
		return
	}

	items := make([]ProjectItemModel, 0, len(projects))
	for _, p := range projects {
		item := ProjectItemModel{
			ID:        types.StringValue(p.ID),
			Name:      types.StringValue(p.Name),
			Slug:      types.StringValue(p.Slug),
			CreatedAt: types.StringValue(p.CreatedAt),
			UpdatedAt: types.StringValue(p.UpdatedAt),
		}
		if p.Description != nil {
			item.Description = types.StringValue(*p.Description)
		} else {
			item.Description = types.StringNull()
		}
		if p.Color != nil {
			item.Color = types.StringValue(*p.Color)
		} else {
			item.Color = types.StringNull()
		}
		items = append(items, item)
	}

	state := ProjectsDataSourceModel{Projects: items}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
