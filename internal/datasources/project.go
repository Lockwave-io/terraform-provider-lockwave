package datasources

import (
	"context"
	"fmt"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectDataSource{}

// ProjectDataSource fetches a single Lockwave project by ID.
type ProjectDataSource struct {
	client *client.Client
}

// ProjectDataSourceModel is the Terraform model for the lockwave_project data source.
type ProjectDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

func (d *ProjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Lockwave project by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the project to look up.",
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
	}
}

func (d *ProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := d.client.GetProject(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}

	state := ProjectDataSourceModel{
		ID:        types.StringValue(project.ID),
		Name:      types.StringValue(project.Name),
		Slug:      types.StringValue(project.Slug),
		CreatedAt: types.StringValue(project.CreatedAt),
		UpdatedAt: types.StringValue(project.UpdatedAt),
	}

	if project.Description != nil {
		state.Description = types.StringValue(*project.Description)
	} else {
		state.Description = types.StringNull()
	}

	if project.Color != nil {
		state.Color = types.StringValue(*project.Color)
	} else {
		state.Color = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
