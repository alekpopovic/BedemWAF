package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bedemwaf/bedemwaf/services/control-api/internal/models"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	Ping(context.Context) error
	ListTenants(context.Context) ([]models.Tenant, error)
	CreateTenant(context.Context, models.CreateTenantRequest) (models.Tenant, error)
	ListApps(context.Context) ([]models.App, error)
	CreateApp(context.Context, models.CreateAppRequest, *url.URL) (models.App, error)
	GetApp(context.Context, string) (models.App, error)
	UpdateApp(context.Context, string, models.UpdateAppRequest, *url.URL) (models.App, error)
	ListPoliciesByApp(context.Context, string) ([]models.Policy, error)
	CreatePolicy(context.Context, string, models.CreatePolicyRequest) (models.Policy, error)
	GetPolicy(context.Context, string) (models.Policy, error)
	PublishPolicy(context.Context, string) (models.PublishPolicyResponse, error)
	ListEvents(context.Context, int) ([]models.EventRef, error)
	GetEvent(context.Context, string) (models.EventRef, error)
	Close()
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func (r *PostgresRepository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *PostgresRepository) ListTenants(ctx context.Context) ([]models.Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, slug, status, metadata, created_at, updated_at
		FROM tenants
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var tenant models.Tenant
		if err := rows.Scan(&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Status, &tenant.Metadata, &tenant.CreatedAt, &tenant.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, tenant)
	}
	return tenants, rows.Err()
}

