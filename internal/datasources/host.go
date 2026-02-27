// Package datasources contains Terraform data source implementations for Lockwave.
package datasources

import (
	"context"
	"fmt"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure HostDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &HostDataSource{}

// HostDataSource reads a single Lockwave host.
type HostDataSource struct {
	client *client.Client
}

// HostDataSourceModel is the Terraform model for the lockwave_host data source.
type HostDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	DisplayName   types.String `tfsdk:"display_name"`
	Hostname      types.String `tfsdk:"hostname"`
	OS            types.String `tfsdk:"os"`
	Arch          types.String `tfsdk:"arch"`
	Status        types.String `tfsdk:"status"`
	DaemonVersion types.String `tfsdk:"daemon_version"`
	LastSeenAt    types.String `tfsdk:"last_seen_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
	HostUsers     types.List   `tfsdk:"host_users"`
}

// hostUserDSAttrTypes defines the object attribute types for the host_users list.
var hostUserDSAttrTypes = map[string]attr.Type{
	"id":                   types.StringType,
	"os_user":              types.StringType,
	"authorized_keys_path": types.StringType,
	"created_at":           types.StringType,
}

// NewHostDataSource is the factory function.
func NewHostDataSource() datasource.DataSource {
	return &HostDataSource{}
}

func (d *HostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (d *HostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Lockwave host by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the host.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable display name.",
			},
			"hostname": schema.StringAttribute{
				Computed:    true,
				Description: "DNS name or IP address.",
			},
			"os": schema.StringAttribute{
				Computed:    true,
				Description: "Operating system.",
			},
			"arch": schema.StringAttribute{
				Computed:    true,
				Description: "CPU architecture.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current sync status.",
			},
			"daemon_version": schema.StringAttribute{
				Computed:    true,
				Description: "Daemon version.",
			},
			"last_seen_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last daemon sync timestamp.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"host_users": schema.ListNestedAttribute{
				Computed:    true,
				Description: "OS users on this host.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                   schema.StringAttribute{Computed: true},
						"os_user":              schema.StringAttribute{Computed: true},
						"authorized_keys_path": schema.StringAttribute{Computed: true},
						"created_at":           schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *HostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state HostDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host, err := d.client.GetHost(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading host", err.Error())
		return
	}

	state.DisplayName = types.StringValue(host.DisplayName)
	state.Hostname = types.StringValue(host.Hostname)
	state.OS = types.StringValue(host.OS)
	state.Arch = types.StringValue(host.Arch)
	state.Status = types.StringValue(host.Status)
	state.DaemonVersion = types.StringValue(host.DaemonVersion)
	state.CreatedAt = types.StringValue(host.CreatedAt)

	if host.LastSeenAt != nil {
		state.LastSeenAt = types.StringValue(*host.LastSeenAt)
	} else {
		state.LastSeenAt = types.StringNull()
	}

	huObjects := make([]attr.Value, 0, len(host.HostUsers))
	for _, hu := range host.HostUsers {
		akp := types.StringNull()
		if hu.AuthorizedKeysPath != nil {
			akp = types.StringValue(*hu.AuthorizedKeysPath)
		}
		obj, d2 := types.ObjectValue(hostUserDSAttrTypes, map[string]attr.Value{
			"id":                   types.StringValue(hu.ID),
			"os_user":              types.StringValue(hu.OsUser),
			"authorized_keys_path": akp,
			"created_at":           types.StringValue(hu.CreatedAt),
		})
		resp.Diagnostics.Append(d2...)
		huObjects = append(huObjects, obj)
	}

	listVal, d2 := types.ListValue(types.ObjectType{AttrTypes: hostUserDSAttrTypes}, huObjects)
	resp.Diagnostics.Append(d2...)
	state.HostUsers = listVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
