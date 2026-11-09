resource "op_item" "identity" {
  vault_id = op_vault.example.id
  title    = "Identity example"
  category = "Identity"
  fields = [
    { title = "first name", type = "Text", value = "Wendy" },
    { title = "last name", type = "Text", value = "Appleseed" },
    { title = "home", type = "Address", address = { street = "123 Main St", city = "Anytown", state = "CA", zip = "12345", country = "USA" } }
  ]
}
