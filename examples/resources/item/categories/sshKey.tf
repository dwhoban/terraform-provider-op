resource "op_item" "sshKey" {
  vault_id = op_vault.example.id
  title    = "SshKey example"
  category = "SshKey"
  fields = [
    { id = "private_key", title = "private key", type = "SshKey", value = "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----" }
  ]
}
