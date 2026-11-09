resource "op_item" "socialSecurityNumber" {
  vault_id = op_vault.example.id
  title    = "SocialSecurityNumber example"
  category = "SocialSecurityNumber"
  fields = [
    { title = "number", type = "Concealed", value = "123-45-6789" }
  ]
}
