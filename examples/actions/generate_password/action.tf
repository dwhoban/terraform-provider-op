action "op_generate_password" "rotate_db" {
  config {
    mode            = "random"
    length          = 32
    include_symbols = true
    vault_id        = op_vault.example.id
    item_id         = op_item.db.id
    field_title     = "password"
  }
}

# Wire the rotation to run after the item is created or updated:
resource "op_item" "db" {
  vault_id = op_vault.example.id
  title    = "Production database"
  category = "Login"

  fields = [
    { id = "username", title = "username", type = "Text", value = "app" },
    { id = "password", title = "password", type = "Concealed", value = "initial-value" },
  ]

  lifecycle {
    action_trigger {
      events  = [after_create, after_update]
      actions = [action.op_generate_password.rotate_db]
    }
  }
}
