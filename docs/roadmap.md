# Roadmap

This roadmap describes the planned BedemWAF product path from MVP through
enterprise-oriented capabilities. It is intentionally implementation-oriented:
each item includes user value, technical scope, risks, and acceptance criteria.

The roadmap does not add offensive features. BedemWAF remains a defensive WAF
and policy management platform for protecting applications behind NGINX origins.

## MVP

MVP proves the core managed-WAF loop on a single local stack: configure an app,
publish defensive policy, enforce at the gateway, and inspect redacted events.

### Reverse Proxy

Value: Gives teams a working data plane that can sit in front of NGINX origins
and make allow/block decisions before traffic reaches the application.

Technical scope: HTTP reverse proxy in Go; host-based app lookup; request ID
generation; safe client IP extraction; health endpoint; origin error handling;
timeouts and max header bytes.

Risks: Incorrect Host normalization can route traffic to the wrong app; origin
proxy failures can look like WAF blocks; request body handling can break
upstream behavior if not restored correctly.

Acceptance criteria: Gateway proxies allowed traffic to the configured origin;
unknown hosts return JSON 404; origin failures return JSON 502; request bodies
arrive intact upstream; `go test ./...` passes for gateway.

### Local Policies

Value: Lets developers run and tune BedemWAF without needing the Control API or
database during early development.

Technical scope: YAML config for apps, hostnames, origins, policy mode, default
action, IP sets, IP allowlists, IP blocklists, custom rules, and rate limits.

Risks: Local and remote policy schemas can drift; defaults may be too permissive
or too strict; invalid config must fail closed at startup rather than failing at
request time.

Acceptance criteria: Local YAML can fully configure a demo app; unsupported
modes/actions/operators are rejected; malformed CIDRs and origins fail startup;
local mode remains available after remote policies are introduced.

### Custom Rules

Value: Enables simple defensive rules without requiring users to write code or
unsafe regular expressions.

Technical scope: Deterministic priority-sorted evaluator; supported operators
for method, path, host, headers, query parameters, and IP set membership;
`allow`, `count`, and `block` actions; `terminal_allow` support.

Risks: Rule ordering bugs can turn count rules into bypasses; overly broad
terminal allows can override important blocks; query/header matching must avoid
unsafe parsing and scripting.

Acceptance criteria: First terminal block wins; count rules do not block or
prevent later block checks; terminal allow only short-circuits when explicitly
configured; invalid rule schema is rejected; unit tests cover every operator.

### Rate Limiting

Value: Protects high-risk endpoints and reduces noisy abuse with WAF-style
rate-limit rules.

Technical scope: Redis fixed-window limiter; atomic Lua script; hashed
high-cardinality key parts; key types for IP, host, path, header, and API key
placeholder; fail-open/fail-closed configuration.

Risks: Redis outages can either allow abuse or block legitimate traffic;
per-request Redis connections can become a bottleneck; raw API keys must never
appear in Redis keys or logs.

Acceptance criteria: Below-limit requests are allowed; over-limit block rules
return 429 in block mode; count-mode/rule count records would-rate-limit but
allows; raw tokens are hashed; Redis disabled mode is a no-op.

### Audit Logs

Value: Gives operators explainability for WAF decisions without storing full
sensitive request bodies.

Technical scope: Structured audit event model; JSON stdout sink; asynchronous
bounded dispatcher; query redaction; sensitive header redaction; event/drop
metrics placeholders.

Risks: Audit logging can leak secrets if redaction is incomplete; slow sinks can
impact request handling if dispatch is not bounded; dropped events reduce
investigation quality.

Acceptance criteria: Events are one JSON object per line; request bodies,
Authorization, and Cookie headers are not logged; sensitive query parameters are
redacted; full queues drop events with metrics/log warnings.

### Coraza Integration

Value: Provides OWASP CRS-compatible WAF inspection through a maintained
defensive engine.

Technical scope: Coraza-backed `Engine` implementation; rule engine config;
request body access with explicit limit; sample `coraza.conf`; interruption to
decision mapping.

Risks: Request body limits can create false negatives if too low; DetectionOnly
must never block; rule files can be misconfigured and cause startup or runtime
failures.

Acceptance criteria: Gateway works with `waf.enabled=false`; Coraza blocks in
block mode when rules interrupt; DetectionOnly/count mode logs would-block and
allows; harmless local test rule is covered by unit tests.

### Control API

Value: Establishes the management plane for tenants, apps, policies, publishing,
and events.

