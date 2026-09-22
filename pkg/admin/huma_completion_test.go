package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

func TestHumaCompletionRoutesPreserveGatesAndSpecificity(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin-store.json"))
	world := testWorld(t)
	handler, err := NewRouter(world, nil, NewLogBuffer(10), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	request := func(method, path, role, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if role != "" {
			req.Header.Set("Authorization", "Bearer "+generateTestToken(t, role))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	zones := world.GetAllZones()
	if len(zones) == 0 {
		t.Fatal("test world has no zones")
	}
	if got := request(http.MethodGet, "/admin/zones/"+strconv.Itoa(zones[0].Number), "builder", ""); got.Code != http.StatusOK {
		t.Fatalf("zone detail = %d, want 200; body: %s", got.Code, got.Body.String())
	}
	if got := request(http.MethodGet, "/admin/zones/reset", "admin", ""); got.Code != http.StatusMethodNotAllowed || got.Body.String() != "{\"error\":\"method not allowed\"}\n" {
		t.Errorf("GET literal reset route = %d %q, want legacy 405 bytes", got.Code, got.Body.String())
	}
	if got := request(http.MethodPost, "/admin/zones/reset", "admin", ""); got.Code != http.StatusNotImplemented || got.Body.String() != "{\"error\":\"not implemented\",\"message\":\"Zone reset trigger is not yet wired to the zone dispatcher\"}\n" {
		t.Errorf("POST placeholder reset route = %d %q, want legacy 501 bytes", got.Code, got.Body.String())
	}
	if got := request(http.MethodPost, "/admin/zones/30/reset", "builder", ""); got.Code != http.StatusForbidden {
		t.Errorf("builder zone reset = %d, want 403", got.Code)
	}
	if got := request(http.MethodPost, "/admin/save-world", "builder", ""); got.Code != http.StatusForbidden {
		t.Errorf("builder save-world = %d, want 403", got.Code)
	}
	if got := request(http.MethodGet, "/admin/findings/not-a-number", "builder", ""); got.Code != http.StatusMethodNotAllowed {
		t.Errorf("wrong-method finding detail = %d, want legacy 405", got.Code)
	}
}

func TestHumaLoginRoundTrip(t *testing.T) {
	setJWTSecret(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	database := &fakeLoginDB{rec: &db.PlayerRecord{Name: "Aidan", Password: string(hash), Level: 40}}
	attempts := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{Threshold: 10, Lockout: 15 * time.Minute})
	t.Cleanup(attempts.Stop)
	mux := http.NewServeMux()
	registerLoginOperation(apidoc.New().NewInternalAPI(mux), database, attempts)

	post := func(password string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"player_name":"Aidan","password":"`+password+`"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	if got := post("wrong"); got.Code != http.StatusUnauthorized || got.Body.String() != authFailureBody+"\n" {
		t.Fatalf("bad password = %d %q, want byte-compatible 401", got.Code, got.Body.String())
	}
	got := post("correct")
	if got.Code != http.StatusOK {
		t.Fatalf("correct password = %d; body: %s", got.Code, got.Body.String())
	}
	var response loginResponse
	if err := json.Unmarshal(got.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.PlayerName != "Aidan" || response.Role != "admin" || response.Token == "" {
		t.Fatalf("unexpected login response: %+v", response)
	}
	if _, err := auth.ValidateJWT(response.Token); err != nil {
		t.Fatalf("issued token does not validate: %v", err)
	}
	for range 10 {
		post("wrong")
	}
	if locked := post("wrong"); locked.Code != http.StatusTooManyRequests || !strings.Contains(locked.Body.String(), "too many failed attempts") {
		t.Fatalf("locked login = %d %q, want 429 lockout", locked.Code, locked.Body.String())
	}
}

func TestHumaLoginPreservesPublicFailureBytes(t *testing.T) {
	setJWTSecret(t)
	t.Setenv("ADMIN_STORE_PATH", filepath.Join(t.TempDir(), "admin-store.json"))
	handler, err := NewRouter(testWorld(t), nil, NewLogBuffer(10), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"player_name":"Aidan","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != "{\"error\":\"database not available, use token auth\"}\n" {
		t.Fatalf("login without database = %d %q; want legacy 503 bytes", rec.Code, rec.Body.String())
	}
}
