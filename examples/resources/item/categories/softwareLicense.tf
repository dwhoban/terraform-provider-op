resource "op_item" "softwareLicense" {
  vault_id = op_vault.example.id
  title    = "SoftwareLicense example"
  category = "SoftwareLicense"
  fields = [
    { title = "license key", type = "Concealed", value = "ABCD-1234-EFGH-5678" }
  ]
}
