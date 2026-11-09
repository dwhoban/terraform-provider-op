resource "op_item" "bankAccount" {
  vault_id = op_vault.example.id
  title    = "BankAccount example"
  category = "BankAccount"
  fields = [
    { title = "bank", type = "Text", value = "First National" },
    { title = "account type", type = "Text", value = "checking" },
    { title = "routing number", type = "Text", value = "123456789" }
  ]
}
