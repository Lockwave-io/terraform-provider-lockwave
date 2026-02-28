package resources

import (
	"context"
	"fmt"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AssignmentResource satisfies the resource.Resource interface.
var _ resource.Resource = &AssignmentResource{}
var _ resource.ResourceWithImportState = &AssignmentResource{}

// AssignmentResource manages a Lockwave SSH key-to-host-user assignment.
type AssignmentResource struct {
	client *client.Client
}

// AssignmentResourceModel is the Terraform state model for an assignment.
type AssignmentResourceModel struct {
	ID            types.String `tfsdk:"id"`
	SshKeyID      types.String `tfsdk:"ssh_key_id"`
	HostUserID    types.String `tfsdk:"host_user_id"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	Description   types.String `tfsdk:"description"`
	CreatedByType types.String `tfsdk:"created_by_type"`
	CreatedById   types.String `tfsdk:"created_by_id"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

// NewAssignmentResource is the factory function for AssignmentResource.
func NewAssignmentResource() resource.Resource {
	return &AssignmentResource{}
}

func (r *AssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assignment"
}

func (r *AssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Lockwave assignment that grants an SSH key access to an OS user on a host.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the assignment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the SSH key to assign.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host_user_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the host user to assign the key to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional ISO 8601 expiry timestamp for the assignment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional human-readable description for the assignment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by_type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the actor that created this assignment (e.g. user, team).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by_id": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the actor that created this assignment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the assignment was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateAssignmentRequest{
		SshKeyID:   plan.SshKeyID.ValueString(),
		HostUserID: plan.HostUserID.ValueString(),
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		createReq.ExpiresAt = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}

	assignment, err := r.client.CreateAssignment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating assignment", err.Error())
		return
	}

	flattenAssignmentToState(assignment, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assignment, err := r.client.GetAssignment(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading assignment", err.Error())
		return
	}

	flattenAssignmentToState(assignment, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateAssignmentRequest{}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		updateReq.ExpiresAt = &v
	}

	assignment, err := r.client.UpdateAssignment(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating assignment", err.Error())
		return
	}

	flattenAssignmentToState(assignment, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAssignment(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting assignment", err.Error())
		}
	}
}

func (r *AssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	assignment, err := r.client.GetAssignment(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing assignment", err.Error())
		return
	}

	var state AssignmentResourceModel
	flattenAssignmentToState(assignment, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenAssignmentToState maps a client.Assignment onto an AssignmentResourceModel.
func flattenAssignmentToState(a *client.Assignment, m *AssignmentResourceModel) {
	m.ID = types.StringValue(a.ID)
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.CreatedByType = types.StringValue(a.CreatedByType)
	m.CreatedById = types.StringValue(a.CreatedById)

	// Prefer the nested objects when available; fall back to flat ID fields.
	if a.SshKey != nil {
		m.SshKeyID = types.StringValue(a.SshKey.ID)
	} else if a.SshKeyID != "" {
		m.SshKeyID = types.StringValue(a.SshKeyID)
	}

	if a.HostUser != nil {
		m.HostUserID = types.StringValue(a.HostUser.ID)
	} else if a.HostUserID != "" {
		m.HostUserID = types.StringValue(a.HostUserID)
	}

	if a.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(*a.ExpiresAt)
	} else {
		m.ExpiresAt = types.StringNull()
	}

	if a.Description != nil {
		m.Description = types.StringValue(*a.Description)
	} else {
		m.Description = types.StringNull()
	}
}
