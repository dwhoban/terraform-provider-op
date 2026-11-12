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
	_ datasource.DataSourceWithConfigure = &groupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *onepassword.Client
}

type groupDataSourceModel struct {
	ID          tfsdk.String `tfsdk:"id"`
	Title       tfsdk.String `tfsdk:"title"`
	Description tfsdk.String `tfsdk:"description"`
	GroupType   tfsdk.String `tfsdk:"group_type"`
	State       tfsdk.String `tfsdk:"state"`
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a 1Password group by its ID.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Required: true, MarkdownDescription: "Group ID."},
			"title":       schema.StringAttribute{Computed: true, MarkdownDescription: "Group title."},
			"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Group description."},
			"group_type":  schema.StringAttribute{Computed: true, MarkdownDescription: "Group type, e.g. `userDefined`."},
			"state":       schema.StringAttribute{Computed: true, MarkdownDescription: "`active`, `deleted`, or `unsupported`."},
		},
	}
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data groupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := d.client.Groups().Get(ctx, data.ID.ValueString(), onepassword.GroupGetParams{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Group",
			fmt.Sprintf("reading Group (%s): %s", data.ID.ValueString(), err),
		)
		return
	}

	data.ID = tfsdk.StringValue(group.ID)
	data.Title = tfsdk.StringValue(group.Title)
	data.Description = tfsdk.StringValue(group.Description)
	data.GroupType = tfsdk.StringValue(string(group.GroupType))
	data.State = tfsdk.StringValue(string(group.State))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
