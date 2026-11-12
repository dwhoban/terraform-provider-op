resource "op_item_login" "login" {
  vault_id = op_vault.example.id
  title    = "Login example"
  username = "wendy_appleseed"
  password = "correct-horse-battery-staple"

  custom_fields = [
    { title = "one-time password", type = "Totp", value = "otpauth://totp/my-example-otp?secret=jncrjgbdjnrncbjsr&issuer=1Password" }
  ]

  websites = [{ url = "example.com", label = "website", autofill_behavior = "AnywhereOnWebsite" }]
}
