resource "op_item" "person" {
  vault_id = op_vault.example.id
  title    = "Person example"
  category = "Person"
  fields = [
    { title = "first name", type = "Text", value = "Wendy" },
    { title = "last name", type = "Text", value = "Appleseed" }
  ]
}
