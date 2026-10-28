package admin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

func TestDashboardEndpointRecordedHistoryAndGaps(t *testing.T) {
	now := time.Now().UTC()
	h := telemetry.New(now.Add(-2 * time.Hour))
	h.Sample(now.Add(-2*time.Hour), 4)
	telemetry.Login()
	telemetry.Character()
	telemetry.Death()
	telemetry.RecordPlayerDeath(telemetry.Activity{Player: "Avery", MinLevel: 31})
	h.Sample(now, 2)
	mux := http.NewServeMux()
	api := apidoc.New().NewInternalAPI(mux)
	registerDashboard(api, testWorld(t), h)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/admin/dashboard?range=24h", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	var body dashboardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Recording || len(body.Points) != 24 || body.Points[22].Online != nil || body.Points[23].Logins != 1 || body.Points[23].Characters != 1 || body.Points[23].Deaths != 1 || body.Points[23].PlayerDeaths != 1 || body.ActivitySince == nil {
		t.Fatalf("wrong observed history: %+v", body)
	}
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/admin/dashboard?range=fake", nil))
	if rr.Code != 422 {
		t.Fatalf("invalid range accepted: %d", rr.Code)
	}
	absent := dashboardSnapshot(testWorld(t), nil, "7d", now)
	if absent.Recording || len(absent.Points) > 0 {
		t.Fatal("missing recorder looked populated")
	}
}

func TestLogbookFilterExportAndVisibility(t *testing.T) {
	feed := newMudlogFeed(t)
	feed.Tap("visible", "normal, message", 1, 31, true)
	feed.Tap("hidden", "secret", 1, 40, true)
	feed.Tap("file", "file-only secret", 1, -1, true)
	feed.Tap("visible", "=spreadsheet()", 3, 31, true)
	waitForSeq(t, feed, 4)
	logs := NewLogBuffer(10)
	for _, line := range []string{"INFO 2026-10-27T12:00:00Z ready", "ERROR 2026-10-27T13:00:00Z database, failed", "WARN 2026-10-27T14:00:00Z warning"} {
		if _, err := logs.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	api := apidoc.New().NewInternalAPI(mux)
	registerLogbook(api, logs, feed, func(context.Context) int { return 31 })
	get := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		return r
	}
	for _, format := range []string{"csv", "log"} {
		r := get("/admin/logbook/export?source=mudlog&level=40&format=" + format)
		if r.Code != 200 || strings.Contains(r.Body.String(), "secret") {
			t.Fatalf("visibility bypass %d %s", r.Code, r.Body.String())
		}
		if !strings.Contains(r.Header().Get("Content-Disposition"), "attachment") || r.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("bad download headers")
		}
	}
	r := get("/admin/logbook/export?source=server&severity=ERROR&search=database&after=2026-10-27T12:30:00Z&before=2026-10-27T13:30:00Z&format=csv")
	rows, err := csv.NewReader(strings.NewReader(r.Body.String())).ReadAll()
	if err != nil || len(rows) != 2 || rows[1][3] != "database, failed" {
		t.Fatalf("bad CSV %s %v", r.Body.String(), err)
	}
	r = get("/admin/logbook/export?source=server&severity=ERROR&format=log")
	if r.Body.String() != "ERROR 2026-10-27T13:00:00Z database, failed\n" {
		t.Fatalf("raw log altered: %q", r.Body.String())
	}
	r = get("/admin/logbook/export?source=mudlog&type=1&key=visible&format=csv")
	if strings.Contains(r.Body.String(), "spreadsheet") || !strings.Contains(r.Body.String(), "normal, message") {
		t.Fatalf("type/key filters failed: %s", r.Body.String())
	}
	r = get("/admin/logbook/export?source=mudlog&format=csv")
	if !strings.Contains(r.Body.String(), "'=spreadsheet()") {
		t.Fatal("spreadsheet formula not neutralized")
	}
	for _, q := range []string{"after=bad", "before=bad", "after=2026-10-28T00:00:00Z&before=2026-10-27T00:00:00Z"} {
		if r := get("/admin/logbook?" + q); r.Code != 400 {
			t.Fatalf("bad date accepted: %d", r.Code)
		}
	}
}

func TestObservabilityRoutesRequireBuilder(t *testing.T) {
	setJWTSecret(t)
	handler, err := NewRouter(testWorld(t), nil, NewLogBuffer(1), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, path := range []string{"/admin/activity/events", "/admin/dashboard", "/admin/logbook", "/admin/logbook/export"} {
		for _, role := range []string{"", "player", "builder"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", i+1)
			req.Header.Set("X-Requested-With", "darkpawns-admin")
			if role != "" {
				req.Header.Set("Authorization", "Bearer "+generateTestToken(t, role))
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			_, _ = io.Copy(io.Discard, rr.Result().Body)
			if role == "builder" && rr.Code != 200 {
				t.Fatalf("builder denied %s: %d %s", path, rr.Code, rr.Body.String())
			}
			if role != "builder" && rr.Code == 200 {
				t.Fatalf("unauthorized %s can read %s", role, path)
			}
		}
	}
}
