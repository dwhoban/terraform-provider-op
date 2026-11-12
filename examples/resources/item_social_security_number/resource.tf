resource "op_item_social_security_number" "social_security_number" {
  vault_id = op_vault.example.id
  title    = "SocialSecurityNumber example"
  name     = "Wendy Appleseed"
  number   = "123-45-6789"
}
