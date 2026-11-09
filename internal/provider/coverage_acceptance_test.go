// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func testAccClient(t *testing.T) *onepassword.Client {
	t.Helper()
	testAccPreCheck(t)
	client, err := onepassword.NewClient(context.Background(), append([]onepassword.ClientOption{
		onepassword.WithIntegrationInfo("terraform-provider-op-tests", "test"),
	}, authOptions()...)...)
	if err != nil {
		t.Fatalf("client: %s", err)
	}
	return client
}

func authOptions() []onepassword.ClientOption {
	if token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"); token != "" {
		return []onepassword.ClientOption{onepassword.WithServiceAccountToken(token)}
	}
	return []onepassword.ClientOption{onepassword.WithDesktopAppIntegration(os.Getenv("OP_DESKTOP_ACCOUNT_NAME"))}
}

func testAccVaultID(t *testing.T) string {
	t.Helper()
	if id := os.Getenv("OP_ACC_VAULT_ID"); id != "" {
		return id
	}
	t.Skip("OP_ACC_VAULT_ID not set; set it to a disposable vault ID for item tests")
	return ""
}

func TestAccItem_disappears(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	vaultID := testAccVaultID(t)
	resourceName := "op_item.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfigVault(vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("title"), knownvalue.StringExact(rName)),
				},
			},
			{
				Config: testAccItemConfigVault(vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					deleteItemOutOfBand(t, vaultID, rName),
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

type deleteItemCheck struct {
	t     *testing.T
	vault string
	title string
}

func (d deleteItemCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	itemID := ""
	for _, r := range req.State.Values.RootModule.Resources {
		if r.Address == "op_item.test" {
			if v, ok := r.AttributeValues["id"].(string); ok {
				itemID = v
			}
			break
		}
	}
	if itemID == "" {
		resp.Error = fmt.Errorf("op_item.test not found in state")
		return
	}
	if err := testAccClient(d.t).Items().Delete(ctx, d.vault, itemID); err != nil {
		resp.Error = fmt.Errorf("deleting item out-of-band: %w", err)
	}
}

func deleteItemOutOfBand(t *testing.T, vaultID, title string) statecheck.StateCheck {
	return deleteItemCheck{t: t, vault: vaultID, title: title}
}

func itemIDForTitle(t *testing.T, vaultID, title string) string {
	t.Helper()
	overviews, err := testAccClient(t).Items().List(context.Background(), vaultID)
	if err != nil {
		t.Fatalf("listing items: %s", err)
	}
	for _, o := range overviews {
		if o.Title == title {
			return o.ID
		}
	}
	t.Fatalf("item %q not found in vault %s", title, vaultID)
	return ""
}

func testAccItemConfigVault(vaultID, rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = %[1]q
  title    = %[2]q
  category = "SecureNote"
}
`, vaultID, rName)
}

func TestAccVaultsDataSource_lists(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "op_vaults" "all" {}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.op_vaults.all", tfjsonpath.New("vaults"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccVaultsDataSource_findsCreatedVault(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVaultsConfig_withVault(rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.op_vaults.all", tfjsonpath.New("vaults"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccVaultsConfig_withVault(rName string) string {
	return fmt.Sprintf(`
resource "op_vault" "test" {
  title = "tf-acctest-%[1]s"
}

data "op_vaults" "all" {}
`, rName)
}

func TestAccItemsDataSource_lists(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	vaultID := testAccVaultID(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemsDataSourceConfig(vaultID, rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.op_items.in_vault", tfjsonpath.New("items"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccItemsDataSourceConfig(vaultID, rName string) string {
	return fmt.Sprintf(`
resource "op_item" "test" {
  vault_id = %[1]q
  title    = %[2]q
  category = "SecureNote"
}

data "op_items" "in_vault" {
  vault_id = %[1]q
}
`, vaultID, rName)
}

func TestAccGroupDataSource(t *testing.T) {
	if os.Getenv("OP_ACC_GROUP_ID") == "" {
		t.Skip("OP_ACC_GROUP_ID not set; skipping group data source acceptance test")
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`data "op_group" "test" { id = %q }`, os.Getenv("OP_ACC_GROUP_ID")),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.op_group.test", tfjsonpath.New("id"), knownvalue.StringExact(os.Getenv("OP_ACC_GROUP_ID"))),
				},
			},
		},
	})
}

func TestAccEnvironmentVariablesDataSource(t *testing.T) {
	if os.Getenv("OP_ACC_ENV_ID") == "" {
		t.Skip("OP_ACC_ENV_ID not set; skipping environment data source acceptance test")
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`data "op_environment_variables" "test" { environment_id = %q }`, os.Getenv("OP_ACC_ENV_ID")),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.op_environment_variables.test", tfjsonpath.New("variables"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccGeneratePasswordAction_validation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_14_0)},
		Steps: []resource.TestStep{
			{
				Config: `
resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.op_generate_password.bad]
    }
  }
}

action "op_generate_password" "bad" {
  config {
    mode        = "warp-driver"
    vault_id    = "irrelevant"
    item_id     = "irrelevant"
    field_title = "password"
  }
}
`,
				ExpectError: regexp.MustCompile(`(?s)Invalid password recipe|warp-driver`),
			},
		},
	})
}
