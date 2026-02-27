package resources

import (
	"context"
	"fmt"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure WebhookEndpointResource satisfies the resource.Resource interface.
var _ resource.Resource = &WebhookEndpointResource{}
var _ resource.ResourceWithImportState = &WebhookEndpointResource{}

// WebhookEndpointResource manages a Lockwave webhook endpoint.
type WebhookEndpointResource struct {
	client *client.Client
}

// WebhookEndpointResourceModel is the Terraform state model for a webhook endpoint.
type WebhookEndpointResourceModel struct {
	ID           types.String `tfsdk:"id"`
	URL          types.String `tfsdk:"url"`
	Description  types.String `tfsdk:"description"`
	Events       types.List   `tfsdk:"events"`
	IsActive     types.Bool   `tfsdk:"is_active"`
	FailureCount types.Int64  `tfsdk:"failure_count"`
	DisabledAt   types.String `tfsdk:"disabled_at"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// NewWebhookEndpointResource is the factory function for WebhookEndpointResource.
func NewWebhookEndpointResource() resource.Resource {
	return &WebhookEndpointResource{}
}

func (r *WebhookEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_endpoint"
}

func (r *WebhookEndpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Lockwave webhook endpoint that receives event notifications.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the webhook endpoint.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "HTTPS URL to deliver webhook payloads to.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Human-readable description for the webhook endpoint.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"events": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of event types to subscribe to (e.g. host.synced, ssh_key.created).",
			},
			"is_active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the webhook endpoint is currently active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"failure_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Cumulative delivery failure count.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"disabled_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp when the endpoint was automatically disabled, or null.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the endpoint was created.",
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

func (r *WebhookEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	events, d := stringListFromTF(ctx, plan.Events)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateWebhookEndpointRequest{
		URL:    plan.URL.ValueString(),
		Events: events,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}

	endpoint, err := r.client.CreateWebhookEndpoint(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook endpoint", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenWebhookEndpointToState(ctx, endpoint, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint, err := r.client.GetWebhookEndpoint(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook endpoint", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenWebhookEndpointToState(ctx, endpoint, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	events, d := stringListFromTF(ctx, plan.Events)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateWebhookEndpointRequest{
		URL:    plan.URL.ValueString(),
		Events: events,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}

	endpoint, err := r.client.UpdateWebhookEndpoint(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook endpoint", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenWebhookEndpointToState(ctx, endpoint, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteWebhookEndpoint(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting webhook endpoint", err.Error())
		}
	}
}

func (r *WebhookEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	endpoint, err := r.client.GetWebhookEndpoint(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing webhook endpoint", err.Error())
		return
	}

	var state WebhookEndpointResourceModel
	resp.Diagnostics.Append(flattenWebhookEndpointToState(ctx, endpoint, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenWebhookEndpointToState maps a client.WebhookEndpoint onto a WebhookEndpointResourceModel.
func flattenWebhookEndpointToState(ctx context.Context, e *client.WebhookEndpoint, m *WebhookEndpointResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(e.ID)
	m.URL = types.StringValue(e.URL)
	m.IsActive = types.BoolValue(e.IsActive)
	m.FailureCount = types.Int64Value(int64(e.FailureCount))
	m.CreatedAt = types.StringValue(e.CreatedAt)
	m.UpdatedAt = types.StringValue(e.UpdatedAt)

	if e.Description != nil {
		m.Description = types.StringValue(*e.Description)
	} else {
		m.Description = types.StringNull()
	}

	if e.DisabledAt != nil {
		m.DisabledAt = types.StringValue(*e.DisabledAt)
	} else {
		m.DisabledAt = types.StringNull()
	}

	eventVals, d := types.ListValueFrom(ctx, types.StringType, e.Events)
	diags.Append(d...)
	m.Events = eventVals

	return diags
}

// stringListFromTF converts a types.List of strings to a []string.
func stringListFromTF(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}
