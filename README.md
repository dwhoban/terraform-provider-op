# Terraform Provider Onepassword

A personal 1Password Terraform provider built for full compliance with the
[1Password Go SDK](https://github.com/1Password/onepassword-sdk-go) — it does
**not** use the Connect SDK.

## Features

- **Provider auth**: 1Password service account token
  (`OP_SERVICE_ACCOUNT_TOKEN`), desktop app integration
  (`OP_DESKTOP_ACCOUNT_NAME`), or — when neither is set — the local
  `op` CLI's default account automatically.
- **Resources**: `op_vault`, `op_item` (fields, sections,
  tags, notes; `delete_mode = "delete" | "archive"`), and one
  `op_item_<category>` per item category (e.g. `op_item_login`,
  `op_item_ssh_key`) with typed template fields plus `custom_fields`.
- **Data sources**: `op_vault`, `op_vaults`,
  `op_item`.
- **Ephemeral resource**: `op_secret_ref` resolves `op://` secret
  references without persisting values to state.

## Develop

    go build -o /tmp/terraform-provider-op .
    go test ./...

Acceptance tests run with `TF_ACC=1` and credentials set via
`OP_SERVICE_ACCOUNT_TOKEN` or `OP_DESKTOP_ACCOUNT_NAME` (otherwise they
skip).
