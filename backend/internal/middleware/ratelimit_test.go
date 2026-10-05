package middleware

import (
	"aulaquest/internal/models"
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"strconv"
	"testing"
)

// fakeSession resolves a user from a header so the per-user limiter can be
// exercised without a database. It must run before PerUser.
func fakeSession(c *fiber.Ctx) error {
	id, _ := strconv.ParseInt(c.Get("X-Test-User"), 10, 64)
	c.Locals("user", models.User{ID: id})
	return c.Next()
}

func request(t *testing.T, app *fiber.App, user, forwarded string) (int, string, string) {
	t.Helper()
	req := httptest.NewRequest("GET", "/x", nil)
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	res, e := app.Test(req, 5000)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	return res.StatusCode, res.Header.Get(fiber.HeaderRetryAfter), res.Header.Get("X-Client-IP")
}

// serve builds an app with the given middleware applied to the probed route. The
// handler is registered after the middleware, the order Fiber requires.
func serve(t *testing.T, cfg fiber.Config, handlers ...fiber.Handler) *fiber.App {
	t.Helper()
	app := fiber.New(cfg)
	if len(handlers) > 0 {
		args := make([]interface{}, len(handlers))
		for i, handler := range handlers {
			args[i] = handler
		}
		app.Use(args...)
	}
	app.Get("/x", func(c *fiber.Ctx) error { c.Set("X-Client-IP", ClientIP(c)); return c.SendString("ok") })
	return app
}

func TestPerUserBudgetsAreIndependent(t *testing.T) {
	app := serve(t, fiber.Config{DisableStartupMessage: true}, fakeSession, PerUser(2))
	for i := 1; i <= 2; i++ {
		if status, _, _ := request(t, app, "7", ""); status != 200 {
			t.Fatalf("request %d for user 7 got %d", i, status)
		}
	}
	status, retry, _ := request(t, app, "7", "")
	if status != 429 || retry != "60" {
		t.Fatalf("expected 429 with Retry-After 60, got %d %q", status, retry)
	}
	// Another user keeps a full budget: one classroom address is not one quota.
	for i := 1; i <= 2; i++ {
		if status, _, _ := request(t, app, "8", ""); status != 200 {
			t.Fatalf("request %d for user 8 got %d", i, status)
		}
	}
	if status, _, _ := request(t, app, "8", ""); status != 429 {
		t.Fatalf("user 8 exceeded its own budget: %d", status)
	}
	if status, _, _ := request(t, app, "7", ""); status != 429 {
		t.Fatalf("user 7 recovered early: %d", status)
	}
}

// TestPerIPIgnoresSpoofedHeaderWithoutTrustedProxy mirrors how the application
// wires Fiber without trusted proxies: no ProxyHeader at all, because Fiber reads
// a disabled trusted-proxy check as "trust every peer".
func TestPerIPIgnoresSpoofedHeaderWithoutTrustedProxy(t *testing.T) {
	app := serve(t, fiber.Config{DisableStartupMessage: true}, PerIP(2))
	for _, forwarded := range []string{"1.1.1.1", "2.2.2.2"} {
		if status, _, ip := request(t, app, "", forwarded); status != 200 || ip != "0.0.0.0" {
			t.Fatalf("rotating header moved the bucket: %s got %d %q", forwarded, status, ip)
		}
	}
	for _, forwarded := range []string{"3.3.3.3", "garbage", ""} {
		status, retry, _ := request(t, app, "", forwarded)
		if status != 429 || retry != "60" {
			t.Fatalf("spoofed header escaped the limit: %q got %d %q", forwarded, status, retry)
		}
	}
}

func TestPerIPHonorsHeaderFromTrustedProxy(t *testing.T) {
	app := serve(t, fiber.Config{DisableStartupMessage: true, ProxyHeader: fiber.HeaderXForwardedFor, EnableTrustedProxyCheck: true, TrustedProxies: []string{"0.0.0.0"}}, PerIP(2))
	for i := 1; i <= 2; i++ {
		if status, _, ip := request(t, app, "", "1.1.1.1"); status != 200 || ip != "1.1.1.1" {
			t.Fatalf("trusted request %d got %d %q", i, status, ip)
		}
	}
	if status, _, _ := request(t, app, "", "1.1.1.1"); status != 429 {
		t.Fatalf("trusted address not limited: %d", status)
	}
	// A different forwarded address is a different client behind the same proxy.
	if status, _, ip := request(t, app, "", "2.2.2.2"); status != 200 || ip != "2.2.2.2" {
		t.Fatalf("second client behind the proxy got %d %q", status, ip)
	}
}

func TestClientIPFallsBackToPeer(t *testing.T) {
	app := serve(t, fiber.Config{DisableStartupMessage: true, ProxyHeader: fiber.HeaderXForwardedFor, EnableTrustedProxyCheck: true, TrustedProxies: []string{"0.0.0.0"}})
	for _, tc := range []struct{ forwarded, want string }{
		{"9.9.9.9", "9.9.9.9"},
		{"2001:db8::1", "2001:db8::1"},
		// Only a single address is honored; anything else falls back to the peer, so
		// a spoofed or multi-value header cannot move the bucket.
		{"9.9.9.9, 8.8.8.8", "0.0.0.0"},
		{"garbage", "0.0.0.0"},
		{"", "0.0.0.0"},
	} {
		if _, _, ip := request(t, app, "", tc.forwarded); ip != tc.want {
			t.Fatalf("forwarded %q resolved to %q, want %q", tc.forwarded, ip, tc.want)
		}
	}
}
