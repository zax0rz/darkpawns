package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
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

// handleLogin creates a new login handler bound to the given database.
// POST /admin/login — authenticates a player and returns a JWT.
func handleLogin(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
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

		// Look up the player
		rec, err := database.GetPlayer(req.PlayerName)
		if err != nil {
			http.Error(w, `{"error":"player not found"}`, http.StatusUnauthorized)
			return
		}

		// Verify password
		if rec.Password == "" {
			http.Error(w, `{"error":"no password set for this player"}`, http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(req.Password)); err != nil {
			http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
			return
		}

		// Determine role from level
		role := "player"
		if rec.Level >= 50 {
			role = "admin"
		} else if rec.Level >= 33 {
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
