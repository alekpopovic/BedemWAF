# BedemWAF Gateway

The gateway is the HTTP data plane for BedemWAF. The MVP gateway accepts HTTP
requests, identifies the protected app by `Host`, evaluates an in-memory policy,
optionally checks Redis-backed rate limits, inspects requests with Coraza when
enabled, emits structured JSON audit logs, and reverse proxies allowed traffic to
the configured NGINX origin.

## Run Locally

Start a simple origin:

```bash
python3 -m http.server 9000
```

Run the gateway:

```bash
go run ./cmd/gateway -config config.example.yaml
```

Send a request:

```bash
curl -H 'Host: localhost' http://localhost:8080/
```

Validate the sample Coraza rule:

```bash
curl -i -H 'Host: localhost' -H 'X-Bedem-Test: block-me' http://localhost:8080/
```

With the sample config, Coraza runs in `DetectionOnly` and the app policy runs in
`count`, so the request is logged as a would-block event but still reaches the
origin. To enforce a `403`, set `waf.rule_engine` to `On` and change the app
policy mode to `block`.

## Configuration

See [config.example.yaml](config.example.yaml).

Important defaults:

- `server.listen_addr` defaults to `:8080`.
- Client IP comes from `RemoteAddr` by default.
- `X-Forwarded-For` is only trusted when `server.trusted_proxies` is configured
  and the immediate peer IP matches that list.
- Policy mode defaults to `count`.
- Redis rate limiting is disabled unless `redis.enabled` is `true`.
- Coraza runs when `waf.enabled` is `true`.
- `waf.rule_engine` defaults to `DetectionOnly`.
- Request bodies are read only up to `waf.request_body_limit_bytes`, then
  restored before proxying.
- Full request bodies are never logged.

## Audit Logs

Audit events are newline-delimited JSON written to stdout. They include:

- `timestamp`
- `request_id`
- `app_id`
- `host`
- `client_ip`
- `method`
- `path`
- `action`
- `mode`
- `status`
- `reason`
- `matched_rule_id`
- `user_agent`
- `latency_ms`

Request bodies are not logged.

## Coraza Rules

The sample rules live in [rules/coraza.conf](rules/coraza.conf).

To use OWASP CRS-compatible rules, mount CRS into the gateway container and
uncomment the placeholder `Include` lines in `rules/coraza.conf`. See
[rules/README.md](rules/README.md).

## Custom Rules

Custom rules are defined under each app policy. They are deterministic,
priority-ordered, and do not support scripts or regex.

Supported actions:

- `allow`
- `count`
- `block`

Supported conditions:

- `method_equals`
- `path_equals`
- `path_starts_with`
- `host_equals`
- `header_contains`
- `header_equals`
- `query_parameter_contains`
- `client_ip_in_ip_set`
- `client_ip_not_in_ip_set`
- `all`
- `any`

Example:

```yaml
policy:
  mode: "count"
  default_action: "allow"
  ip_sets:
    office_ips:
      - "198.51.100.0/24"
  custom_rules:
    - id: "rule-admin-office-only"
      name: "Admin only from office IPs"
      priority: 100
      enabled: true
      action: "block"
      status_code: 403
      when:
        all:
          - path_starts_with: "/admin"
          - client_ip_not_in_ip_set: "office_ips"

    - id: "rule-block-bad-ua"
      name: "Block suspicious user agent"
      priority: 200
      enabled: true
      action: "block"
      status_code: 403
      when:
        header_contains:
          name: "User-Agent"
          value: "bad-bot-test"
```

Rules are sorted by `priority` ascending. The first matching `block` rule wins.
`count` rules are logged but do not block. An `allow` rule only short-circuits
later custom block rules when `terminal_allow: true` is set. Header names are
canonicalized before evaluation.

## Tests

```bash
go test ./...
```

## TODO

- Add hot policy reload from the control plane.
- Add richer Redis rate-limit keys.
- Add Prometheus metrics.
