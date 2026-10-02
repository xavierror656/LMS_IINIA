package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"bytes"
	"encoding/json"
	"fmt"
	socket "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Uses a new schema on a database whose name ends in _test. Never clears existing data.
func TestIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: PostgreSQL integration not executed")
	}
	parsed, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("test database name must end in _test")
	}
	base, e := database.Open(dsn)
	if e != nil {
		t.Fatal(e)
	}
	pool, _ := base.DB()
	defer pool.Close()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
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
	p, _ := db.DB()
	defer p.Close()
	t.Setenv("APP_ENV", "test")
	t.Setenv("SEED_STUDENT_PASSWORD", "synthetic-student-pass")
	t.Setenv("SEED_TEACHER_PASSWORD", "synthetic-teacher-pass")
	for i := 0; i < 2; i++ {
		if e = migrations.Apply(db); e != nil {
			t.Fatal(e)
		}
		if e = services.Seed(db); e != nil {
			t.Fatal(e)
		}
	}
	var count int64
	db.Raw("SELECT count(*) FROM users").Scan(&count)
	if count != 3 {
		t.Fatal("seed duplicated users")
	}
	app, hub := New(db, config.Config{Origin: "http://localhost:4321"})
	defer hub.Close()
	defer app.Shutdown()
	request := func(method, path, body, cookie, origin string) *http.Response {
		r := httptestRequest(method, path, body, cookie, origin)
		res, e := app.Test(r, 10000)
		if e != nil {
			t.Fatal(e)
		}
		return res
	}
	expect := func(res *http.Response, status int) {
		t.Helper()
		defer res.Body.Close()
		if res.StatusCode != status {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status %d expected %d: %s", res.StatusCode, status, b)
		}
	}
	login := func(name, password string) string {
		res := request("POST", "/api/v1/auth/login", fmt.Sprintf(`{"username":%q,"password":%q}`, name, password), "", "http://localhost:4321")
		if res.StatusCode != 200 {
			t.Fatalf("login: %d", res.StatusCode)
		}
		defer res.Body.Close()
		cookies := res.Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
			t.Fatal("cookie policy")
		}
		return cookies[0].Name + "=" + cookies[0].Value
	}
	luna := login("luna", "synthetic-student-pass")
	sol := login("sol", "synthetic-student-pass")
	teacher := login("profe", "synthetic-teacher-pass")
	var lunaUser, solUser models.User
	db.Where("username='luna'").Take(&lunaUser)
	db.Where("username='sol'").Take(&solUser)
	var reading, code, h5p models.Lesson
	db.Where("type='reading'").Order("id").First(&reading)
	db.Where("type='code'").First(&code)
	db.Where("type='h5p'").First(&h5p)
	t.Run("LMS001_002_auth_and_origin", func(t *testing.T) {
		expect(request("GET", "/api/v1/courses", "", "", ""), 401)
		expect(request("POST", "/api/v1/auth/login", `{"username":"luna","password":"synthetic-student-pass"}`, "", "https://evil.example"), 403)
		expect(request("POST", "/api/v1/auth/login", `{"username":"luna","password":"wrong-password"}`, "", "http://localhost:4321"), 401)
		expect(request("GET", "/api/v1/teacher/students", "", luna, ""), 403)
		expect(request("GET", "/api/v1/courses", "", teacher, ""), 403)
	})
	t.Run("LMS003_courses", func(t *testing.T) {
		expect(request("GET", "/api/v1/courses", "", luna, ""), 200)
		expect(request("GET", "/api/v1/courses?page=0", "", luna, ""), 400)
		expect(request("GET", fmt.Sprintf("/api/v1/lessons/%d", reading.ID), "", luna, ""), 200)
		expect(request("GET", "/api/v1/courses/99999", "", luna, ""), 404)
	})
	t.Run("LMS004_concurrent_idempotence", func(t *testing.T) {
		var wg sync.WaitGroup
		statuses := make(chan int, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				r := httptestRequest("POST", fmt.Sprintf("/api/v1/lessons/%d/complete", reading.ID), "{}", luna, "http://localhost:4321")
				r.Header.Set("Idempotency-Key", fmt.Sprint(i))
				res, e := app.Test(r, 10000)
				if e != nil {
					statuses <- 0
					return
				}
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				statuses <- res.StatusCode
			}(i)
		}
		wg.Wait()
		close(statuses)
		for s := range statuses {
			if s != 200 {
				t.Fatalf("concurrent status %d", s)
			}
		}
		var result models.Progress
		res := request("GET", "/api/v1/me/progress", "", luna, "")
		json.NewDecoder(res.Body).Decode(&result)
		res.Body.Close()
		if result.XP != 25 || result.Stars != 1 || result.Gems != 2 || result.Completed != 1 {
			t.Fatalf("duplicated rewards %+v", result)
		}
		db.Raw("SELECT count(*) FROM reward_events WHERE user_id=?", lunaUser.ID).Scan(&count)
		if count != 1 {
			t.Fatal("reward count", count)
		}
		expect(request("POST", fmt.Sprintf("/api/v1/lessons/%d/complete", reading.ID), `{"xp_to_add":999}`, luna, "http://localhost:4321"), 400)
		expect(request("POST", fmt.Sprintf("/api/v1/lessons/%d/complete", code.ID), `{}`, luna, "http://localhost:4321"), 409)
	})
	t.Run("LMS002_005_object_authorization", func(t *testing.T) {
		expect(request("GET", "/api/v1/teacher/students", "", teacher, ""), 200)
		expect(request("GET", fmt.Sprintf("/api/v1/teacher/students/%d/progress", lunaUser.ID), "", teacher, ""), 200)
		expect(request("GET", fmt.Sprintf("/api/v1/teacher/students/%d/progress", solUser.ID), "", teacher, ""), 404)
		expect(request("GET", fmt.Sprintf("/api/v1/teacher/students/%d/progress", lunaUser.ID), "", sol, ""), 403)
		if e = db.Exec("DELETE FROM enrollments WHERE user_id=?", solUser.ID).Error; e != nil {
			t.Fatal(e)
		}
		expect(request("GET", fmt.Sprintf("/api/v1/lessons/%d", reading.ID), "", sol, ""), 404)
		expect(request("POST", fmt.Sprintf("/api/v1/lessons/%d/complete", reading.ID), `{}`, sol, "http://localhost:4321"), 404)
	})
	t.Run("LMS007_untrusted_attempt", func(t *testing.T) {
		expect(request("POST", fmt.Sprintf("/api/v1/lessons/%d/attempts", h5p.ID), `{"verb":"passed","score":1}`, luna, "http://localhost:4321"), 200)
		var xp int
		db.Raw("SELECT xp FROM gamification_profiles WHERE user_id=?", lunaUser.ID).Scan(&xp)
		if xp != 25 {
			t.Fatal("H5P awarded points")
		}
		expect(request("POST", fmt.Sprintf("/api/v1/lessons/%d/attempts", h5p.ID), `{"verb":"passed","score":1,"user_id":2}`, luna, "http://localhost:4321"), 400)
	})
	t.Run("LMS006_websocket", func(t *testing.T) {
		listener, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		go app.Listener(listener)
		wsURL := "ws://" + listener.Addr().String() + "/ws/code"
		_, res, e := socket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"http://evil.example"}, "Cookie": []string{luna}})
		if e == nil || res.StatusCode != 403 {
			t.Fatal("foreign origin accepted")
		}
		c, _, e := socket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"http://localhost:4321"}, "Cookie": []string{luna}})
		if e != nil {
			t.Fatal(e)
		}
		defer c.Close()
		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		start := func(id string) {
			if e := c.WriteJSON(map[string]any{"v": 1, "type": "run.start", "requestId": id, "payload": map[string]any{"lessonId": code.ID, "language": "javascript", "code": "throw new Error()"}}); e != nil {
				t.Fatal(e)
			}
		}
		read := func() map[string]any {
			var event map[string]any
			if e := c.ReadJSON(&event); e != nil {
				t.Fatal(e)
			}
			return event
		}
		start("first")
		accepted := read()
		if accepted["type"] != "run.accepted" {
			t.Fatal(accepted)
		}
		run := accepted["runId"]
		if read()["type"] != "run.stdout" {
			t.Fatal("missing stdout")
		}
		c.WriteJSON(map[string]any{"v": 1, "type": "run.cancel", "requestId": "cancel", "runId": run, "payload": map[string]any{}})
		for {
			event := read()
			if event["type"] == "run.finished" {
				if event["payload"].(map[string]any)["status"] != "cancelled" {
					t.Fatal(event)
				}
				break
			}
		}
		start("second")
		if read()["type"] != "run.accepted" {
			t.Fatal("cancellation did not release execution")
		}
		c.Close()
		time.Sleep(30 * time.Millisecond)
		next, _, e := socket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"http://localhost:4321"}, "Cookie": []string{luna}})
		if e != nil {
			t.Fatal(e)
		}
		defer next.Close()
		next.SetReadDeadline(time.Now().Add(5 * time.Second))
		next.WriteJSON(map[string]any{"v": 1, "type": "run.start", "requestId": "third", "payload": map[string]any{"lessonId": code.ID, "language": "python", "code": "print('hello')"}})
		var event map[string]any
		if e = next.ReadJSON(&event); e != nil || event["type"] != "run.accepted" {
			t.Fatal("disconnect did not release connection", e, event)
		}
	})
	t.Run("LMS001_logout", func(t *testing.T) {
		expect(request("POST", "/api/v1/auth/logout", "", luna, "http://localhost:4321"), 200)
		expect(request("GET", "/api/v1/auth/session", "", luna, ""), 401)
	})
}
func httptestRequest(method, path, body, cookie, origin string) *http.Request {
	r, _ := http.NewRequest(method, "http://localhost:4321"+path, bytes.NewBufferString(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		r.Header.Set("Cookie", cookie)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

var _ *fiber.App
