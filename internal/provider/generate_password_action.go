// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	stringvalidator "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ action.ActionWithConfigure = &generatePasswordAction{}
)

func NewGeneratePasswordAction() action.Action {
	return &generatePasswordAction{}
}

type generatePasswordAction struct {
	client *onepassword.Client
}

type generatePasswordActionModel struct {
	Mode           types.String `tfsdk:"mode"`
	Length         types.Int64  `tfsdk:"length"`
	IncludeDigits  types.Bool   `tfsdk:"include_digits"`
	IncludeSymbols types.Bool   `tfsdk:"include_symbols"`
	Capitalize     types.Bool   `tfsdk:"capitalize"`
	WordCount      types.Int64  `tfsdk:"word_count"`
	VaultID        types.String `tfsdk:"vault_id"`
	ItemID         types.String `tfsdk:"item_id"`
	FieldTitle     types.String `tfsdk:"field_title"`
}

func (a *generatePasswordAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generate_password"
}

func (a *generatePasswordAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*onepassword.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf("Expected *onepassword.Client, got: %T.", req.ProviderData),
		)
		return
	}
	a.client = client
}

func (a *generatePasswordAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generates a 1Password password (random, PIN, or memorable) using the SDK password generator and writes it into a target item field, rotating the previous value.",
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Recipe: `random`, `pin`, or `memorable`.",
				Validators:          []validator.String{stringvalidator.OneOf("random", "pin", "memorable")},
			},
			"length": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Length for `random` (default 32) and `pin` (default 6) recipes.",
			},
			"include_digits": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Random recipe: include digits (default true).",
			},
			"include_symbols": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Random recipe: include symbols (default true).",
			},
			"capitalize": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Memorable recipe: capitalize words (default true).",
			},
			"word_count": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Memorable recipe: word count (default 4).",
			},
			"vault_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Vault ID of the item whose field receives the generated password.",
			},
			"item_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Item ID whose field receives the generated password.",
			},
			"field_title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Title of the item field to write the generated password into. Created as a Concealed field if absent.",
			},
		},
	}
}

func (a *generatePasswordAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data generatePasswordActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recipe, err := passwordRecipeFromConfig(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid password recipe", err.Error())
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: "Generating password via 1Password"})
	generated, err := onepassword.Secrets.GeneratePassword(ctx, recipe)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error generating password",
			fmt.Sprintf("generating %s password: %s", data.Mode.ValueString(), err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Writing password into item field %q", data.FieldTitle.ValueString())})
	item, err := a.client.Items().Get(ctx, data.VaultID.ValueString(), data.ItemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading item",
			fmt.Sprintf("reading item (%s/%s): %s", data.VaultID.ValueString(), data.ItemID.ValueString(), err),
		)
		return
	}

	fieldTitle := data.FieldTitle.ValueString()
	updated := false
	for i := range item.Fields {
		if item.Fields[i].Title == fieldTitle {
			item.Fields[i].Value = generated.Password
			item.Fields[i].FieldType = onepassword.ItemFieldTypeConcealed
			updated = true
			break
		}
	}
	if !updated {
		item.Fields = append(item.Fields, onepassword.ItemField{
			Title:     fieldTitle,
			FieldType: onepassword.ItemFieldTypeConcealed,
			Value:     generated.Password,
		})
	}

	if _, err := a.client.Items().Put(ctx, item); err != nil {
		resp.Diagnostics.AddError(
			"Error updating item",
			fmt.Sprintf("writing generated password into item (%s/%s): %s", data.VaultID.ValueString(), data.ItemID.ValueString(), err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: "Password generated and written to item"})
}

func passwordRecipeFromConfig(data *generatePasswordActionModel) (onepassword.PasswordRecipe, error) {
	length := uint32(0)
	if !data.Length.IsNull() && !data.Length.IsUnknown() {
		if data.Length.ValueInt64() < 1 || data.Length.ValueInt64() > 64 {
			return onepassword.PasswordRecipe{}, fmt.Errorf("length must be between 1 and 64, got %d", data.Length.ValueInt64())
		}
		length = uint32(data.Length.ValueInt64())
	}

	switch data.Mode.ValueString() {
	case "random":
		if length == 0 {
			length = 32
		}
		includeDigits := true
		if !data.IncludeDigits.IsNull() && !data.IncludeDigits.IsUnknown() {
			includeDigits = data.IncludeDigits.ValueBool()
		}
		includeSymbols := true
		if !data.IncludeSymbols.IsNull() && !data.IncludeSymbols.IsUnknown() {
			includeSymbols = data.IncludeSymbols.ValueBool()
		}
		return onepassword.NewPasswordRecipeTypeVariantRandom(&onepassword.PasswordRecipeRandomInner{
			Length:         length,
			IncludeDigits:  includeDigits,
			IncludeSymbols: includeSymbols,
		}), nil
	case "pin":
		if length == 0 {
			length = 6
		}
		return onepassword.NewPasswordRecipeTypeVariantPin(&onepassword.PasswordRecipePinInner{
			Length: length,
		}), nil
	case "memorable":
		wordCount := uint32(4)
		if !data.WordCount.IsNull() && !data.WordCount.IsUnknown() {
			wordCount = uint32(data.WordCount.ValueInt64())
		}
		capitalize := true
		if !data.Capitalize.IsNull() && !data.Capitalize.IsUnknown() {
			capitalize = data.Capitalize.ValueBool()
		}
		return onepassword.NewPasswordRecipeTypeVariantMemorable(&onepassword.PasswordRecipeMemorableInner{
			WordCount:     wordCount,
			Capitalize:    capitalize,
			WordListType:  onepassword.WordListTypeFullWords,
			SeparatorType: onepassword.SeparatorTypeHyphens,
		}), nil
	default:
		return onepassword.PasswordRecipe{}, fmt.Errorf("unknown mode %q", data.Mode.ValueString())
	}
}
