// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
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
	_ resource.Resource                   = &categoryItemResource{}
	_ resource.ResourceWithConfigure      = &categoryItemResource{}
	_ resource.ResourceWithImportState    = &categoryItemResource{}
	_ resource.ResourceWithValidateConfig = &categoryItemResource{}
)

// NewCategoryItemResource returns a resource for one item category, driven
// entirely by its spec.
func NewCategoryItemResource(spec categorySpec) resource.Resource {
	return &categoryItemResource{spec: spec}
}

// categoryItemResource is a generic spec-driven 1Password item resource.
// Because its attributes differ per category, it decodes and encodes the
// whole resource as a types.Object instead of a tfsdk-tagged model.
type categoryItemResource struct {
	spec   categorySpec
	client *onepassword.Client
}

func (r *categoryItemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.TypeName
}

func (r *categoryItemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *categoryItemResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if !r.spec.GenerateSSHKey {
		return
	}
	resourcevalidator.Conflicting(
		path.MatchRoot("generate_ssh_key"),
		path.MatchRoot("private_key"),
	).ValidateResource(ctx, req, resp)
}

func (r *categoryItemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = categoryItemSchema(r.spec)
}

// categoryItemSchema builds the schema for one category spec: the common
// op_item attributes plus the spec's typed fields.
func categoryItemSchema(spec categorySpec) schema.Schema {
	attributes := map[string]schema.Attribute{
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
		"notes": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Item notes (stored as the item's `notesPlain` field).",
		},
		"tags": schema.ListAttribute{
			ElementType:         tfsdk.StringType,
			Optional:            true,
			MarkdownDescription: "Item tags.",
		},
		"custom_fields": schema.ListNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Custom fields beyond this category's typed fields.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: customFieldObjectSchema(),
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
		"delete_mode": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Destroy behavior: `delete` (default) permanently deletes the item, `archive` archives it instead.",
			Validators:          []validator.String{stringvalidator.OneOf("", "delete", "archive")},
		},
	}

	for _, f := range spec.Fields {
		if f.Address {
			attributes[f.Attr] = schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: fieldDescription(f),
				Attributes:          addressObjectSchema(),
			}
			continue
		}
		description := fieldDescription(f)
		if hint, ok := menuValueHint(spec.TypeName, f.FieldID); ok {
			description = fmt.Sprintf("%s Menu field; common values: %s.", strings.TrimSuffix(description, "."), hint)
		}
		// private_key is Optional+Computed: with generate_ssh_key the value
		// comes from the provider, so it must survive unrelated updates.
		computed := f.Type == onepassword.ItemFieldTypeSSHKey
		attributes[f.Attr] = schema.StringAttribute{
			Optional:            true,
			Computed:            computed,
			Sensitive:           f.Sensitive,
			MarkdownDescription: description,
		}
	}

	for _, d := range spec.SSHDetails {
		description := "SSH key detail computed by 1Password."
		switch d.DetailKey {
		case "public_key":
			description = "Public key."
		case "fingerprint":
			description = "Key fingerprint."
		case "key_type":
			description = "Key type, e.g. `Ed25519` or `RSA, 4096-bit`."
		}
		attributes[d.Attr] = schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: description,
		}
	}

	if spec.Websites {
		attributes["websites"] = schema.ListNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Websites for autofill (Login and Password categories).",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"url":               schema.StringAttribute{Required: true, MarkdownDescription: "Website URL."},
					"label":             schema.StringAttribute{Optional: true, MarkdownDescription: "Website label, e.g. 'website'."},
					"autofill_behavior": schema.StringAttribute{Optional: true, MarkdownDescription: "One of AnywhereOnWebsite, ExactDomain, Never."},
				},
			},
		}
	}

	if spec.GenerateSSHKey {
		attributes["generate_ssh_key"] = schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Generate a new private key of this type (ed25519, rsa2048, rsa3072, rsa4096) and store it in the item's private key field; 1Password derives the public key, fingerprint, and key type. Changing this replaces the item.",
			Validators:          []validator.String{stringvalidator.OneOf("ed25519", "rsa2048", "rsa3072", "rsa4096")},
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
	}

	return schema.Schema{
		MarkdownDescription: spec.Description,
		Attributes:          attributes,
	}
}

