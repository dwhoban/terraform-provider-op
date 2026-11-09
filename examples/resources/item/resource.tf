resource "op_item" "db" {
  vault_id = op_vault.infra.id
  title    = "Production database"
  category = "Login"
  tags     = ["terraform", "production"]

  fields = [
    { title = "username", type = "Text", value = "app" },
    { title = "password", type = "Concealed", value = var.db_password },
  ]

  delete_mode = "archive" # archive instead of permanent delete on destroy
}
