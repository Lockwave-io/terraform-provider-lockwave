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

// Ensure HostsDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &HostsDataSource{}

// HostsDataSource reads all Lockwave hosts for the current team.
type HostsDataSource struct {
	client *client.Client
}

// HostsDataSourceModel is the Terraform model for the lockwave_hosts data source.
type HostsDataSourceModel struct {
	Hosts []HostDataSourceModel `tfsdk:"hosts"`
}

// NewHostsDataSource is the factory function.
func NewHostsDataSource() datasource.DataSource {
	return &HostsDataSource{}
}

func (d *HostsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosts"
}

func (d *HostsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	hostAttrs := map[string]schema.Attribute{
		"id":             schema.StringAttribute{Computed: true, Description: "UUID of the host."},
		"display_name":   schema.StringAttribute{Computed: true, Description: "Display name."},
		"hostname":       schema.StringAttribute{Computed: true, Description: "DNS name or IP address."},
		"os":             schema.StringAttribute{Computed: true, Description: "Operating system."},
		"arch":           schema.StringAttribute{Computed: true, Description: "CPU architecture."},
		"status":         schema.StringAttribute{Computed: true, Description: "Sync status."},
		"daemon_version": schema.StringAttribute{Computed: true, Description: "Daemon version."},
		"last_seen_at":   schema.StringAttribute{Computed: true, Description: "Last sync timestamp."},
		"created_at":     schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
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
	}

	resp.Schema = schema.Schema{
		Description: "Fetches all Lockwave hosts for the current team.",
		Attributes: map[string]schema.Attribute{
			"hosts": schema.ListNestedAttribute{
				Computed:             true,
				Description:          "List of hosts.",
				NestedObject:         schema.NestedAttributeObject{Attributes: hostAttrs},
			},
		},
	}
}

func (d *HostsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HostsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	hosts, err := d.client.ListHosts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing hosts", err.Error())
		return
	}

	state := HostsDataSourceModel{
		Hosts: make([]HostDataSourceModel, 0, len(hosts)),
	}

	for _, h := range hosts {
		m := HostDataSourceModel{
			ID:            types.StringValue(h.ID),
			DisplayName:   types.StringValue(h.DisplayName),
			Hostname:      types.StringValue(h.Hostname),
			OS:            types.StringValue(h.OS),
			Arch:          types.StringValue(h.Arch),
			Status:        types.StringValue(h.Status),
			DaemonVersion: types.StringValue(h.DaemonVersion),
			CreatedAt:     types.StringValue(h.CreatedAt),
		}
		if h.LastSeenAt != nil {
			m.LastSeenAt = types.StringValue(*h.LastSeenAt)
		} else {
			m.LastSeenAt = types.StringNull()
		}

		huObjects := make([]attr.Value, 0, len(h.HostUsers))
		for _, hu := range h.HostUsers {
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
		m.HostUsers = listVal

		state.Hosts = append(state.Hosts, m)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
