package resources

import (
	"context"
	"fmt"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NotificationChannelResource satisfies the resource.Resource interface.
var _ resource.Resource = &NotificationChannelResource{}
var _ resource.ResourceWithImportState = &NotificationChannelResource{}

// NotificationChannelResource manages a Lockwave notification channel.
type NotificationChannelResource struct {
	client *client.Client
}

// NotificationChannelConfigModel holds the flattened config block.
// Only the fields relevant to the chosen channel type need to be populated;
// the other fields remain null in state.
type NotificationChannelConfigModel struct {
	// Slack fields.
	WebhookURL types.String `tfsdk:"webhook_url"`
	// Email fields.
	Recipients types.List `tfsdk:"recipients"`
}

// NotificationChannelResourceModel is the Terraform state model for a notification channel.
type NotificationChannelResourceModel struct {
	ID        types.String                    `tfsdk:"id"`
	Type      types.String                    `tfsdk:"type"`
	Name      types.String                    `tfsdk:"name"`
	Config    *NotificationChannelConfigModel `tfsdk:"config"`
	IsActive  types.Bool                      `tfsdk:"is_active"`
	CreatedAt types.String                    `tfsdk:"created_at"`
	UpdatedAt types.String                    `tfsdk:"updated_at"`
}

// NewNotificationChannelResource is the factory function for NotificationChannelResource.
func NewNotificationChannelResource() resource.Resource {
	return &NotificationChannelResource{}
}

func (r *NotificationChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *NotificationChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Lockwave notification channel. Supported types are \"slack\" and \"email\".",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the notification channel.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Channel type. One of: slack, email. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name for the notification channel.",
			},
			"config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Type-specific configuration block. Populate only the fields relevant to the chosen type.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Incoming webhook URL (slack channels only).",
					},
					"recipients": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "List of recipient email addresses (email channels only).",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"is_active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the notification channel is currently active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the notification channel was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of the last update.",
			},
		},
	}
}

func (r *NotificationChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NotificationChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, d := buildChannelConfig(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateNotificationChannelRequest{
		Type:   plan.Type.ValueString(),
		Name:   plan.Name.ValueString(),
		Config: cfg,
	}

	ch, err := r.client.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating notification channel", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenNotificationChannelToState(ctx, ch, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotificationChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := r.client.GetNotificationChannel(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading notification channel", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenNotificationChannelToState(ctx, ch, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NotificationChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NotificationChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, d := buildChannelConfig(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateNotificationChannelRequest{
		Name:   plan.Name.ValueString(),
		Config: cfg,
	}

	ch, err := r.client.UpdateNotificationChannel(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating notification channel", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenNotificationChannelToState(ctx, ch, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotificationChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNotificationChannel(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting notification channel", err.Error())
		}
	}
}

func (r *NotificationChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ch, err := r.client.GetNotificationChannel(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing notification channel", err.Error())
		return
	}

	var state NotificationChannelResourceModel
	resp.Diagnostics.Append(flattenNotificationChannelToState(ctx, ch, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildChannelConfig converts the Terraform config model into the API map payload.
// It only includes fields that are set, keeping the payload minimal and correct
// for each channel type.
func buildChannelConfig(ctx context.Context, m *NotificationChannelConfigModel) (client.NotificationChannelConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m == nil {
		diags.AddError("Missing config block", "A config block is required for notification channels.")
		return nil, diags
	}

	cfg := make(client.NotificationChannelConfig)

	if !m.WebhookURL.IsNull() && !m.WebhookURL.IsUnknown() {
		cfg["webhook_url"] = m.WebhookURL.ValueString()
	}

	if !m.Recipients.IsNull() && !m.Recipients.IsUnknown() {
		var recipients []string
		diags.Append(m.Recipients.ElementsAs(ctx, &recipients, false)...)
		if !diags.HasError() {
			cfg["recipients"] = recipients
		}
	}

	return cfg, diags
}

// flattenNotificationChannelToState maps a client.NotificationChannel onto a
// NotificationChannelResourceModel.
func flattenNotificationChannelToState(ctx context.Context, ch *client.NotificationChannel, m *NotificationChannelResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(ch.ID)
	m.Type = types.StringValue(ch.Type)
	m.Name = types.StringValue(ch.Name)
	m.IsActive = types.BoolValue(ch.IsActive)
	m.CreatedAt = types.StringValue(ch.CreatedAt)
	m.UpdatedAt = types.StringValue(ch.UpdatedAt)

	// Preserve the existing config model pointer so we don't lose fields that
	// are not returned by the API (e.g. webhook_url is sensitive and may be
	// omitted on reads). We only overwrite what the API actually returns.
	if m.Config == nil {
		m.Config = &NotificationChannelConfigModel{}
	}

	// Flatten webhook_url (slack).
	if v, ok := ch.Config["webhook_url"]; ok {
		if s, ok := v.(string); ok {
			m.Config.WebhookURL = types.StringValue(s)
		}
	}

	// Flatten recipients (email).
	if v, ok := ch.Config["recipients"]; ok {
		// The JSON decoder produces []interface{} for arrays; convert to []string.
		switch rv := v.(type) {
		case []interface{}:
			strs := make([]string, 0, len(rv))
			for _, item := range rv {
				if s, ok := item.(string); ok {
					strs = append(strs, s)
				}
			}
			listVal, d := types.ListValueFrom(ctx, types.StringType, strs)
			diags.Append(d...)
			m.Config.Recipients = listVal
		case []string:
			listVal, d := types.ListValueFrom(ctx, types.StringType, rv)
			diags.Append(d...)
			m.Config.Recipients = listVal
		}
	}

	return diags
}
