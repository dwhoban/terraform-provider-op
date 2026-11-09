resource "op_item" "cryptoWallet" {
  vault_id = op_vault.example.id
  title    = "CryptoWallet example"
  category = "CryptoWallet"
  fields = [
    { title = "recovery phrase", type = "Concealed", value = "word1 word2 word3" }
  ]
}
