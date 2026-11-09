// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	stringvalidator "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &vaultResource{}
	_ resource.ResourceWithConfigure   = &vaultResource{}
	_ resource.ResourceWithImportState = &vaultResource{}
)

func NewVaultResource() resource.Resource {
	return &vaultResource{}
}

type vaultResource struct {
	client *onepassword.Client
}

type vaultResourceModel struct {
	ID               tfsdk.String `tfsdk:"id"`
	Title            tfsdk.String `tfsdk:"title"`
	Description      tfsdk.String `tfsdk:"description"`
	AllowAdminsSetup tfsdk.Bool   `tfsdk:"allow_admins_access"`
	VaultType        tfsdk.String `tfsdk:"vault_type"`
	ActiveItemCount  tfsdk.Int64  `tfsdk:"active_item_count"`
	ContentVersion   tfsdk.Int64  `tfsdk:"content_version"`
	AttributeVersion tfsdk.Int64  `tfsdk:"attribute_version"`
}

func (r *vaultResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (r *vaultResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*onepassword.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *onepassword.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *vaultResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a 1Password vault.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vault ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Vault title.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Vault description.",
			},
			"allow_admins_access": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Grant administrators access to this vault. Only settable at creation.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"vault_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vault type, e.g. `personal`.",
			},
			"active_item_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of active items in the vault.",
			},
			"content_version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Vault content version, incremented on item changes.",
			},
			"attribute_version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Vault attribute version, incremented on presentation changes.",
			},
		},
	}
}

func (r *vaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := onepassword.VaultCreateParams{
		Title:             data.Title.ValueString(),
		Description:       data.Description.ValueStringPointer(),
		AllowAdminsAccess: data.AllowAdminsSetup.ValueBoolPointer(),
	}

	vault, err := r.client.Vaults().Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Vault",
			fmt.Sprintf("creating Vault (%s): %s", data.Title.ValueString(), err),
		)
		return
	}

	setVaultResourceState(&data, vault)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vaultResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vault, err := r.client.Vaults().Get(ctx, data.ID.ValueString(), onepassword.VaultGetParams{})
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Vault",
			fmt.Sprintf("reading Vault (%s): %s", data.ID.ValueString(), err),
		)
		return
	}

	setVaultResourceState(&data, vault)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := onepassword.VaultUpdateParams{
		Title:       plan.Title.ValueStringPointer(),
		Description: plan.Description.ValueStringPointer(),
	}

	vault, err := r.client.Vaults().Update(ctx, state.ID.ValueString(), params)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Vault",
			fmt.Sprintf("updating Vault (%s): %s", state.ID.ValueString(), err),
		)
		return
	}

	setVaultResourceState(&plan, vault)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vaultResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Vaults().Delete(ctx, data.ID.ValueString())
	if isNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Vault",
			fmt.Sprintf("deleting Vault (%s): %s", data.ID.ValueString(), err),
		)
	}
}

func (r *vaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setVaultResourceState(data *vaultResourceModel, vault onepassword.Vault) {
	data.ID = tfsdk.StringValue(vault.ID)
	data.Title = tfsdk.StringValue(vault.Title)
	if vault.Description == "" {
		data.Description = tfsdk.StringNull()
	} else {
		data.Description = tfsdk.StringValue(vault.Description)
	}
	data.VaultType = tfsdk.StringValue(string(vault.VaultType))
	data.ActiveItemCount = tfsdk.Int64Value(int64(vault.ActiveItemCount))
	data.ContentVersion = tfsdk.Int64Value(int64(vault.ContentVersion))
	data.AttributeVersion = tfsdk.Int64Value(int64(vault.AttributeVersion))
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not in an active state")
}
