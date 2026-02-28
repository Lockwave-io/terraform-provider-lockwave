package datasources

import (
	"context"
	"fmt"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SshKeyDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &SshKeyDataSource{}

// SshKeyDataSource reads a single Lockwave SSH key.
type SshKeyDataSource struct {
	client *client.Client
}

// SshKeyDataSourceModel is the Terraform model for the lockwave_ssh_key data source.
type SshKeyDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	FingerprintSHA256 types.String `tfsdk:"fingerprint_sha256"`
	KeyType           types.String `tfsdk:"key_type"`
	KeyBits           types.Int64  `tfsdk:"key_bits"`
	Comment           types.String `tfsdk:"comment"`
	BlockedUntil      types.String `tfsdk:"blocked_until"`
	BlockedIndefinite types.Bool   `tfsdk:"blocked_indefinite"`
	CreatedAt         types.String `tfsdk:"created_at"`
}

// NewSshKeyDataSource is the factory function.
func NewSshKeyDataSource() datasource.DataSource {
	return &SshKeyDataSource{}
}

func (d *SshKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (d *SshKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Lockwave SSH key by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the SSH key.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the key.",
			},
			"fingerprint_sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 fingerprint.",
			},
			"key_type": schema.StringAttribute{
				Computed:    true,
				Description: "Key algorithm (ed25519 or rsa).",
			},
			"key_bits": schema.Int64Attribute{
				Computed:    true,
				Description: "RSA key size (null for ed25519).",
			},
			"comment": schema.StringAttribute{
				Computed:    true,
				Description: "Key comment.",
			},
			"blocked_until": schema.StringAttribute{
				Computed:    true,
				Description: "Blocked-until timestamp, or null.",
			},
			"blocked_indefinite": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the key is blocked indefinitely.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp.",
			},
		},
	}
}

func (d *SshKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SshKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state SshKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := d.client.GetSshKey(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SSH key", err.Error())
		return
	}

	flattenSshKeyDSToState(key, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenSshKeyDSToState maps a client.SshKey onto a SshKeyDataSourceModel.
func flattenSshKeyDSToState(k *client.SshKey, m *SshKeyDataSourceModel) {
	m.ID = types.StringValue(k.ID)
	m.Name = types.StringValue(k.Name)
	m.FingerprintSHA256 = types.StringValue(k.FingerprintSHA256)
	m.KeyType = types.StringValue(k.KeyType)
	m.BlockedIndefinite = types.BoolValue(k.BlockedIndefinite)
	m.CreatedAt = types.StringValue(k.CreatedAt)

	if k.KeyBits != nil {
		m.KeyBits = types.Int64Value(int64(*k.KeyBits))
	} else {
		m.KeyBits = types.Int64Null()
	}

	if k.Comment != nil {
		m.Comment = types.StringValue(*k.Comment)
	} else {
		m.Comment = types.StringNull()
	}

	if k.BlockedUntil != nil {
		m.BlockedUntil = types.StringValue(*k.BlockedUntil)
	} else {
		m.BlockedUntil = types.StringNull()
	}
}
