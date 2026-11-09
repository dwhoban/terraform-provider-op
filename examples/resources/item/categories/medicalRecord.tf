resource "op_item" "medicalRecord" {
  vault_id = op_vault.example.id
  title    = "MedicalRecord example"
  category = "MedicalRecord"
  fields = [
    { title = "record date", type = "Date", value = "2026-01-15" }
  ]
}
