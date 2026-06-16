# Production Checklist

Use this checklist before running BedemWAF in front of real applications.

## Network And Origin Locking

- Put BedemWAF Gateway in the only public ingress path to the application.
- Lock NGINX origins so they accept traffic only from BedemWAF gateway IPs or the private gateway network.
- Reject direct public access to origins at firewall, security group, or load balancer level.
- Preserve the original client IP only through trusted proxy CIDRs.
- Do not trust `X-Forwarded-For` from arbitrary clients.

## TLS

- Terminate TLS at a trusted load balancer or gateway listener.
- Use modern TLS versions and managed certificate renewal.
- Redirect HTTP to HTTPS where applicable.
- Set HSTS only after confirming all tenant hostnames are HTTPS-ready.
- Ensure Control API and Dashboard are HTTPS-only outside local development.

## Access Control

- Replace MVP static admin API key with proper user session auth before production.
- Store gateway API keys in a secret manager.
- Rotate API keys regularly.
- Restrict Control API access by network policy, VPN, SSO, or private ingress.
- Use least-privilege database users.
- Require MFA for dashboard users once real auth exists.

## Policy Safety

- Start new policies in count mode.
- Review audit events before switching to block mode.
- Keep emergency rollback paths for active policy deployments.
- Validate custom rules in staging before publishing.
- Do not enable body preview logging in production unless a short-lived incident procedure requires it.

## Data Protection

- Do not log full request bodies.
- Redact sensitive query parameters and headers.
- Define retention periods for ClickHouse audit events.
- Encrypt disks or volumes that store Postgres and ClickHouse data.
- Back up Postgres configuration data.
- Back up ClickHouse event data if audit retention requirements demand it.
- Test restore procedures.

## Monitoring

- Monitor gateway allowed, blocked, counted, and rate-limited request totals.
- Monitor audit event queue drops.
- Alert on Redis, ClickHouse, Postgres, and Control API outages.
- Alert on gateway policy fetch failures and stale policy use.
- Track origin 5xx rates and origin latency.
- Centralize structured service logs.

## Runtime Hardening

- Run containers as non-root where practical.
- Use read-only filesystems where practical.
- Mount only required configuration files.
- Avoid exposing Postgres, Redis, and ClickHouse to public networks.
- Apply CPU and memory limits in the production orchestrator.
- Keep images patched and rebuild regularly.

## Database Operations

- Run migrations as an explicit deployment step.
- Restrict migration privileges.
- Monitor Postgres connection pool saturation.
- Monitor ClickHouse disk growth and partition retention.
- Keep Redis protected on a private network.

## Incident Readiness

- Keep a documented process to switch policies back to count mode.
- Keep a documented process to disable a faulty managed rule set.
- Preserve request IDs in user-facing errors and audit events.
- Ensure on-call engineers can find gateway, Control API, and ClickHouse logs.

## MVP Blockers Before Real Production

- Replace dashboard localStorage API key auth.
- Add persistent admin API rate limiting.
- Add role-based authorization.
- Add production deployment manifests.
- Add automated database migration workflow.
- Add backup and restore automation.
- Add managed rules update verification.
