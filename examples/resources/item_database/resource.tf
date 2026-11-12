resource "op_item_database" "database" {
  vault_id = op_vault.example.id
  title    = "Database example"
  type     = "mysql"
  hostname = "db.example.com"
  database = "appdb"
  username = "app"
  password = "hunter2"
}
