// Copyright (c) HashiCorp, Inc.
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
	_ datasource.DataSourceWithConfigure = &itemsDataSource{}
)

func NewItemsDataSource() datasource.DataSource {
	return &itemsDataSource{}
}

type itemsDataSource struct {
	client *onepassword.Client
}

type itemsDataSourceModelReal struct {
	VaultID         types.String `tfsdk:"vault_id"`
	IncludeArchived types.Bool   `tfsdk:"include_archived"`
	Items           types.List   `tfsdk:"items"`
}

var itemOverviewAttrs = map[string]attr.Type{
	"id":       types.StringType,
	"title":    types.StringType,
	"category": types.StringType,
	"vault_id": types.StringType,
	"tags":     types.ListType{ElemType: types.StringType},
	"state":    types.StringType,
}

func (d *itemsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_items"
}

func (d *itemsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *itemsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists items in a vault.",
		Attributes: map[string]schema.Attribute{
			"vault_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Vault ID.",
			},
			"include_archived": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Also list archived items. Defaults to active items only.",
			},
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Item overviews.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true, MarkdownDescription: "Item ID."},
						"title":    schema.StringAttribute{Computed: true, MarkdownDescription: "Item title."},
						"category": schema.StringAttribute{Computed: true, MarkdownDescription: "Item category."},
						"vault_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Vault ID."},
						"tags":     schema.ListAttribute{ElementType: types.StringType, Computed: true, MarkdownDescription: "Item tags."},
						"state":    schema.StringAttribute{Computed: true, MarkdownDescription: "`active` or `archived`."},
					},
				},
			},
		},
	}
}

func (d *itemsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data itemsDataSourceModelReal
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var filters []onepassword.ItemListFilter
	if data.IncludeArchived.ValueBool() {
		active := true
		archived := true
		filters = append(filters, onepassword.NewItemListFilterTypeVariantByState(&onepassword.ItemListFilterByStateInner{
			Active:   active,
			Archived: archived,
		}))
	}

	overviews, err := d.client.Items().List(ctx, data.VaultID.ValueString(), filters...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing Items",
			fmt.Sprintf("listing Items in vault (%s): %s", data.VaultID.ValueString(), err),
		)
		return
	}

	objType := types.ObjectType{AttrTypes: itemOverviewAttrs}
	objects := make([]attr.Value, 0, len(overviews))
	for _, overview := range overviews {
		tags, diags := types.ListValueFrom(ctx, types.StringType, overview.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		obj, diags := types.ObjectValueFrom(ctx, objType.AttrTypes, map[string]attr.Value{
			"id":       types.StringValue(overview.ID),
			"title":    types.StringValue(overview.Title),
			"category": types.StringValue(string(overview.Category)),
			"vault_id": types.StringValue(overview.VaultID),
			"tags":     tags,
			"state":    types.StringValue(string(overview.State)),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objects = append(objects, obj)
	}

	list, diags := types.ListValueFrom(ctx, objType, objects)
	resp.Diagnostics.Append(diags...)
	data.Items = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
