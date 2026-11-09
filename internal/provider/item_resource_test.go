// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccItem_basicUpdateImport(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "op_item.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfig_basic(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("category"), knownvalue.StringExact("Login")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccItemConfig_fields(rName, "updateduser"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_mode"},
			},
		},
	})
}

func TestAccItem_archiveMode(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "op_item.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfig_archive(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("delete_mode"), knownvalue.StringExact("archive")),
				},
			},
		},
	})
}

func testAccItemConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = "tf-test-vault"
  title    = %[1]q
  category = "Login"
}
`, rName)
}

func testAccItemConfig_fields(rName, username string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = "tf-test-vault"
  title    = %[1]q
  category = "Login"
  tags     = ["terraform", "test"]
  fields = [
    { title = "username", type = "Text", value = %[2]q },
    { title = "password", type = "Concealed", value = "correct-horse-battery-staple" },
  ]
}
`, rName, username)
}

func testAccItemConfig_archive(rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id    = "tf-test-vault"
  title       = %[1]q
  category    = "SecureNote"
  delete_mode = "archive"
}
`, rName)
}
