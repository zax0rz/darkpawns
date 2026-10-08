package admin

// VULN-043: the admin credential lives in an HttpOnly cookie, not the
// console's localStorage, and every authenticated admin request carries the
// custom X-Requested-With header as CSRF defence in depth. Each test below
// fails with the fix reverted: before it, the login response carried the raw
// token in its JSON body, no cookie was set, and requests without the header
// were served.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

func TestAdminLoginCookieAttributes(t *testing.T) {
	setJWTSecret(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	database := &fakeLoginDB{rec: &db.PlayerRecord{Name: "Aidan", Password: string(hash), Level: 40}}
	tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{Threshold: 10, Lockout: 15 * time.Minute})
	t.Cleanup(tracker.Stop)

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"player_name":"Aidan","password":"secret"}`))
	rec := httptest.NewRecorder()
	handleLogin(database, tracker).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login = %d: %s", rec.Code, rec.Body.String())
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("login set no cookie — the token is back to body-only transport")
	}
	for _, want := range []string{
		adminTokenCookie + "=",
		"HttpOnly",
		"Secure",
		"SameSite=Strict",
		"Path=/admin",
		"Max-Age=86400",
	} {
		if !strings.Contains(setCookie, want) {
			t.Errorf("Set-Cookie missing %q: %s", want, setCookie)
		}
	}
	if body := rec.Body.String(); strings.Contains(body, "token") {
		t.Errorf("login body must not carry the token: %s", body)
	}

	value, _, _ := strings.Cut(strings.TrimPrefix(setCookie, adminTokenCookie+"="), ";")
	claims, err := auth.ValidateJWT(value)
	if err != nil {
		t.Fatalf("cookie token does not validate: %v", err)
	}
	if claims.PlayerName != "Aidan" || !claims.HasRole("admin") {
		t.Errorf("cookie claims = %+v", claims)
	}
}

// cookieFixture assembles the full admin router over a world whose TestPlayer
// holds the given level, so the gates run exactly as in production.
func cookieFixture(t *testing.T, level int) http.Handler {
	t.Helper()
	return newOLCTestRouter(t, level, 0, nil)
}

func doCookieRequest(handler http.Handler, method, path, cookie, header string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if cookie != "" {
		req.Header.Set("Cookie", adminTokenCookie+"="+cookie)
	}
	if header != "" {
		req.Header.Set(adminCSRFHeader, header)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAdminSessionFromCookieWithCSRFHeader(t *testing.T) {
	handler := cookieFixture(t, 40)
	token := generateTestToken(t, "builder")

	rec := doCookieRequest(handler, http.MethodGet, "/admin/session", token, "test")
	if rec.Code != http.StatusOK {
		t.Fatalf("session = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"player_name":"TestPlayer"`) || !strings.Contains(rec.Body.String(), `"role":"builder"`) {
		t.Errorf("session body = %s", rec.Body.String())
	}
}

func TestAdminCookieWithoutCSRFHeaderRejected(t *testing.T) {
	handler := cookieFixture(t, 40)
	token := generateTestToken(t, "builder")

	// The cookie rides automatically on any request the browser makes; the
	// custom header is what a cross-site page cannot add.
	rec := doCookieRequest(handler, http.MethodGet, "/admin/session", token, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie without CSRF header = %d, want 401", rec.Code)
	}
}

func TestAdminBearerWithoutCSRFHeaderRejected(t *testing.T) {
	handler := cookieFixture(t, 40)
	token := generateTestToken(t, "builder")

	req := httptest.NewRequest(http.MethodGet, "/admin/session", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Bearer without CSRF header = %d, want 401 (the header check must be uniform)", rec.Code)
	}
}

func TestAdminInvalidCookieRejected(t *testing.T) {
	handler := cookieFixture(t, 40)

	rec := doCookieRequest(handler, http.MethodGet, "/admin/session", "garbage.token.here", "test")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid cookie = %d, want 401", rec.Code)
	}
}

func TestAdminStaleCookieFallsBackToBearer(t *testing.T) {
	handler := cookieFixture(t, 40)
	token := generateTestToken(t, "builder")

	req := httptest.NewRequest(http.MethodGet, "/admin/session", nil)
	req.Header.Set("Cookie", adminTokenCookie+"=stale-token")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(adminCSRFHeader, "test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stale cookie + fresh Bearer = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminLogoutClearsCookie(t *testing.T) {
	handler := cookieFixture(t, 40)

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", rec.Code, rec.Body.String())
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "Max-Age=0") || !strings.Contains(setCookie, adminTokenCookie+"=") {
		t.Errorf("logout Set-Cookie must expire the cookie: %s", setCookie)
	}
}

// The OLC huma gates resolve credentials independently of requireRole; they
// must honour the cookie and demand the header too, or the console's OLC
// surface breaks (or opens) after the localStorage cutover.
func TestAdminOLCGateCookieAndCSRFHeader(t *testing.T) {
	handler := cookieFixture(t, 40)
	token := generateTestToken(t, "builder")

	rec := doCookieRequest(handler, http.MethodGet, "/admin/olc/held", token, "test")
	if rec.Code != http.StatusOK {
		t.Fatalf("OLC read via cookie = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	rec = doCookieRequest(handler, http.MethodGet, "/admin/olc/held", token, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("OLC read without CSRF header = %d, want 401", rec.Code)
	}
}