func (r *PostgresRepository) CreateTenant(ctx context.Context, req models.CreateTenantRequest) (models.Tenant, error) {
	metadata := req.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	var tenant models.Tenant
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, metadata)
		VALUES ($1, $2, $3)
		RETURNING id::text, name, slug, status, metadata, created_at, updated_at`,
		req.Name, req.Slug, metadata,
	).Scan(&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Status, &tenant.Metadata, &tenant.CreatedAt, &tenant.UpdatedAt)
	return tenant, err
}

func (r *PostgresRepository) ListApps(ctx context.Context) ([]models.App, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, tenant_id::text, name, slug, hostnames, status, metadata, created_at, updated_at
		FROM apps
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []models.App
	for rows.Next() {
		var app models.App
		if err := rows.Scan(&app.ID, &app.TenantID, &app.Name, &app.Slug, &app.Hostnames, &app.Status, &app.Metadata, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *PostgresRepository) CreateApp(ctx context.Context, req models.CreateAppRequest, originURL *url.URL) (models.App, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.App{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	metadata := req.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	var app models.App
	err = tx.QueryRow(ctx, `
		INSERT INTO apps (tenant_id, name, slug, hostnames, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, tenant_id::text, name, slug, hostnames, status, metadata, created_at, updated_at`,
		req.TenantID, req.Name, req.Slug, req.Hostnames, metadata,
	).Scan(&app.ID, &app.TenantID, &app.Name, &app.Slug, &app.Hostnames, &app.Status, &app.Metadata, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return models.App{}, err
	}
	origin, err := insertOrigin(ctx, tx, app.TenantID, app.ID, originURL)
	if err != nil {
		return models.App{}, err
	}
	app.Origins = []models.Origin{origin}
	if err := tx.Commit(ctx); err != nil {
		return models.App{}, err
	}
	return app, nil
}

func (r *PostgresRepository) GetApp(ctx context.Context, id string) (models.App, error) {
	var app models.App
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, name, slug, hostnames, status, metadata, created_at, updated_at
		FROM apps
		WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&app.ID, &app.TenantID, &app.Name, &app.Slug, &app.Hostnames, &app.Status, &app.Metadata, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.App{}, ErrNotFound
	}
	if err != nil {
		return models.App{}, err
	}
	origins, err := r.listOrigins(ctx, app.ID)
	if err != nil {
		return models.App{}, err
	}
	app.Origins = origins
	return app, nil
}

func (r *PostgresRepository) UpdateApp(ctx context.Context, id string, req models.UpdateAppRequest, originURL *url.URL) (models.App, error) {
	current, err := r.GetApp(ctx, id)
	if err != nil {
		return models.App{}, err
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if len(req.Hostnames) > 0 {
		current.Hostnames = req.Hostnames
	}
	if req.Status != nil {
		current.Status = *req.Status
	}
	metadata := current.Metadata
	if len(req.Metadata) > 0 {
		metadata = req.Metadata
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.App{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	err = tx.QueryRow(ctx, `
		UPDATE apps
		SET name = $2, hostnames = $3, status = $4, metadata = $5, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, tenant_id::text, name, slug, hostnames, status, metadata, created_at, updated_at`,
		id, current.Name, current.Hostnames, current.Status, metadata,
	).Scan(&current.ID, &current.TenantID, &current.Name, &current.Slug, &current.Hostnames, &current.Status, &current.Metadata, &current.CreatedAt, &current.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.App{}, ErrNotFound
	}
	if err != nil {
		return models.App{}, err
	}
	if originURL != nil {
		if _, err := tx.Exec(ctx, `UPDATE origins SET deleted_at = now(), updated_at = now() WHERE app_id = $1 AND deleted_at IS NULL`, id); err != nil {
			return models.App{}, err
		}
		origin, err := insertOrigin(ctx, tx, current.TenantID, current.ID, originURL)
		if err != nil {
			return models.App{}, err
		}
		current.Origins = []models.Origin{origin}
	}
	if err := tx.Commit(ctx); err != nil {
		return models.App{}, err
	}
	if originURL == nil {
		current.Origins, err = r.listOrigins(ctx, current.ID)
		if err != nil {
			return models.App{}, err
		}
	}
	return current, nil
}

func (r *PostgresRepository) ListPoliciesByApp(ctx context.Context, appID string) ([]models.Policy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, tenant_id::text, app_id::text, name, mode, enabled,
		       COALESCE(active_version_id::text, ''), created_at, updated_at
		FROM policies
		WHERE app_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []models.Policy
	for rows.Next() {
		var policy models.Policy
		if err := rows.Scan(&policy.ID, &policy.TenantID, &policy.AppID, &policy.Name, &policy.Mode, &policy.Enabled, &policy.ActiveVersionID, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func (r *PostgresRepository) CreatePolicy(ctx context.Context, appID string, req models.CreatePolicyRequest) (models.Policy, error) {
	app, err := r.GetApp(ctx, appID)
	if err != nil {
		return models.Policy{}, err
	}
	snapshot := req.Snapshot
	if len(snapshot) == 0 {
		snapshot = []byte(`{}`)
	}
	metadata := []byte(fmt.Sprintf(`{"snapshot":%s}`, snapshot))
	var policy models.Policy
	err = r.pool.QueryRow(ctx, `
		INSERT INTO policies (tenant_id, app_id, name, mode, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, tenant_id::text, app_id::text, name, mode, enabled,
		          COALESCE(active_version_id::text, ''), created_at, updated_at`,
		app.TenantID, appID, req.Name, req.Mode, metadata,
	).Scan(&policy.ID, &policy.TenantID, &policy.AppID, &policy.Name, &policy.Mode, &policy.Enabled, &policy.ActiveVersionID, &policy.CreatedAt, &policy.UpdatedAt)
	policy.Snapshot = snapshot
	return policy, err
}

func (r *PostgresRepository) GetPolicy(ctx context.Context, id string) (models.Policy, error) {
	var policy models.Policy
	err := r.pool.QueryRow(ctx, `
		SELECT p.id::text, p.tenant_id::text, p.app_id::text, p.name, p.mode, p.enabled,
		       COALESCE(p.active_version_id::text, ''), COALESCE(v.snapshot, '{}'::jsonb),
		       p.created_at, p.updated_at
		FROM policies p
		LEFT JOIN policy_versions v ON v.id = p.active_version_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`, id,
	).Scan(&policy.ID, &policy.TenantID, &policy.AppID, &policy.Name, &policy.Mode, &policy.Enabled, &policy.ActiveVersionID, &policy.Snapshot, &policy.CreatedAt, &policy.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Policy{}, ErrNotFound
	}
	return policy, err
}

func (r *PostgresRepository) PublishPolicy(ctx context.Context, id string) (models.PublishPolicyResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.PublishPolicyResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	policy, err := getPolicyForUpdate(ctx, tx, id)
	if err != nil {
		return models.PublishPolicyResponse{}, err
	}
	var version int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM policy_versions WHERE policy_id = $1`, id).Scan(&version); err != nil {
		return models.PublishPolicyResponse{}, err
	}
	snapshot := policy.Snapshot
	if len(snapshot) == 0 {
		snapshot = []byte(fmt.Sprintf(`{"policy_id":%q,"mode":%q}`, policy.ID, policy.Mode))
	}
	var versionID string
	var publishedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO policy_versions (tenant_id, app_id, policy_id, version, mode, snapshot)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at`,
		policy.TenantID, policy.AppID, policy.ID, version, policy.Mode, snapshot,
	).Scan(&versionID, &publishedAt)
	if err != nil {
		return models.PublishPolicyResponse{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE policies SET active_version_id = $2, updated_at = now() WHERE id = $1`, policy.ID, versionID); err != nil {
		return models.PublishPolicyResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.PublishPolicyResponse{}, err
	}
	return models.PublishPolicyResponse{PolicyID: policy.ID, PolicyVersionID: versionID, Version: version, PublishedAt: publishedAt}, nil
}

