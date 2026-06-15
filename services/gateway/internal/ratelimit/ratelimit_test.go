package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bedemwaf/bedemwaf/services/gateway/internal/decision"
	"github.com/bedemwaf/bedemwaf/services/gateway/internal/policy"
)

func TestBelowLimitAllowed(t *testing.T) {
	limiter := testLimiter(NewMemoryStore(), false)
	got := limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), testRule("block"))
	if got.Action != decision.ActionAllow {
		t.Fatalf("action = %q, want allow", got.Action)
	}
	if got.RateLimit == nil || got.RateLimit.Remaining != 1 {
		t.Fatalf("rate limit info = %+v, want remaining 1", got.RateLimit)
	}
}

func TestOverLimitBlocked(t *testing.T) {
	limiter := testLimiter(NewMemoryStore(), false)
	rule := testRule("block")
	_ = limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	_ = limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	got := limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	if got.Action != decision.ActionRateLimit {
		t.Fatalf("action = %q, want rate_limit", got.Action)
	}
	if got.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", got.StatusCode)
	}
}

func TestCountActionLogsButAllows(t *testing.T) {
	limiter := testLimiter(NewMemoryStore(), false)
	rule := testRule("count")
	_ = limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	_ = limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	got := limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	if got.Action != decision.ActionCount {
		t.Fatalf("action = %q, want count", got.Action)
	}
}

func TestDisabledRuleIgnored(t *testing.T) {
	limiter := testLimiter(NewMemoryStore(), false)
	rule := testRule("block")
	rule.Enabled = false
	got := limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	if got.Action != decision.ActionAllow {
		t.Fatalf("action = %q, want allow", got.Action)
	}
}

func TestSeparateIPsHaveSeparateCounters(t *testing.T) {
	limiter := testLimiter(NewMemoryStore(), false)
	rule := testRule("block")
	_ = limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	_ = limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), rule)
	got := limiter.Check(context.Background(), testApp(), testRequest("198.51.100.11"), rule)
	if got.Action != decision.ActionAllow {
		t.Fatalf("second IP action = %q, want allow", got.Action)
	}
}

func TestRedisFailureFailOpen(t *testing.T) {
	limiter := testLimiter(FailingStore{Err: fmt.Errorf("redis down")}, false)
	got := limiter.Check(context.Background(), testApp(), testRequest("198.51.100.10"), testRule("block"))
	if got.Action != decision.ActionAllow {
		t.Fatalf("action = %q, want allow", got.Action)
	}
}

func TestKeyHashingDoesNotExposeRawTokens(t *testing.T) {
	store := &recordingStore{}
	limiter := testLimiter(store, false)
	rule := testRule("block")
	rule.KeyType = "api_key_placeholder"
	req := testRequest("198.51.100.10")
	req.Headers.Set("X-API-Key", "secret-token-value")

	_ = limiter.Check(context.Background(), testApp(), req, rule)

	if strings.Contains(store.key, "secret-token-value") {
		t.Fatalf("redis key exposed raw token: %q", store.key)
	}
	if !strings.Contains(store.key, HashKeyPart("secret-token-value")) {
		t.Fatalf("redis key = %q, want hashed token component", store.key)
	}
}

func testLimiter(store Store, failClosed bool) *FixedWindowLimiter {
	limiter := NewFixedWindowLimiter(store, failClosed, nil)
	limiter.now = func() time.Time { return time.Unix(120, 0).UTC() }
	return limiter
}

func testApp() *policy.App {
	return &policy.App{TenantID: "tenant-local", ID: "app-local"}
}

func testRule(action string) policy.RateLimitRule {
	return policy.RateLimitRule{
		ID:            "rl-global",
		Name:          "Global",
		Enabled:       true,
		KeyType:       "ip",
		Limit:         2,
		WindowSeconds: 60,
		Action:        decision.Action(action),
		StatusCode:    http.StatusTooManyRequests,
	}
}

func testRequest(ip string) Request {
	return Request{
		ClientIP: ip,
		Method:   http.MethodGet,
		Host:     "example.local",
		Path:     "/",
		Headers:  http.Header{},
		Query:    url.Values{},
	}
}

type recordingStore struct {
	key string
}

func (r *recordingStore) IncrementFixedWindow(_ context.Context, key string, limit int, _ int) (int64, int, error) {
	r.key = key
	return 1, limit - 1, nil
}
