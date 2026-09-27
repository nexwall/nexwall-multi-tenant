# Nexwall Multi-Tenant

Cloud-hosted, multi-customer central management for Nexwall Firewalls — the layer that turns a single `nexwall-controller` instance (one customer) into a fleet of isolated controller instances managed from one place, comparable to Sophos Central / FortiManager Cloud.

This repo **does not** contain the controller's own source (API, UI, VPN, proxy) — that lives in [`nexwall-controller`](https://github.com/nexwall/nexwall-controller) and [`nexwall-ui`](https://github.com/nexwall/nexwall-ui), and its images are consumed here as-is (`ghcr.io/nexwall/nethsecurity-*`). This repo adds everything needed to run **many** of those stacks, isolated per customer, on shared infrastructure:

| Piece | What it is | Where |
|---|---|---|
| Helm chart | Packages one customer's full stack (vpn, db, api, ui, proxy, loki, promtail, prometheus, grafana) as a k8s release | `charts/nexwall-controller/` |
| Management Plane | New service: provisions/suspends/deletes tenants, gives the MSP a single-pane view across all customers | `management-plane/` |
| Infra | k3s cluster bootstrap for the master + worker VMs | `infra/k3s/` |
| Docs | Architecture, decision records, phased roadmap, API contracts | `docs/` |

## Why this is a separate repo

`nexwall-controller` is single-tenant by design (confirmed against upstream NethSecurity: no `tenant_id` anywhere in its schema or routes). Rather than rewrite its data layer, we isolate customers at the **infrastructure** level — one full stack per customer namespace — and keep that orchestration logic here, decoupled from the controller's own release cycle.

## Start here

1. [`docs/adr/`](docs/adr) — *why* k3s, *why* isolated-namespace-per-tenant, *why* this network scheme. Read these before touching infra.
2. [`docs/phases/`](docs/phases) — what's being built, in what order, and what "done" means for each phase.
3. [`docs/contracts/`](docs/contracts) — the Management Plane's API contract and the Helm chart's `values.schema.json`. Both are the source of truth other components are built against.

## Status

Phase 0 (this scaffold) — infra VMs not yet provisioned. See [`docs/phases/phase-0-foundations.md`](docs/phases/phase-0-foundations.md).

## License

GPL-3.0, inherited from `nexwall-controller`/NethSecurity — see `NOTICE.md`.
