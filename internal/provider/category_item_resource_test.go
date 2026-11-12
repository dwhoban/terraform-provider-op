// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestCategoryItemSpecs_valid(t *testing.T) {
	typeNames := map[string]bool{}
	for _, spec := range categoryItemSpecs {
		if spec.TypeName == "" {
			t.Fatalf("spec with empty TypeName (category %s)", spec.Category)
		}
		if typeNames[spec.TypeName] {
			t.Errorf("duplicate TypeName %q", spec.TypeName)
		}
		typeNames[spec.TypeName] = true

		sectionIDs := map[string]bool{}
		for _, s := range spec.DefaultSections {
			if s.ID == "" || s.Title == "" {
				t.Errorf("%s: default section %+v has empty ID or Title", spec.TypeName, s)
			}
			sectionIDs[s.ID] = true
		}

		fieldIDs := map[string]bool{}
		attrs := map[string]bool{}
		for _, f := range spec.Fields {
			if f.Attr == "" || f.FieldID == "" || f.Title == "" {
				t.Errorf("%s: field %+v has empty Attr, FieldID, or Title", spec.TypeName, f)
			}
			if fieldIDs[f.FieldID] {
				t.Errorf("%s: duplicate FieldID %q", spec.TypeName, f.FieldID)
			}
			fieldIDs[f.FieldID] = true
			if attrs[f.Attr] {
				t.Errorf("%s: duplicate Attr %q", spec.TypeName, f.Attr)
			}
			attrs[f.Attr] = true

			switch f.Type {
			case onepassword.ItemFieldTypeConcealed, onepassword.ItemFieldTypeCreditCardNumber, onepassword.ItemFieldTypeSSHKey:
				if !f.Sensitive {
					t.Errorf("%s: field %q of type %s must be Sensitive", spec.TypeName, f.Attr, f.Type)
				}
			}
			if f.SectionID != "" && !sectionIDs[f.SectionID] {
				t.Errorf("%s: field %q references unknown section %q", spec.TypeName, f.Attr, f.SectionID)
			}
		}

		for _, d := range spec.SSHDetails {
			if d.Attr == "" || d.FieldID == "" || d.DetailKey == "" {
				t.Errorf("%s: ssh detail %+v has empty Attr, FieldID, or DetailKey", spec.TypeName, d)
			}
			if !fieldIDs[d.FieldID] {
				t.Errorf("%s: ssh detail %q references unknown field %q", spec.TypeName, d.Attr, d.FieldID)
			}
		}
	}
	if len(categoryItemSpecs) != 21 {
		t.Errorf("expected 21 category specs, got %d", len(categoryItemSpecs))
	}
}

func TestCategoryItemSchema_valid(t *testing.T) {
	ctx := context.Background()
	for _, spec := range categoryItemSpecs {
		r := &categoryItemResource{spec: spec}
		resp := &tfresource.SchemaResponse{}
		r.Schema(ctx, tfresource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%s: schema error: %s", spec.TypeName, resp.Diagnostics.Errors())
			continue
		}

		for _, name := range []string{"id", "vault_id", "title", "notes", "tags", "custom_fields", "sections", "version", "delete_mode"} {
			if _, ok := resp.Schema.Attributes[name]; !ok {
				t.Errorf("%s: schema missing common attribute %q", spec.TypeName, name)
			}
		}
		for _, f := range spec.Fields {
			a, ok := resp.Schema.Attributes[f.Attr]
			if !ok {
				t.Errorf("%s: schema missing typed attribute %q", spec.TypeName, f.Attr)
				continue
			}
			_ = a
		}
		for _, d := range spec.SSHDetails {
			a, ok := resp.Schema.Attributes[d.Attr].(schema.StringAttribute)
			if !ok {
				t.Errorf("%s: ssh detail %q is not a StringAttribute", spec.TypeName, d.Attr)
				continue
			}
			if !a.Computed {
				t.Errorf("%s: ssh detail %q must be Computed", spec.TypeName, d.Attr)
			}
		}
		_, hasWebsites := resp.Schema.Attributes["websites"]
		if hasWebsites != spec.Websites {
			t.Errorf("%s: websites presence %t, want %t", spec.TypeName, hasWebsites, spec.Websites)
		}
		_, hasGenerate := resp.Schema.Attributes["generate_ssh_key"]
		if hasGenerate != spec.GenerateSSHKey {
			t.Errorf("%s: generate_ssh_key presence %t, want %t", spec.TypeName, hasGenerate, spec.GenerateSSHKey)
		}
	}
}

