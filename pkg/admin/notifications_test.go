package admin

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
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
	t.Setenv("DP_NOTIFY_GOSSIP_WEBHOOK", "https://discord.com/api/webhooks/456/gossip-mirror-secret")
	t.Setenv("DP_NOTIFY_HEARTBEAT_URL", "https://hc-ping.com/12345678-1234-1234-1234-123456789012")
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
		for _, op := range []struct{ method, path string }{{"GET", "/admin/notifications"}, {"PUT", "/admin/notifications"}, {"POST", "/admin/notifications/discord/test"}, {"POST", "/admin/notifications/gossip/test"}, {"POST", "/admin/notifications/heartbeat/test"}} {
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
	// The console tests every channel it can configure. A provider outside the
	// enum is refused before it reaches the service.
	for _, provider := range []string{notify.Gossip, notify.Heartbeat} {
		out = request("POST", "/admin/notifications/"+provider+"/test", nil, admin, true)
		if out.Code != 202 {
			t.Fatalf("%s test %d %s", provider, out.Code, out.Body.String())
		}
	}
	out = request("POST", "/admin/notifications/ntfy/test", nil, admin, true)
	if out.Code != 422 {
		t.Fatalf("unknown provider test %d %s", out.Code, out.Body.String())
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

func TestWriteOnlyNotificationCredentials(t *testing.T) {
	for _, provider := range []string{notify.Discord, notify.Gossip} {
		t.Run(provider, func(t *testing.T) {
			setJWTSecret(t)
			t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin.json"))
			t.Setenv("DP_NOTIFICATION_KEY_PATH", "")
			database := newOLCTestDatabase(t, game.LVL_IMPL, 1)
			path := filepath.Join(t.TempDir(), "notifications.json")
			service, err := notify.New(path)
			if err != nil {
				t.Fatal(err)
			}
			router, err := newRouter(newOLCTestWorld(t), nil, NewLogBuffer(10), database, nil, WithNotifications(service))
			if err != nil {
				t.Fatal(err)
			}
			token := generateTestToken(t, "admin")
			calls := 0
			request := func(method, path, body, credential string, csrf bool) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				calls++
				req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", calls)
				req.Header.Set("Content-Type", "application/json")
				if credential != "" {
					req.Header.Set("Authorization", "Bearer "+credential)
				}
				if csrf {
					req.Header.Set("X-Requested-With", "darkpawns-admin")
				}
				out := httptest.NewRecorder()
				router.handler.ServeHTTP(out, req)
				if strings.Contains(out.Body.String(), "secret-marker") {
					t.Fatal("credential echoed by response", out.Code)
				}
				return out
			}
			endpoint := "/admin/notifications/" + provider + "/credentials"
			secret := "https://discord.com/api/webhooks/123/secret-marker"
			body := fmt.Sprintf(`{"revision":1,"url":%q}`, secret)
			for _, method := range []string{"PUT", "DELETE"} {
				if out := request(method, endpoint, body, "", true); out.Code != 401 {
					t.Fatal("unauthenticated credential mutation", out.Code)
				}
				if out := request(method, endpoint, body, token, false); out.Code != 401 {
					t.Fatal("credential mutation without CSRF", out.Code)
				}
				if out := request(method, endpoint, body, generateTestToken(t, "builder"), true); out.Code != 403 {
					t.Fatal("builder credential mutation", out.Code)
				}
			}
			for _, bad := range []string{
				`{"revision":"secret-marker","url":"secret-marker"}`,
				`{"revision":1,"url":{"secret-marker":true}}`,
				`{"revision":1,"unknown":"secret-marker"}`,
				`{"revision":1,"url":"secret-marker"}`,
				`{"revision":1,"url":"secret-marker"} trailing`,
			} {
				if out := request("PUT", endpoint, bad, token, true); out.Code != 422 {
					t.Fatalf("invalid credential input %d %s", out.Code, out.Body.String())
				}
			}
			if out := request("PUT", endpoint, body, token, true); out.Code != 200 {
				t.Fatalf("credential save %d %s", out.Code, out.Body.String())
			}
			raw, err := os.ReadFile(path)
			if err != nil || strings.Contains(string(raw), "secret-marker") {
				t.Fatal("plaintext credential storage", err)
			}
			if out := request("GET", "/admin/notifications", "", token, true); out.Code != 200 {
				t.Fatal("credential status unavailable", out.Code)
			}
			// Stale writes and malformed types never return secret values.
			if out := request("PUT", endpoint, body, token, true); out.Code != 409 {
				t.Fatal("stale credential write", out.Code)
			}
			current := service.View().Config
			if provider == notify.Gossip {
				current.Gossip = true
			} else {
				current.Discord = notify.Destination{Enabled: true, Events: []string{notify.Login}}
			}
			if err := service.Save(current); err != nil {
				t.Fatal(err)
			}
			out := request("DELETE", endpoint, fmt.Sprintf(`{"revision":%d}`, service.View().Config.Revision), token, true)
			if out.Code != 200 || service.View().Status[provider].Ready || (provider == notify.Gossip && service.View().Config.Gossip) || (provider == notify.Discord && service.View().Config.Discord.Enabled) {
				t.Fatal("credential removal failed", out.Code)
			}
			// The documented schema is JSON with write-only secret fields, not RawBody's
			// default binary schema; responses contain only Config/Status.
			schema := router.api.OpenAPI().Paths["/admin/notifications/{provider}/credentials"].Put.RequestBody.Content["application/json"].Schema
			if schema.Type != "object" || !schema.Properties["url"].WriteOnly || !schema.Properties["token"].WriteOnly || !schema.Properties["user"].WriteOnly {
				t.Fatal("missing write-only OpenAPI contract")
			}
			if _, err := database.Exec("UPDATE players SET level=? WHERE name=?", game.LVL_GOD, "TestPlayer"); err != nil {
				t.Fatal(err)
			}
			if out := request("PUT", endpoint, body, token, true); out.Code != 403 {
				t.Fatal("demoted staff retained credential access", out.Code)
			}
		})
	}
}
