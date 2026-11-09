resource "op_item" "email" {
  vault_id = op_vault.example.id
  title    = "Email example"
  category = "Email"
  fields = [
    { title = "email", type = "Email", value = "wendy@example.com" },
    { title = "password", type = "Concealed", value = "hunter2" }
  ]
}
