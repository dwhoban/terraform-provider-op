resource "op_item" "login" {
  vault_id = op_vault.example.id
  title    = "Login example"
  category = "Login"
  fields = [
    { title = "username", type = "Text", value = "wendy_appleseed" },
    { id = "password", title = "password", type = "Concealed", value = "correct-horse-battery-staple" },
    { title = "one-time password", type = "Totp", value = "otpauth://totp/my-example-otp?secret=jncrjgbdjnrncbjsr&issuer=1Password", section_id = "extra" }
  ]

  websites = [{ url = "example.com", label = "website", autofill_behavior = "AnywhereOnWebsite" }]
}
