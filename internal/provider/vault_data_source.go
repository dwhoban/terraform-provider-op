// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSourceWithConfigure = &vaultDataSource{}
)

func NewVaultDataSource() datasource.DataSource {
	return &vaultDataSource{}
}

type vaultDataSource struct {
	client *onepassword.Client
}

type vaultDataSourceModel struct {
	ID               tfsdk.String `tfsdk:"id"`
	Title            tfsdk.String `tfsdk:"title"`
	Description      tfsdk.String `tfsdk:"description"`
	VaultType        tfsdk.String `tfsdk:"vault_type"`
	ActiveItemCount  tfsdk.Int64  `tfsdk:"active_item_count"`
	ContentVersion   tfsdk.Int64  `tfsdk:"content_version"`
	AttributeVersion tfsdk.Int64  `tfsdk:"attribute_version"`
}

func (d *vaultDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (d *vaultDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*onepassword.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *onepassword.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *vaultDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a 1Password vault by its ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Vault ID.",
			},
			"title": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vault title.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vault description.",
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
				MarkdownDescription: "Vault content version.",
			},
			"attribute_version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Vault attribute version.",
			},
		},
	}
}

func (d *vaultDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vaultDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vault, err := d.client.Vaults().Get(ctx, data.ID.ValueString(), onepassword.VaultGetParams{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Vault",
			fmt.Sprintf("reading Vault (%s): %s", data.ID.ValueString(), err),
		)
		return
	}

	data.ID = tfsdk.StringValue(vault.ID)
	data.Title = tfsdk.StringValue(vault.Title)
	data.Description = tfsdk.StringValue(vault.Description)
	data.VaultType = tfsdk.StringValue(string(vault.VaultType))
	data.ActiveItemCount = tfsdk.Int64Value(int64(vault.ActiveItemCount))
	data.ContentVersion = tfsdk.Int64Value(int64(vault.ContentVersion))
	data.AttributeVersion = tfsdk.Int64Value(int64(vault.AttributeVersion))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
