package ws

import (
	"aulaquest/internal/models"
	"aulaquest/internal/runner"
	"context"
	socket "github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
	"time"
)

func (h *Hub) serve(c *socket.Conn) {
	u := c.Locals("user").(models.User)
	hash := c.Locals("sessionHash").(string)
	h.mu.Lock()
	_, exists := h.clients[u.ID]
	if exists || h.closed {
		h.mu.Unlock()
		c.WriteControl(socket.CloseMessage, socket.FormatCloseMessage(1008, "Solo una conexión por estudiante"), time.Now().Add(time.Second))
		c.Close()
		return
	}
	h.clients[u.ID] = connection{hash: hash, conn: c}
	h.mu.Unlock()
	ctx, closeContext := context.WithCancel(context.Background())
	defer func() { closeContext(); c.Close(); h.mu.Lock(); delete(h.clients, u.ID); h.mu.Unlock() }()
	c.SetReadLimit(24 * 1024)
	c.SetReadDeadline(time.Now().Add(45 * time.Second))
	c.SetPongHandler(func(string) error { return c.SetReadDeadline(time.Now().Add(45 * time.Second)) })
	incoming := make(chan []byte, 8)
	go func() {
		defer closeContext()
		for {
			kind, b, e := c.ReadMessage()
			if e != nil {
				return
			}
			if kind != socket.TextMessage {
				return
			}
			select {
			case incoming <- b:
			case <-ctx.Done():
				return
			default:
				return
			}
		}
	}()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	outputs := make(chan runner.Output, 8)
	var cancel context.CancelFunc
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	runID, requestID := "", ""
	seq := 0
	write := func(kind, req, run string, n int, payload any) bool {
		c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		return c.WriteJSON(Event{V: 1, Type: kind, RequestID: req, RunID: run, Seq: n, Payload: payload}) == nil
	}
	fail := func(req, code, message string) bool {
		return write("run.failed", req, "", 0, map[string]string{"code": code, "message": message})
	}
	validSession := func() bool {
		var n int64
		e := h.Repo.DB.Raw("SELECT count(*) FROM sessions WHERE token_hash=? AND expires_at>now()", hash).Scan(&n).Error
		return e == nil && n == 1
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !validSession() {
				return
			}
			if e := c.WriteControl(socket.PingMessage, nil, time.Now().Add(5*time.Second)); e != nil {
				return
			}
		case b := <-incoming:
			var event ClientEvent
			if strict(b, &event) != nil || event.V != 1 || len(event.RequestID) < 1 || len(event.RequestID) > 64 {
				if !fail("", "invalid_message", "Mensaje inválido") {
					return
				}
				continue
			}
			if !validSession() {
				return
			}
			switch event.Type {
			case "run.start":
				if runID != "" {
					if !fail(event.RequestID, "busy", "Ya hay una ejecución activa") {
						return
					}
					continue
				}
				var p struct {
					LessonID int64  `json:"lessonId"`
					Language string `json:"language"`
					Code     string `json:"code"`
				}
				if strict(event.Payload, &p) != nil || p.LessonID < 1 || (p.Language != "javascript" && p.Language != "python") || len(p.Code) > 16384 || event.RunID != "" {
					if !fail(event.RequestID, "invalid_payload", "Código o lenguaje inválido") {
						return
					}
					continue
				}
				l, e := h.Repo.Lesson(u.ID, p.LessonID)
				if e != nil || l.Type != "code" {
					if !fail(event.RequestID, "forbidden", "Lección no disponible") {
						return
					}
					continue
				}
				if e = h.Repo.DB.Exec("INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'in_progress') ON CONFLICT DO NOTHING", u.ID, p.LessonID).Error; e != nil {
					if !fail(event.RequestID, "unavailable", "No se pudo guardar el intento") {
						return
					}
					continue
				}
				runID, requestID, seq = uuid.NewString(), event.RequestID, 1
				if !write("run.accepted", requestID, runID, seq, map[string]bool{"simulated": true}) {
					return
				}
				runContext, stopRun := context.WithTimeout(ctx, 10*time.Second)
				cancel = stopRun
				go func() {
					defer stopRun()
					h.Runner.Run(runContext, runner.Request{Language: p.Language, Code: p.Code}, func(o runner.Output) bool {
						select {
						case outputs <- o:
							return true
						case <-ctx.Done():
							return false
						}
					})
				}()

			case "run.cancel":
				var empty struct{}
				if strict(event.Payload, &empty) != nil || event.RunID == "" || event.RunID != runID {
					if !fail(event.RequestID, "invalid_run", "Ejecución no disponible") {
						return
					}
					continue
				}
				cancel()
			default:
				if !fail(event.RequestID, "invalid_type", "Evento desconocido") {
					return
				}
			}
		case o := <-outputs:
			seq++
			payload := map[string]string{"text": o.Text}
			if o.Kind == "finished" {
				payload = map[string]string{"status": o.Status}
			}
			if !write("run."+o.Kind, requestID, runID, seq, payload) {
				return
			}
			if o.Kind == "finished" {
				cancel()
				cancel = nil
				runID = ""
				requestID = ""
			}
		}
	}
}