func TestProviderResourcesIncludeCategoryItems(t *testing.T) {
	ctx := context.Background()
	names := map[string]bool{}
	for _, factory := range New("test")().Resources(ctx) {
		resp := &tfresource.MetadataResponse{}
		factory().Metadata(ctx, tfresource.MetadataRequest{ProviderTypeName: "op"}, resp)
		names[resp.TypeName] = true
	}
	for _, spec := range categoryItemSpecs {
		want := "op_" + spec.TypeName
		if !names[want] {
			t.Errorf("Resources() missing %q", want)
		}
	}
}

// --- round-trip tests ---

func nullAttrs(attrTypes map[string]attr.Type) map[string]attr.Value {
	m := make(map[string]attr.Value, len(attrTypes))
	for k, t := range attrTypes {
		switch t := t.(type) {
		case basetypes.StringType:
			m[k] = tfsdk.StringNull()
		case basetypes.Int64Type:
			m[k] = tfsdk.Int64Null()
		case basetypes.ListType:
			m[k] = tfsdk.ListNull(t.ElementType())
		case basetypes.ObjectType:
			m[k] = tfsdk.ObjectNull(t.AttrTypes)
		default:
			m[k] = nil
		}
	}
	return m
}

func newCustomField(title, typ, value string) itemFieldModel {
	return itemFieldModel{
		ID:        tfsdk.StringNull(),
		Title:     tfsdk.StringValue(title),
		SectionID: tfsdk.StringNull(),
		FieldType: tfsdk.StringValue(typ),
		Value:     tfsdk.StringValue(value),
		Address:   tfsdk.ObjectNull(itemAddressObjectAttrs()),
		SSHKey:    tfsdk.ObjectNull(itemSSHKeyObjectAttrs()),
	}
}

func customFieldsList(t *testing.T, models []itemFieldModel) tfsdk.List {
	t.Helper()
	list, diags := tfsdk.ListValueFrom(context.Background(), tfsdk.ObjectType{AttrTypes: itemFieldObjectAttrs()}, models)
	if diags.HasError() {
		t.Fatalf("custom fields list: %s", diags.Errors())
	}
	return list
}

func mustAttrs(t *testing.T, r *categoryItemResource, attrs map[string]attr.Value) tfsdk.Object {
	t.Helper()
	obj, diags := tfsdk.ObjectValue(r.attrTypes(), attrs)
	if diags.HasError() {
		t.Fatalf("object value: %s", diags.Errors())
	}
	return obj
}

func assertStringAttr(t *testing.T, attrs map[string]attr.Value, key, want string) {
	t.Helper()
	got, ok := attrs[key].(basetypes.StringValue)
	if !ok || got.ValueString() != want {
		t.Errorf("attr %q = %v, want %q", key, attrs[key], want)
	}
}

func assertCustomFields(t *testing.T, attrs map[string]attr.Value, wantIDs ...string) {
	t.Helper()
	list, ok := attrs["custom_fields"].(basetypes.ListValue)
	if !ok {
		t.Fatalf("custom_fields is %T, want list", attrs["custom_fields"])
	}
	if len(wantIDs) == 0 {
		if !list.IsNull() && len(list.Elements()) > 0 {
			t.Errorf("custom_fields = %v, want empty", list)
		}
		return
	}
	var models []itemFieldModel
	if diags := list.ElementsAs(context.Background(), &models, false); diags.HasError() {
		t.Fatalf("decoding custom_fields: %s", diags.Errors())
	}
	got := map[string]string{}
	for _, m := range models {
		got[m.ID.ValueString()] = m.Value.ValueString()
	}
	if len(models) != len(wantIDs) {
		t.Errorf("custom_fields has %d entries (%v), want %d", len(models), got, len(wantIDs))
		return
	}
	for _, id := range wantIDs {
		if _, ok := got[id]; !ok {
			t.Errorf("custom_fields missing %q (has %v)", id, got)
		}
	}
}

