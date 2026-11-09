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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccItem_basicUpdateImport(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	vaultID := testAccVaultRef(t)
	resourceName := "op_item.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfig_basic(vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("category"), knownvalue.StringExact("Login")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccItemConfig_fields(vaultID, rName, "updateduser"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       importItemStateID(resourceName),
				ImportStateVerifyIgnore: []string{"delete_mode"},
			},
		},
	})
}

func importItemStateID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}
		return rs.Primary.Attributes["vault_id"] + "/" + rs.Primary.ID, nil
	}
}

func TestAccItem_archiveMode(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	vaultID := testAccVaultRef(t)
	resourceName := "op_item.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfig_archive(vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("delete_mode"), knownvalue.StringExact("archive")),
				},
			},
		},
	})
}

func testAccVaultRef(t *testing.T) string {
	t.Helper()
	return testAccVaultID(t)
}

func testAccItemConfig_basic(vaultID, rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = %[1]q
  title    = %[2]q
  category = "Login"
}
`, vaultID, rName)
}

func testAccItemConfig_fields(vaultID, rName, username string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = %[1]q
  title    = %[2]q
  category = "Login"
  tags     = ["terraform", "test"]
  fields = [
    { title = "username", type = "Text", value = %[3]q },
    { title = "password", type = "Concealed", value = "correct-horse-battery-staple" },
  ]
}
`, vaultID, rName, username)
}

func testAccItemConfig_archive(vaultID, rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id    = %[1]q
  title       = %[2]q
  category    = "SecureNote"
  delete_mode = "archive"
}
`, vaultID, rName)
}
