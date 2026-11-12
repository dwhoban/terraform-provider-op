resource "op_item_medical_record" "medical_record" {
  vault_id   = op_vault.example.id
  title      = "MedicalRecord example"
  date       = "2026-01-15"
  patient    = "Wendy Appleseed"
  location   = "Anytown Clinic"
  medication = "ibuprofen"
  dosage     = "200mg"
}
