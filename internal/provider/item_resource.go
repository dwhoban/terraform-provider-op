// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	stringvalidator "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &itemResource{}
	_ resource.ResourceWithConfigure   = &itemResource{}
	_ resource.ResourceWithImportState = &itemResource{}
)

func NewItemResource() resource.Resource {
	return &itemResource{}
}

type itemResource struct {
	client *onepassword.Client
}

type itemFieldModel struct {
	ID        tfsdk.String `tfsdk:"id"`
	Title     tfsdk.String `tfsdk:"title"`
	SectionID tfsdk.String `tfsdk:"section_id"`
	FieldType tfsdk.String `tfsdk:"type"`
	Value     tfsdk.String `tfsdk:"value"`
}

type itemSectionModel struct {
	ID    tfsdk.String `tfsdk:"id"`
	Title tfsdk.String `tfsdk:"title"`
}

type itemResourceModel struct {
	ID         tfsdk.String       `tfsdk:"id"`
	VaultID    tfsdk.String       `tfsdk:"vault_id"`
	Title      tfsdk.String       `tfsdk:"title"`
	Category   tfsdk.String       `tfsdk:"category"`
	Notes      tfsdk.String       `tfsdk:"notes"`
	Tags       tfsdk.List         `tfsdk:"tags"`
	Fields     []itemFieldModel   `tfsdk:"fields"`
	Sections   []itemSectionModel `tfsdk:"sections"`
	Version    tfsdk.Int64        `tfsdk:"version"`
	DeleteMode tfsdk.String       `tfsdk:"delete_mode"`
}

func (r *itemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item"
}

func (r *itemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *itemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a 1Password item.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Item ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"vault_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the vault containing the item.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"title": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Item title.",
			},
			"category": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Item category, e.g. `Login`, `Password`, `SecureNote`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"notes": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Item notes.",
			},
			"tags": schema.ListAttribute{
				ElementType:         tfsdk.StringType,
				Optional:            true,
				MarkdownDescription: "Item tags.",
			},
			"fields": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Item fields.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Field ID.",
							PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"title": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Field label.",
						},
						"section_id": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Section the field belongs to.",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("Text"),
							MarkdownDescription: "Field type, e.g. `Text`, `Concealed`, `Totp`, `Url`.",
							Validators: []validator.String{stringvalidator.OneOf(
								"Text", "Concealed", "CreditCardType", "CreditCardNumber", "Phone", "Url",
								"Totp", "Email", "Reference", "SshKey", "MonthYear", "Address", "Date",
							)},
						},
						"value": schema.StringAttribute{
							Optional:            true,
							Sensitive:           true,
							MarkdownDescription: "Field value. Treated as sensitive.",
						},
					},
				},
			},
			"sections": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Item sections.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Section ID.",
							PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"title": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Section title.",
						},
					},
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Item version.",
			},
			"delete_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("delete"),
				MarkdownDescription: "Destroy behavior: `delete` permanently deletes the item, `archive` archives it instead.",
				Validators:          []validator.String{stringvalidator.OneOf("delete", "archive")},
			},
		},
	}
}

func (r *itemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data itemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := onepassword.ItemCreateParams{
		Category: onepassword.ItemCategory(data.Category.ValueString()),
		VaultID:  data.VaultID.ValueString(),
		Title:    data.Title.ValueString(),
	}
	if !data.Notes.IsNull() && !data.Notes.IsUnknown() {
		notes := data.Notes.ValueString()
		params.Notes = &notes
	}
	var tags []string
	resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params.Tags = tags
	params.Fields = sdkFieldsFromModel(data.Fields)
	params.Sections = sdkSectionsFromModel(data.Sections)

	item, err := r.client.Items().Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Item",
			fmt.Sprintf("creating Item (%s): %s", data.Title.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(setItemResourceState(ctx, &data, item)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *itemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data itemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, err := r.client.Items().Get(ctx, data.VaultID.ValueString(), data.ID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Item",
			fmt.Sprintf("reading Item (%s/%s): %s", data.VaultID.ValueString(), data.ID.ValueString(), err),
		)
		return
	}

	deleteMode := data.DeleteMode
	resp.Diagnostics.Append(setItemResourceState(ctx, &data, item)...)
	data.DeleteMode = deleteMode
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *itemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state itemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item := onepassword.Item{
		ID:       state.ID.ValueString(),
		VaultID:  plan.VaultID.ValueString(),
		Title:    plan.Title.ValueString(),
		Category: onepassword.ItemCategory(plan.Category.ValueString()),
	}
	if !plan.Notes.IsNull() && !plan.Notes.IsUnknown() {
		notes := plan.Notes.ValueString()
		item.Notes = notes
	}
	var tags []string
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	item.Tags = tags
	item.Fields = sdkFieldsFromModel(plan.Fields)
	item.Sections = sdkSectionsFromModel(plan.Sections)

	updated, err := r.client.Items().Put(ctx, item)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Item",
			fmt.Sprintf("updating Item (%s/%s): %s", plan.VaultID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(setItemResourceState(ctx, &plan, updated)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *itemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data itemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if strings.EqualFold(data.DeleteMode.ValueString(), "archive") {
		err = r.client.Items().Archive(ctx, data.VaultID.ValueString(), data.ID.ValueString())
	} else {
		err = r.client.Items().Delete(ctx, data.VaultID.ValueString(), data.ID.ValueString())
	}
	if isNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Item",
			fmt.Sprintf("deleting Item (%s/%s): %s", data.VaultID.ValueString(), data.ID.ValueString(), err),
		)
	}
}

func (r *itemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format vault_id/item_id.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vault_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func sdkFieldsFromModel(models []itemFieldModel) []onepassword.ItemField {
	fields := make([]onepassword.ItemField, 0, len(models))
	for _, f := range models {
		field := onepassword.ItemField{
			Title:     f.Title.ValueString(),
			FieldType: onepassword.ItemFieldType(f.FieldType.ValueString()),
			Value:     f.Value.ValueString(),
		}
		if !f.SectionID.IsNull() {
			sectionID := f.SectionID.ValueString()
			field.SectionID = &sectionID
		}
		fields = append(fields, field)
	}
	return fields
}

func sdkSectionsFromModel(models []itemSectionModel) []onepassword.ItemSection {
	sections := make([]onepassword.ItemSection, 0, len(models))
	for _, s := range models {
		sections = append(sections, onepassword.ItemSection{
			Title: s.Title.ValueString(),
		})
	}
	return sections
}

func setItemResourceState(ctx context.Context, data *itemResourceModel, item onepassword.Item) diag.Diagnostics {
	data.ID = tfsdk.StringValue(item.ID)
	data.Title = tfsdk.StringValue(item.Title)
	data.Category = tfsdk.StringValue(string(item.Category))
	data.VaultID = tfsdk.StringValue(item.VaultID)
	data.Notes = tfsdk.StringValue(item.Notes)
	data.Version = tfsdk.Int64Value(int64(item.Version))

	tags := make([]tfsdk.String, 0, len(item.Tags))
	for _, t := range item.Tags {
		tags = append(tags, tfsdk.StringValue(t))
	}
	tagsList, diags := tfsdk.ListValueFrom(ctx, tfsdk.StringType, item.Tags)
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
	return diags
}
