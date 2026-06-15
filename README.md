# BedemWAF

Project name: BedemWAF

Description:
BedemWAF is a self-hosted managed WAF platform that sits in front of NGINX origins.
It provides WAF policies, OWASP CRS inspection, custom rules, IP sets, rate limiting,
audit logs, dashboard visibility, and safe count/block rollout.

Core architecture:
Internet → BedemWAF Gateway → NGINX Origin → Application

Main services:
- gateway: high-performance reverse proxy and WAF data plane
- control-api: REST API for tenants, apps, origins, policies, rule groups and events
- dashboard: web UI for managing policies and viewing security events
- worker: async log processing and rule update jobs
- redis: rate limiting and distributed counters
- postgres: configuration, tenants, policies, users
- clickhouse: high-volume WAF event analytics
