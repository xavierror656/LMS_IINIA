package config

import "testing"

func TestProductionRequiresHTTPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://local/test")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ORIGIN", "http://localhost:4321")
	if _, e := Load(); e == nil {
		t.Fatal("insecure production accepted")
	}
	t.Setenv("APP_ORIGIN", "https://school.example")
	if _, e := Load(); e != nil {
		t.Fatal(e)
	}
}
func TestOriginRejectsPath(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://local/test")
	t.Setenv("APP_ORIGIN", "https://school.example/path")
	if _, e := Load(); e == nil {
		t.Fatal("origin with path accepted")
	}
}
func TestRateLimitBudgets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://local/test")
	t.Setenv("APP_ORIGIN", "https://school.example")
	c, e := Load()
	if e != nil {
		t.Fatal(e)
	}
	if c.RateLimitUser != DefaultRateLimitUser || c.RateLimitIP != DefaultRateLimitIP {
		t.Fatalf("defaults %+v", c)
	}
	t.Setenv("RATE_LIMIT_USER", "42")
	t.Setenv("RATE_LIMIT_IP", "7")
	if c, e = Load(); e != nil || c.RateLimitUser != 42 || c.RateLimitIP != 7 {
		t.Fatalf("explicit budgets %+v %v", c, e)
	}
	for _, bad := range []string{"0", "-1", "abc", "100001", "1.5"} {
		t.Setenv("RATE_LIMIT_USER", bad)
		if _, e = Load(); e == nil {
			t.Fatalf("invalid RATE_LIMIT_USER accepted: %s", bad)
		}
	}
	// A partially built Config must never reach Fiber, which reads Max <= 0 as its
	// own default of five requests.
	if user, ip := (Config{}).Budgets(); user != DefaultRateLimitUser || ip != DefaultRateLimitIP {
		t.Fatalf("zero budgets %d %d", user, ip)
	}
	if user, ip := (Config{RateLimitUser: 5, RateLimitIP: 9}).Budgets(); user != 5 || ip != 9 {
		t.Fatalf("explicit budgets changed %d %d", user, ip)
	}
}
func TestTrustedProxiesParsing(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://local/test")
	t.Setenv("APP_ORIGIN", "https://school.example")
	t.Setenv("TRUSTED_PROXIES", " 10.0.0.1 , 172.16.0.0/12 ,, ")
	c, e := Load()
	if e != nil {
		t.Fatal(e)
	}
	if len(c.TrustedProxies) != 2 || c.TrustedProxies[0] != "10.0.0.1" || c.TrustedProxies[1] != "172.16.0.0/12" {
		t.Fatalf("proxies %+v", c.TrustedProxies)
	}
	for _, bad := range []string{"not-an-ip", "10.0.0.1/99", "::gg"} {
		t.Setenv("TRUSTED_PROXIES", bad)
		if _, e = Load(); e == nil {
			t.Fatalf("invalid trusted proxy accepted: %s", bad)
		}
	}
	t.Setenv("TRUSTED_PROXIES", "")
	if c, e = Load(); e != nil || len(c.TrustedProxies) != 0 {
		t.Fatalf("empty proxies %+v %v", c.TrustedProxies, e)
	}
}
