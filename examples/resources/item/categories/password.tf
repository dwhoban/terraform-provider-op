resource "op_item" "password" {
  vault_id = op_vault.example.id
  title    = "Password example"
  category = "Password"
  fields = [
    { id = "password", title = "password", type = "Concealed", value = "correct-horse-battery-staple" }
  ]

  websites = [{ url = "example.com", label = "website", autofill_behavior = "AnywhereOnWebsite" }]
}
