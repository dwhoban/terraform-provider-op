resource "op_item" "router" {
  vault_id = op_vault.example.id
  title    = "Router example"
  category = "Router"
  fields = [
    { title = "password", type = "Concealed", value = "admin" },
    { title = "base station name", type = "Text", value = "Living Room" }
  ]
}
