package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/middleware"
	"aulaquest/internal/repositories"
	"aulaquest/internal/runner"
	"aulaquest/internal/services"
	"aulaquest/internal/ws"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

var attachmentUploadPath = regexp.MustCompile(`^/api/v1/(lessons/[1-9][0-9]*/submission/attachments|teacher/activities/[1-9][0-9]*/attachments)$`)

// friendlyError keeps the child-facing wording for framework errors. A handler
// that raises its own message keeps that message instead, so the specific texts
// for a closed task, a closed quiz or a duplicate group name actually reach the
// client.
var friendlyError = map[int]string{
	400: "Revisa los datos enviados",
	401: "Revisa tu acceso o vuelve a iniciar sesión",
	403: "No tienes acceso a esta acción",
	404: "Esta aventura no está disponible",
	409: "Hay cambios más recientes o esta actividad ya no admite esa acción. Revisa la versión guardada antes de continuar",
	413: "El archivo o la solicitud supera el tamaño permitido",
	429: "Hagamos una pausa. Inténtalo en un minuto",
	503: "No podemos conectar. Vuelve a intentarlo",
}

func New(db *gorm.DB, cfg config.Config) (*fiber.App, *ws.Hub) {
	// Client addresses come from X-Forwarded-For only when the direct peer is a
	// configured trusted proxy. Leaving ProxyHeader unset otherwise is essential:
	// Fiber reads "no trusted proxy check" as "trust everything", so a permanent
	// ProxyHeader would let any client rotate the header to escape its limit.
	trustedProxies := len(cfg.TrustedProxies) > 0
	proxyHeader := ""
	if trustedProxies {
		proxyHeader = fiber.HeaderXForwardedFor
	}
	app := fiber.New(fiber.Config{DisableStartupMessage: true, BodyLimit: services.MaxAttachmentBytes + 64*1024, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, ProxyHeader: proxyHeader, EnableTrustedProxyCheck: trustedProxies, TrustedProxies: cfg.TrustedProxies, ErrorHandler: func(c *fiber.Ctx, e error) error {
		status := 500
		message := "No se pudo completar la solicitud"
		var f *fiber.Error
		if errors.As(e, &f) {
			status = f.Code
			if f.Message != "" && f.Message != http.StatusText(f.Code) {
				message = f.Message
			} else if friendly := friendlyError[status]; friendly != "" {
				message = friendly
			}
		}
		return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": http.StatusText(status), "message": message, "requestId": c.GetRespHeader("X-Request-ID")}})
	}})
	app.Use(requestid.New(), recover.New())
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		c.Set("X-Content-Type-Options", "nosniff")
		start := time.Now()
		e := c.Next()
		status := c.Response().StatusCode()
		if e != nil {
			status = 500
			var f *fiber.Error
			if errors.As(e, &f) {
				status = f.Code
			}
		}
		slog.Info("request", "requestId", c.GetRespHeader("X-Request-ID"), "method", c.Method(), "status", status, "duration_ms", time.Since(start).Milliseconds())
		return e
	})
	app.Use(func(c *fiber.Ctx) error {
		if len(c.Body()) > 128*1024 && !(c.Method() == "POST" && attachmentUploadPath.MatchString(c.Path())) {
			return fiber.ErrRequestEntityTooLarge
		}
		return c.Next()
	})
	// Rate limits live on the route groups: the public group is keyed by client
	// address and the private group by session user, so a classroom behind one
	// address no longer shares a single budget. Liveness probes stay unlimited on
	// purpose: an orchestrator poll must not consume anyone's quota.
	_, rateLimitIP := cfg.Budgets()
	repo := repositories.Repository{DB: db}
	hub := ws.New(repo, runner.MockRunner{})
	(API{Repo: repo, Config: cfg, Revoke: hub.Revoke}).Register(app)
	hub.Register(app, cfg.Origin, middleware.PerIP(rateLimitIP))
	return app, hub
}
