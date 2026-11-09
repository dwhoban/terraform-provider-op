resource "op_item" "rewards" {
  vault_id = op_vault.example.id
  title    = "Rewards example"
  category = "Rewards"
  fields = [
    { title = "member ID", type = "Text", value = "RW-1234" }
  ]
}
