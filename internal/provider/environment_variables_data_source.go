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
	_ datasource.DataSourceWithConfigure = &environmentVariablesDataSource{}
)

func NewEnvironmentVariablesDataSource() datasource.DataSource {
	return &environmentVariablesDataSource{}
}

type environmentVariablesDataSource struct {
	client *onepassword.Client
}

type environmentVariablesDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	Variables     types.List   `tfsdk:"variables"`
}

var environmentVariableAttrs = map[string]attr.Type{
	"name":   types.StringType,
	"value":  types.StringType,
	"masked": types.BoolType,
}

func (d *environmentVariablesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variables"
}

func (d *environmentVariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *environmentVariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the environment variables of a 1Password Environment (beta). Variable values are sensitive.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Environment ID.",
			},
			"variables": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Environment variables.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "Variable name."},
						"value":  schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Variable value."},
						"masked": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the variable is masked."},
					},
				},
			},
		},
	}
}

func (d *environmentVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data environmentVariablesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.Environments().GetVariables(ctx, data.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Environment variables",
			fmt.Sprintf("reading Environment (%s) variables: %s", data.EnvironmentID.ValueString(), err),
		)
		return
	}

	objType := types.ObjectType{AttrTypes: environmentVariableAttrs}
	objects := make([]attr.Value, 0, len(result.Variables))
	for _, v := range result.Variables {
		obj, diags := types.ObjectValueFrom(ctx, objType.AttrTypes, map[string]attr.Value{
			"name":   types.StringValue(v.Name),
			"value":  types.StringValue(v.Value),
			"masked": types.BoolValue(v.Masked),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objects = append(objects, obj)
	}

	list, diags := types.ListValueFrom(ctx, objType, objects)
	resp.Diagnostics.Append(diags...)
	data.Variables = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
