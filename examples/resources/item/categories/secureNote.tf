resource "op_item" "secureNote" {
  vault_id = op_vault.example.id
  title    = "SecureNote example"
  category = "SecureNote"
  fields = [
    { title = "notes", type = "Text", value = "Recovery codes: ..." }
  ]
}
