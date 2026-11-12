resource "op_item_password" "password" {
  vault_id = op_vault.example.id
  title    = "Password example"
  password = "correct-horse-battery-staple"

  websites = [{ url = "example.com", label = "website", autofill_behavior = "AnywhereOnWebsite" }]
}
