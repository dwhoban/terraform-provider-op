resource "op_item_credit_card" "credit_card" {
  vault_id     = op_vault.example.id
  title        = "CreditCard example"
  cardholder   = "Wendy Appleseed"
  card_type    = "visa"
  card_number  = "4111111111111111"
  cvv          = "123"
  expiry       = "03/2028"
  issuing_bank = "First National"

  custom_fields = [
    { title = "discount", type = "Text", value = "member 10%" }
  ]
}
