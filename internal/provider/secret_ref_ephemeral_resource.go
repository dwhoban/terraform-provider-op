// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/1password/onepassword-sdk-go"
	stringvalidator "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResourceWithConfigure = &secretRefEphemeralResource{}
)

var secretRefPattern = regexp.MustCompile(`^op://[^\s/]+/[^\s/]+/[^\s/]+(/[^\s/]+)?$`)

func NewSecretRefEphemeralResource() ephemeral.EphemeralResource {
	return &secretRefEphemeralResource{}
}

type secretRefEphemeralResource struct {
	client *onepassword.Client
}

type secretRefEphemeralResourceModel struct {
	Reference tfsdk.String `tfsdk:"reference"`
	Secret    tfsdk.String `tfsdk:"secret"`
}

func (r *secretRefEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_ref"
}

func (r *secretRefEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*onepassword.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *onepassword.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *secretRefEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resolves a 1Password secret reference (`op://vault/item/field`) to its value at apply time. The value is never persisted to state or plan.",
		Attributes: map[string]schema.Attribute{
			"reference": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Secret reference URI, e.g. `op://vault/item/field` or `op://vault/item/section/field`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(secretRefPattern, "must be a secret reference URI like op://vault/item/field"),
				},
			},
			"secret": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The resolved secret value. Never persisted to state.",
			},
		},
	}
}

func (r *secretRefEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data secretRefEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.Secrets().Resolve(ctx, data.Reference.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error resolving secret reference",
			fmt.Sprintf("resolving secret reference (%s): %s", data.Reference.ValueString(), err),
		)
		return
	}

	data.Secret = tfsdk.StringValue(secret)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
