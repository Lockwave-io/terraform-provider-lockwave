package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure HostUserResource satisfies the resource.Resource interface.
var _ resource.Resource = &HostUserResource{}
var _ resource.ResourceWithImportState = &HostUserResource{}

// HostUserResource manages a Lockwave host OS user.
type HostUserResource struct {
	client *client.Client
}

// HostUserResourceModel is the Terraform state model for a host user.
type HostUserResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	HostID             types.String `tfsdk:"host_id"`
	OsUser             types.String `tfsdk:"os_user"`
	AuthorizedKeysPath types.String `tfsdk:"authorized_keys_path"`
	CreatedAt          types.String `tfsdk:"created_at"`
}

// NewHostUserResource is the factory function for HostUserResource.
func NewHostUserResource() resource.Resource {
	return &HostUserResource{}
}

func (r *HostUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_user"
}

func (r *HostUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OS user record on a Lockwave host. These records define which authorized_keys files the daemon manages.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the host user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the parent host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_user": schema.StringAttribute{
				Required:    true,
				Description: "OS username (e.g. ubuntu, ec2-user, deploy).",
			},
			"authorized_keys_path": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Absolute path to the authorized_keys file. Defaults to the standard location for the OS user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the host user was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *HostUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HostUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HostUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateHostUserRequest{
		OsUser: plan.OsUser.ValueString(),
	}
	if !plan.AuthorizedKeysPath.IsNull() && !plan.AuthorizedKeysPath.IsUnknown() {
		v := plan.AuthorizedKeysPath.ValueString()
		createReq.AuthorizedKeysPath = &v
	}

	hu, err := r.client.CreateHostUser(ctx, plan.HostID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating host user", err.Error())
		return
	}

	flattenHostUserToState(hu, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HostUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hu, err := r.client.GetHostUser(ctx, state.HostID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading host user", err.Error())
		return
	}

	flattenHostUserToState(hu, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HostUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HostUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HostUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateHostUserRequest{
		OsUser: plan.OsUser.ValueString(),
	}
	if !plan.AuthorizedKeysPath.IsNull() && !plan.AuthorizedKeysPath.IsUnknown() {
		v := plan.AuthorizedKeysPath.ValueString()
		updateReq.AuthorizedKeysPath = &v
	}

	hu, err := r.client.UpdateHostUser(ctx, state.HostID.ValueString(), state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating host user", err.Error())
		return
	}

	flattenHostUserToState(hu, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HostUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteHostUser(ctx, state.HostID.ValueString(), state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting host user", err.Error())
		}
	}
}

// ImportState supports `terraform import lockwave_host_user.<name> <host_id>/<user_id>`.
func (r *HostUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format <host_id>/<user_id>",
		)
		return
	}

	hostID, userID := parts[0], parts[1]

	hu, err := r.client.GetHostUser(ctx, hostID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing host user", err.Error())
		return
	}

	var state HostUserResourceModel
	state.HostID = types.StringValue(hostID)
	flattenHostUserToState(hu, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenHostUserToState maps a client.HostUser onto a HostUserResourceModel.
func flattenHostUserToState(hu *client.HostUser, m *HostUserResourceModel) {
	m.ID = types.StringValue(hu.ID)
	m.OsUser = types.StringValue(hu.OsUser)
	m.CreatedAt = types.StringValue(hu.CreatedAt)

	if hu.AuthorizedKeysPath != nil {
		m.AuthorizedKeysPath = types.StringValue(*hu.AuthorizedKeysPath)
	} else {
		m.AuthorizedKeysPath = types.StringNull()
	}
}
