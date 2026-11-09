resource "op_item" "creditCard" {
  vault_id = op_vault.example.id
  title    = "CreditCard example"
  category = "CreditCard"
  fields = [
    { title = "card type", type = "CreditCardType", value = "visa" },
    { title = "number", type = "CreditCardNumber", value = "4111111111111111" },
    { title = "expiry", type = "MonthYear", value = "03/2028" },
    { title = "cvv", type = "Concealed", value = "123" }
  ]
}
