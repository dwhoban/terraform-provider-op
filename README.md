# Terraform Provider Onepassword

A personal 1Password Terraform provider built for full compliance with the
[1Password Go SDK](https://github.com/1Password/onepassword-sdk-go) — it does
**not** use the Connect SDK.

## Features

- **Provider auth**: 1Password service account token
  (`OP_SERVICE_ACCOUNT_TOKEN`) or desktop app integration
  (`OP_DESKTOP_ACCOUNT_NAME`).
- **Resources**: `op_vault`, `op_item` (fields, sections,
  tags, notes; `delete_mode = "delete" | "archive"`).
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