// customFieldObjectSchema mirrors op_item's `fields` nested attributes so
// custom_fields entries behave identically.
func customFieldObjectSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
			Attributes:          addressObjectSchema(),
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
	}
}

func addressObjectSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"street":  schema.StringAttribute{Required: true, MarkdownDescription: "Street."},
		"city":    schema.StringAttribute{Required: true, MarkdownDescription: "City."},
		"state":   schema.StringAttribute{Optional: true, MarkdownDescription: "State."},
		"zip":     schema.StringAttribute{Optional: true, MarkdownDescription: "ZIP / postal code."},
		"country": schema.StringAttribute{Required: true, MarkdownDescription: "Country."},
	}
}

// menuValueHint documents common menu values where the template dump knows
// them; menu options are not enumerated, so no OneOf validator is added.
func menuValueHint(typeName, fieldID string) (string, bool) {
	switch {
	case typeName == "item_api_credentials" && fieldID == "type":
		return "e.g. `bearer`", true
	case typeName == "item_database" && fieldID == "database_type":
		return "e.g. `mysql`, `postgresql`", true
	}
	return "", false
}

// attrTypes derives the object attribute types from the schema so the schema
// stays the single source of truth.
func (r *categoryItemResource) attrTypes() map[string]attr.Type {
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	objType, isObj := resp.Schema.Type().(basetypes.ObjectType)
	if !isObj {
		panic(fmt.Sprintf("category item schema type is %T, want basetypes.ObjectType", resp.Schema.Type()))
	}
	return objType.AttrTypes
}

// categoryItemCommon carries the common attributes shared with op_item,
// decoded from a types.Object attribute map.
type categoryItemCommon struct {
	VaultID        tfsdk.String
	Title          tfsdk.String
	Notes          tfsdk.String
	Tags           []string
	CustomFields   []itemFieldModel
	Sections       []itemSectionModel
	DeleteMode     tfsdk.String
	GenerateSSHKey tfsdk.String
	Websites       []itemWebsiteModel
}

func categoryCommonFromAttrs(ctx context.Context, attrs map[string]attr.Value) (categoryItemCommon, diag.Diagnostics) {
	var result diag.Diagnostics
	var common categoryItemCommon

	common.VaultID, _ = attrs["vault_id"].(tfsdk.String)
	common.Title, _ = attrs["title"].(tfsdk.String)
	common.Notes, _ = attrs["notes"].(tfsdk.String)
	common.DeleteMode, _ = attrs["delete_mode"].(tfsdk.String)

	if tags, ok := attrs["tags"].(tfsdk.List); ok && !tags.IsNull() && !tags.IsUnknown() {
		result.Append(tags.ElementsAs(ctx, &common.Tags, false)...)
	}
	if fields, ok := attrs["custom_fields"].(tfsdk.List); ok && !fields.IsNull() && !fields.IsUnknown() {
		result.Append(fields.ElementsAs(ctx, &common.CustomFields, false)...)
	}
	if sections, ok := attrs["sections"].(tfsdk.List); ok && !sections.IsNull() && !sections.IsUnknown() {
		result.Append(sections.ElementsAs(ctx, &common.Sections, false)...)
	}
	if websites, ok := attrs["websites"].(tfsdk.List); ok && !websites.IsNull() && !websites.IsUnknown() {
		result.Append(websites.ElementsAs(ctx, &common.Websites, false)...)
	}
	if key, ok := attrs["generate_ssh_key"].(tfsdk.String); ok {
		common.GenerateSSHKey = key
	}

	return common, result
}

