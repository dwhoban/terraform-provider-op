resource "op_item_bank_account" "bank_account" {
  vault_id       = op_vault.example.id
  title          = "BankAccount example"
  bank_name      = "First National"
  account_type   = "checking"
  routing_number = "123456789"
  account_number = "000123456789"
  branch_phone   = "555-0100"
}