// mustSpec returns the category spec with the given type name (without the
// provider prefix), failing the test if none matches.
func mustSpec(t *testing.T, typeName string) categorySpec {
	t.Helper()
	for _, s := range categoryItemSpecs {
		if s.TypeName == typeName {
			return s
		}
	}
	t.Fatalf("no category spec %q", typeName)
	return categorySpec{}
}

// TestCategoryItem_roundTrip drives the engine's pure conversion functions:
// attr map → SDK fields → fake server item → state fill, for a plain
// category (login), a multi-section category (credit_card), and ssh_key.
func TestCategoryItem_roundTrip(t *testing.T) {
	ctx := context.Background()

	t.Run("login", func(t *testing.T) {
		spec := mustSpec(t, "item_login")
		r := &categoryItemResource{spec: spec}

		attrs := nullAttrs(r.attrTypes())
		attrs["vault_id"] = tfsdk.StringValue("v1")
		attrs["title"] = tfsdk.StringValue("T")
		attrs["username"] = tfsdk.StringValue("wendy")
		attrs["custom_fields"] = customFieldsList(t, []itemFieldModel{
			newCustomField("custom note", "Text", "hello"),
		})

		fields := typedFieldsFromAttrs(ctx, spec, attrs)
		var custom []itemFieldModel
		customList, isList := attrs["custom_fields"].(basetypes.ListValue)
		if !isList {
			t.Fatalf("custom_fields is %T", attrs["custom_fields"])
		}
		diags := customList.ElementsAs(ctx, &custom, false)
		if diags.HasError() {
			t.Fatalf("decode: %s", diags.Errors())
		}
		fields = append(fields, sdkFieldsFromModel(ctx, custom)...)
		if len(fields) != 2 {
			t.Fatalf("got %d fields, want 2: %+v", len(fields), fields)
		}
		if fields[0].ID != "username" || fields[0].FieldType != onepassword.ItemFieldTypeText || fields[0].Value != "wendy" {
			t.Errorf("typed field mismatch: %+v", fields[0])
		}
		if fields[1].ID != "custom_note" || fields[1].Value != "hello" {
			t.Errorf("custom field mismatch: %+v", fields[1])
		}
		if got := categorySectionsToSend(spec, nil, fields); len(got) != 0 {
			t.Errorf("login sent %d sections, want 0", len(got))
		}

		item := onepassword.Item{
			ID: "i1", Title: "T", VaultID: "v1", Category: spec.Category,
			Fields: append(fields,
				onepassword.ItemField{ID: "notesPlain", Title: "notesPlain", FieldType: onepassword.ItemFieldTypeText, Value: "mynote"},
				onepassword.ItemField{ID: "bonus", Title: "bonus", FieldType: onepassword.ItemFieldTypeText, Value: "surprise"},
			),
			Notes: "mynote", Tags: []string{"a"}, Version: 3,
		}

		state := nullAttrs(r.attrTypes())
		if diags := setCategoryStateFromItem(ctx, spec, state, item); diags.HasError() {
			t.Fatalf("state fill: %s", diags.Errors())
		}
		mustAttrs(t, r, state)

		assertStringAttr(t, state, "id", "i1")
		assertStringAttr(t, state, "username", "wendy")
		assertStringAttr(t, state, "notes", "mynote")
		if !state["password"].IsNull() {
			t.Errorf("password = %v, want null (field absent on server)", state["password"])
		}
		assertCustomFields(t, state, "custom_note", "bonus")
	})

	t.Run("credit_card sections", func(t *testing.T) {
		spec := mustSpec(t, "item_credit_card")
		r := &categoryItemResource{spec: spec}

		attrs := nullAttrs(r.attrTypes())
		attrs["cvv"] = tfsdk.StringValue("123")                          // main section
		attrs["pin"] = tfsdk.StringValue("999")                          // details section
		attrs["website"] = tfsdk.StringValue("https://bank.example.com") // contactInfo section

		fields := typedFieldsFromAttrs(ctx, spec, attrs)
		if len(fields) != 3 {
			t.Fatalf("got %d fields, want 3", len(fields))
		}
		if fields[0].ID != "cvv" || fields[0].SectionID != nil {
			t.Errorf("cvv field mismatch (template has no section): %+v", fields[0])
		}
		if fields[1].ID != "website" || fields[1].SectionID == nil || *fields[1].SectionID != "contactInfo" {
			t.Errorf("website field mismatch: %+v", fields[1])
		}
		if fields[2].ID != "pin" || fields[2].SectionID == nil || *fields[2].SectionID != "details" {
			t.Errorf("pin field mismatch: %+v", fields[2])
		}

		sections := categorySectionsToSend(spec, nil, fields)
		if len(sections) != 2 {
			t.Fatalf("got %d sections, want 2: %+v", len(sections), sections)
		}
		byID := map[string]string{}
		for _, s := range sections {
			byID[s.ID] = s.Title
		}
		if byID["details"] != "Additional Details" || byID["contactInfo"] != "Contact Information" {
			t.Errorf("sections mismatch: %+v", byID)
		}

		// A custom section duplicating a default section's title must not
		// produce a duplicate default section.
		dup := categorySectionsToSend(spec, []onepassword.ItemSection{{Title: "Additional Details"}}, fields)
		if len(dup) != 2 {
			t.Errorf("got %d sections with duplicate title, want 2: %+v", len(dup), dup)
		}

		item := onepassword.Item{
			ID: "i2", Title: "T", VaultID: "v1", Category: spec.Category,
			Fields:   fields,
			Sections: []onepassword.ItemSection{{ID: "details", Title: "Additional Details"}, {ID: "contactInfo", Title: "Contact Information"}},
		}
		state := nullAttrs(r.attrTypes())
		if diags := setCategoryStateFromItem(ctx, spec, state, item); diags.HasError() {
			t.Fatalf("state fill: %s", diags.Errors())
		}
		mustAttrs(t, r, state)

		assertStringAttr(t, state, "cvv", "123")
		assertStringAttr(t, state, "pin", "999")
		assertStringAttr(t, state, "website", "https://bank.example.com")
		assertCustomFields(t, state)
	})

	t.Run("ssh_key details", func(t *testing.T) {
		spec := mustSpec(t, "item_ssh_key")
		r := &categoryItemResource{spec: spec}

		details := onepassword.NewItemFieldDetailsTypeVariantSSHKey(&onepassword.SSHKeyAttributes{
			PublicKey:   "ssh-ed25519 AAAA",
			Fingerprint: "SHA256:xyz",
			KeyType:     "Ed25519",
		})
		item := onepassword.Item{
			ID: "i3", Title: "T", VaultID: "v1", Category: spec.Category,
			Fields: []onepassword.ItemField{{
				ID: "private_key", Title: "private key",
				FieldType: onepassword.ItemFieldTypeSSHKey, Value: "PEM", Details: &details,
			}},
		}

		state := nullAttrs(r.attrTypes())
		if diags := setCategoryStateFromItem(ctx, spec, state, item); diags.HasError() {
			t.Fatalf("state fill: %s", diags.Errors())
		}
		mustAttrs(t, r, state)

		assertStringAttr(t, state, "private_key", "PEM")
		assertStringAttr(t, state, "public_key", "ssh-ed25519 AAAA")
		assertStringAttr(t, state, "fingerprint", "SHA256:xyz")
		assertStringAttr(t, state, "key_type", "Ed25519")
		assertCustomFields(t, state)
	})

	t.Run("identity address", func(t *testing.T) {
		spec := mustSpec(t, "item_identity")
		r := &categoryItemResource{spec: spec}

		attrs := nullAttrs(r.attrTypes())
		addr, diags := tfsdk.ObjectValue(itemAddressObjectAttrs(), map[string]attr.Value{
			"street": tfsdk.StringValue("123 Main St"), "city": tfsdk.StringValue("Anytown"),
			"state": tfsdk.StringValue("CA"), "zip": tfsdk.StringNull(), "country": tfsdk.StringValue("USA"),
		})
		if diags.HasError() {
			t.Fatalf("address: %s", diags.Errors())
		}
		attrs["address"] = addr

		fields := typedFieldsFromAttrs(ctx, spec, attrs)
		if len(fields) != 1 {
			t.Fatalf("got %d fields, want 1", len(fields))
		}
		if fields[0].ID != "address" || fields[0].FieldType != onepassword.ItemFieldTypeAddress || fields[0].Value != "" {
			t.Errorf("address field mismatch: %+v", fields[0])
		}
		if fields[0].Details == nil || fields[0].Details.Address() == nil || fields[0].Details.Address().City != "Anytown" {
			t.Errorf("address details mismatch: %+v", fields[0].Details)
		}

		sections := categorySectionsToSend(spec, nil, fields)
		if len(sections) != 1 || sections[0].ID != "address" {
			t.Errorf("sections mismatch: %+v", sections)
		}

		// Echo the address back through the server item.
		item := onepassword.Item{ID: "i4", Title: "T", VaultID: "v1", Category: spec.Category, Fields: fields}
		state := nullAttrs(r.attrTypes())
		if diags := setCategoryStateFromItem(ctx, spec, state, item); diags.HasError() {
			t.Fatalf("state fill: %s", diags.Errors())
		}
		mustAttrs(t, r, state)

		got, ok := state["address"].(basetypes.ObjectValue)
		if !ok || got.IsNull() {
			t.Fatalf("address attr = %v, want object", state["address"])
		}
		city, _ := got.Attributes()["city"].(basetypes.StringValue)
		if city.ValueString() != "Anytown" {
			t.Errorf("address.city = %v, want Anytown", got)
		}
	})

	t.Run("server-side ID case normalization", func(t *testing.T) {
		// A server ID differing only in case still fills the typed attr and
		// is excluded from custom_fields.
		spec := mustSpec(t, "item_ssh_key")
		r := &categoryItemResource{spec: spec}

		item := onepassword.Item{
			ID: "i5", Title: "T", VaultID: "v1", Category: spec.Category,
			Fields: []onepassword.ItemField{
				{ID: "PRIVATE_KEY", Title: "private key", FieldType: onepassword.ItemFieldTypeSSHKey, Value: "PEM"},
			},
		}
		state := nullAttrs(r.attrTypes())
		if diags := setCategoryStateFromItem(ctx, spec, state, item); diags.HasError() {
			t.Fatalf("state fill: %s", diags.Errors())
		}
		mustAttrs(t, r, state)

		assertStringAttr(t, state, "private_key", "PEM")
		assertCustomFields(t, state)
	})

	t.Run("server-side ID structural normalization", func(t *testing.T) {
		// A server ID that differs structurally from the spec's FieldID
		// (e.g. "full_name" vs "fullname") is still matched via its title,
		// but ID-based custom_fields exclusion cannot see the equivalence,
		// so the field also surfaces in custom_fields. If observed live,
		// fix the spec's FieldID from the server value.
		spec := mustSpec(t, "item_driver_license")
		r := &categoryItemResource{spec: spec}

		item := onepassword.Item{
			ID: "i6", Title: "T", VaultID: "v1", Category: spec.Category,
			Fields: []onepassword.ItemField{
				{ID: "full_name", Title: "full name", FieldType: onepassword.ItemFieldTypeText, Value: "Wendy"},
				{ID: "number", Title: "number", FieldType: onepassword.ItemFieldTypeText, Value: "D1"},
			},
		}
		state := nullAttrs(r.attrTypes())
		if diags := setCategoryStateFromItem(ctx, spec, state, item); diags.HasError() {
			t.Fatalf("state fill: %s", diags.Errors())
		}
		mustAttrs(t, r, state)

		assertStringAttr(t, state, "full_name", "Wendy")
		assertStringAttr(t, state, "number", "D1")
		assertCustomFields(t, state, "full_name")
	})
}

