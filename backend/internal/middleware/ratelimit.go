package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"net"
	"strconv"
	"time"
)

// RateLimitWindow is the fixed window used by every limiter in the application.
const RateLimitWindow = time.Minute

// PerUser limits authenticated traffic by session user. One student's load never
// consumes another student's budget, even when a whole classroom shares the same
// public address. It must be registered after Session so the user is resolved.
func PerUser(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          max,
		Expiration:   RateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string { return "user:" + strconv.FormatInt(User(c).ID, 10) },
		LimitReached: tooManyRequests,
	})
}

// PerIP limits unauthenticated traffic by the real client address.
func PerIP(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          max,
		Expiration:   RateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string { return "ip:" + ClientIP(c) },
		LimitReached: tooManyRequests,
	})
}

// ClientIP returns the client address. It honors X-Forwarded-For only when the
// direct peer is a configured trusted proxy, which Fiber decides, and always
// falls back to the peer when the resolved value is not a real address. A client
// cannot move to another bucket by rotating a spoofed header.
func ClientIP(c *fiber.Ctx) string {
	if ip := net.ParseIP(c.IP()); ip != nil {
		return ip.String()
	}
	if peer := c.Context().RemoteIP(); peer != nil {
		return peer.String()
	}
	return "unknown"
}

// tooManyRequests answers 429 announcing the whole window, the conservative wait
// a well-behaved client should observe.
func tooManyRequests(c *fiber.Ctx) error {
	c.Set(fiber.HeaderRetryAfter, strconv.Itoa(int(RateLimitWindow.Seconds())))
	return fiber.ErrTooManyRequests
}
