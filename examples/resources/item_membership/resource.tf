resource "op_item_membership" "membership" {
  vault_id    = op_vault.example.id
  title       = "Membership example"
  group       = "Costco"
  website     = "https://www.costco.com"
  member_id   = "M-12345"
  member_name = "Wendy Appleseed"
}
