package middleware

import (
	"aulaquest/internal/models"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Hash(token string) string { s := sha256.Sum256([]byte(token)); return hex.EncodeToString(s[:]) }
func Session(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var u models.User
		token := c.Cookies("aq_session")
		if len(token) != 64 {
			return fiber.ErrUnauthorized
		}
		e := db.Raw("SELECT u.* FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>now()", Hash(token)).Scan(&u).Error
		if e != nil {
			return fiber.ErrServiceUnavailable
		}
		if u.ID == 0 {
			return fiber.ErrUnauthorized
		}
		c.Locals("user", u)
		c.Locals("sessionHash", Hash(token))
		return c.Next()
	}
}
func User(c *fiber.Ctx) models.User { return c.Locals("user").(models.User) }
func Student(c *fiber.Ctx) error {
	if User(c).Role != "student" {
		return fiber.ErrForbidden
	}
	return c.Next()
}
func Teacher(c *fiber.Ctx) error {
	if User(c).Role != "teacher" {
		return fiber.ErrForbidden
	}
	return c.Next()
}
func Origin(origin string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && c.Get("Origin") != origin {
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}