// --- acceptance tests ---

type categoryItemFixture struct {
	typeName    string
	createBody  string // resource body with one typed field + one custom field + tag
	updateBody  string // body with the typed value (or custom value) changed
	checkPath   string // typed attribute asserted; "" asserts custom_fields only
	createValue string
	updateValue string
}

// categoryItemFixtures covers every category spec that 1Password can
// create; fixture values reuse the
// examples/resources/item/categories/*.tf values where types match.
var categoryItemFixtures = []categoryItemFixture{
	{typeName: "op_item_login", checkPath: "username", createValue: "wendy_appleseed", updateValue: "wendy_updated",
		createBody: "  username = \"wendy_appleseed\"", updateBody: "  username = \"wendy_updated\""},
	{typeName: "op_item_credit_card", checkPath: "card_number", createValue: "4111111111111111", updateValue: "4222222222222222",
		createBody: "  card_type = \"visa\"\n  card_number = \"4111111111111111\"\n  expiry = \"03/2028\"\n  cvv = \"123\"", updateBody: "  card_type = \"visa\"\n  card_number = \"4222222222222222\"\n  expiry = \"03/2028\"\n  cvv = \"123\""},
	{typeName: "op_item_secure_note",
		createBody: "", updateBody: ""},
	{typeName: "op_item_identity", checkPath: "first_name", createValue: "Wendy", updateValue: "Wendeline",
		createBody: "  first_name = \"Wendy\"\n  last_name = \"Appleseed\"", updateBody: "  first_name = \"Wendeline\"\n  last_name = \"Appleseed\""},
	{typeName: "op_item_password", checkPath: "password", createValue: "correct-horse-battery-staple", updateValue: "updated-horse-battery-staple",
		createBody: "  password = \"correct-horse-battery-staple\"", updateBody: "  password = \"updated-horse-battery-staple\""},
	{typeName: "op_item_software_license", checkPath: "license_key", createValue: "ABCD-1234-EFGH-5678", updateValue: "EFGH-5678-ABCD-1234",
		createBody: "  license_key = \"ABCD-1234-EFGH-5678\"", updateBody: "  license_key = \"EFGH-5678-ABCD-1234\""},
	{typeName: "op_item_bank_account", checkPath: "account_number", createValue: "1234567890", updateValue: "0987654321",
		createBody: "  bank_name = \"First National\"\n  account_type = \"checking\"\n  account_number = \"1234567890\"", updateBody: "  bank_name = \"First National\"\n  account_type = \"checking\"\n  account_number = \"0987654321\""},
	{typeName: "op_item_database", checkPath: "type", createValue: "mysql", updateValue: "postgresql",
		createBody: "  type = \"mysql\"\n  hostname = \"db.example.com\"\n  database = \"appdb\"", updateBody: "  type = \"postgresql\"\n  hostname = \"db.example.com\"\n  database = \"appdb\""},
	{typeName: "op_item_driver_license", checkPath: "number", createValue: "D1234567", updateValue: "D7654321",
		createBody: "  number = \"D1234567\"\n  expiry_date = \"03/2030\"", updateBody: "  number = \"D7654321\"\n  expiry_date = \"03/2030\""},
	{typeName: "op_item_outdoor_license", checkPath: "full_name", createValue: "Wendy Appleseed", updateValue: "W. Appleseed",
		createBody: "  full_name = \"Wendy Appleseed\"", updateBody: "  full_name = \"W. Appleseed\""},
	{typeName: "op_item_membership", checkPath: "member_id", createValue: "M-12345", updateValue: "M-67890",
		createBody: "  member_id = \"M-12345\"", updateBody: "  member_id = \"M-67890\""},
	{typeName: "op_item_passport", checkPath: "number", createValue: "P1234567", updateValue: "P7654321",
		createBody: "  number = \"P1234567\"\n  expiry_date = \"2031-08-01\"", updateBody: "  number = \"P7654321\"\n  expiry_date = \"2031-08-01\""},
	{typeName: "op_item_rewards", checkPath: "member_id", createValue: "RW-1234", updateValue: "RW-4321",
		createBody: "  company_name = \"Acme Rewards\"\n  member_id = \"RW-1234\"", updateBody: "  company_name = \"Acme Rewards\"\n  member_id = \"RW-4321\""},
	{typeName: "op_item_social_security_number", checkPath: "number", createValue: "123-45-6789", updateValue: "987-65-4321",
		createBody: "  name = \"Wendy Appleseed\"\n  number = \"123-45-6789\"", updateBody: "  name = \"Wendy Appleseed\"\n  number = \"987-65-4321\""},
	{typeName: "op_item_server", checkPath: "username", createValue: "root", updateValue: "admin",
		createBody: "  url = \"https://server.example.com\"\n  username = \"root\"", updateBody: "  url = \"https://server.example.com\"\n  username = \"admin\""},
	{typeName: "op_item_email", checkPath: "username", createValue: "wendy", updateValue: "wappleseed",
		createBody: "  username = \"wendy\"\n  server = \"pop.example.com\"", updateBody: "  username = \"wappleseed\"\n  server = \"pop.example.com\""},
	{typeName: "op_item_api_credentials", checkPath: "username", createValue: "svc-account", updateValue: "svc-account-2",
		createBody: "  username = \"svc-account\"\n  credential = \"sk-live-xxxx\"\n  type = \"bearer\"", updateBody: "  username = \"svc-account-2\"\n  credential = \"sk-live-xxxx\"\n  type = \"bearer\""},
	{typeName: "op_item_medical_record", checkPath: "patient", createValue: "Wendy Appleseed", updateValue: "W. Appleseed",
		createBody: "  date = \"2026-01-15\"\n  patient = \"Wendy Appleseed\"", updateBody: "  date = \"2026-01-15\"\n  patient = \"W. Appleseed\""},
	{typeName: "op_item_ssh_key",
		createBody: "  generate_ssh_key = \"ed25519\"", updateBody: "  generate_ssh_key = \"ed25519\""},
	{typeName: "op_item_crypto_wallet", checkPath: "wallet_address", createValue: "0x123abc", updateValue: "0x456def",
		createBody: "  wallet_address = \"0x123abc\"", updateBody: "  wallet_address = \"0x456def\""},
}

