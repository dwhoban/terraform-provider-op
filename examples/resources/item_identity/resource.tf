resource "op_item_identity" "identity" {
  vault_id   = op_vault.example.id
  title      = "Identity example"
  first_name = "Wendy"
  last_name  = "Appleseed"
  address = {
    street  = "123 Main St"
    city    = "Anytown"
    state   = "CA"
    zip     = "12345"
    country = "USA"
  }
}
