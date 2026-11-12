resource "op_item_server" "server" {
  vault_id              = op_vault.example.id
  title                 = "Server example"
  url                   = "https://server.example.com"
  username              = "root"
  password              = "hunter2"
  admin_console_url     = "https://server.example.com/admin"
  hosting_provider_name = "Acme Hosting"
}
