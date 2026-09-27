# Phase 1 — Cluster bootstrap + one tenant via Helm

**Status**: In progress — k3s is up (see ADR 0001 addendum for real deployment
notes), chart skeleton exists, first pilot install not yet done.
**Goal**: k3s running on both VMs, one real customer stack deployed and
reachable through it — no Management Plane yet, tenant created by hand.
**Repos touched**: this repo only (`infra/`, `charts/nexwall-controller/`).

For every task below: **Study** = read this before writing anything.
**Reimplement** = adapt an existing, proven pattern into the new location.
**Build new** = no reference exists anywhere in our repos or upstream; design
it from scratch.

---

## 1.1 — k3s cluster bootstrap

**Status: done.** `infra/k3s/install-master.sh` / `install-worker.sh` exist
and are validated against real hardware — see the "Real deployment notes"
section of `infra/k3s/README.md` for the two gotchas already found
(`node-ip`/`flannel-iface` must target the private `10.10.1.0/24` interface,
not the public one; the control-plane taint does not survive a restart).

**Remaining work**:
- **Build new**: a startup check (systemd unit override or a cron `@reboot`
  entry on the master) that re-applies `kubectl taint nodes <master>
  node-role.kubernetes.io/control-plane=:NoSchedule --overwrite` after every
  boot/`systemctl restart k3s`. No reference exists for this — it's a gap
  specific to our own taint-immediately decision (ADR 0001 addendum). Until
  this exists, re-tainting after any master restart is a **manual step you
  must remember**, not yet automated. Low urgency (doesn't cause an outage,
  only a silent scheduling risk) but cheap to close — do it before Phase 1
  sign-off, not blocking cluster bring-up itself.

**Acceptance criteria**: master reboots, taint reappears within 60s with no
manual intervention; verified by `kubectl describe node <master> | grep
Taints` immediately after a forced reboot.

---

## 1.2 — cert-manager + wildcard TLS

**Status: not started.**

- **Study**: cert-manager's own docs for DNS-01 `ClusterIssuer` (external —
  no Nethesis-specific reference for this, it's generic k8s tooling). Also
  re-read ADR 0005's TLS-termination open item — **this decision must be
  made before this task starts**: certbot on the VPS (consistent with how
  `license`/`updates`/`lists.nexwall.com.br` already work there) vs.
  cert-manager inside the cluster. ADR 0005 leans toward the VPS option but
  marks it not final — resolve this first, update ADR 0005's status, then
  proceed.
