resource "op_item_outdoor_license" "outdoor_license" {
  vault_id          = op_vault.example.id
  title             = "OutdoorLicense example"
  full_name         = "Wendy Appleseed"
  valid_from        = "2026-01-01"
  expires           = "2027-01-01"
  approved_wildlife = "deer"
  state             = "CA"
}
