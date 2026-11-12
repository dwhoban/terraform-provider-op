resource "op_item_software_license" "software_license" {
  vault_id         = op_vault.example.id
  title            = "SoftwareLicense example"
  license_key      = "ABCD-1234-EFGH-5678"
  version          = "3.2.1"
  licensed_to      = "Wendy Appleseed"
  registered_email = "wendy@example.com"
}
