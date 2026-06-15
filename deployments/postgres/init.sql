-- BedemWAF local development database bootstrap.
-- TODO: replace with versioned migrations for tenants, apps, origins, policies,
-- rule groups, IP sets, rate limits, users, and API audit metadata.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
