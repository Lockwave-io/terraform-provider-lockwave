package datasources

import (
	"context"
	"fmt"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure SshKeysDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &SshKeysDataSource{}

// SshKeysDataSource reads all SSH keys for the current team.
type SshKeysDataSource struct {
	client *client.Client
}

// SshKeysDataSourceModel is the Terraform model for the lockwave_ssh_keys data source.
type SshKeysDataSourceModel struct {
	Keys []SshKeyDataSourceModel `tfsdk:"keys"`
}

// NewSshKeysDataSource is the factory function.
func NewSshKeysDataSource() datasource.DataSource {
	return &SshKeysDataSource{}
}

func (d *SshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (d *SshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	keyAttrs := map[string]schema.Attribute{
		"id":                  schema.StringAttribute{Computed: true, Description: "UUID of the key."},
		"name":                schema.StringAttribute{Computed: true, Description: "Name."},
		"fingerprint_sha256":  schema.StringAttribute{Computed: true, Description: "SHA-256 fingerprint."},
		"key_type":            schema.StringAttribute{Computed: true, Description: "Algorithm."},
		"key_bits":            schema.Int64Attribute{Computed: true, Description: "RSA key bits."},
		"comment":             schema.StringAttribute{Computed: true, Description: "Comment."},
		"blocked_until":       schema.StringAttribute{Computed: true, Description: "Blocked until."},
		"blocked_indefinite":  schema.BoolAttribute{Computed: true, Description: "Blocked indefinitely."},
		"created_at":          schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
	}

	resp.Schema = schema.Schema{
		Description: "Fetches all Lockwave SSH keys for the current team.",
		Attributes: map[string]schema.Attribute{
			"keys": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of SSH keys.",
				NestedObject: schema.NestedAttributeObject{Attributes: keyAttrs},
			},
		},
	}
}

func (d *SshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SshKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListSshKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing SSH keys", err.Error())
		return
	}

	state := SshKeysDataSourceModel{
		Keys: make([]SshKeyDataSourceModel, 0, len(keys)),
	}

	for _, k := range keys {
		var m SshKeyDataSourceModel
		flattenSshKeyDSToState(&k, &m)
		state.Keys = append(state.Keys, m)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
