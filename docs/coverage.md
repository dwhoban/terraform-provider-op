# onepassword-sdk-go Coverage Matrix

How this provider maps the onepassword-sdk-go (v0.4.1) surface. Every API is
implemented, N/A (does not fit Terraform semantics), or deferred with a reason.

| SDK API | Provider surface | Status |
|---|---|---|
| Secrets.Resolve | ephemeral op_secret_ref | implemented |
| Secrets.ResolveAll | — | N/A: map-of-references semantics; use one ephemeral per reference |
| Secrets.ValidateSecretReference | op_secret_ref reference validator (regex) | approximated |
| Secrets.GeneratePassword (Pin/Random/Memorable) | action op_generate_password | implemented |
| Items.Create | resource op_item | implemented |
| Items.Get | resource + data source op_item | implemented |
| Items.Put | resource op_item | implemented |
| Items.Delete / Items.Archive | resource op_item delete_mode | implemented |
| Items.List (ByState filter) | data source op_items (include_archived) | implemented |
| Items.CreateAll / GetAll / DeleteAll | — | N/A: per-item Terraform resources already batch plans |
| Items.Files (Attach/Read/Delete/ReplaceDocument) | — | deferred: file/document management resource planned |
| Items.Shares (policy/validate/create) | — | deferred: share-link resource planned |
| Vaults.Create/Get/List/Update/Delete | resource op_vault, data sources op_vault / op_vaults | implemented |
| Vaults.Grant/Revoke/UpdateGroupPermissions | — | deferred: vault group permission resource planned |
| Groups.Get | data source op_group | implemented |
| Environments.GetVariables | data source op_environment_variables | implemented (SDK beta) |
| Item field details (Address/SSHKey/OTP) | fields.address (input), fields.ssh_key (computed) | implemented; OTP code is derivable via op_secret_ref |

Supported item categories with examples: see examples/resources/item/categories/
(23 categories; the SDK also lists Unsupported, which is not creatable).
