package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"golang.org/x/crypto/ssh"
)

// Ensure SshKeyResource satisfies the resource.Resource interface.
var _ resource.Resource = &SshKeyResource{}
var _ resource.ResourceWithImportState = &SshKeyResource{}

// SshKeyResource manages a Lockwave SSH key.
type SshKeyResource struct {
	client *client.Client
}

// SshKeyResourceModel is the Terraform state model for an SSH key.
type SshKeyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Mode              types.String `tfsdk:"mode"`
	PublicKey         types.String `tfsdk:"public_key"`
	KeyType           types.String `tfsdk:"key_type"`
	KeyBits           types.Int64  `tfsdk:"key_bits"`
	Comment           types.String `tfsdk:"comment"`
	FingerprintSHA256 types.String `tfsdk:"fingerprint_sha256"`
	BlockedUntil      types.String `tfsdk:"blocked_until"`
	BlockedIndefinite types.Bool   `tfsdk:"blocked_indefinite"`
	PrivateKey        types.String `tfsdk:"private_key"`
	CreatedAt         types.String `tfsdk:"created_at"`
}

// NewSshKeyResource is the factory function for SshKeyResource.
func NewSshKeyResource() resource.Resource {
	return &SshKeyResource{}
}

func (r *SshKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *SshKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Lockwave SSH key. Keys can be server-generated (mode=generate) or imported (mode=import).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the SSH key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name for the SSH key.",
			},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: "Creation mode. One of: generate, import. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OpenSSH public key. Required when mode=import. Computed when mode=generate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"key_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Key algorithm. One of: ed25519, rsa. Required when mode=generate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"key_bits": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "RSA key size. One of: 3072, 4096. Only relevant when key_type=rsa.",
			},
			"comment": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional comment embedded in the public key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint_sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 fingerprint of the public key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"blocked_until": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp until which the key is blocked, or null.",
			},
			"blocked_indefinite": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the key is blocked indefinitely.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Private key returned only on creation when mode=generate. Store this securely; it is never returned again.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the key was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SshKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *SshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SshKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateSshKeyRequest{
		Name: plan.Name.ValueString(),
		Mode: plan.Mode.ValueString(),
	}

	if !plan.PublicKey.IsNull() && !plan.PublicKey.IsUnknown() {
		v := plan.PublicKey.ValueString()
		createReq.PublicKey = &v
	}
	if !plan.KeyType.IsNull() && !plan.KeyType.IsUnknown() {
		v := plan.KeyType.ValueString()
		createReq.KeyType = &v
	}
	if !plan.KeyBits.IsNull() && !plan.KeyBits.IsUnknown() {
		v := int(plan.KeyBits.ValueInt64())
		createReq.KeyBits = &v
	}

	key, err := r.client.CreateSshKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSH key", err.Error())
		return
	}

	flattenSshKeyToState(key, &plan)

	// If the API returned a private key but no public key, derive the public key.
	if key.PrivateKey != "" && (plan.PublicKey.IsNull() || plan.PublicKey.ValueString() == "") {
		signer, err := ssh.ParsePrivateKey([]byte(key.PrivateKey))
		if err == nil {
			plan.PublicKey = types.StringValue(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetSshKey(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SSH key", err.Error())
		return
	}

	// Preserve computed-once fields that the API does not return on GET.
	savedPrivateKey := state.PrivateKey
	savedPublicKey := state.PublicKey
	savedMode := state.Mode
	flattenSshKeyToState(key, &state)
	state.PrivateKey = savedPrivateKey
	if state.PublicKey.IsNull() || state.PublicKey.ValueString() == "" {
		state.PublicKey = savedPublicKey
	}
	state.Mode = savedMode

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SshKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateSshKeyRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		updateReq.Comment = plan.Comment.ValueString()
	}

	key, err := r.client.UpdateSshKey(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SSH key", err.Error())
		return
	}

	savedPrivateKey := state.PrivateKey
	savedPublicKey := state.PublicKey
	flattenSshKeyToState(key, &plan)
	plan.PrivateKey = savedPrivateKey
	if plan.PublicKey.IsNull() || plan.PublicKey.ValueString() == "" {
		plan.PublicKey = savedPublicKey
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SshKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSshKey(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting SSH key", err.Error())
		}
	}
}

func (r *SshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	key, err := r.client.GetSshKey(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing SSH key", err.Error())
		return
	}

	var state SshKeyResourceModel
	state.Mode = types.StringValue("import") // Imported keys are treated as imported.
	flattenSshKeyToState(key, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenSshKeyToState maps a client.SshKey onto a SshKeyResourceModel.
func flattenSshKeyToState(k *client.SshKey, m *SshKeyResourceModel) {
	m.ID = types.StringValue(k.ID)
	m.Name = types.StringValue(k.Name)
	m.FingerprintSHA256 = types.StringValue(k.FingerprintSHA256)
	m.KeyType = types.StringValue(k.KeyType)
	m.BlockedIndefinite = types.BoolValue(k.BlockedIndefinite)
	m.CreatedAt = types.StringValue(k.CreatedAt)

	if k.PublicKey != "" {
		m.PublicKey = types.StringValue(k.PublicKey)
	} else {
		m.PublicKey = types.StringNull()
	}

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

	if k.PrivateKey != "" {
		m.PrivateKey = types.StringValue(k.PrivateKey)
	}
}
