resource "op_item" "outdoorLicense" {
  vault_id = op_vault.example.id
  title    = "OutdoorLicense example"
  category = "OutdoorLicense"
  fields = [
    { title = "license number", type = "Text", value = "OL-999" }
  ]
}