Technical scope: Go REST API; health/ready endpoints; static admin and gateway
API keys for MVP; tenant context middleware; JSON errors; input validation;
OpenAPI document.

Risks: Tenant isolation bugs can leak configuration; static API keys are only an
MVP auth model; shallow validation can publish policies that gateways reject.

Acceptance criteria: `/v1` admin routes require bearer auth; gateway policy
route uses a separate gateway bearer key; tenant-scoped routes require tenant
context; errors use the documented shape; OpenAPI matches implemented endpoints.

### Dashboard MVP

Value: Gives users a basic admin workflow for apps, policy JSON, and event
search without relying only on curl.

Technical scope: Next.js TypeScript app; login page using development API key
storage; sidebar/topbar layout; app list/create/detail pages; policy editor;
event search/detail pages.

Risks: LocalStorage API key storage is not production authentication; dashboard
can drift from API schemas; unclear count/block UI can lead to unsafe rollout.

Acceptance criteria: TypeScript strict checks pass; dashboard works against the
documented Control API; development auth warning is visible; no hardcoded
secrets are committed.

## v0.2

v0.2 turns the MVP into a more realistic managed-policy loop with remote policy
distribution, event analytics, and operational visibility.

### Remote Policy Cache

Value: Allows gateways to consume the active policy from the Control API instead
of local files.

Technical scope: Gateway policy client; in-memory cache with TTL; stale policy
fallback; per-host fetch locking; configurable fail behavior.

Risks: Fail-open behavior can allow traffic with no policy; fail-closed can
cause outages; stale policy use must be visible in audit events and logs.

Acceptance criteria: Gateway fetches by normalized Host; cache hit/miss/expiry
tests pass; stale fallback is used when available; local YAML mode still works.

### Policy Versions

Value: Makes policy publishing auditable and reversible by storing immutable
snapshots.

Technical scope: Postgres `policies`, `policy_versions`, and
`policy_deployments`; optimistic locking for draft updates; active deployment
pointer.

Risks: Draft and published snapshots can diverge unexpectedly; active pointer
updates must be atomic; published versions must not be mutated.

Acceptance criteria: Publishing creates an immutable version; active policy by
app/hostname returns the published snapshot; draft updates do not change old
versions; tests cover optimistic lock conflicts.

### ClickHouse Analytics

Value: Supports high-volume event search and dashboard analytics without
overloading Postgres.

Technical scope: `waf_events` table; gateway ClickHouse sink; Control API event
search filters; request ID lookup; ClickHouse Docker initialization.

Risks: ClickHouse outages must not block requests; query filters must be
parameterized; events must never expose full request bodies.

Acceptance criteria: Docker Compose initializes the event table; gateway can
write events to stdout and ClickHouse; Control API enforces tenant-scoped
filters and max limits; query builder tests pass.

### Safe Rollout Summary

Value: Helps users tune in count mode before enforcing block mode.

Technical scope: `enforced` and `would_block` event fields; simulation summary
endpoint; dashboard display of would-block rules, unique IPs, top paths, and
sample request IDs.

Risks: Misclassifying count events can cause premature blocking decisions;
summary queries can be expensive without limits; sample request IDs must remain
tenant scoped.

Acceptance criteria: Count mode never blocks because of WAF/custom/rate rules;
block mode enforces block/rate-limit actions; simulation summary validates time
ranges; dashboard clearly distinguishes count and block behavior.

### Prometheus Metrics

Value: Gives operators visibility into request volume, blocks, latency, cache
behavior, rate limiting, audit drops, API errors, and worker jobs.

Technical scope: `/metrics` endpoints for gateway, Control API, and worker;
Prometheus client library; optional Compose profile; Grafana dashboard
placeholder.

Risks: Metrics labels can expose secrets or create high cardinality; metrics
collection should not slow request handling.

Acceptance criteria: Metrics endpoints work without auth-sensitive data; labels
are bounded; optional Prometheus/Grafana profile starts; README documents local
observability.

## v0.3

v0.3 focuses on operating multiple gateways, improving administration, and
closing lifecycle gaps around policies, keys, rules, and event retention.

### Multi-Gateway Deployment

Value: Lets teams run multiple gateway replicas behind a load balancer for
availability and scale.

Technical scope: Gateway node identity; gateway group metadata; active
deployment status per gateway group; health and last-seen tracking.

Risks: Gateways can run different policy versions during rollout; cache
invalidation and stale handling need clear semantics; clock skew can affect
last-seen data.

Acceptance criteria: Control API can represent multiple gateway nodes/groups;
policy deployment status includes active version per group; docs explain rollout
and stale behavior.

