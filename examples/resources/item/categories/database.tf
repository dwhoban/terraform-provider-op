resource "op_item" "database" {
  vault_id = op_vault.example.id
  title    = "Database example"
  category = "Database"
  fields = [
    { title = "username", type = "Text", value = "app" },
    { title = "password", type = "Concealed", value = "hunter2" },
    { title = "hostname", type = "Text", value = "db.example.com" },
    { title = "database", type = "Text", value = "appdb" }
  ]
}
