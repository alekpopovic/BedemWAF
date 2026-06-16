# Security Hardening

This document tracks defensive hardening controls for BedemWAF. The goal is to make safe behavior the default for local development and provide clear production knobs before any deployment protects real applications.

## Gateway

The gateway is the highest-risk component because it receives untrusted internet traffic before forwarding requests to origins.

### Request Limits

The gateway supports a configurable request body inspection limit:

```yaml
waf:
  request_body_limit_bytes: 1048576
```

If a request body exceeds the configured limit, the gateway creates a WAF decision. In block mode it returns a deny response. In count mode it records the would-block decision and allows the request according to policy.

The gateway must not log full request bodies. Body previews are disabled by default and, when enabled for debugging, only emit a short redacted preview marker rather than raw content.

### Server Timeouts

Configure HTTP server resource limits:

```yaml
server:
  read_header_timeout_millis: 5000
  read_timeout_millis: 30000
  write_timeout_millis: 30000
  idle_timeout_millis: 60000
  max_header_bytes: 1048576
```

These settings reduce exposure to slowloris-style resource exhaustion and oversized header abuse.

### Client IP Handling

The gateway ignores `X-Forwarded-For` unless the direct peer IP matches a configured trusted proxy CIDR:

```yaml
server:
  trusted_proxies:
    - "10.0.0.0/8"
```

For MVP, deployments should leave `trusted_proxies` empty unless the gateway is behind a known load balancer or reverse proxy.

### Host Validation

The gateway normalizes the `Host` header before app lookup. Malformed hosts, hosts with paths/schemes, whitespace, or invalid label characters are rejected before policy lookup. Unknown valid hosts return a JSON 404 response.

### Panic Recovery

Gateway request handling includes panic recovery that returns a structured 500 response and logs the panic without request bodies or sensitive headers.

## Control API

The Control API is an administrative surface and should not be exposed publicly without additional access control.

### Authentication

MVP authentication uses static bearer tokens:

- `BEDEMWAF_ADMIN_API_KEY` for `/v1` administrative routes.
- `BEDEMWAF_GATEWAY_API_KEY` for gateway policy fetch routes.

Health endpoints do not require auth. Gateway policy endpoints reject admin tokens and require the separate gateway token.

TODO: replace static API keys with session auth for humans, scoped service tokens for gateways, rotation, expiration, and audit trails.

### Request Size Limit

Configure the maximum request body size:

```env
BEDEMWAF_REQUEST_BODY_LIMIT_BYTES=1048576
```

The API uses JSON decoders with unknown-field rejection and request size limiting.

### CORS

CORS is deny-by-default except for configured origins:

```env
BEDEMWAF_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

Wildcard origins are ignored. Production should set this to the exact dashboard origin.

### Error Handling

Errors use a consistent JSON shape:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "name is required",
    "request_id": "..."
  }
}
```

Internal errors are logged server-side with `request_id` and return a generic `internal_error` message to clients.

### Admin API Rate Limiting

The current code includes a placeholder at the admin auth boundary. Production should add a persistent per-token and per-client-IP limiter before exposing the Control API outside a trusted network.

## Dashboard

The dashboard sets baseline browser security headers through Next.js:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`
- restrictive `Permissions-Policy`
- a starter `Content-Security-Policy`

The MVP login page clearly warns that the admin API key is stored in `localStorage`. This is development-only. Production must replace it with proper session authentication, CSRF protection, secure cookies, and role-based authorization.

## Docker Compose

The local Compose stack is for development, but it still follows basic hardening:

- App containers run as non-root where practical.
- App containers use `read_only: true` where practical.
- Runtime scratch paths use `tmpfs`.
- `no-new-privileges` is enabled for app containers.
- Postgres, Redis, and ClickHouse are exposed only to the Compose network by default.
- Healthchecks are configured for services that support them.

Stateful databases are not read-only because they need writable data directories.

## Secrets

Do not commit real secrets. `.env.example` contains local placeholders only. Production secrets should come from a secrets manager or orchestrator secret store.

## Audit Data

Audit events must not contain:

- full request bodies
- `Authorization` headers
- `Cookie` headers
- sensitive query values such as `password`, `token`, `api_key`, `secret`, or `code`

ClickHouse event storage should be protected as security telemetry and covered by retention policy.