// typedFieldsFromAttrs converts the spec's typed attributes that the user
// set into canonical 1Password fields. Null (or unknown, at create time for
// computed typed attributes) attributes are omitted entirely.
func typedFieldsFromAttrs(ctx context.Context, spec categorySpec, attrs map[string]attr.Value) []onepassword.ItemField {
	var fields []onepassword.ItemField
	for _, f := range spec.Fields {
		v, ok := attrs[f.Attr]
		if !ok || v == nil || v.IsNull() || v.IsUnknown() {
			continue
		}
		field := onepassword.ItemField{
			ID:        f.FieldID,
			Title:     f.Title,
			FieldType: f.Type,
		}
		if f.Address {
			obj, isObj := v.(basetypes.ObjectValue)
			if !isObj {
				continue
			}
			var addr itemFieldAddressModel
			if diags := obj.As(ctx, &addr, basetypes.ObjectAsOptions{}); diags.HasError() {
				continue
			}
			field.Value = ""
			details := onepassword.NewItemFieldDetailsTypeVariantAddress(&onepassword.AddressFieldDetails{
				Street:  addr.Street.ValueString(),
				City:    addr.City.ValueString(),
				State:   addr.State.ValueString(),
				Zip:     addr.Zip.ValueString(),
				Country: addr.Country.ValueString(),
			})
			field.Details = &details
		} else {
			s, isString := v.(tfsdk.String)
			if !isString {
				continue
			}
			field.Value = s.ValueString()
		}
		if f.SectionID != "" {
			sectionID := f.SectionID
			field.SectionID = &sectionID
		}
		fields = append(fields, field)
	}
	return fields
}

// categorySectionsToSend returns the sections to send to 1Password: the
// configured custom sections plus every default section referenced by a
// written field's SectionID. Default sections are skipped when a section
// with the same title is already being sent, so preserved state sections
// don't duplicate them.
func categorySectionsToSend(spec categorySpec, custom []onepassword.ItemSection, fields []onepassword.ItemField) []onepassword.ItemSection {
	sections := make([]onepassword.ItemSection, 0, len(custom))
	sections = append(sections, custom...)

	referenced := map[string]bool{}
	for _, f := range fields {
		if f.SectionID != nil && *f.SectionID != "" {
			referenced[*f.SectionID] = true
		}
	}
	sentTitles := map[string]bool{}
	for _, s := range sections {
		sentTitles[strings.ToLower(s.Title)] = true
	}
	seenIDs := map[string]bool{}
	for _, s := range spec.DefaultSections {
		if !referenced[s.ID] || seenIDs[s.ID] || sentTitles[strings.ToLower(s.Title)] {
			continue
		}
		seenIDs[s.ID] = true
		sentTitles[strings.ToLower(s.Title)] = true
		sections = append(sections, onepassword.ItemSection{ID: s.ID, Title: s.Title})
	}
	return sections
}

// findServerFieldForSpec matches a server field for a typed spec field:
// exact ID, then case-insensitive ID, then exact title. The title stage
// only considers server fields whose ID belongs to no spec field, so a
// present field (e.g. pop_username, title "username") cannot satisfy a
// different absent spec field with the same title (e.g. smtp_username).
func findServerFieldForSpec(spec categorySpec, fields []onepassword.ItemField, want categoryFieldSpec) *onepassword.ItemField {
	for i := range fields {
		if fields[i].ID == want.FieldID {
			return &fields[i]
		}
	}
	for i := range fields {
		if fields[i].ID != "" && strings.EqualFold(fields[i].ID, want.FieldID) {
			return &fields[i]
		}
	}
	for i := range fields {
		if fields[i].ID != "" && !isSpecFieldID(spec, fields[i].ID) && fields[i].Title == want.Title {
			return &fields[i]
		}
	}
	return nil
}

