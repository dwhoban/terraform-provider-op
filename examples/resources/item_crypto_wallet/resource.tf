resource "op_item_crypto_wallet" "crypto_wallet" {
  vault_id        = op_vault.example.id
  title           = "CryptoWallet example"
  recovery_phrase = "word1 word2 word3"
  password        = "correct-horse-battery-staple"
  wallet_address  = "0x123abc"
}
