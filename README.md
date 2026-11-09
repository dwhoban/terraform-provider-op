# Terraform Provider Onepassword

A personal 1Password Terraform provider built for full compliance with the
[1Password Go SDK](https://github.com/1Password/onepassword-sdk-go) — it does
**not** use the Connect SDK.

## Features

- **Provider auth**: 1Password service account token
  (`OP_SERVICE_ACCOUNT_TOKEN`) or desktop app integration
  (`OP_DESKTOP_ACCOUNT_NAME`).
- **Resources**: `onepassword_vault`, `onepassword_item` (fields, sections,
  tags, notes; `delete_mode = "delete" | "archive"`).
- **Data sources**: `onepassword_vault`, `onepassword_vaults`,
  `onepassword_item`.
- **Ephemeral resource**: `onepassword_secret_ref` resolves `op://` secret
  references without persisting values to state.

## Develop

    go build -o /tmp/terraform-provider-onepassword .
    go test ./...

Acceptance tests run with `TF_ACC=1` and credentials set via
`OP_SERVICE_ACCOUNT_TOKEN` or `OP_DESKTOP_ACCOUNT_NAME` (otherwise they
skip).
