package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Default budgets. The per-user budget covers the server-render fan-out of a
// private page; the per-IP budget only applies to unauthenticated traffic.
const (
	DefaultRateLimitUser = 600
	DefaultRateLimitIP   = 180
	maxRateLimit         = 100000
)

type Config struct {
	DatabaseURL, Origin, Address string
	TrustedProxies               []string
	Production                   bool
	RateLimitUser                int
	RateLimitIP                  int
}

func Load() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), Origin: os.Getenv("APP_ORIGIN"), Address: os.Getenv("LISTEN_ADDR"), Production: os.Getenv("APP_ENV") == "production"}
	if c.Address == "" {
		c.Address = "127.0.0.1:8080"
	}
	u, e := url.Parse(c.Origin)
	if c.DatabaseURL == "" || e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return c, fmt.Errorf("DATABASE_URL y APP_ORIGIN válidos son obligatorios")
	}
	if c.Production && u.Scheme != "https" {
		return c, fmt.Errorf("producción requiere HTTPS")
	}
	if c.RateLimitUser, e = rateLimit("RATE_LIMIT_USER", DefaultRateLimitUser); e != nil {
		return c, e
	}
	if c.RateLimitIP, e = rateLimit("RATE_LIMIT_IP", DefaultRateLimitIP); e != nil {
		return c, e
	}
	for _, proxy := range splitList(os.Getenv("TRUSTED_PROXIES")) {
		if !validProxy(proxy) {
			return c, fmt.Errorf("TRUSTED_PROXIES contiene un valor que no es una dirección ni un rango CIDR: %q", proxy)
		}
		c.TrustedProxies = append(c.TrustedProxies, proxy)
	}
	return c, nil
}

// Budgets returns the effective rate limits. A non-positive value falls back to
// the documented default: Fiber's limiter silently treats Max <= 0 as its own
// default of five requests, so an incompletely built Config must never reach it.
func (c Config) Budgets() (user, ip int) {
	user, ip = c.RateLimitUser, c.RateLimitIP
	if user < 1 {
		user = DefaultRateLimitUser
	}
	if ip < 1 {
		ip = DefaultRateLimitIP
	}
	return user, ip
}

// rateLimit reads a positive budget and keeps the default when unset. An invalid
// value stops the process instead of silently weakening a protection.
func rateLimit(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, e := strconv.Atoi(raw)
	if e != nil || value < 1 || value > maxRateLimit {
		return 0, fmt.Errorf("%s debe ser un entero entre 1 y %d", name, maxRateLimit)
	}
	return value, nil
}

func splitList(raw string) []string {
	out := []string{}
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// validProxy accepts a single address or a CIDR range, the two forms Fiber
// understands for trusted proxies.
func validProxy(value string) bool {
	if _, _, e := net.ParseCIDR(value); e == nil {
		return true
	}
	return net.ParseIP(value) != nil
}
