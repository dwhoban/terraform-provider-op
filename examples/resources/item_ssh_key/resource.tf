# Option 1: let the provider generate the private key; 1Password derives
# the public key, fingerprint, and key type (computed attributes).
resource "op_item_ssh_key" "ssh_key" {
  vault_id         = op_vault.example.id
  title            = "SshKey example"
  generate_ssh_key = "ed25519"

  custom_fields = [
    { title = "host", type = "Text", value = "github.com" }
  ]
}

# Option 2: supply your own decrypted PEM private key.
# resource "op_item_ssh_key" "ssh_key" {
#   vault_id    = op_vault.example.id
#   title       = "SshKey example"
#   private_key = tls_private_key.this.private_key_pem
# }
