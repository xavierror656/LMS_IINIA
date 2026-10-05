package ws

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/repositories"
	"aulaquest/internal/runner"
	"bytes"
	"encoding/json"
	socket "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"io"
	"sync"
)

type ClientEvent struct {
	V         int             `json:"v"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId"`
	RunID     string          `json:"runId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}
type Event struct {
	V         int    `json:"v"`
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	RunID     string `json:"runId,omitempty"`
	Seq       int    `json:"seq"`
	Payload   any    `json:"payload"`
}
type connection struct {
	hash string
	conn *socket.Conn
}
type Hub struct {
	Repo    repositories.Repository
	Runner  runner.Runner
	mu      sync.Mutex
	clients map[int64]connection
	closed  bool
}

func New(repo repositories.Repository, r runner.Runner) *Hub {
	return &Hub{Repo: repo, Runner: r, clients: map[int64]connection{}}
}
func (h *Hub) Revoke(hash string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.clients {
		if c.hash == hash {
			c.conn.Close()
		}
	}
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, c := range h.clients {
		c.conn.Close()
	}
}

// Register mounts the code socket. The caller passes the rate limiter that keeps
// the upgrade attempt bounded per client address, since the socket sits outside
// the authenticated API group.
func (h *Hub) Register(app *fiber.App, origin string, limit fiber.Handler) {
	app.Get("/ws/code", limit, middleware.Session(h.Repo.DB), middleware.Student, func(c *fiber.Ctx) error {
		if c.Get("Origin") != origin {
			return fiber.ErrForbidden
		}
		if !socket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		return c.Next()
	}, socket.New(h.handle, socket.Config{Origins: []string{origin}, ReadBufferSize: 4096, WriteBufferSize: 4096}))
}
func strict(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if e := d.Decode(&struct{}{}); e != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (h *Hub) handle(c *socket.Conn) { h.serve(c) }
