# Phase 3 — Group-level firewall policy push

**Status**: Not started.
**Goal**: close the real product gap found during evaluation — NethSecurity's
own controller has no way to push a policy to a group of units (confirmed:
not in `nexwall-controller`'s code, not in upstream, and openly tracked as
unimplemented in `NethServer/nethsecurity#1895`).
**Repos touched**: `nexwall-controller` (API + UI) primarily; this repo only
for the version-bump rollout at the end.
**Reference material**: check
`dev-nethsec-reference/future-development/nethsecurity.md` before starting —
that's our periodic snapshot of upstream's open backlog, and issue #1895
should appear there if it's still open. If upstream has since posted a
design proposal or started their own PR against #1895, read it before
designing ours independently — not to block on them, but to avoid an
incompatible approach if we ever want to stay mergeable with upstream
patterns, or at minimum to learn from any design discussion already had
there.

---

## 3.1 — `Policy` model

- **Study**: `nexwall-controller`'s `api/models/unit.go` — specifically how
  `UnitGroup` is currently modeled. It exists today but is **RBAC-only**
  (scoped to which users can see which units, confirmed during ADR 0002's
  research — "no `tenant_id` anywhere... `UnitGroup` is RBAC-only, scoped to
  users not tenants"). This model needs to grow a second purpose (policy
  targeting) without breaking its first (access control) — read how it's
  used in both the API and `UnitGroupsView.vue` before changing its shape.
- **Build new**: a `Policy` model associated with `UnitGroup`. No existing
  reference for the policy's own shape (rule set, scope, versioning) —
  design this against whatever firewall-rule schema `nexwall-controller`
  (or the units themselves, via UCI) already uses for a *single* unit's
  rules, so a group policy is expressible as "the same rule shape, applied
  to N units" rather than a parallel, incompatible format.

## 3.2 — Push mechanism

- **Study**: the existing `Actions → Update systems` feature — phase doc
  already identifies this as "the closest existing analog": a per-unit
  action, triggerable in bulk, that goes out over the same authenticated
  VPN channel already used for SSH key distribution and remote updates.
  Trace this end-to-end in `nexwall-controller`'s API before designing the
  policy-push path — the goal is to **generalize**, not replace.
- **Reimplement**: extend that same bulk-action mechanism to accept an
  arbitrary config payload (a serialized `Policy`) instead of only an
  "update to version X" instruction. The transport/auth/bulk-targeting
  logic is proven and existing — only the payload shape and the
  receiving-end handler (on the unit, presumably `ns-api`'s side, applying
  the pushed policy via UCI) are new.
- **Build new**: the unit-side handler for receiving and applying a pushed
  policy, if `ns-api` doesn't already have a generic "apply this config
  payload" entrypoint beyond the update mechanism. Check upstream
  `nethsecurity`'s `ns-api` package first — if a generic config-push
  primitive already exists there for some other feature, reuse its shape
  rather than inventing a second one.

## 3.3 — UI: "Policies" view under Unit Groups

- **Study**: `UnitGroupsView.vue` — match its existing patterns (list/
  create/edit/delete flow, how it fetches units within a group) rather than
  introducing a new UI convention.
- **Reimplement**: same list/detail/edit shell, new content — a policy
  editor instead of a membership editor. If `nexwall-ui`'s DPI rule-builder
  components (`dpi.ts`, the `DpiRule`/`DpiAppOrProtocol` types, and
  whatever `.vue` components render them for a single unit) already have a
  usable rule-editing UI, that component is a strong candidate to reuse
  for the group-policy editor rather than building a second rule-editing
  UI from scratch — the underlying rule *shape* differs (single-unit DPI
  criteria vs. a broader firewall policy) but the editing *interaction
  pattern* (add/remove criteria, category pickers, etc.) is likely directly
  reusable.

## 3.4 — Rollout (this repo)

- **Reimplement**: no infra changes — once 3.1–3.3 land in
  `nexwall-controller`/`nexwall-ui` and are tagged, this repo's role is
  exactly what any other version bump requires: update `image.tag` in
  `charts/nexwall-controller/values.yaml` (or per-tenant override) and
  `helm upgrade` each tenant. Nothing new to design here.

**Acceptance criteria (= Phase 3 exit criteria, unchanged)**: an MSP
operator applies one firewall rule change to a group of 10 units in one
action, from one tenant's controller UI.

## Open question to resolve before starting

Is this feature valuable enough, and general enough, to eventually propose
upstream against `NethServer/nethsecurity#1895` (contributing back), or is
it staying Nexwall-specific? This doesn't block starting the work, but it
does affect a few design choices early (e.g. whether the `Policy` payload
format tries to stay close to what upstream's issue discussion has
proposed, if anything). Worth a explicit decision, not a default.
