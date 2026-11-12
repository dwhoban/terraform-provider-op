// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSecretRef_ephemeral(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	vaultID := testAccVaultID(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretRefItemConfig(vaultID, rName),
			},
			{
				Config: testAccSecretRefConfig(vaultID, rName),
			},
		},
	})
}

func testAccSecretRefItemConfig(vaultID, rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = %[1]q
  title    = %[2]q
  category = "Password"
  fields = [
    { id = "password", title = "password", type = "Concealed", value = "hunter2" },
  ]
}
`, vaultID, rName)
}

func testAccSecretRefConfig(vaultID, rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = %[1]q
  title    = %[2]q
  category = "Password"
  fields = [
    { id = "password", title = "password", type = "Concealed", value = "hunter2" },
  ]
}

ephemeral "op_secret_ref" "ref" {
  reference = "op://%[1]s/%[2]s/password"
}
`, vaultID, rName)
}