// customSectionsFromSDK strips template default sections (by ID or title)
// from the server's section list.
func customSectionsFromSDK(spec categorySpec, sections []onepassword.ItemSection) []onepassword.ItemSection {
	var out []onepassword.ItemSection
	for _, s := range sections {
		if isDefaultSection(spec, s) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func isDefaultSection(spec categorySpec, s onepassword.ItemSection) bool {
	for _, d := range spec.DefaultSections {
		if s.ID == d.ID || strings.EqualFold(s.ID, d.ID) || strings.EqualFold(s.Title, d.Title) {
			return true
		}
	}
	return false
}

// isSpecFieldID reports whether a server field ID belongs to the spec's
// typed fields (exact or case-insensitive).
func isSpecFieldID(spec categorySpec, fieldID string) bool {
	for _, f := range spec.Fields {
		if f.FieldID == fieldID || strings.EqualFold(f.FieldID, fieldID) {
			return true
		}
	}
	return false
}

func setCategoryStateFromItem(ctx context.Context, spec categorySpec, attrs map[string]attr.Value, item onepassword.Item) diag.Diagnostics {
	var result diag.Diagnostics

	attrs["id"] = tfsdk.StringValue(item.ID)
	attrs["title"] = tfsdk.StringValue(item.Title)
	attrs["vault_id"] = tfsdk.StringValue(item.VaultID)
	if item.Notes == "" {
		attrs["notes"] = tfsdk.StringNull()
	} else {
		attrs["notes"] = tfsdk.StringValue(item.Notes)
	}
	attrs["version"] = tfsdk.Int64Value(int64(item.Version))

	if len(item.Tags) > 0 {
		tagsList, diags := tfsdk.ListValueFrom(ctx, tfsdk.StringType, item.Tags)
		result.Append(diags...)
		attrs["tags"] = tagsList
	} else {
		attrs["tags"] = tfsdk.ListNull(tfsdk.StringType)
	}

	for _, f := range spec.Fields {
		server := findServerFieldForSpec(spec, item.Fields, f)
		if server == nil {
			if f.Address {
				attrs[f.Attr] = tfsdk.ObjectNull(itemAddressObjectAttrs())
			} else {
				attrs[f.Attr] = tfsdk.StringNull()
			}
			continue
		}
		if f.Address {
			address := tfsdk.ObjectNull(itemAddressObjectAttrs())
			if server.Details != nil {
				if a := server.Details.Address(); a != nil {
					var diags diag.Diagnostics
					address, diags = tfsdk.ObjectValue(itemAddressObjectAttrs(), map[string]attr.Value{
						"street":  tfsdk.StringValue(a.Street),
						"city":    tfsdk.StringValue(a.City),
						"state":   tfsdk.StringValue(a.State),
						"zip":     tfsdk.StringValue(a.Zip),
						"country": tfsdk.StringValue(a.Country),
					})
					result.Append(diags...)
				}
			}
			attrs[f.Attr] = address
			continue
		}
		attrs[f.Attr] = tfsdk.StringValue(server.Value)
	}

	for _, d := range spec.SSHDetails {
		value := tfsdk.StringNull()
		if server := findServerFieldForSpec(spec, item.Fields, categoryFieldSpec{FieldID: d.FieldID}); server != nil && server.Details != nil {
			if k := server.Details.SSHKey(); k != nil {
				switch d.DetailKey {
				case "public_key":
					value = tfsdk.StringValue(k.PublicKey)
				case "fingerprint":
					value = tfsdk.StringValue(k.Fingerprint)
				case "key_type":
					value = tfsdk.StringValue(k.KeyType)
				}
			}
		}
		attrs[d.Attr] = value
	}

	// `sections` tracks only user-defined sections: template default
	// sections are managed via the typed fields that reference them, and
	// echoing them here would make every plan propose an update for the
	// config-null Optional+Computed attribute.
	sectionList, diags := itemSectionListFromSDK(ctx, customSectionsFromSDK(spec, item.Sections))
	result.Append(diags...)
	attrs["sections"] = sectionList

	if spec.Websites {
		if len(item.Websites) > 0 {
			websiteList, diags := tfsdk.ListValueFrom(ctx, tfsdk.ObjectType{AttrTypes: itemWebsiteObjectAttrs()}, modelWebsitesFromSDK(item.Websites))
			result.Append(diags...)
			attrs["websites"] = websiteList
		} else {
			attrs["websites"] = tfsdk.ListNull(tfsdk.ObjectType{AttrTypes: itemWebsiteObjectAttrs()})
		}
	}

	var customFields []onepassword.ItemField
	for _, f := range item.Fields {
		if f.ID == "notesPlain" || isSpecFieldID(spec, f.ID) {
			continue
		}
		customFields = append(customFields, f)
	}
	customList, diags := itemFieldListFromSDK(customFields)
	result.Append(diags...)
	attrs["custom_fields"] = customList

	return result
}

func itemWebsiteObjectAttrs() map[string]attr.Type {
	return map[string]attr.Type{
		"url": tfsdk.StringType, "label": tfsdk.StringType, "autofill_behavior": tfsdk.StringType,
	}
}

func attrString(attrs map[string]attr.Value, key string) string {
	if v, ok := attrs[key].(tfsdk.String); ok {
		return v.ValueString()
	}
	return ""
}

func (r *categoryItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var obj tfsdk.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &obj)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := obj.Attributes()

	common, diags := categoryCommonFromAttrs(ctx, attrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := onepassword.ItemCreateParams{
		Category: r.spec.Category,
		VaultID:  common.VaultID.ValueString(),
		Title:    common.Title.ValueString(),
	}
	if !common.Notes.IsNull() && !common.Notes.IsUnknown() {
		notes := common.Notes.ValueString()
		params.Notes = &notes
	}
	params.Tags = common.Tags
	params.Fields = typedFieldsFromAttrs(ctx, r.spec, attrs)
	params.Fields = append(params.Fields, sdkFieldsFromModel(ctx, common.CustomFields)...)

	if !common.GenerateSSHKey.IsNull() && !common.GenerateSSHKey.IsUnknown() {
		pemKey, err := generateSSHKeyPEM(common.GenerateSSHKey.ValueString())
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

	params.Sections = categorySectionsToSend(r.spec, sdkSectionsFromModel(common.Sections), params.Fields)
	params.Websites = sdkWebsitesFromModel(common.Websites)

	item, err := r.client.Items().Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Item",
			fmt.Sprintf("creating Item (%s): %s", common.Title.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(setCategoryStateFromItem(ctx, r.spec, attrs, item)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, diags := tfsdk.ObjectValue(r.attrTypes(), attrs)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}

func (r *categoryItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var obj tfsdk.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &obj)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := obj.Attributes()

	item, err := r.client.Items().Get(ctx, attrString(attrs, "vault_id"), attrString(attrs, "id"))
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Item",
			fmt.Sprintf("reading Item (%s/%s): %s", attrString(attrs, "vault_id"), attrString(attrs, "id"), err),
		)
		return
	}

	resp.Diagnostics.Append(setCategoryStateFromItem(ctx, r.spec, attrs, item)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, diags := tfsdk.ObjectValue(r.attrTypes(), attrs)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}

func (r *categoryItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tfsdk.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	planAttrs := plan.Attributes()
	stateAttrs := state.Attributes()

	common, diags := categoryCommonFromAttrs(ctx, planAttrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	item := onepassword.Item{
		ID:       attrString(stateAttrs, "id"),
		VaultID:  common.VaultID.ValueString(),
		Title:    common.Title.ValueString(),
		Category: r.spec.Category,
	}
	if !common.Notes.IsNull() && !common.Notes.IsUnknown() {
		item.Notes = common.Notes.ValueString()
	}
	item.Tags = common.Tags
	item.Fields = typedFieldsFromAttrs(ctx, r.spec, planAttrs)
	item.Fields = append(item.Fields, sdkFieldsFromModel(ctx, common.CustomFields)...)
	item.Sections = categorySectionsToSend(r.spec, sdkSectionsFromModel(common.Sections), item.Fields)

	current, err := r.client.Items().Get(ctx, attrString(stateAttrs, "vault_id"), attrString(stateAttrs, "id"))
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
	item.Websites = sdkWebsitesFromModel(common.Websites)

	updated, err := r.client.Items().Put(ctx, item)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Item",
			fmt.Sprintf("updating Item (%s/%s): %s", common.VaultID.ValueString(), attrString(stateAttrs, "id"), err),
		)
		return
	}

	resp.Diagnostics.Append(setCategoryStateFromItem(ctx, r.spec, planAttrs, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, diags := tfsdk.ObjectValue(r.attrTypes(), planAttrs)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}

func (r *categoryItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var obj tfsdk.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &obj)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := obj.Attributes()
	vaultID := attrString(attrs, "vault_id")
	itemID := attrString(attrs, "id")

	var err error
	if strings.EqualFold(attrString(attrs, "delete_mode"), "archive") {
		err = r.client.Items().Archive(ctx, vaultID, itemID)
	} else {
		err = r.client.Items().Delete(ctx, vaultID, itemID)
	}
	if isNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Item",
			fmt.Sprintf("deleting Item (%s/%s): %s", vaultID, itemID, err),
		)
	}
}

func (r *categoryItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