- **Build new**: whichever `ClusterIssuer` config is chosen, plus the
  cross-namespace secret distribution mechanism `infra/k3s/README.md` step 4
  explicitly flags as unsolved — either a Helm post-install hook that copies
  `wildcard-painel-tls` into each new tenant namespace, or
  [reflector](https://github.com/emberstack/kubernetes-reflector) configured
  to mirror it automatically. No existing pattern in `nexwall-controller` or
  upstream NethSecurity to draw on here — single-tenant NethSecurity has
  never needed multi-namespace cert sharing. This is a genuinely new,
  Nexwall-specific piece of infrastructure.

**Acceptance criteria**: a brand-new namespace, created with nothing but
`kubectl create namespace`, has a valid `wildcard-painel-tls` secret present
within 30s without any manual `kubectl cp`/`create secret` step.

---

## 1.3 — Finish `charts/nexwall-controller/templates/`

The chart README's own mapping table already did the 1:1 translation design
work. Four concrete TODOs remain, each independently completable:

### 1.3.a — Proxy env vars (`templates/proxy-deployment.yaml`)
- **Study**: `nexwall-controller`'s `proxy/entrypoint.sh`, line by line — the
  chart README states this was "inferred, not yet verified" against it.
  This is the single highest-risk unverified piece in the whole chart: if
  the proxy's actual required env vars differ from what's templated, the
  Ingress → Traefik-in-proxy-container routing silently breaks in a way
  that's hard to diagnose (proxy starts, but routes nothing).
- **Reimplement**: once verified, this is a direct value-for-value port —
  no new design needed, just correctness.

### 1.3.b — Grafana dashboards as a ConfigMap
- **Study**: the two dashboard JSON files from the single-VM dev setup
  (referenced by UID in the UI's hardcoded links: `W3S__804z`, `liz0yRCZz`
  — find these files in `nexwall-controller`'s dev environment, likely
  under a `grafana/dashboards/` or `monitoring/` path there).
- **Reimplement**: wrap each JSON file as a `ConfigMap` with the
  `grafana_dashboard: "1"` label (Grafana's sidecar dashboard-provisioning
  convention — same mechanism upstream `ns8-nethsecurity-controller` uses
  for its own controller-side Grafana, worth double-checking that repo's
  own dashboard ConfigMap pattern for the exact label/annotation Grafana's
  sidecar expects, since getting the label wrong means dashboards silently
  don't appear).
- **Acceptance criteria**: a freshly-installed tenant's Grafana shows both
  dashboards with no manual import, and the UI's hardcoded dashboard links
  resolve (not a 404) on first login.

### 1.3.c — VPN → cluster-Service routing for promtail
- **Study**: this is the one item in the whole chart with a real open
  technical question, not just unverified-but-probably-fine work. Under
  Podman host networking (the single-VM dev setup), a connected firewall's
  VPN traffic trivially reaches the promtail process because everything
  shares one network namespace. Under k8s with a CNI pod network, the `vpn`
  pod and the `promtail` pod are in *different* pod IPs, reachable only via
  the `promtail` ClusterIP Service — this path has never been proven to
  work end-to-end. Read the TODO comment already left in
  `templates/secrets.yaml` for the exact concern.
- **Build new**: this needs an actual test, not just a code read — deploy a
  pilot tenant (this may need to happen alongside 1.4 rather than strictly
  before it), connect a real or simulated firewall unit over its VPN, and
  confirm a syslog line makes it from the unit through the `vpn` pod to
  `promtail`'s Service to Loki. If it doesn't route cleanly, likely fixes
  are: routing table rule inside the `vpn` pod pointing pod-CIDR traffic at
  the CNI gateway (should already work by default with most CNIs — worth
  checking k3s's default flannel behavior specifically), or, if the `vpn`
  container needs `hostNetwork: true` to replicate Podman's behavior, that
  has its own consequences (loses per-pod IP isolation, needs
  `vpnPort` uniqueness enforcement to move from "Service port" to "actual
  host port" — reopens part of ADR 0004's port-allocation reasoning).
- **Acceptance criteria**: a real VPN-connected test unit's syslog output
  is visible in the tenant's Grafana (Loki datasource) within the same
  session as connecting it — no pod restart, no manual network fix needed.

### 1.3.d — Secrets externalization
- **Study**: `values.yaml`'s `secrets:` block — every value defaults to
  `CHANGE_ME*`, by design (chart README already states these are not safe
  production defaults).
- **Build new**: decide and document the actual secret-injection mechanism
  for real tenant deploys — `--set-file` per tenant (simplest, but secrets
  end up in shell history / CI logs unless handled carefully), or an
  external-secrets-operator source (more infrastructure, better hygiene).
  No existing pattern to reuse — single-tenant NethSecurity has no concept
  of "many tenants' secrets managed from one place." This decision should
  be made once, documented as an ADR addendum or a new ADR 0006, and then
  applied consistently — don't let it be decided ad hoc per tenant.

---

## 1.4 — First real pilot install

- **Reimplement**: `helm install tenant-pilot charts/nexwall-controller -f
  values-pilot.yaml` — mechanically straightforward once 1.2 and 1.3 are
  done; the actual values file just needs `tenantId: 1`, a real `slug`, and
  the network allocation from ADR 0004 applied by hand (Management Plane
  doesn't exist yet in this phase — this is the "created by hand" step the
  phase's own goal calls out).
- **Study**: the unit registration flow validated in the single-VM dev
  setup — `ns-plug` → `/units/register` in `nexwall-controller`'s API. This
  flow itself needs no changes; the goal here is only proving it still
  works when the controller it's registering against is one Helm release
  in one k8s namespace instead of a set of Podman containers on one host.

**Acceptance criteria (= Phase 1 exit criteria, unchanged from before)**:
one paying-customer-grade stack reachable at `pilot.painel.<domain>`, a
real firewall registered against it end-to-end — UI login works, unit shows
online, metrics AND logs both flowing (logs specifically depend on 1.3.c
being solved first, not just assumed).

---

## Order of execution

1.1 (close out) → 1.2 → 1.3.a/b/d (independent, parallelizable) → 1.3.c
(needs a real or test tenant to validate against, so do this alongside or
right after 1.4's first install attempt, not strictly before) → 1.4.

## Carries into Phase 2

- The exact `values-<slug>.yaml` shape produced by hand in 1.4 is what
  `internal/k8s.InstallOrUpgradeRelease` (currently a stub) needs to
  generate programmatically — keep notes on anything done by hand here
  that isn't yet captured in `values.schema.json`.
- The wildcard-cert-per-namespace mechanism from 1.2 needs to be triggered
  automatically by the Management Plane at tenant-creation time, per the
  TODO already in `infra/k3s/README.md` step 4 — Phase 1 only needs it
  working when triggered by hand.
