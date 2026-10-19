package admin

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/notify"
)

func TestNotificationAPI(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
	t.Setenv("DP_NOTIFY_DISCORD_WEBHOOK", "https://discord.com/api/webhooks/123/never-return-this-secret")
	database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
	service, err := notify.New(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, nil, WithNotifications(service))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	request := func(method, path string, body any, token string, csrf bool) *httptest.ResponseRecorder {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		calls++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", calls)
		req.Header.Set("Content-Type", "application/json")
		if csrf {
			req.Header.Set("X-Requested-With", "darkpawns-admin")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, req)
		return out
	}
	admin := generateTestToken(t, "admin")
	builder := generateTestToken(t, "builder")
	for _, tc := range []struct {
		token  string
		csrf   bool
		status int
	}{{"", true, 401}, {admin, false, 401}, {builder, true, 403}} {
		for _, op := range []struct{ method, path string }{{"GET", "/admin/notifications"}, {"PUT", "/admin/notifications"}, {"POST", "/admin/notifications/discord/test"}} {
			out := request(op.method, op.path, nil, tc.token, tc.csrf)
			if out.Code != tc.status {
				t.Fatalf("auth %s: %d %s", op.method, out.Code, out.Body.String())
			}
		}
	}
	out := request("GET", "/admin/notifications", nil, admin, true)
	if out.Code != 200 || strings.Contains(out.Body.String(), "never-return-this-secret") || out.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unsafe read %d %s", out.Code, out.Body.String())
	}
	c := service.View().Config
	c.Discord = notify.Destination{Enabled: true, Events: []string{notify.Level}}
	out = request("PUT", "/admin/notifications", c, admin, true)
	if out.Code != 200 {
		t.Fatalf("save %d %s", out.Code, out.Body.String())
	}
	out = request("PUT", "/admin/notifications", c, admin, true)
	if out.Code != 409 {
		t.Fatalf("stale save %d %s", out.Code, out.Body.String())
	}
	out = request("POST", "/admin/notifications/discord/test", nil, admin, true)
	if out.Code != 202 {
		t.Fatalf("test %d %s", out.Code, out.Body.String())
	}
	c = service.View().Config
	c.Discord.Events = []string{"mudlog.raw"}
	out = request("PUT", "/admin/notifications", c, admin, true)
	if out.Code != 422 {
		t.Fatalf("invalid event %d", out.Code)
	}
	// JWT claims alone cannot retain access after a demotion in storage.
	if _, err := database.Exec("UPDATE players SET level=? WHERE name=?", game.LVL_GOD, "TestPlayer"); err != nil {
		t.Fatal(err)
	}
	out = request("GET", "/admin/notifications", nil, admin, true)
	if out.Code != 403 {
		t.Fatalf("demoted admin retained access %d", out.Code)
	}
}
