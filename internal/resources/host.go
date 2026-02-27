// Package resources contains Terraform resource implementations for Lockwave.
package resources

import (
	"context"
	"fmt"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure HostResource satisfies the resource.Resource interface.
var _ resource.Resource = &HostResource{}
var _ resource.ResourceWithImportState = &HostResource{}

// HostResource manages a Lockwave host.
type HostResource struct {
	client *client.Client
}

// HostResourceModel is the Terraform state model for a host.
type HostResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DisplayName   types.String `tfsdk:"display_name"`
	Hostname      types.String `tfsdk:"hostname"`
	OS            types.String `tfsdk:"os"`
	Arch          types.String `tfsdk:"arch"`
	Status        types.String `tfsdk:"status"`
	DaemonVersion types.String `tfsdk:"daemon_version"`
	LastSeenAt    types.String `tfsdk:"last_seen_at"`
	Credential    types.String `tfsdk:"credential"`
	CreatedAt     types.String `tfsdk:"created_at"`
	HostUsers     types.List   `tfsdk:"host_users"`
}

// hostUserAttrTypes defines the object attribute types for host_users list elements.
var hostUserAttrTypes = map[string]attr.Type{
	"id":                   types.StringType,
	"os_user":              types.StringType,
	"authorized_keys_path": types.StringType,
	"created_at":           types.StringType,
}

// NewHostResource is the factory function for HostResource.
func NewHostResource() resource.Resource {
	return &HostResource{}
}

func (r *HostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *HostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Lockwave host. Creating a host also provisions a one-time daemon credential stored in the `credential` attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable display name for the host.",
			},
			"hostname": schema.StringAttribute{
				Required:    true,
				Description: "DNS name or IP address of the host.",
			},
			"os": schema.StringAttribute{
				Required:    true,
				Description: "Operating system of the host. One of: linux, darwin, freebsd.",
			},
			"arch": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "CPU architecture. One of: x86_64, aarch64, amd64, arm64.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current sync status of the host.",
			},
			"daemon_version": schema.StringAttribute{
				Computed:    true,
				Description: "Version of the Lockwave daemon running on this host.",
			},
			"last_seen_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last daemon sync.",
			},
			"credential": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "One-time daemon credential returned only on host creation. Store this securely.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the host was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_users": schema.ListNestedAttribute{
				Computed:    true,
				Description: "OS users registered on this host.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "UUID of the host user.",
						},
						"os_user": schema.StringAttribute{
							Computed:    true,
							Description: "OS username.",
						},
						"authorized_keys_path": schema.StringAttribute{
							Computed:    true,
							Description: "Path to the authorized_keys file.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "ISO 8601 creation timestamp.",
						},
					},
				},
			},
		},
	}
}

func (r *HostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateHostRequest{
		DisplayName: plan.DisplayName.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		OS:          plan.OS.ValueString(),
	}
	if !plan.Arch.IsNull() && !plan.Arch.IsUnknown() {
		createReq.Arch = plan.Arch.ValueString()
	}

	host, err := r.client.CreateHost(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating host", err.Error())
		return
	}

	resp.Diagnostics.Append(flattenHostToState(ctx, host, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host, err := r.client.GetHost(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading host", err.Error())
		return
	}

	// Preserve the one-time credential from prior state — it is not returned on GET.
	resp.Diagnostics.Append(flattenHostToState(ctx, host, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateHostRequest{
		DisplayName: plan.DisplayName.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		OS:          plan.OS.ValueString(),
	}
	if !plan.Arch.IsNull() && !plan.Arch.IsUnknown() {
		updateReq.Arch = plan.Arch.ValueString()
	}

	host, err := r.client.UpdateHost(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating host", err.Error())
		return
	}

	// Carry forward the one-time credential from state; PATCH does not return it.
	// Set it before flatten so flattenHostToState does not overwrite it with an
	// empty value (the API returns "" for credential on updates).
	plan.Credential = state.Credential
	resp.Diagnostics.Append(flattenHostToState(ctx, host, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Re-assert after flatten in case the API ever starts returning a credential
	// on PATCH (which it currently does not).
	plan.Credential = state.Credential

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteHost(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error deleting host", err.Error())
		}
	}
}

func (r *HostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	host, err := r.client.GetHost(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing host", err.Error())
		return
	}

	var state HostResourceModel
	resp.Diagnostics.Append(flattenHostToState(ctx, host, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenHostToState maps a client.Host onto a HostResourceModel.
func flattenHostToState(ctx context.Context, h *client.Host, m *HostResourceModel) fwdiag.Diagnostics {
	var diags fwdiag.Diagnostics

	m.ID = types.StringValue(h.ID)
	m.DisplayName = types.StringValue(h.DisplayName)
	m.Hostname = types.StringValue(h.Hostname)
	m.OS = types.StringValue(h.OS)
	m.Arch = types.StringValue(h.Arch)
	m.Status = types.StringValue(h.Status)
	m.DaemonVersion = types.StringValue(h.DaemonVersion)
	m.CreatedAt = types.StringValue(h.CreatedAt)

	if h.LastSeenAt != nil {
		m.LastSeenAt = types.StringValue(*h.LastSeenAt)
	} else {
		m.LastSeenAt = types.StringNull()
	}

	// Credential is only returned by CreateHost; subsequent GETs/PATCHes return "".
	// When the API provides one we store it; otherwise we leave the model field
	// untouched so callers can preserve the value they already have in state.
	if h.Credential != "" {
		m.Credential = types.StringValue(h.Credential)
	}

	huObjects := make([]attr.Value, 0, len(h.HostUsers))
	for _, hu := range h.HostUsers {
		akp := types.StringNull()
		if hu.AuthorizedKeysPath != nil {
			akp = types.StringValue(*hu.AuthorizedKeysPath)
		}
		obj, d := types.ObjectValue(hostUserAttrTypes, map[string]attr.Value{
			"id":                   types.StringValue(hu.ID),
			"os_user":              types.StringValue(hu.OsUser),
			"authorized_keys_path": akp,
			"created_at":           types.StringValue(hu.CreatedAt),
		})
		diags.Append(d...)
		huObjects = append(huObjects, obj)
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: hostUserAttrTypes}, huObjects)
	diags.Append(d...)
	m.HostUsers = listVal

	return diags
}
