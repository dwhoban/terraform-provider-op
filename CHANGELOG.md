## 0.1.0 (unreleased)

FEATURES:

- resource/op_item_*: one dedicated resource per creatable item category (21, e.g. op_item_login, op_item_ssh_key) exposing each category's default template fields as typed attributes plus custom_fields; typed field IDs/titles/sections mirror `op item template get` dumps. The Email item's `provider` field is exposed as `email_provider` because Terraform reserves `provider` as an attribute name. No Document or Person resources: the API rejects creating them (use `op_item`)
- provider: when no credentials are configured, fall back to the local `op` CLI's default account for desktop app integration (no `OP_DESKTOP_ACCOUNT_NAME` needed for local execution)
- resource/op_item: fix crash and plan/state inconsistencies (SDK details variant dispatch, null normalization for tags/sections/notes/websites, Optional+Computed fields/sections)

- action/op_generate_password: generate random, PIN, or memorable passwords and write them into an item field
- data-source/op_items: list items in a vault with optional archived filter
- data-source/op_group: retrieve a group
- data-source/op_environment_variables: read 1Password Environment variables (beta)
- resource/op_item: add websites, Address field details (input), SSH key field details (computed), and optional input field IDs with SDK field-type constraint docs
- docs: coverage matrix (docs/coverage.md) and per-category item examples

- provider: 1Password SDK client with service-account and desktop-app auth
- resource/op_vault: manage vaults
- resource/op_item: manage items with fields, sections, tags, notes; archive-on-destroy option
- data-source/op_vault, op_vaults, op_item
- ephemeral/op_secret_ref: resolve op:// secret references without persisting values
