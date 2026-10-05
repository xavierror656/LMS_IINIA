package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/middleware"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// rateLimitFixture prepares an isolated schema with synthetic accounts and their
// session cookies.
func rateLimitFixture(t *testing.T) (*gorm.DB, map[string]string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: academic PostgreSQL integration not executed")
	}
	parsed, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("separate _test database required")
	}
	base, e := database.Open(dsn)
	if e != nil {
		t.Fatal(e)
	}
	bp, _ := base.DB()
	t.Cleanup(func() { bp.Close() })
	schema := fmt.Sprintf("ratelimit_%d", time.Now().UnixNano())
	if e = base.Exec("CREATE SCHEMA " + schema).Error; e != nil {
		t.Fatal(e)
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	db, e := database.Open(parsed.String())
	if e != nil {
		t.Fatal(e)
	}
	pool, _ := db.DB()
	t.Cleanup(func() { pool.Close() })
	t.Setenv("APP_ENV", "test")
	t.Setenv("SEED_STUDENT_PASSWORD", "academic-student-pass")
	t.Setenv("SEED_TEACHER_PASSWORD", "academic-teacher-pass")
	for range 2 {
		if e = migrations.Apply(db); e != nil {
			t.Fatal(e)
		}
		if e = services.Seed(db); e != nil {
			t.Fatal(e)
		}
	}
	cookies := map[string]string{}
	for i, name := range []string{"profe", "luna", "sol"} {
		token := fmt.Sprintf("%064d", i+1)
		if e = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) SELECT ?,id,now()+interval '1 hour' FROM users WHERE username=?`, middleware.Hash(token), name).Error; e != nil {
			t.Fatal(e)
		}
		cookies[name] = "aq_session=" + token
	}
	return db, cookies
}

// call sends one request through the real application and reports the status and
// the Retry-After header.
func call(t *testing.T, app *fiber.App, method, path, body, forwarded, cookie string) (int, string) {
	t.Helper()
	req := httptestRequest(method, path, body, cookie, "http://localhost:4321")
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	res, e := app.Test(req, 10000)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	return res.StatusCode, res.Header.Get(fiber.HeaderRetryAfter)
}

// TSEC1: authenticated traffic is budgeted per user, so one classroom address is
// not a shared quota; unauthenticated traffic stays budgeted per address; the
// stricter login limit survives; and liveness probes never consume a budget.
func TestRateLimitIntegration(t *testing.T) {
	db, cookies := rateLimitFixture(t)
	// Small budgets make the boundaries observable in a handful of requests.
	app, hub := New(db, config.Config{Origin: "http://localhost:4321", RateLimitUser: 3, RateLimitIP: 30})
	defer hub.Close()
	defer app.Shutdown()
	for i := 1; i <= 3; i++ {
		if status, _ := call(t, app, "GET", "/api/v1/auth/session", "", "", cookies["luna"]); status != 200 {
			t.Fatalf("request %d for luna got %d", i, status)
		}
	}
	status, retry := call(t, app, "GET", "/api/v1/auth/session", "", "", cookies["luna"])
	if status != 429 || retry != "60" {
		t.Fatalf("luna exceeded her budget: %d %q", status, retry)
	}
	// Two other accounts keep full budgets even though every request comes from the
	// same address: the classroom is no longer one shared quota.
	if status, _ := call(t, app, "GET", "/api/v1/auth/session", "", "", cookies["profe"]); status != 200 {
		t.Fatalf("profe shared luna's exhausted budget: %d", status)
	}
	if status, _ := call(t, app, "GET", "/api/v1/auth/session", "", "", cookies["sol"]); status != 200 {
		t.Fatalf("sol shared luna's exhausted budget: %d", status)
	}
	// The strict login limit (10 per minute per address) is still enforced and is
	// reached before the generous address budget of this app.
	bad := `{"username":"luna","password":"wrong-password-value"}`
	for i := 1; i <= 10; i++ {
		if status, _ := call(t, app, "POST", "/api/v1/auth/login", bad, "", ""); status != 401 {
			t.Fatalf("login attempt %d got %d", i, status)
		}
	}
	if status, retry = call(t, app, "POST", "/api/v1/auth/login", bad, "", ""); status != 429 || retry != "60" {
		t.Fatalf("login limiter relaxed: %d %q", status, retry)
	}
	// Liveness probes answer repeatedly: an orchestrator poll must not be throttled
	// nor consume the budget of real users.
	for i := 1; i <= 8; i++ {
		if status, _ := call(t, app, "GET", "/healthz", "", "", ""); status != 200 {
			t.Fatalf("healthz probe %d got %d", i, status)
		}
	}
}

// TSEC1: X-Forwarded-For is honored only from a configured trusted proxy. Without
// one, rotating the header must not create fresh budgets.
func TestForwardedHeaderTrustIntegration(t *testing.T) {
	db, cookies := rateLimitFixture(t)
	bad := `{"username":"luna","password":"wrong-password-value"}`
	// No trusted proxy: the header is ignored, so the third address is still the
	// second request of the same bucket.
	untrusted, hubA := New(db, config.Config{Origin: "http://localhost:4321", RateLimitUser: 3, RateLimitIP: 2})
	defer hubA.Close()
	defer untrusted.Shutdown()
	for _, forwarded := range []string{"1.1.1.1", "2.2.2.2"} {
		if status, _ := call(t, untrusted, "POST", "/api/v1/auth/login", bad, forwarded, ""); status != 401 {
			t.Fatalf("spoofed header moved the bucket: %s got %d", forwarded, status)
		}
	}
	if status, retry := call(t, untrusted, "POST", "/api/v1/auth/login", bad, "3.3.3.3", ""); status != 429 || retry != "60" {
		t.Fatalf("spoofed header escaped the limit: %d %q", status, retry)
	}
	// A configured trusted proxy is believed: distinct forwarded addresses are
	// distinct clients and each keeps its own budget.
	trusted, hubB := New(db, config.Config{Origin: "http://localhost:4321", RateLimitUser: 3, RateLimitIP: 2, TrustedProxies: []string{"0.0.0.0"}})
	defer hubB.Close()
	defer trusted.Shutdown()
	for i := 1; i <= 2; i++ {
		if status, _ := call(t, trusted, "POST", "/api/v1/auth/login", bad, "1.1.1.1", ""); status != 401 {
			t.Fatalf("trusted request %d got %d", i, status)
		}
	}
	if status, _ := call(t, trusted, "POST", "/api/v1/auth/login", bad, "1.1.1.1", ""); status != 429 {
		t.Fatalf("trusted client not limited: %d", status)
	}
	if status, _ := call(t, trusted, "POST", "/api/v1/auth/login", bad, "9.9.9.9", ""); status != 401 {
		t.Fatalf("second client behind the trusted proxy got %d", status)
	}
	// Authenticated traffic keeps working through the trusted proxy.
	if status, _ := call(t, trusted, "GET", "/api/v1/auth/session", "", "7.7.7.7", cookies["luna"]); status != 200 {
		t.Fatalf("session behind a trusted proxy got %d", status)
	}
}
