# Personal 1Password Terraform Provider — onepassword-sdk-go only

## Context
- Repo: untouched `terraform-provider-scaffolding-framework` skeleton (go 1.25.8, plugin-framework v1.19.0).
- Data plane: `github.com/1Password/onepassword-sdk-go` v0.4.1 ONLY (no connect-sdk-go).
- Tier: HEAVY. Child agents unavailable on this machine → all work in root session.

## SDK surface (v0.4.1, verified via pkg.go.dev)
- Client: `NewClient(ctx, WithServiceAccountToken | WithDesktopAppIntegration(account), WithIntegrationInfo(name, ver))`
- Items: Create/CreateAll/Get/GetAll/Put/Delete/DeleteAll/Archive/List(ByState); Files (Attach/Read/Delete/ReplaceDocument); Shares (GetAccountPolicy/ValidateRecipients/Create)
- Vaults: Create/Delete/Get/GetOverview/List/Update/Grant|Revoke|UpdateGroupPermissions
- Groups: Get. Environments: GetVariables (beta). Secrets: Resolve/ResolveAll (op:// refs)
- Errors: RateLimitExceededError, DesktopSessionExpiredError, typed variant enums

## Delegation topology
Root session only (no usable child models): sequential phases below.

## Phases
1. **Foundation** — rename module, provider schema (auth config: service-account token via env var default OP_SERVICE_ACCOUNT_TOKEN; optional desktop-app account name), client factory, remove example_* code, go.mod tidy.
2. **Vaults** — `vault` data source (Get/List) then `vault` resource (Create/Update/Delete).
3. **Items** — `item` data source; `item` resource (CRUD + archive-on-delete option), full field/section/category/tags model; WriteOnly/Ephemeral handling for concealed fields.
4. **Secrets** — `op://` reference resolution: ephemeral resource `secret_ref` (preferred) + optional `resolve` function.
5. **Optional** (user decides) — files/documents, shares, vault permissions, environments.
6. **Verification** — unit tests; TF_ACC acceptance against live account (desktop app present on this machine?) or stubbed plan smoke; golangci-lint; docs/examples regenerated.

## Success criteria
Per goal 01a1bfc2: build+vet+lint clean; repo tests green; real-surface proof; zero connect-sdk deps.

## Open decisions (asked user)
- Provider address (e.g. dwh/onepassword)
- v1 scope (items+vaults+secrets vs more)
- Secrets surface (ephemeral vs data source vs both)
- Delete behavior (archive vs delete)
