# Events

BedemWAF stores high-volume gateway audit events in ClickHouse. The gateway can
still emit newline-delimited JSON to stdout for local development and log
collection, while production deployments can enable the ClickHouse audit sink.

## Pipeline

```text
Gateway request decision
  |
  v
audit.Event
  |
  +--> JSON stdout sink
  |
  +--> ClickHouse sink
          |
          v
      bedemwaf.waf_events
          |
          v
      Control API /v1/events
```

## ClickHouse Table

The local Docker Compose stack initializes `bedemwaf.waf_events` from
`deployments/clickhouse/init.sql`.

Key columns:

- `timestamp`
- `request_id`
- `tenant_id`
- `app_id`
- `policy_id`
- `policy_version_id`
- `host`
- `client_ip`
- `method`
- `path`
- `action`
- `mode`
- `status`
- `reason`
- `matched_rule_id`
- `matched_rule_name`
- `rule_group`
- `tags`
- `anomaly_score`
- `user_agent`
- `latency_ms`
- `origin_status`
- `origin_latency_ms`

Full request bodies are not written to ClickHouse. Query strings are already
redacted in gateway audit events, and the ClickHouse schema intentionally omits
request-body fields.

## Gateway Configuration

Stdout JSON remains enabled. ClickHouse is optional:

```yaml
clickhouse:
  enabled: true
  url: "http://localhost:8123"
  database: "bedemwaf"
  username: "bedemwaf"
  password: "${BEDEMWAF_CLICKHOUSE_PASSWORD}"
```

The gateway writes events with ClickHouse `JSONEachRow`. If ClickHouse is
temporarily unavailable, the async audit dispatcher logs a warning and request
processing continues.

## Control API Search

`GET /v1/events` requires the admin bearer token and supports these filters:

- `tenant_id`
- `app_id`
- `host`
- `action`
- `client_ip`
- `matched_rule_id`
- `from`
- `to`
- `limit`

Defaults:

- `limit` defaults to `100`.
- `limit` is capped at `1000`.
- `from` and `to` must be RFC3339 timestamps.
- `from` must be before `to` when both are provided.

Example:

```bash
curl -s "$API/v1/events?tenant_id=$TENANT_ID&action=block&limit=50" \
  -H "Authorization: Bearer $BEDEMWAF_ADMIN_API_KEY"
```

Fetch by request ID:

```bash
curl -s "$API/v1/events/$REQUEST_ID" \
  -H "Authorization: Bearer $BEDEMWAF_ADMIN_API_KEY"
```

The Control API uses ClickHouse HTTP query parameters for filter values instead
of string-interpolating user input into SQL.

## Later Phase

- Batched gateway inserts.
- Compression for ClickHouse HTTP writes.
- Event retention settings per tenant.
- Event enrichment jobs in the worker.
- Cursor-based pagination for large result sets.
