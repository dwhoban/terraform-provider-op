resource "op_item" "membership" {
  vault_id = op_vault.example.id
  title    = "Membership example"
  category = "Membership"
  fields = [
    { title = "member number", type = "Text", value = "M-12345" }
  ]
}
