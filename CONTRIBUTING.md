# Contributing

This repo is currently maintained solely through Claude, by explicit agreement — see the project owner before making manual changes, to avoid conflicting edits.

## Where things belong

| Change | Goes in |
|---|---|
| New/changed k8s resource for the tenant stack | `charts/nexwall-controller/templates/` — update `docs/README.md`'s mapping table too if it's a new component |
| Provisioning/lifecycle logic | `management-plane/internal/` |
| A new architectural decision (orchestrator, tenancy, networking, etc.) | A new numbered ADR in `docs/adr/`, never edit a past one — supersede it with a new one instead |
| API surface change | `docs/contracts/management-plane-openapi.yaml` **first**, then the handler in `management-plane/internal/api/` — the contract is the source of truth, code follows it |
| Chart input change | `values.schema.json` **first**, then `values.yaml` defaults, then the templates that consume it |

## Before opening a PR

1. `helm lint charts/nexwall-controller` and `helm template` against a dummy tenant (see `.github/workflows/lint-chart.yml` for the exact invocation) — must pass locally before CI.
2. `cd management-plane && go vet ./... && go test ./...`
3. If you touched an ADR's "Consequences" section materially, flag it in the PR description — consequences drifting silently is how architecture docs stop being trusted.
