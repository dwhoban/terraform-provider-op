## 0.1.0 (unreleased)

FEATURES:

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
