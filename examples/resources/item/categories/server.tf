resource "op_item" "server" {
  vault_id = op_vault.example.id
  title    = "Server example"
  category = "Server"
  fields = [
    { title = "username", type = "Text", value = "root" },
    { title = "password", type = "Concealed", value = "hunter2" },
    { title = "URL", type = "Url", value = "https://server.example.com" }
  ]
}
