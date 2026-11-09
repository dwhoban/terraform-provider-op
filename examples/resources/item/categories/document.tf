resource "op_item" "document" {
  vault_id = op_vault.example.id
  title    = "Document example"
  category = "Document"
}
