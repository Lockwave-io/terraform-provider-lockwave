package resources

import (
	"context"
	"fmt"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AuditLogStreamResource satisfies the resource.Resource interface.
var _ resource.Resource = &AuditLogStreamResource{}
var _ resource.ResourceWithImportState = &AuditLogStreamResource{}

// AuditLogStreamResource manages a Lockwave audit log stream.
type AuditLogStreamResource struct {
	client *client.Client
}

// AuditLogStreamConfigModel is the Terraform state model for the nested config block.
// Fields are split by stream type; unused fields remain null in state.
type AuditLogStreamConfigModel struct {
	// webhook fields
	URL    types.String `tfsdk:"url"`
	Secret types.String `tfsdk:"secret"`

	// s3 fields
	Bucket          types.String `tfsdk:"bucket"`
	Region          types.String `tfsdk:"region"`
	Prefix          types.String `tfsdk:"prefix"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

// AuditLogStreamResourceModel is the Terraform state model for an audit log stream.
type AuditLogStreamResourceModel struct {
	ID        types.String              `tfsdk:"id"`
	Type      types.String              `tfsdk:"type"`
	Config    AuditLogStreamConfigModel `tfsdk:"config"`
	IsActive  types.Bool                `tfsdk:"is_active"`
	CreatedAt types.String              `tfsdk:"created_at"`
	UpdatedAt types.String              `tfsdk:"updated_at"`
}

// NewAuditLogStreamResource is the factory function for AuditLogStreamResource.
func NewAuditLogStreamResource() resource.Resource {
	return &AuditLogStreamResource{}
}

func (r *AuditLogStreamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audit_log_stream"
}

func (r *AuditLogStreamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Lockwave audit log stream that forwards audit events to an external destination (webhook or S3).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the audit log stream.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Stream destination type. Must be \"webhook\" or \"s3\". Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the audit log stream is currently active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the stream was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of the last update.",
			},
			"config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Type-specific configuration for the audit log stream. Provide only the fields relevant to the chosen type.",
				Attributes: map[string]schema.Attribute{
					// ------ webhook fields ------
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "(webhook) HTTPS URL to deliver audit log payloads to.",
					},
					"secret": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "(webhook) Optional HMAC secret used to sign payloads.",
					},
					// ------ s3 fields ------
					"bucket": schema.StringAttribute{
						Optional:    true,
						Description: "(s3) Name of the S3 bucket to write audit logs to.",
					},
					"region": schema.StringAttribute{
						Optional:    true,
						Description: "(s3) AWS region where the bucket resides (e.g. us-east-1).",
					},
					"prefix": schema.StringAttribute{
						Optional:    true,
						Description: "(s3) Optional key prefix for objects written to the bucket.",
					},
					"access_key_id": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "(s3) AWS access key ID with write access to the bucket.",
					},
					"secret_access_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "(s3) AWS secret access key corresponding to access_key_id.",
					},
				},
			},
		},
	}
}

func (r *AuditLogStreamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AuditLogStreamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateAuditLogStreamRequest{
		Type:   plan.Type.ValueString(),
		Config: configFromModel(plan.Config),
	}

	stream, err := r.client.CreateAuditLogStream(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating audit log stream", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenAuditLogStreamToState(stream, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuditLogStreamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.GetAuditLogStream(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading audit log stream", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenAuditLogStreamToState(stream, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuditLogStreamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateAuditLogStreamRequest{
		Config: configFromModel(plan.Config),
	}

	stream, err := r.client.UpdateAuditLogStream(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating audit log stream", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenAuditLogStreamToState(stream, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuditLogStreamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAuditLogStream(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting audit log stream", err.Error())
		}
	}
}

func (r *AuditLogStreamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	stream, err := r.client.GetAuditLogStream(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing audit log stream", err.Error())
		return
	}

	var state AuditLogStreamResourceModel
	resp.Diagnostics.Append(flattenAuditLogStreamToState(stream, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// configFromModel converts an AuditLogStreamConfigModel into the client struct.
// Fields that are null or unknown are left as zero values (omitted via omitempty in JSON).
func configFromModel(m AuditLogStreamConfigModel) client.AuditLogStreamConfig {
	cfg := client.AuditLogStreamConfig{}

	if !m.URL.IsNull() && !m.URL.IsUnknown() {
		cfg.URL = m.URL.ValueString()
	}
	if !m.Secret.IsNull() && !m.Secret.IsUnknown() {
		cfg.Secret = m.Secret.ValueString()
	}
	if !m.Bucket.IsNull() && !m.Bucket.IsUnknown() {
		cfg.Bucket = m.Bucket.ValueString()
	}
	if !m.Region.IsNull() && !m.Region.IsUnknown() {
		cfg.Region = m.Region.ValueString()
	}
	if !m.Prefix.IsNull() && !m.Prefix.IsUnknown() {
		cfg.Prefix = m.Prefix.ValueString()
	}
	if !m.AccessKeyID.IsNull() && !m.AccessKeyID.IsUnknown() {
		cfg.AccessKeyID = m.AccessKeyID.ValueString()
	}
	if !m.SecretAccessKey.IsNull() && !m.SecretAccessKey.IsUnknown() {
		cfg.SecretAccessKey = m.SecretAccessKey.ValueString()
	}

	return cfg
}

// flattenAuditLogStreamToState maps a client.AuditLogStream onto an AuditLogStreamResourceModel.
//
// Sensitive config fields (secret, access_key_id, secret_access_key) are returned by the
// API only on create. On subsequent reads the API omits them, so we preserve whatever
// value is already in state rather than overwriting with an empty string. Callers must
// pass the current model so that we can copy state-preserved values for those fields.
func flattenAuditLogStreamToState(s *client.AuditLogStream, m *AuditLogStreamResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(s.ID)
	m.Type = types.StringValue(s.Type)
	m.IsActive = types.BoolValue(s.IsActive)
	m.CreatedAt = types.StringValue(s.CreatedAt)
	m.UpdatedAt = types.StringValue(s.UpdatedAt)

	// Non-sensitive config fields are always refreshed from the API response.
	if s.Config.URL != "" {
		m.Config.URL = types.StringValue(s.Config.URL)
	} else if m.Config.URL.IsUnknown() {
		m.Config.URL = types.StringNull()
	}

	if s.Config.Bucket != "" {
		m.Config.Bucket = types.StringValue(s.Config.Bucket)
	} else if m.Config.Bucket.IsUnknown() {
		m.Config.Bucket = types.StringNull()
	}

	if s.Config.Region != "" {
		m.Config.Region = types.StringValue(s.Config.Region)
	} else if m.Config.Region.IsUnknown() {
		m.Config.Region = types.StringNull()
	}

	if s.Config.Prefix != "" {
		m.Config.Prefix = types.StringValue(s.Config.Prefix)
	} else if m.Config.Prefix.IsUnknown() {
		m.Config.Prefix = types.StringNull()
	}

	// Sensitive fields: only overwrite from API when the API actually returned a value.
	// If the API omits the field (empty string after JSON decode), we leave the existing
	// state value in place so Terraform does not produce a spurious diff.
	if s.Config.Secret != "" {
		m.Config.Secret = types.StringValue(s.Config.Secret)
	} else if m.Config.Secret.IsUnknown() {
		m.Config.Secret = types.StringNull()
	}
	// else: preserve whatever is already in m.Config.Secret (state carry-forward)

	if s.Config.AccessKeyID != "" {
		m.Config.AccessKeyID = types.StringValue(s.Config.AccessKeyID)
	} else if m.Config.AccessKeyID.IsUnknown() {
		m.Config.AccessKeyID = types.StringNull()
	}

	if s.Config.SecretAccessKey != "" {
		m.Config.SecretAccessKey = types.StringValue(s.Config.SecretAccessKey)
	} else if m.Config.SecretAccessKey.IsUnknown() {
		m.Config.SecretAccessKey = types.StringNull()
	}

	return diags
}
