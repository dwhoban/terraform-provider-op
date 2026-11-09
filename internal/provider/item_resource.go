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
	ID        tfsdk.String           `tfsdk:"id"`
	Title     tfsdk.String           `tfsdk:"title"`
	SectionID tfsdk.String           `tfsdk:"section_id"`
	FieldType tfsdk.String           `tfsdk:"type"`
	Value     tfsdk.String           `tfsdk:"value"`
	Address   *itemFieldAddressModel `tfsdk:"address"`
	SSHKey    *itemFieldSSHKeyModel  `tfsdk:"ssh_key"`
}

type itemFieldAddressModel struct {
	Street  tfsdk.String `tfsdk:"street"`
	City    tfsdk.String `tfsdk:"city"`
	State   tfsdk.String `tfsdk:"state"`
	Zip     tfsdk.String `tfsdk:"zip"`
	Country tfsdk.String `tfsdk:"country"`
}

type itemFieldSSHKeyModel struct {
	PublicKey   tfsdk.String `tfsdk:"public_key"`
	Fingerprint tfsdk.String `tfsdk:"fingerprint"`
	KeyType     tfsdk.String `tfsdk:"key_type"`
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
	Websites   []itemWebsiteModel `tfsdk:"websites"`
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
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Field ID. Set to a built-in field ID (e.g. `username`, `password`) where applicable; otherwise computed.",
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
	params.Websites = sdkWebsitesFromModel(data.Websites)

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

func sdkFieldsFromModel(models []itemFieldModel) []onepassword.ItemField {
	fields := make([]onepassword.ItemField, 0, len(models))
	for _, f := range models {
		field := onepassword.ItemField{
			ID:        f.ID.ValueString(),
			Title:     f.Title.ValueString(),
			FieldType: onepassword.ItemFieldType(f.FieldType.ValueString()),
			Value:     f.Value.ValueString(),
		}
		if !f.SectionID.IsNull() {
			sectionID := f.SectionID.ValueString()
			field.SectionID = &sectionID
		}
		if f.Address != nil {
			details := onepassword.NewItemFieldDetailsTypeVariantAddress(&onepassword.AddressFieldDetails{
				Street:  f.Address.Street.ValueString(),
				City:    f.Address.City.ValueString(),
				State:   f.Address.State.ValueString(),
				Zip:     f.Address.Zip.ValueString(),
				Country: f.Address.Country.ValueString(),
			})
			field.Details = &details
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
		if f.Details != nil {
			if address := f.Details.Address(); address != nil {
				field.Address = &itemFieldAddressModel{
					Street:  tfsdk.StringValue(address.Street),
					City:    tfsdk.StringValue(address.City),
					State:   tfsdk.StringValue(address.State),
					Zip:     tfsdk.StringValue(address.Zip),
					Country: tfsdk.StringValue(address.Country),
				}
			}
			if sshKey := f.Details.SSHKey(); sshKey != nil {
				field.SSHKey = &itemFieldSSHKeyModel{
					PublicKey:   tfsdk.StringValue(sshKey.PublicKey),
					Fingerprint: tfsdk.StringValue(sshKey.Fingerprint),
					KeyType:     tfsdk.StringValue(sshKey.KeyType),
				}
			}
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
	data.Websites = modelWebsitesFromSDK(item.Websites)
	return diags
}

func modelWebsitesFromSDK(websites []onepassword.Website) []itemWebsiteModel {
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
