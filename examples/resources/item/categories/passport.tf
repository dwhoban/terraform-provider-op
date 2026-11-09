resource "op_item" "passport" {
  vault_id = op_vault.example.id
  title    = "Passport example"
  category = "Passport"
  fields = [
    { title = "number", type = "Text", value = "P1234567" },
    { title = "expiry date", type = "Date", value = "2031-08-01" }
  ]
}