const (
	categoryCustomFieldCreate = `{ title = "custom note", type = "Text", value = "hello" }`
	categoryCustomFieldUpdate = `{ title = "custom note", type = "Text", value = "hello again" }`
)

func testAccCategoryItemConfig(typeName, vaultID, body, customField string) string {
	return fmt.Sprintf(`
resource %q "test" {
  vault_id = %[2]q
  title    = "tf-acc category item"
  tags     = ["terraform"]

%[3]s

  custom_fields = [%[4]s]
}
`, typeName, vaultID, body, customField)
}

func TestAccCategoryItem_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	vaultID := testAccVaultID(t)

	for _, fx := range categoryItemFixtures {
		t.Run(fx.typeName, func(t *testing.T) {
			resourceAddress := fx.typeName + ".test"

			checks := []statecheck.StateCheck{
				statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("id"), knownvalue.NotNull()),
				statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("custom_fields"), knownvalue.ListSizeExact(1)),
			}
			updateChecks := []statecheck.StateCheck{
				statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("id"), knownvalue.NotNull()),
				statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("custom_fields"), knownvalue.ListSizeExact(1)),
			}
			if fx.checkPath != "" {
				checks = append(checks, statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New(fx.checkPath), knownvalue.StringExact(fx.createValue)))
				updateChecks = append(updateChecks, statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New(fx.checkPath), knownvalue.StringExact(fx.updateValue)))
			}

			importIgnore := []string{"delete_mode"}
			if fx.typeName == "op_item_ssh_key" {
				// generate_ssh_key is config-only; imported state cannot
				// reproduce it.
				importIgnore = append(importIgnore, "generate_ssh_key")
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:            testAccCategoryItemConfig(fx.typeName, vaultID, fx.createBody, categoryCustomFieldCreate),
						ConfigStateChecks: checks,
					},
					{
						Config:            testAccCategoryItemConfig(fx.typeName, vaultID, fx.updateBody, categoryCustomFieldUpdate),
						ConfigStateChecks: updateChecks,
					},
					{
						ResourceName:            resourceAddress,
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateIdFunc:       importItemStateID(resourceAddress),
						ImportStateVerifyIgnore: importIgnore,
					},
				},
			})
		})
	}
}

func TestAccItemSSHKey_generate(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	vaultID := testAccVaultID(t)
	resourceName := "op_item_ssh_key.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "op_item_ssh_key" "test" {
  vault_id         = %[1]q
  title            = "tf-acc generated ssh key"
  generate_ssh_key = "ed25519"
}
`, vaultID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("key_type"), knownvalue.StringExact("Ed25519")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("public_key"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccCategoryItem_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	vaultID := testAccVaultID(t)
	resourceName := "op_item_ssh_key.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "op_item_ssh_key" "test" {
  vault_id         = %[1]q
  title            = %[2]q
  generate_ssh_key = "ed25519"
}
`, vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
				},
			},
			{
				Config: fmt.Sprintf(`
resource "op_item_ssh_key" "test" {
  vault_id         = %[1]q
  title            = %[2]q
  generate_ssh_key = "ed25519"
}
`, vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					deleteItemOutOfBand(t, vaultID, rName, resourceName),
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
