package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/crypto/bcrypt"
)

// loginRequest is the JSON body for admin login.
type loginRequest struct {
	PlayerName string `json:"player_name"`
	Password   string `json:"password"`
}

// loginResponse is the JSON shape returned on successful login.
type loginResponse struct {
	Token      string `json:"token"`
	PlayerName string `json:"player_name"`
	Role       string `json:"role"`
}

// loginPlayerDB is the narrow database surface used by the login handler.
type loginPlayerDB interface {
	GetPlayer(name string) (*db.PlayerRecord, error)
}

// handleLogin creates a new login handler bound to the given database.
// authFailureBody is the single answer for every authentication failure:
// unknown player, player without a password, and wrong password all return it.
const authFailureBody = `{"error":"invalid credentials"}`

// timingDecoyHash is a valid bcrypt hash of a value no caller can produce. It
// gives the no-such-player path the same cost as a real comparison.
const timingDecoyHash = `$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy` // #nosec G101 -- not a credential; a fixed decoy hash for constant-time behaviour

// POST /admin/login — authenticates a player and returns a JWT.
func handleLogin(database loginPlayerDB, loginAttempts *auth.LoginAttemptTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		ip := auth.GetIPFromRequest(r)

		// Check lockout BEFORE parsing body — don't burn JSON decode on locked IPs
		if locked, remaining := loginAttempts.IsLocked(ip); locked {
			mins := int(remaining.Minutes()) + 1
			http.Error(w, fmt.Sprintf(`{"error":"too many failed attempts, try again in %d minutes"}`, mins), http.StatusTooManyRequests)
			audit.LogSecurityEvent("login_locked_out", "Admin login locked out", "", ip)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.PlayerName == "" || req.Password == "" {
			http.Error(w, `{"error":"player_name and password are required"}`, http.StatusBadRequest)
			return
		}

		// Database not available — allow token-only auth
		if database == nil {
			http.Error(w, `{"error":"database not available, use token auth"}`, http.StatusServiceUnavailable)
			return
		}

		// Every authentication failure answers identically. Distinguishing
		// "no such player" from "wrong password" lets a stranger enumerate
		// character names one request at a time, and a character name is the
		// public half of a player's credentials.
		rec, err := database.GetPlayer(req.PlayerName)

		// The comparison runs even when there is no player, against a fixed
		// hash, so a missing account costs the same time as a wrong password.
		// Without this the timing answers the question the message no longer
		// does. The hash below is bcrypt of a value nothing can present.
		stored := timingDecoyHash
		if rec != nil && err == nil && rec.Password != "" {
			stored = rec.Password
		}
		passwordOK := bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.Password)) == nil

		if err != nil || rec == nil || rec.Password == "" || !passwordOK {
			loginAttempts.RecordFailure(ip)
			http.Error(w, authFailureBody, http.StatusUnauthorized)
			return
		}

		loginAttempts.RecordSuccess(ip)

		// Panel roles follow the game's own ladder rather than magic numbers.
		// Level 50 did not exist: LVL_IMPL (40) is the ceiling ported from
		// structs.h:610, so no character created through the game could ever
		// hold "admin" and the first player, the Implementor, signed in as a
		// builder. 33 matched nothing either; C gates every OLC editor on
		// LVL_BUILDER, which olc.h:54 aliases to LVL_IMMORT (31), so that is
		// what "builder" means here.
		//
		// WARNING: this panel role is NOT the OLC authorization gate. C gates
		// the *editor surface* at LVL_IMMORT but confines levels 31-34 to
		// their assigned zone (olcAuthorized: unconfined only at LVL_GOD+1).
		// Any route that mutates world state through the OLC machinery must
		// apply olcAuthorized's zone confinement on top of this role check —
		// mounting such a route on requireRole("builder") alone hands levels
		// 31-34 unconfined edit rights over the whole world (the webOLC
		// design review calls this B2). Reads are fine at this role.
		role := "player"
		switch {
		case rec.Level >= game.LVL_IMPL:
			role = "admin"
		case rec.Level >= game.LVL_IMMORT:
			role = "builder"
		}

		// Generate JWT
		token, err := auth.GenerateJWT(req.PlayerName, false, 0, role)
		if err != nil {
			slog.Error("admin login JWT generation failed", "error", err)
			http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(loginResponse{
			Token:      token,
			PlayerName: req.PlayerName,
			Role:       role,
		}); err != nil {
			slog.Warn("admin login encode failed", "error", err)
		}
	}
}
