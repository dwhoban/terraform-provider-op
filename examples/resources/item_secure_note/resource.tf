resource "op_item_secure_note" "secure_note" {
  vault_id = op_vault.example.id
  title    = "SecureNote example"
  notes    = "Recovery codes: ..."
}
