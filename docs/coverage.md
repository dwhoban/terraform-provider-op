# onepassword-sdk-go Coverage Matrix

How this provider maps the onepassword-sdk-go (v0.4.1) surface. Every API is
implemented, N/A (does not fit Terraform semantics), or deferred with a reason.

| SDK API | Provider surface | Status |
|---|---|---|
| Secrets.Resolve | ephemeral op_secret_ref | implemented |
| Secrets.ResolveAll | — | N/A: map-of-references semantics; use one ephemeral per reference |
| Secrets.ValidateSecretReference | op_secret_ref reference validator (regex) | approximated |
| Secrets.GeneratePassword (Pin/Random/Memorable) | action op_generate_password | implemented |
| Items.Create | resource op_item, per-category resources op_item_* | implemented |
| Items.Get | resource + data source op_item, per-category resources op_item_* | implemented |
| Items.Put | resource op_item, per-category resources op_item_* | implemented |
| Items.Delete / Items.Archive | resource op_item and op_item_* delete_mode | implemented |
| Vaults.Create/Get/List/Update/Delete | resource op_vault, data sources op_vault / op_vaults | implemented |
| Vaults.Grant/Revoke/UpdateGroupPermissions | — | deferred: vault group permission resource planned |
| Groups.Get | data source op_group | implemented |
| Environments.GetVariables | data source op_environment_variables | implemented (SDK beta) |
| Item field details (Address/SSHKey/OTP) | fields.address (input), fields.ssh_key (computed) | implemented; OTP code is derivable via op_secret_ref |

Per-category resources (`op_item_login`, `op_item_ssh_key`, …; one per
category) expose each category's default template fields as typed
attributes plus `custom_fields` for arbitrary extras. Field IDs, titles,
types, and sections mirror `op item template get <category>` dumps
(op 2.42.0-beta.01, 2026-11-12). (The Email template's `provider` field is
exposed as `email_provider`; Terraform reserves `provider` as a resource
attribute name.)

No per-category resources exist for Document and Person: the API rejects
creating Document items without an attached file (deferred with
Items.Files) and rejects every Person create ("bad request body").

Supported item categories with examples: see examples/resources/item/categories/
(21 categories; Document and Person are omitted because the API rejects creating them).
Per-category resources cover the 21 categories excluding Document and Person.
