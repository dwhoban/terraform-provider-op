resource "op_item_driver_license" "driver_license" {
  vault_id    = op_vault.example.id
  title       = "DriverLicense example"
  full_name   = "Wendy Appleseed"
  number      = "D1234567"
  birth_date  = "1990-01-01"
  expiry_date = "03/2030"
}
