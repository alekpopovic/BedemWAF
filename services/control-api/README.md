# BedemWAF Control API

The Control API is the REST management plane for BedemWAF. It manages tenants,
apps, origins, policy drafts, policy publishes, and audit event references.

This is an MVP skeleton with production-oriented boundaries:

- `net/http` router and middleware
- structured JSON logging
- request IDs
- static admin bearer token authentication
- Postgres connection pool through `pgxpool`
- repository interfaces for testability
- consistent JSON error responses
- static OpenAPI spec at `../../docs/openapi.yaml`

## Configuration

Environment variables:

```bash
export BEDEMWAF_CONTROL_API_ADDR=":8081"
export BEDEMWAF_DATABASE_URL="postgres://bedemwaf:bedemwaf_dev_password@localhost:5432/bedemwaf?sslmode=disable"
export BEDEMWAF_ADMIN_API_KEY="local-dev-admin-key-change-me"
```

`BEDEMWAF_ADMIN_API_KEY` is required. Do not commit real production tokens.

## Run

From this directory:

```bash
go run ./cmd/control-api
```

From the repository root with local infrastructure:

```bash
./scripts/dev-up.sh
BEDEMWAF_ADMIN_API_KEY="local-dev-admin-key-change-me" \
BEDEMWAF_DATABASE_URL="postgres://bedemwaf:bedemwaf_dev_password@localhost:5432/bedemwaf?sslmode=disable" \
go run ./services/control-api/cmd/control-api
```

## Health

Health endpoints do not require authentication:

```bash
curl -s http://localhost:8081/healthz
curl -s http://localhost:8081/readyz
```

## Authenticated Requests

All `/v1` routes require:

```http
Authorization: Bearer <BEDEMWAF_ADMIN_API_KEY>
```

Set a shell helper:

```bash
API=http://localhost:8081
TOKEN=local-dev-admin-key-change-me
AUTH="Authorization: Bearer $TOKEN"
```

Create a tenant:

```bash
curl -s -X POST "$API/v1/tenants" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo Tenant","slug":"demo"}'
```

List tenants:

```bash
curl -s "$API/v1/tenants" -H "$AUTH"
```

Create an app and primary origin:

```bash
curl -s -X POST "$API/v1/apps" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id":"00000000-0000-0000-0000-000000000000",
    "name":"Demo App",
    "slug":"demo-app",
    "hostnames":["app.example.local"],
    "origin_url":"http://localhost:9000"
  }'
```

Create a policy draft:

```bash
curl -s -X POST "$API/v1/apps/<app_id>/policies" \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Default policy",
    "mode":"count",
    "snapshot":{
      "mode":"count",
      "ip_blocklist":["203.0.113.10/32"],
      "rules":[{"id":"rule-login","action":"count"}]
    }
  }'
```

Publish a policy:

```bash
curl -s -X POST "$API/v1/policies/<policy_id>/publish" -H "$AUTH"
```

Search event references:

```bash
curl -s "$API/v1/events?limit=50" -H "$AUTH"
curl -s "$API/v1/events/<event_id>" -H "$AUTH"
```

## Error Shape

All API errors use:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "hostnames must contain at least one hostname",
    "request_id": "..."
  }
}
```

## MVP Notes

- Authentication is a single static admin API key.
- Policy snapshots are stored as JSON in policy metadata until publish creates an
  immutable `policy_versions` row.
- Event endpoints read `audit_event_refs`; full ClickHouse event search is later
  phase work.
- The API validates hostnames, origin URLs, modes, rule actions in policy
  snapshots, and CIDR values in policy snapshots.
