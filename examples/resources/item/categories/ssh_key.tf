# Option 1: let the provider generate the private key; 1Password derives
# the public key, fingerprint, and key type (computed in fields[].ssh_key).
resource "op_item" "ssh_key" {
  vault_id         = op_vault.example.id
  title            = "SshKey example"
  category         = "SshKey"
  generate_ssh_key = "ed25519"
}

# Option 2: supply your own decrypted PEM private key.
# resource "op_item" "ssh_key" {
#   vault_id = op_vault.example.id
#   title    = "SshKey example"
#   category = "SshKey"
#   fields = [{
#     id    = "private_key"
#     title = "private key"
#     type  = "SshKey"
#     value = tls_private_key.this.private_key_pem
#   }]
# }
