// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The op_item data source's fields schema must agree with the field values it
// stores: item fields carry address/ssh_key objects, so the schema has to
// declare them too (otherwise every read fails with a Value Conversion Error).
func TestItemDataSourceSchemaMatchesFieldValues(t *testing.T) {
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	NewItemDataSource().Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema error: %s", schemaResp.Diagnostics.Errors())
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	fieldList, diags := itemFieldListFromSDK([]onepassword.ItemField{{
		ID: "password", Title: "password", FieldType: onepassword.ItemFieldTypeConcealed, Value: "s3cret",
	}})
	if diags.HasError() {
		t.Fatalf("field list: %s", diags.Errors())
	}
	model := itemDataSourceModel{
		ID:       types.StringValue("item"),
		VaultID:  types.StringValue("vault"),
		Title:    types.StringValue("title"),
		Category: types.StringValue("Login"),
		Notes:    types.StringValue(""),
		Tags:     types.ListValueMust(types.StringType, nil),
		Fields:   fieldList,
		Sections: []itemSectionModel{},
		Version:  types.Int64Value(1),
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("setting state from itemDataSourceModel: %s", diags.Errors())
	}
}

// op_vaults must return real vault titles, not the SDK's "[Encrypted]"
// placeholder (returned when List is called without decryptDetails).
func TestAccVaultsDataSource_returnsDecryptedTitles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "op_vaults" "all" {}`,
				Check: func(s *terraform.State) error {
					attrs := s.RootModule().Resources["data.op_vaults.all"].Primary.Attributes
					n, err := strconv.Atoi(attrs["vaults.#"])
					if err != nil || n == 0 {
						return fmt.Errorf("expected at least one vault, got %q", attrs["vaults.#"])
					}
					for i := range n {
						if title := attrs[fmt.Sprintf("vaults.%d.title", i)]; title == "" || title == "[Encrypted]" {
							return fmt.Errorf("vaults.%d.title = %q, want a decrypted title", i, title)
						}
					}
					return nil
				},
			},
		},
	})
}
