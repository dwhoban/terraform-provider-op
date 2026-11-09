resource "op_item" "apiCredentials" {
  vault_id = op_vault.example.id
  title    = "ApiCredentials example"
  category = "ApiCredentials"
  fields = [
    { title = "credential", type = "Concealed", value = "sk-live-xxxx" }
  ]
}
