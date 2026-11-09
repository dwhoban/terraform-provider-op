resource "op_item" "driverLicense" {
  vault_id = op_vault.example.id
  title    = "DriverLicense example"
  category = "DriverLicense"
  fields = [
    { title = "number", type = "Text", value = "D1234567" },
    { title = "expiry date", type = "Date", value = "2030-03-15" }
  ]
}
