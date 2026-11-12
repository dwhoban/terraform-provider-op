// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSourceWithConfigure = &vaultsDataSource{}
)

func NewVaultsDataSource() datasource.DataSource {
	return &vaultsDataSource{}
}

type vaultsDataSource struct {
	client *onepassword.Client
}

type vaultsDataSourceModel struct {
	Vaults types.List `tfsdk:"vaults"`
}

var vaultOverviewAttrs = map[string]attr.Type{
	"id":                types.StringType,
	"title":             types.StringType,
	"description":       types.StringType,
	"vault_type":        types.StringType,
	"active_item_count": types.Int64Type,
	"content_version":   types.Int64Type,
	"attribute_version": types.Int64Type,
}

func (d *vaultsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vaults"
}

func (d *vaultsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vaultsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all 1Password vaults accessible to the configured credentials.",
		Attributes: map[string]schema.Attribute{
			"vaults": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of accessible vaults.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Computed: true, MarkdownDescription: "Vault ID."},
						"title":             schema.StringAttribute{Computed: true, MarkdownDescription: "Vault title."},
						"description":       schema.StringAttribute{Computed: true, MarkdownDescription: "Vault description."},
						"vault_type":        schema.StringAttribute{Computed: true, MarkdownDescription: "Vault type, e.g. `personal`."},
						"active_item_count": schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of active items."},
						"content_version":   schema.Int64Attribute{Computed: true, MarkdownDescription: "Vault content version."},
						"attribute_version": schema.Int64Attribute{Computed: true, MarkdownDescription: "Vault attribute version."},
					},
				},
			},
		},
	}
}

func (d *vaultsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vaultsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	overviews, err := d.client.Vaults().List(ctx, onepassword.VaultListParams{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing Vaults",
			fmt.Sprintf("listing Vaults: %s", err),
		)
		return
	}

	objects := make([]attr.Value, 0, len(overviews))
	for _, overview := range overviews {
		objTypes := types.ObjectType{AttrTypes: vaultOverviewAttrs}
		obj, diags := types.ObjectValue(objTypes.AttrTypes, map[string]attr.Value{
			"id":                types.StringValue(overview.ID),
			"title":             types.StringValue(overview.Title),
			"description":       types.StringValue(overview.Description),
			"vault_type":        types.StringValue(string(overview.VaultType)),
			"active_item_count": types.Int64Value(int64(overview.ActiveItemCount)),
			"content_version":   types.Int64Value(int64(overview.ContentVersion)),
			"attribute_version": types.Int64Value(int64(overview.AttributeVersion)),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objects = append(objects, obj)
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: vaultOverviewAttrs}, objects)
	resp.Diagnostics.Append(diags...)
	data.Vaults = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
