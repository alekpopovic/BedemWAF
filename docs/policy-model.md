# Policy Model

BedemWAF models WAF configuration as tenant-owned applications protected by
policies. Policies combine origins, rule groups, IP sets, rate limits, and
rollout mode.

## Tenant

A tenant is an administrative boundary.

Fields to define:

- `id`
- `name`
- `status`
- `created_at`
- `updated_at`

Tenant-owned resources include apps, origins, policies, rule groups, IP sets,
rate limits, users, and audit events.

## App

An app represents a protected HTTP property, such as `api.example.com`.

Fields to define:

- `id`
- `tenant_id`
- `name`
- `hostnames`
- `default_origin_id`
- `active_policy_id`

## Origin

An origin is the upstream NGINX endpoint that receives allowed traffic from the
gateway.

Fields to define:

- `id`
- `tenant_id`
- `app_id`
- `name`
- `scheme`
- `host`
- `port`
- `health_check_path`
- `tls_server_name`

Origin security requirements:

- Operators should lock down origin ingress to BedemWAF gateway addresses
- Gateway should set and preserve explicit forwarding headers
- Direct-to-origin traffic should be considered a deployment misconfiguration

## Policy

A policy defines how BedemWAF evaluates traffic for an app.

Fields to define:

- `id`
- `tenant_id`
- `app_id`
- `name`
- `mode`: `count` or `block`
- `rule_group_ids`
- `ip_set_ids`
- `rate_limit_ids`
- `enabled`

Mode behavior:

- `count` records matching decisions but allows the request
- `block` enforces blocking decisions when rules match

## Rule Group

A rule group is an ordered collection of rules. BedemWAF will support managed
OWASP CRS-compatible groups and future custom defensive rules.

Fields to define:

- `id`
- `tenant_id`
- `name`
- `source`: `managed` or `custom`
- `version`
- `rules`
- `enabled`

## Rule

A rule is a single match or inspection unit.

Fields to define:

- `id`
- `rule_group_id`
- `name`
- `description`
- `severity`
- `action`: `count`, `block`, or `allow`
- `enabled`

Initial MVP rule execution should be delegated to Coraza and CRS-compatible rule
files rather than hand-rolled parsing.

## IP Set

An IP set is a reusable collection of CIDR ranges.

Fields to define:

- `id`
- `tenant_id`
- `name`
- `description`
- `cidrs`
- `action`: `allow`, `block`, or `count`

## Rate Limit

A rate limit defines request thresholds over a time window.

Fields to define:

- `id`
- `tenant_id`
- `app_id`
- `name`
- `key`: `source_ip`, `host`, `path`, or future composite keys
- `limit`
- `window_seconds`
- `action`: `count` or `block`

## Evaluation Order

```text
Request
  |
  v
Resolve tenant/app by hostname
  |
  v
Resolve active policy
  |
  v
Evaluate IP sets
  |
  v
Evaluate rate limits
  |
  v
Run WAF rule groups
  |
  v
Apply policy mode and rule action
```

TODO:

- Define conflict resolution between allow/block IP sets
- Define rule priority and override model
- Define policy versioning and rollback semantics
- Define generated gateway configuration snapshot format
