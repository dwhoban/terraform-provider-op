// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	stringvalidator "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	Address   tfsdk.Object `tfsdk:"address"`
	SSHKey    tfsdk.Object `tfsdk:"ssh_key"`
}

type itemFieldAddressModel struct {
	Street  tfsdk.String `tfsdk:"street"`
	City    tfsdk.String `tfsdk:"city"`
	State   tfsdk.String `tfsdk:"state"`
	Zip     tfsdk.String `tfsdk:"zip"`
	Country tfsdk.String `tfsdk:"country"`
}

type itemSectionModel struct {
	ID    tfsdk.String `tfsdk:"id"`
	Title tfsdk.String `tfsdk:"title"`
}

type itemWebsiteModel struct {
	URL              tfsdk.String `tfsdk:"url"`
	Label            tfsdk.String `tfsdk:"label"`
	AutofillBehavior tfsdk.String `tfsdk:"autofill_behavior"`
}

type itemResourceModel struct {
	ID             tfsdk.String       `tfsdk:"id"`
	VaultID        tfsdk.String       `tfsdk:"vault_id"`
	Title          tfsdk.String       `tfsdk:"title"`
	Category       tfsdk.String       `tfsdk:"category"`
	Notes          tfsdk.String       `tfsdk:"notes"`
	Tags           tfsdk.List         `tfsdk:"tags"`
	Fields         tfsdk.List         `tfsdk:"fields"`
	Sections       tfsdk.List         `tfsdk:"sections"`
	Version        tfsdk.Int64        `tfsdk:"version"`
	DeleteMode     tfsdk.String       `tfsdk:"delete_mode"`
	GenerateSSHKey tfsdk.String       `tfsdk:"generate_ssh_key"`
	Websites       []itemWebsiteModel `tfsdk:"websites"`
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
				Computed:            true,
				MarkdownDescription: "Item fields.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Field ID. Set to a built-in field ID (e.g. `username`, `password`) where applicable; otherwise computed.",
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
							Computed:            true,
							Sensitive:           true,
							MarkdownDescription: "Field value, treated as sensitive. Constraints: `Date` fields must be `YYYY-MM-DD`; `MonthYear` must be `MM/YYYY`; `Reference` must be a 26-character item ID in the same vault; `SshKey` must be a decrypted PEM private key (one per SshKey-category item); `Totp` accepts an `otpauth://` URI.",
						},
						"address": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Address details; required for `Address` fields (the value is derived from these).",
							Attributes: map[string]schema.Attribute{
								"street":  schema.StringAttribute{Required: true, MarkdownDescription: "Street."},
								"city":    schema.StringAttribute{Required: true, MarkdownDescription: "City."},
								"state":   schema.StringAttribute{Optional: true, MarkdownDescription: "State."},
								"zip":     schema.StringAttribute{Optional: true, MarkdownDescription: "ZIP / postal code."},
								"country": schema.StringAttribute{Required: true, MarkdownDescription: "Country."},
							},
						},
						"ssh_key": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "SSH key attributes computed by 1Password for `SshKey` fields.",
							Attributes: map[string]schema.Attribute{
								"public_key":  schema.StringAttribute{Computed: true, MarkdownDescription: "Public key."},
								"fingerprint": schema.StringAttribute{Computed: true, MarkdownDescription: "Key fingerprint."},
								"key_type":    schema.StringAttribute{Computed: true, MarkdownDescription: "Key type, e.g. `Ed25519` or `RSA, 4096-bit`."},
							},
						},
					},
				},
			},
			"sections": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
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
			"websites": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Websites for autofill (Login and Password categories).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url":               schema.StringAttribute{Required: true, MarkdownDescription: "Website URL."},
						"label":             schema.StringAttribute{Optional: true, MarkdownDescription: "Website label, e.g. 'website'."},
						"autofill_behavior": schema.StringAttribute{Optional: true, MarkdownDescription: "One of AnywhereOnWebsite, ExactDomain, Never."},
					},
				},
			},
			"generate_ssh_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Generate a new private key of this type (ed25519, rsa2048, rsa3072, rsa4096) and store it in the item's private key field; 1Password derives the public key, fingerprint, and key type. Only for SshKey items. Changing this replaces the item.",
				Validators:          []validator.String{stringvalidator.OneOf("ed25519", "rsa2048", "rsa3072", "rsa4096")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"delete_mode": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Destroy behavior: `delete` (default) permanently deletes the item, `archive` archives it instead.",
				Validators:          []validator.String{stringvalidator.OneOf("", "delete", "archive")},
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
	var planFields []itemFieldModel
	if !data.Fields.IsUnknown() && !data.Fields.IsNull() {
		resp.Diagnostics.Append(data.Fields.ElementsAs(ctx, &planFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	params.Fields = sdkFieldsFromModel(ctx, planFields)
	var planSections []itemSectionModel
	if !data.Sections.IsUnknown() && !data.Sections.IsNull() {
		resp.Diagnostics.Append(data.Sections.ElementsAs(ctx, &planSections, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	params.Sections = sdkSectionsFromModel(planSections)
	params.Websites = sdkWebsitesFromModel(data.Websites)

	if !data.GenerateSSHKey.IsNull() && !data.GenerateSSHKey.IsUnknown() {
		pemKey, err := generateSSHKeyPEM(data.GenerateSSHKey.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error generating SSH key", err.Error())
			return
		}
		params.Fields = append(params.Fields, onepassword.ItemField{
			ID:        "private_key",
			Title:     "private key",
			FieldType: onepassword.ItemFieldTypeSSHKey,
			Value:     pemKey,
		})
	}

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
	var updateFields []itemFieldModel
	if !plan.Fields.IsUnknown() && !plan.Fields.IsNull() {
		resp.Diagnostics.Append(plan.Fields.ElementsAs(ctx, &updateFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	item.Fields = sdkFieldsFromModel(ctx, updateFields)
	var updateSections []itemSectionModel
	if !plan.Sections.IsUnknown() && !plan.Sections.IsNull() {
		resp.Diagnostics.Append(plan.Sections.ElementsAs(ctx, &updateSections, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	item.Sections = sdkSectionsFromModel(updateSections)

	current, err := r.client.Items().Get(ctx, state.VaultID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Item before update", err.Error())
		return
	}
	idsByTitle := map[string]string{}
	for _, cf := range current.Fields {
		if cf.ID != "" {
			idsByTitle[cf.Title] = cf.ID
		}
	}
	for i := range item.Fields {
		if item.Fields[i].ID == "" {
			item.Fields[i].ID = idsByTitle[item.Fields[i].Title]
		}
	}
	item.CreatedAt = current.CreatedAt
	item.UpdatedAt = current.UpdatedAt
	item.Version = current.Version
	item.Websites = sdkWebsitesFromModel(plan.Websites)

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

func sdkFieldsFromModel(ctx context.Context, models []itemFieldModel) []onepassword.ItemField {
	fields := make([]onepassword.ItemField, 0, len(models))
	seen := map[string]int{}
	for _, f := range models {
		id := f.ID.ValueString()
		if id == "" {
			id = fieldIDFromTitle(f.Title.ValueString())
			if n, dup := seen[id]; dup {
				seen[id] = n + 1
				id = fmt.Sprintf("%s_%d", id, n+1)
			} else {
				seen[id] = 1
			}
		}
		field := onepassword.ItemField{
			ID:        id,
			Title:     f.Title.ValueString(),
			FieldType: onepassword.ItemFieldType(f.FieldType.ValueString()),
			Value:     f.Value.ValueString(),
		}
		if !f.SectionID.IsNull() {
			sectionID := f.SectionID.ValueString()
			field.SectionID = &sectionID
		}
		if !f.Address.IsNull() && !f.Address.IsUnknown() {
			var addr itemFieldAddressModel
			diags := f.Address.As(ctx, &addr, basetypes.ObjectAsOptions{})
			_ = diags
			details := onepassword.NewItemFieldDetailsTypeVariantAddress(&onepassword.AddressFieldDetails{
				Street:  addr.Street.ValueString(),
				City:    addr.City.ValueString(),
				State:   addr.State.ValueString(),
				Zip:     addr.Zip.ValueString(),
				Country: addr.Country.ValueString(),
			})
			field.Details = &details
		}
		fields = append(fields, field)
	}
	return fields
}

func fieldIDFromTitle(title string) string {
	slug := strings.ToLower(strings.Join(strings.Fields(title), "_"))
	if slug == "" {
		slug = "field"
	}
	return slug
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
	var result diag.Diagnostics
	data.ID = tfsdk.StringValue(item.ID)
	data.Title = tfsdk.StringValue(item.Title)
	data.Category = tfsdk.StringValue(string(item.Category))
	data.VaultID = tfsdk.StringValue(item.VaultID)
	if item.Notes == "" {
		data.Notes = tfsdk.StringNull()
	} else {
		data.Notes = tfsdk.StringValue(item.Notes)
	}
	data.Version = tfsdk.Int64Value(int64(item.Version))

	data.Tags = tfsdk.ListNull(tfsdk.StringType)
	if len(item.Tags) > 0 {
		tagsList, diags := tfsdk.ListValueFrom(ctx, tfsdk.StringType, item.Tags)
		result = append(result, diags...)
		data.Tags = tagsList
	}

	fieldList, diags := itemFieldListFromSDK(item.Fields)
	result = append(result, diags...)
	data.Fields = fieldList

	sectionList, diags := itemSectionListFromSDK(ctx, item.Sections)
	result = append(result, diags...)
	data.Sections = sectionList

	data.Websites = modelWebsitesFromSDK(item.Websites)
	return result
}

// itemFieldListFromSDK converts server fields into a `fields`-list value,
// including Address/SSHKey Details handling. Returns a null list for empty
// input.
func itemFieldListFromSDK(fields []onepassword.ItemField) (tfsdk.List, diag.Diagnostics) {
	var result diag.Diagnostics
	fieldObjType := tfsdk.ObjectType{AttrTypes: itemFieldObjectAttrs()}
	var fieldObjs []attr.Value
	for _, f := range fields {
		sectionID := tfsdk.StringNull()
		if f.SectionID != nil && *f.SectionID != "" {
			sectionID = tfsdk.StringValue(*f.SectionID)
		}
		address := tfsdk.ObjectNull(itemAddressObjectAttrs())
		sshKey := tfsdk.ObjectNull(itemSSHKeyObjectAttrs())
		if f.Details != nil {
			switch f.Details.Type {
			case onepassword.ItemFieldDetailsTypeVariantAddress:
				if a := f.Details.Address(); a != nil {
					address = tfsdk.ObjectValueMust(itemAddressObjectAttrs(), map[string]attr.Value{
						"street": tfsdk.StringValue(a.Street), "city": tfsdk.StringValue(a.City), "state": tfsdk.StringValue(a.State),
						"zip": tfsdk.StringValue(a.Zip), "country": tfsdk.StringValue(a.Country),
					})
				}
			case onepassword.ItemFieldDetailsTypeVariantSSHKey:
				if k := f.Details.SSHKey(); k != nil {
					sshKey = tfsdk.ObjectValueMust(itemSSHKeyObjectAttrs(), map[string]attr.Value{
						"public_key": tfsdk.StringValue(k.PublicKey), "fingerprint": tfsdk.StringValue(k.Fingerprint), "key_type": tfsdk.StringValue(k.KeyType),
					})
				}
			}
		}
		obj, diags := tfsdk.ObjectValue(fieldObjType.AttrTypes, map[string]attr.Value{
			"id":         tfsdk.StringValue(f.ID),
			"title":      tfsdk.StringValue(f.Title),
			"section_id": sectionID,
			"type":       tfsdk.StringValue(string(f.FieldType)),
			"value":      tfsdk.StringValue(f.Value),
			"address":    address,
			"ssh_key":    sshKey,
		})
		result = append(result, diags...)
		fieldObjs = append(fieldObjs, obj)
	}
	if len(fieldObjs) == 0 {
		return tfsdk.ListNull(fieldObjType), result
	}
	fieldList, diags := tfsdk.ListValue(fieldObjType, fieldObjs)
	result = append(result, diags...)
	return fieldList, result
}

// itemSectionListFromSDK converts server sections into a `sections`-list
// value. Returns a null list for empty input.
func itemSectionListFromSDK(ctx context.Context, sections []onepassword.ItemSection) (tfsdk.List, diag.Diagnostics) {
	var result diag.Diagnostics
	var models []itemSectionModel
	for _, s := range sections {
		if s.ID == "" && s.Title == "" {
			continue
		}
		models = append(models, itemSectionModel{
			ID:    tfsdk.StringValue(s.ID),
			Title: tfsdk.StringValue(s.Title),
		})
	}
	if len(models) == 0 {
		return tfsdk.ListNull(tfsdk.ObjectType{AttrTypes: itemSectionObjectAttrs()}), result
	}
	sectionList, diags := tfsdk.ListValueFrom(ctx, tfsdk.ObjectType{AttrTypes: itemSectionObjectAttrs()}, models)
	result = append(result, diags...)
	return sectionList, result
}

func modelWebsitesFromSDK(websites []onepassword.Website) []itemWebsiteModel {
	if len(websites) == 0 {
		return nil
	}
	out := make([]itemWebsiteModel, 0, len(websites))
	for _, w := range websites {
		out = append(out, itemWebsiteModel{
			URL:              tfsdk.StringValue(w.URL),
			Label:            tfsdk.StringValue(w.Label),
			AutofillBehavior: tfsdk.StringValue(string(w.AutofillBehavior)),
		})
	}
	return out
}

func sdkWebsitesFromModel(models []itemWebsiteModel) []onepassword.Website {
	websites := make([]onepassword.Website, 0, len(models))
	for _, w := range models {
		websites = append(websites, onepassword.Website{
			URL:              w.URL.ValueString(),
			Label:            w.Label.ValueString(),
			AutofillBehavior: onepassword.AutofillBehavior(w.AutofillBehavior.ValueString()),
		})
	}
	return websites
}

func itemAddressObjectAttrs() map[string]attr.Type {
	return map[string]attr.Type{
		"street": tfsdk.StringType, "city": tfsdk.StringType, "state": tfsdk.StringType,
		"zip": tfsdk.StringType, "country": tfsdk.StringType,
	}
}

func itemSSHKeyObjectAttrs() map[string]attr.Type {
	return map[string]attr.Type{
		"public_key": tfsdk.StringType, "fingerprint": tfsdk.StringType, "key_type": tfsdk.StringType,
	}
}

func itemFieldObjectAttrs() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         tfsdk.StringType,
		"title":      tfsdk.StringType,
		"section_id": tfsdk.StringType,
		"type":       tfsdk.StringType,
		"value":      tfsdk.StringType,
		"address": tfsdk.ObjectType{AttrTypes: map[string]attr.Type{
			"street": tfsdk.StringType, "city": tfsdk.StringType, "state": tfsdk.StringType,
			"zip": tfsdk.StringType, "country": tfsdk.StringType,
		}},
		"ssh_key": tfsdk.ObjectType{AttrTypes: map[string]attr.Type{
			"public_key": tfsdk.StringType, "fingerprint": tfsdk.StringType, "key_type": tfsdk.StringType,
		}},
	}
}

func itemSectionObjectAttrs() map[string]attr.Type {
	return map[string]attr.Type{
		"id":    tfsdk.StringType,
		"title": tfsdk.StringType,
	}
}
