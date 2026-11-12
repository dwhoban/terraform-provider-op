resource "op_item_rewards" "rewards" {
  vault_id     = op_vault.example.id
  title        = "Rewards example"
  company_name = "Acme Rewards"
  member_id    = "RW-1234"
  member_name  = "Wendy Appleseed"
  pin          = "1234"
}