### API Keys

Value: Replaces global static admin keys with scoped credentials suitable for
automation and integration.

Technical scope: API key creation, hashing, prefixes, scopes, expiration,
last-used tracking, and revocation; audit events for key use.

Risks: Key material must never be stored raw; scope checks can be incomplete;
rotation UX must be clear.

Acceptance criteria: Keys are shown only once; hashes are stored; revoked or
expired keys fail; scope checks protect endpoints; key usage is auditable.

### Better Dashboard Auth

Value: Removes development-only localStorage API key login from normal admin
workflows.

Technical scope: Session-based dashboard auth; secure cookies; CSRF protection
where needed; logout; server-side API proxy or token exchange.

Risks: Incorrect cookie settings can expose sessions; auth state can drift
between dashboard and API; migration from dev auth must be smooth.

Acceptance criteria: Dashboard no longer requires storing admin API keys in
localStorage; sessions expire; secure cookie flags are documented and enabled
for production.

### Policy Rollback

Value: Lets operators quickly return to a previous known-good policy version
after false positives or regressions.

Technical scope: Rollback endpoint; version diff metadata; dashboard rollback
action; deployment pointer update to previous immutable version.

Risks: Rollback can re-enable outdated managed rules; concurrent draft updates
can confuse users; rollback must be audited.

Acceptance criteria: Users can select a previous version and make it active;
rollback does not mutate historical snapshots; audit event records actor,
policy, old version, and new version.

### Managed Rules Metadata

Value: Supports manual, reviewed OWASP CRS-compatible rule set lifecycle without
automatic unsafe downloads.

Technical scope: Worker local rules directory scanner; checksum computation;
managed rule set/version metadata in Postgres; placeholder activation endpoint.

Risks: Rule files must not be executed as code; checksums must cover the
expected files; users may assume metadata activation changes enforcement.

Acceptance criteria: Worker records local rule versions and checksums; no remote
downloads occur by default; docs require manual review and explicit policy
publish before enforcement.

### Event Retention

Value: Controls storage growth and aligns event retention with operational and
privacy requirements.

Technical scope: Retention settings; worker cleanup jobs; ClickHouse TTL or
partition cleanup; dashboard visibility of retention policy.

Risks: Deleting too aggressively harms investigations; retaining too long
increases cost and privacy exposure; retention must be tenant-aware.

Acceptance criteria: Default retention is documented; cleanup job is safe and
idempotent; retention can be configured; tests cover retention cutoff logic.

## v1.0

v1.0 is the production-ready milestone: stable APIs, hardened auth, documented
HA operations, deployment artifacts, and recovery procedures.

### HA Deployment Docs

Value: Gives operators a supported path for running BedemWAF with redundancy.

Technical scope: Reference architecture for multiple gateways, Control API
replicas, Postgres, Redis, ClickHouse, load balancers, TLS, and backups.

Risks: HA docs can overpromise if failure modes are not tested; Redis and
ClickHouse HA choices vary by environment.

Acceptance criteria: Docs include topology diagrams, failure behavior, upgrade
strategy, scaling notes, and explicit non-goals such as L3/L4 DDoS protection.

### Kubernetes Helm Chart

Value: Makes BedemWAF easier to install and operate in Kubernetes environments.

Technical scope: Helm chart for gateway, Control API, worker, dashboard,
config/secrets, services, ingress, probes, resources, and optional dependencies.

Risks: Charts can accidentally expose databases or admin APIs; defaults must be
safe but still usable locally.

Acceptance criteria: Helm install renders valid manifests; default values do
not include real secrets; readiness/liveness probes work; docs include minimal
and production examples.

### Stronger Auth And RBAC

Value: Supports least-privilege administrative access across tenants and roles.

Technical scope: Users, roles, permissions, scoped API keys, tenant membership,
admin audit logs, password/session policy or external identity integration
hooks.

Risks: Partial RBAC can create privilege escalation; cross-tenant checks must be
centralized and tested; admin audit logs must avoid secret leakage.

Acceptance criteria: Owner/admin/viewer roles are enforced; tenant isolation
tests cover read/write paths; admin actions are audited; denied access returns
consistent JSON errors.

### Production Hardening

Value: Reduces operational and security risk before declaring a stable release.

Technical scope: Strict request limits, safe CORS, secure headers, panic
recovery, container hardening, dependency scanning, origin locking guidance,
secret management guidance, and egress controls.

Risks: Hardened defaults can break demos or local development; some controls
depend on deployment environment.

