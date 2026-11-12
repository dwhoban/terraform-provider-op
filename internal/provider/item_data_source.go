// Copyright (c) 2026 Dan Hoban
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
	_ datasource.DataSourceWithConfigure = &itemDataSource{}
)

func NewItemDataSource() datasource.DataSource {
	return &itemDataSource{}
}

type itemDataSource struct {
	client *onepassword.Client
}

type itemDataSourceModel struct {
	ID       tfsdk.String       `tfsdk:"id"`
	VaultID  tfsdk.String       `tfsdk:"vault_id"`
	Title    tfsdk.String       `tfsdk:"title"`
	Category tfsdk.String       `tfsdk:"category"`
	Notes    tfsdk.String       `tfsdk:"notes"`
	Tags     tfsdk.List         `tfsdk:"tags"`
	Fields   []itemFieldModel   `tfsdk:"fields"`
	Sections []itemSectionModel `tfsdk:"sections"`
	Version  tfsdk.Int64        `tfsdk:"version"`
}

func itemFieldsDataSourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:            true,
		MarkdownDescription: "Item fields.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id":         schema.StringAttribute{Computed: true, MarkdownDescription: "Field ID."},
				"title":      schema.StringAttribute{Computed: true, MarkdownDescription: "Field label."},
				"section_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Section the field belongs to."},
				"type":       schema.StringAttribute{Computed: true, MarkdownDescription: "Field type."},
				"value":      schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Field value."},
			},
		},
	}
}

func itemSectionsDataSourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:            true,
		MarkdownDescription: "Item sections.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id":    schema.StringAttribute{Computed: true, MarkdownDescription: "Section ID."},
				"title": schema.StringAttribute{Computed: true, MarkdownDescription: "Section title."},
			},
		},
	}
}

func (d *itemDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item"
}

func (d *itemDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *itemDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a 1Password item by vault and item ID. Returned field values are sensitive.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Required: true, MarkdownDescription: "Item ID."},
			"vault_id": schema.StringAttribute{Required: true, MarkdownDescription: "Vault ID."},
			"title":    schema.StringAttribute{Computed: true, MarkdownDescription: "Item title."},
			"category": schema.StringAttribute{Computed: true, MarkdownDescription: "Item category."},
			"notes":    schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Item notes."},
			"tags":     schema.ListAttribute{ElementType: tfsdk.StringType, Computed: true, MarkdownDescription: "Item tags."},
			"fields":   itemFieldsDataSourceAttribute(),
			"sections": itemSectionsDataSourceAttribute(),
			"version":  schema.Int64Attribute{Computed: true, MarkdownDescription: "Item version."},
		},
	}
}

func (d *itemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data itemDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, err := d.client.Items().Get(ctx, data.VaultID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Item",
			fmt.Sprintf("reading Item (%s/%s): %s", data.VaultID.ValueString(), data.ID.ValueString(), err),
		)
		return
	}

	data.ID = tfsdk.StringValue(item.ID)
	data.Title = tfsdk.StringValue(item.Title)
	data.Category = tfsdk.StringValue(string(item.Category))
	data.VaultID = tfsdk.StringValue(item.VaultID)
	data.Notes = tfsdk.StringValue(item.Notes)
	data.Version = tfsdk.Int64Value(int64(item.Version))

	tagsList, diags := tfsdk.ListValueFrom(ctx, tfsdk.StringType, item.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsList

	fields := make([]itemFieldModel, 0, len(item.Fields))
	for _, f := range item.Fields {
		field := itemFieldModel{
			ID:        tfsdk.StringValue(f.ID),
			Title:     tfsdk.StringValue(f.Title),
			FieldType: tfsdk.StringValue(string(f.FieldType)),
			Value:     tfsdk.StringValue(f.Value),
		}
		if f.SectionID != nil {
			field.SectionID = tfsdk.StringValue(*f.SectionID)
		}
		fields = append(fields, field)
	}
	data.Fields = fields

	sections := make([]itemSectionModel, 0, len(item.Sections))
	for _, s := range item.Sections {
		sections = append(sections, itemSectionModel{
			ID:    tfsdk.StringValue(s.ID),
			Title: tfsdk.StringValue(s.Title),
		})
	}
	data.Sections = sections

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
