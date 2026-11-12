resource "op_item_passport" "passport" {
  vault_id        = op_vault.example.id
  title           = "Passport example"
  full_name       = "Wendy Appleseed"
  number          = "P1234567"
  expiry_date     = "2031-08-01"
  issuing_country = "USA"
}