Acceptance criteria: Production checklist is complete; no full request body
logging by default; containers avoid root/read-write where practical; automated
tests cover critical security behavior.

### Backup And Restore

Value: Gives operators a way to recover configuration and analytics after data
loss or migration mistakes.

Technical scope: Postgres backup/restore docs; ClickHouse backup/retention
guidance; restore validation steps; disaster recovery runbook.

Risks: Untested backups create false confidence; restoring analytics and config
may require different RPO/RTO expectations.

Acceptance criteria: Backup commands are documented; restore procedure is tested
in a local or staging environment; RPO/RTO expectations are documented.

### Stable API

Value: Allows dashboard, gateway, scripts, and external automation to depend on
Control API behavior without constant churn.

Technical scope: Versioned API paths; OpenAPI contract; compatibility policy;
error schema stability; deprecation process.

Risks: Freezing unstable endpoints too early creates long-term maintenance
burden; undocumented behavior may become accidental contract.

Acceptance criteria: `docs/openapi.yaml` validates; implemented routes match
the spec; compatibility/deprecation policy is documented; CI checks API docs
where practical.

## Enterprise Later

Enterprise features are useful for larger organizations but should not distract
from a safe, stable core product.

### SSO, SAML, And OIDC

Value: Lets enterprises centralize identity, MFA, and access lifecycle.

Technical scope: OIDC login; SAML support; group/role mapping; session policy;
tenant-level identity provider configuration.

Risks: Misconfigured identity mapping can grant excessive access; SAML/OIDC
edge cases are complex; logout/session revocation behavior varies.

Acceptance criteria: OIDC works with at least one common provider in staging;
SAML metadata validation exists; role mapping is tested; local fallback auth is
controlled and documented.

### Per-Tenant Quotas

Value: Prevents one tenant from consuming disproportionate gateway, API, or
analytics resources.

Technical scope: Quotas for apps, policies, rules, event retention, API
requests, and event ingestion; quota visibility in dashboard.

Risks: Quotas can block legitimate emergency changes; enforcement must be
tenant scoped and race-safe.

Acceptance criteria: Quota checks are enforced consistently; over-quota errors
are clear; dashboard shows usage; tests cover cross-tenant isolation.

### SIEM Integrations

Value: Sends BedemWAF events into existing security operations workflows.

Technical scope: Webhook, syslog, and vendor-neutral JSON exports; retry queue;
redaction preservation; integration health.

Risks: Integrations can leak secrets if schemas diverge; retries can overload
external systems; delivery failures need visibility.

Acceptance criteria: At least one generic webhook/syslog path works; exports do
not include request bodies; failures are observable; retry behavior is bounded.

### Advanced Bot Signals

Value: Helps distinguish automation from normal user traffic without turning
BedemWAF into a tracking product.

Technical scope: Defensive heuristics such as request rate shape, header
consistency, known safe client hints, and optional challenge placeholders.

Risks: Bot signals can create privacy concerns and false positives; challenges
can harm accessibility and user experience.

Acceptance criteria: Signals are explainable in audit events; no invasive
fingerprinting by default; count mode is required before enforcement.

### GraphQL And API Protection

Value: Improves protection for modern APIs beyond simple path/header matching.

Technical scope: GraphQL operation name extraction, depth/complexity limits,
JSON API field matching, schema-aware allow/count/block rules.

Risks: Parsing request bodies increases sensitivity and performance cost; full
body logging must remain disabled; malformed payloads must fail safely.

Acceptance criteria: Body parsing is bounded by explicit limits; parsed metadata
is redacted; harmless test cases cover GraphQL depth and API field matching.

### Terraform Provider

Value: Lets platform teams manage BedemWAF configuration as code.

Technical scope: Terraform provider for tenants, apps, origins, policies, IP
sets, custom rules, rate limits, and policy publish/rollback workflows.

Risks: Terraform state can contain sensitive configuration; publish workflows
must be explicit; API stability is required first.

Acceptance criteria: Provider supports CRUD for core resources; docs warn about
state sensitivity; acceptance tests run against local Control API.

### Multi-Region Gateways

Value: Supports globally distributed gateways close to applications or users.

Technical scope: Gateway region metadata; policy replication strategy;
region-scoped health; event ingestion per region; failover docs.

Risks: Policy consistency across regions is hard; event ordering can vary;
cross-region latency can affect remote policy fetches.

Acceptance criteria: Gateways report region and active policy version; docs
explain consistency model; stale/fail behavior is region-aware.
