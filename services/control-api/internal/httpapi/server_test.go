package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/bedemwaf/bedemwaf/services/control-api/internal/auth"
	"github.com/bedemwaf/bedemwaf/services/control-api/internal/db"
	"github.com/bedemwaf/bedemwaf/services/control-api/internal/models"
)

func TestHealthDoesNotRequireAuth(t *testing.T) {
	handler := testServer(t).Routes()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestV1RequiresBearerToken(t *testing.T) {
	handler := testServer(t).Routes()
	req := httptest.NewRequest(http.MethodGet, "/v1/tenants", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	assertErrorShape(t, rec.Body.Bytes(), "unauthorized")
}

func TestCreateTenant(t *testing.T) {
	handler := testServer(t).Routes()
	body := bytes.NewBufferString(`{"name":"Demo Tenant","slug":"demo"}`)
	req := authedRequest(http.MethodPost, "/v1/tenants", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var got models.Tenant
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Slug != "demo" || got.ID == "" {
		t.Fatalf("tenant = %+v, want created tenant", got)
	}
}

func TestCreateAppValidatesHostnameAndOrigin(t *testing.T) {
	handler := testServer(t).Routes()
	body := bytes.NewBufferString(`{
		"tenant_id":"tenant-1",
		"name":"Bad App",
		"slug":"bad-app",
		"hostnames":["https://example.local/path"],
		"origin_url":"ftp://origin.local"
	}`)
	req := authedRequest(http.MethodPost, "/v1/apps", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	assertErrorShape(t, rec.Body.Bytes(), "invalid_request")
}

func TestCreatePolicyValidatesModeActionAndCIDRInSnapshot(t *testing.T) {
	handler := testServer(t).Routes()
	body := bytes.NewBufferString(`{
		"name":"Bad Policy",
		"mode":"block",
		"snapshot":{
			"ip_blocklist":["not-cidr"],
			"rules":[{"action":"explode"}]
		}
	}`)
	req := authedRequest(http.MethodPost, "/v1/apps/app-1/policies", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	assertErrorShape(t, rec.Body.Bytes(), "invalid_request")
}

func TestReadyzUsesRepositoryPing(t *testing.T) {
	handler := testServer(t).Routes()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func testServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(newFakeRepo(), auth.NewStaticBearer("test-admin-key"), nil)
}

func authedRequest(method, path string, body *bytes.Buffer) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer test-admin-key")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assertErrorShape(t *testing.T, data []byte, code string) {
	t.Helper()
	var got struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode error response: %v\n%s", err, string(data))
	}
	if got.Error.Code != code || got.Error.Message == "" || got.Error.RequestID == "" {
		t.Fatalf("error response = %+v, want code %q with message and request_id", got.Error, code)
	}
}

type fakeRepo struct {
	tenants []models.Tenant
	apps    []models.App
}

func newFakeRepo() *fakeRepo {
	now := time.Now().UTC()
	return &fakeRepo{
		tenants: []models.Tenant{{ID: "tenant-1", Name: "Demo", Slug: "demo", Status: "active", CreatedAt: now, UpdatedAt: now}},
		apps: []models.App{{
			ID: "app-1", TenantID: "tenant-1", Name: "Demo App", Slug: "demo-app",
			Hostnames: []string{"example.local"}, Status: "active", CreatedAt: now, UpdatedAt: now,
		}},
	}
}

func (f *fakeRepo) Ping(context.Context) error { return nil }
func (f *fakeRepo) Close()                     {}

func (f *fakeRepo) ListTenants(context.Context) ([]models.Tenant, error) {
	return f.tenants, nil
}

func (f *fakeRepo) CreateTenant(_ context.Context, req models.CreateTenantRequest) (models.Tenant, error) {
	now := time.Now().UTC()
	tenant := models.Tenant{ID: "tenant-created", Name: req.Name, Slug: req.Slug, Status: "active", Metadata: req.Metadata, CreatedAt: now, UpdatedAt: now}
	f.tenants = append(f.tenants, tenant)
	return tenant, nil
}

func (f *fakeRepo) ListApps(context.Context) ([]models.App, error) {
	return f.apps, nil
}

func (f *fakeRepo) CreateApp(_ context.Context, req models.CreateAppRequest, originURL *url.URL) (models.App, error) {
	now := time.Now().UTC()
	app := models.App{
		ID: "app-created", TenantID: req.TenantID, Name: req.Name, Slug: req.Slug,
		Hostnames: req.Hostnames, Status: "active", CreatedAt: now, UpdatedAt: now,
		Origins: []models.Origin{{Name: "primary", Scheme: originURL.Scheme, Host: originURL.Hostname(), URL: originURL.String()}},
	}
	f.apps = append(f.apps, app)
	return app, nil
}

func (f *fakeRepo) GetApp(_ context.Context, id string) (models.App, error) {
	for _, app := range f.apps {
		if app.ID == id {
			return app, nil
		}
	}
	return models.App{}, db.ErrNotFound
}

func (f *fakeRepo) UpdateApp(ctx context.Context, id string, req models.UpdateAppRequest, originURL *url.URL) (models.App, error) {
	app, err := f.GetApp(ctx, id)
	if err != nil {
		return models.App{}, err
	}
	if req.Name != nil {
		app.Name = *req.Name
	}
	if len(req.Hostnames) > 0 {
		app.Hostnames = req.Hostnames
	}
	if originURL != nil {
		app.Origins = []models.Origin{{Name: "primary", Scheme: originURL.Scheme, Host: originURL.Hostname(), URL: originURL.String()}}
	}
	return app, nil
}

func (f *fakeRepo) ListPoliciesByApp(context.Context, string) ([]models.Policy, error) {
	return []models.Policy{}, nil
}

func (f *fakeRepo) CreatePolicy(_ context.Context, appID string, req models.CreatePolicyRequest) (models.Policy, error) {
	now := time.Now().UTC()
	return models.Policy{ID: "policy-created", TenantID: "tenant-1", AppID: appID, Name: req.Name, Mode: req.Mode, Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (f *fakeRepo) GetPolicy(context.Context, string) (models.Policy, error) {
	now := time.Now().UTC()
	return models.Policy{ID: "policy-1", TenantID: "tenant-1", AppID: "app-1", Name: "Default", Mode: "count", Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (f *fakeRepo) PublishPolicy(context.Context, string) (models.PublishPolicyResponse, error) {
	return models.PublishPolicyResponse{PolicyID: "policy-1", PolicyVersionID: "version-1", Version: 1, PublishedAt: time.Now().UTC()}, nil
}

func (f *fakeRepo) ListEvents(context.Context, int) ([]models.EventRef, error) {
	return []models.EventRef{}, nil
}

func (f *fakeRepo) GetEvent(context.Context, string) (models.EventRef, error) {
	return models.EventRef{}, db.ErrNotFound
}