func (r *PostgresRepository) ListEvents(ctx context.Context, limit int) ([]models.EventRef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, tenant_id::text, COALESCE(app_id::text, ''), COALESCE(policy_id::text, ''),
		       event_id, request_id, COALESCE(source_ip::text, ''), COALESCE(host, ''),
		       COALESCE(path, ''), action, occurred_at, metadata
		FROM audit_event_refs
		ORDER BY occurred_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []models.EventRef
	for rows.Next() {
		var event models.EventRef
		if err := rows.Scan(&event.ID, &event.TenantID, &event.AppID, &event.PolicyID, &event.EventID, &event.RequestID, &event.SourceIP, &event.Host, &event.Path, &event.Action, &event.OccurredAt, &event.Metadata); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *PostgresRepository) GetEvent(ctx context.Context, eventID string) (models.EventRef, error) {
	var event models.EventRef
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, COALESCE(app_id::text, ''), COALESCE(policy_id::text, ''),
		       event_id, request_id, COALESCE(source_ip::text, ''), COALESCE(host, ''),
		       COALESCE(path, ''), action, occurred_at, metadata
		FROM audit_event_refs
		WHERE event_id = $1`, eventID,
	).Scan(&event.ID, &event.TenantID, &event.AppID, &event.PolicyID, &event.EventID, &event.RequestID, &event.SourceIP, &event.Host, &event.Path, &event.Action, &event.OccurredAt, &event.Metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.EventRef{}, ErrNotFound
	}
	return event, err
}

func (r *PostgresRepository) listOrigins(ctx context.Context, appID string) ([]models.Origin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, scheme, host, port
		FROM origins
		WHERE app_id = $1 AND deleted_at IS NULL
		ORDER BY created_at`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var origins []models.Origin
	for rows.Next() {
		var origin models.Origin
		if err := rows.Scan(&origin.ID, &origin.Name, &origin.Scheme, &origin.Host, &origin.Port); err != nil {
			return nil, err
		}
		origin.URL = origin.Scheme + "://" + origin.Host
		if origin.Port > 0 {
			origin.URL += ":" + strconv.Itoa(origin.Port)
		}
		origins = append(origins, origin)
	}
	return origins, rows.Err()
}

func insertOrigin(ctx context.Context, tx pgx.Tx, tenantID, appID string, originURL *url.URL) (models.Origin, error) {
	port := originURL.Port()
	if port == "" {
		if originURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return models.Origin{}, err
	}
	host := strings.ToLower(originURL.Hostname())
	var origin models.Origin
	err = tx.QueryRow(ctx, `
		INSERT INTO origins (tenant_id, app_id, name, scheme, host, port)
		VALUES ($1, $2, 'primary', $3, $4, $5)
		RETURNING id::text, name, scheme, host, port`,
		tenantID, appID, originURL.Scheme, host, portNumber,
	).Scan(&origin.ID, &origin.Name, &origin.Scheme, &origin.Host, &origin.Port)
	if err != nil {
		return models.Origin{}, err
	}
	origin.URL = origin.Scheme + "://" + origin.Host + ":" + strconv.Itoa(origin.Port)
	return origin, nil
}

func getPolicyForUpdate(ctx context.Context, tx pgx.Tx, id string) (models.Policy, error) {
	var policy models.Policy
	err := tx.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, app_id::text, name, mode, enabled,
		       COALESCE(active_version_id::text, ''), COALESCE(metadata->'snapshot', '{}'::jsonb), created_at, updated_at
		FROM policies
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, id,
	).Scan(&policy.ID, &policy.TenantID, &policy.AppID, &policy.Name, &policy.Mode, &policy.Enabled, &policy.ActiveVersionID, &policy.Snapshot, &policy.CreatedAt, &policy.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Policy{}, ErrNotFound
	}
	return policy, err
}
