resource "op_item_email" "email" {
  vault_id       = op_vault.example.id
  title          = "Email example"
  username       = "wendy"
  server         = "pop.example.com"
  password       = "hunter2"
  smtp_server    = "smtp.example.com"
  smtp_port      = "587"
  email_provider = "Example Mail"
}
