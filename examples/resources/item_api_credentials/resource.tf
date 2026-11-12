resource "op_item_api_credentials" "api_credentials" {
  vault_id   = op_vault.example.id
  title      = "ApiCredentials example"
  username   = "svc-account"
  credential = "sk-live-xxxx"
  type       = "bearer"
  hostname   = "api.example.com"

  custom_fields = [
    { title = "rotation reminder", type = "Text", value = "quarterly" }
  ]
}
