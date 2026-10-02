package admin

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

// DP-1380: admin must also retain full-secret bcrypt equivalence.
func TestAdminFullPasswordComparison(t *testing.T) {
	setJWTSecret(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("12345678ab"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	database := &fakeLoginDB{rec: &db.PlayerRecord{Name: "Hero", Password: string(hash), Level: 40}}
	for _, secret := range []string{"12345678cd", "12345678ab"} {
		tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{Threshold: 100, Lockout: time.Minute})
		t.Cleanup(tracker.Stop)
		req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(`{"player_name":"Hero","password":"`+secret+`"}`))
		out := httptest.NewRecorder()
		handleLogin(database, tracker).ServeHTTP(out, req)
		want := http.StatusUnauthorized
		if secret == "12345678ab" {
			want = http.StatusOK
		}
		if out.Code != want {
			t.Fatalf("suffix control: status=%d want=%d", out.Code, want)
		}
	}
}
