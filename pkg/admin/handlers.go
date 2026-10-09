package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// processStartTime captures when the server booted — used for uptime calculation.
var processStartTime = time.Now()

// ProcessStartTime returns the server boot timestamp. Used by the `uptime` command
// (pkg/session) and the admin API to report a faithful C-style "Up since ..." line.
func ProcessStartTime() time.Time { return processStartTime }

// PlayerDB is the interface admin needs from the game database to authenticate logins.
type PlayerDB interface {
	GetPlayer(name string) (*PlayerRecord, error)
}

// PlayerRecord holds the minimal fields admin needs from a player record.
type PlayerRecord struct {
	Name     string
	Password string
	Level    int
}

// zoneResponse is the JSON shape returned by zone endpoints.
type zoneResponse struct {
	Number    int    `json:"number"`
	Name      string `json:"name"`
	TopRoom   int    `json:"top_room"`
	Lifespan  int    `json:"lifespan"`
	ResetMode int    `json:"reset_mode"`
}

// serverInfoResponse is the JSON shape returned by the server info endpoint.
type serverInfoResponse struct {
	Uptime      string `json:"uptime"`
	RoomCount   int    `json:"room_count"`
	PlayerCount int    `json:"player_count"`
	ZoneCount   int    `json:"zone_count"`
}

// mobResponse is the JSON shape returned by mob endpoints.
type mobResponse struct {
	VNum        int      `json:"vnum"`
	Keywords    string   `json:"keywords"`
	ShortDesc   string   `json:"short_desc"`
	LongDesc    string   `json:"long_desc"`
	Level       int      `json:"level"`
	Alignment   int      `json:"alignment"`
	AC          int      `json:"ac"`
	HP          string   `json:"hp"`
	Gold        int      `json:"gold"`
	Exp         int      `json:"exp"`
	Position    int      `json:"position"`
	DefaultPos  int      `json:"default_pos"`
	Sex         int      `json:"sex"`
	Race        int      `json:"race"`
	ActionFlags []string `json:"action_flags"`
	AffectFlags []string `json:"affect_flags"`
	ScriptName  string   `json:"script_name"`
	Str         int      `json:"str"`
	Int         int      `json:"int"`
	Wis         int      `json:"wis"`
	Dex         int      `json:"dex"`
	Con         int      `json:"con"`
	Cha         int      `json:"cha"`
}

// objResponse is the JSON shape returned by object endpoints.
type objResponse struct {
	VNum       int    `json:"vnum"`
	Keywords   string `json:"keywords"`
	ShortDesc  string `json:"short_desc"`
	LongDesc   string `json:"long_desc"`
	TypeFlag   int    `json:"type_flag"`
	Weight     int    `json:"weight"`
	Cost       int    `json:"cost"`
	ExtraFlags [4]int `json:"extra_flags"`
	WearFlags  [4]int `json:"wear_flags"`
	Values     [4]int `json:"values"`
	ScriptName string `json:"script_name"`
}

// roomResponse is the JSON shape returned by room endpoints.
type roomResponse struct {
	VNum        int      `json:"vnum"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Zone        int      `json:"zone"`
	Sector      int      `json:"sector"`
	Flags       []string `json:"flags"`
}

// playerResponse is the JSON shape returned by the players endpoint.
type playerResponse struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
	Room  int    `json:"room"`
}

// playerItemResponse is the JSON shape for a single item in a player's inventory or equipment.
type playerItemResponse struct {
	VNum  int    `json:"vnum"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Slot  string `json:"slot,omitempty"`
}

// playerDetailResponse is the JSON shape returned by the player detail endpoint.
type playerDetailResponse struct {
	Name        string               `json:"name"`
	Level       int                  `json:"level"`
	Class       int                  `json:"class"`
	Race        int                  `json:"race"`
	Sex         int                  `json:"sex"`
	Health      int                  `json:"health"`
	MaxHealth   int                  `json:"max_health"`
	Mana        int                  `json:"mana"`
	MaxMana     int                  `json:"max_mana"`
	Move        int                  `json:"move"`
	MaxMove     int                  `json:"max_move"`
	Alignment   int                  `json:"alignment"`
	Gold        int                  `json:"gold"`
	BankGold    int                  `json:"bank_gold"`
	Exp         int                  `json:"exp"`
	Room        int                  `json:"room"`
	AC          int                  `json:"ac"`
	THAC0       int                  `json:"thac0"`
	Hitroll     int                  `json:"hitroll"`
	Damroll     int                  `json:"damroll"`
	Stats       map[string]int       `json:"stats"`
	Affects     uint64               `json:"affects"`
	ConnectedAt string               `json:"connected_at"`
	LastActive  string               `json:"last_active"`
	Inventory   []playerItemResponse `json:"inventory"`
	Equipment   []playerItemResponse `json:"equipment"`
}

// metricsResponse is the JSON shape returned by the server metrics endpoint.
type metricsResponse struct {
	MemoryAlloc  uint64 `json:"memory_alloc"`
	MemorySys    uint64 `json:"memory_sys"`
	MemoryHeap   uint64 `json:"memory_heap"`
	Goroutines   int    `json:"goroutines"`
	GCCycles     uint32 `json:"gc_cycles"`
	LastGC       string `json:"last_gc"`
	PauseTotalNs uint64 `json:"pause_total_ns"`
	Uptime       string `json:"uptime"`
	PlayerCount  int    `json:"player_count"`
	RoomCount    int    `json:"room_count"`
	ZoneCount    int    `json:"zone_count"`
}

// sessionKicker disconnects a live player session. Satisfied by the session
// manager; kept narrow so this package needs no session import.
type sessionKicker interface {
	Kick(playerName string, notice string) bool
}

// handlePlayerDetail returns full info for a specific player.
// Supports:
//
//	GET /admin/players/{name} — full player detail
//	POST /admin/players/{name}/save — force save
//	POST /admin/players/{name}/kick — disconnect
func handlePlayerDetail(world *game.World, auditLogger *audit.AuditLogger, kicker sessionKicker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse path: /admin/players/{name} or /admin/players/{name}/save or /admin/players/{name}/kick
		path := strings.TrimPrefix(r.URL.Path, "/admin/players/")
		if path == "" {
			http.Error(w, `{"error":"player name required"}`, http.StatusBadRequest)
			return
		}

		// Split into parts: [name] or [name, action]
		parts := strings.SplitN(path, "/", 2)
		playerName := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}

		if playerName == "" {
			http.Error(w, `{"error":"player name required"}`, http.StatusBadRequest)
			return
		}

		// Look up the player
		player, ok := world.GetPlayer(playerName)
		if !ok {
			http.Error(w, `{"error":"player not found"}`, http.StatusNotFound)
			return
		}

		claims, claimsOk := auth.GetClaimsFromContext(r.Context())

		switch {
		case r.Method == http.MethodGet && action == "":
			// GET — return full detail
			resp := playerDetailToResponse(player)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				slog.Warn("admin player detail encode failed", "error", err)
			}

		case r.Method == http.MethodPost && action == "save":
			// POST /save — requires admin role
			if !claimsOk || !claims.HasRole("admin") {
				http.Error(w, `{"error":"forbidden","required":"admin"}`, http.StatusForbidden)
				return
			}

			if res := world.SavePlayerRecord(player, "admin save", game.LoadRoomNowhere, game.SaveCrash); res != game.SaveSucceeded {
				err := fmt.Errorf("store save result: %d", res)
				slog.Error("admin player save failed", "name", playerName, "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("save failed: %v", err)}) //nolint:errcheck
				return
			}

			if auditLogger != nil {
				adminName := ""
				if claimsOk {
					adminName = claims.PlayerName
				}
				auditLogger.Log(audit.AuditEvent{
					IPAddress: auth.GetIPFromRequest(r),
					EventType: "administration",
					User:      adminName,
					Action:    "admin_force_save",
					Details:   fmt.Sprintf("forced save player %s", playerName),
					Success:   true,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "saved"}) //nolint:errcheck

		case r.Method == http.MethodPost && action == "kick":
			// POST /kick — requires admin role
			if !claimsOk || !claims.HasRole("admin") {
				http.Error(w, `{"error":"forbidden","required":"admin"}`, http.StatusForbidden)
				return
			}
			if kicker == nil {
				http.Error(w, `{"error":"session manager unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			// Best-effort save before the disconnect so a kicked player keeps
			// their progress; the kick proceeds regardless of the store's
			// answer (the /save endpoint exists for strict saves).
			if res := world.SavePlayerRecord(player, "admin kick", game.LoadRoomNowhere, game.SaveCrash); res != game.SaveSucceeded {
				slog.Warn("admin kick: pre-kick save failed", "name", playerName, "result", res)
			}
			if !kicker.Kick(playerName, "\r\nYou have been disconnected by an administrator.\r\n") {
				http.Error(w, `{"error":"player not online"}`, http.StatusNotFound)
				return
			}
			if auditLogger != nil {
				adminName := ""
				if claimsOk {
					adminName = claims.PlayerName
				}
				auditLogger.Log(audit.AuditEvent{
					IPAddress: auth.GetIPFromRequest(r),
					EventType: "administration",
					User:      adminName,
					Action:    "admin_kick",
					Details:   fmt.Sprintf("kicked player %s", playerName),
					Success:   true,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "kicked"}) //nolint:errcheck

		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

// playerDetailToResponse converts a game.Player to a playerDetailResponse.
func playerDetailToResponse(p *game.Player) playerDetailResponse {
	stats := map[string]int{
		"str": p.Stats.Str,
		"int": p.Stats.Int,
		"wis": p.Stats.Wis,
		"dex": p.Stats.Dex,
		"con": p.Stats.Con,
		"cha": p.Stats.Cha,
	}

	// Build inventory items (Snapshot: iterating the live Items slice races
	// gameplay mutations — a fatal concurrent map/slice read, VULN-028)
	invItems := make([]playerItemResponse, 0)
	if p.Inventory != nil {
		for _, item := range p.Inventory.Snapshot() {
			name := ""
			if item.Prototype != nil {
				name = item.Prototype.ShortDesc
			}
			invItems = append(invItems, playerItemResponse{
				VNum:  item.VNum,
				Name:  name,
				Count: 1,
			})
		}
	}

	// Build equipment items (Snapshot under eq.mu — same race class, VULN-028)
	equipItems := make([]playerItemResponse, 0)
	if p.Equipment != nil {
		for slot, item := range p.Equipment.Snapshot() {
			name := ""
			if item.Prototype != nil {
				name = item.Prototype.ShortDesc
			}
			equipItems = append(equipItems, playerItemResponse{
				VNum:  item.VNum,
				Name:  name,
				Count: 1,
				Slot:  slot.String(),
			})
		}
	}

	return playerDetailResponse{
		Name:        p.Name,
		Level:       p.Level,
		Class:       p.Class,
		Race:        p.Race,
		Sex:         p.Sex,
		Health:      p.Health,
		MaxHealth:   p.MaxHealth,
		Mana:        p.Mana,
		MaxMana:     p.MaxMana,
		Move:        p.Move,
		MaxMove:     p.MaxMove,
		Alignment:   p.Alignment,
		Gold:        p.Gold,
		BankGold:    p.BankGold,
		Exp:         p.Exp,
		Room:        p.RoomVNum,
		AC:          p.AC,
		THAC0:       p.THAC0,
		Hitroll:     p.Hitroll,
		Damroll:     p.Damroll,
		Stats:       stats,
		Affects:     p.Affects,
		ConnectedAt: p.ConnectedAt.Format(time.RFC3339),
		LastActive:  p.LastActive.Format(time.RFC3339),
		Inventory:   invItems,
		Equipment:   equipItems,
	}
}

// handleFindingByID updates a finding's status by ID (PUT).
func handleFindingByID(store *AgentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/admin/findings/")
		if idStr == "" {
			http.Error(w, `{"error":"finding id required"}`, http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"invalid finding id"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			Status        string `json:"status"`
			LinearIssueID string `json:"linear_issue_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Status == "" && req.LinearIssueID == "" {
			http.Error(w, `{"error":"status or linear_issue_id is required"}`, http.StatusBadRequest)
			return
		}

		var finding *Finding
		var ok bool
		var saveErr error
		if req.LinearIssueID != "" {
			finding, ok, saveErr = store.UpdateFinding(id, req.Status, req.LinearIssueID)
		} else {
			finding, ok, saveErr = store.UpdateFindingStatus(id, req.Status)
		}
		if saveErr != nil {
			slog.Error("admin finding update save failed", "error", saveErr)
			http.Error(w, `{"error":"failed to persist finding update"}`, http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, `{"error":"finding not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(finding); err != nil {
			slog.Warn("admin finding update encode failed", "error", err)
		}
	}
}

// shopResponse is the JSON shape returned by shop endpoints.
type shopResponse struct {
	VNum       int     `json:"vnum"`
	KeeperVNum int     `json:"keeper_vnum"`
	BuyTypes   []int   `json:"buy_types"`
	SellTypes  []int   `json:"sell_types"`
	ProfitBuy  float64 `json:"profit_buy"`
	ProfitSell float64 `json:"profit_sell"`
	KeeperName string  `json:"keeper_name"`
	RoomVNum   int     `json:"room_vnum"`
}

// handleShopByKeeper handles GET /admin/shops/{keeper_vnum}.
func handleShopByKeeper(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/shops/")
		if vnumStr == "" {
			http.Error(w, `{"error":"keeper vnum required"}`, http.StatusBadRequest)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			shop, ok := world.GetShopByKeeper(vnum)
			if !ok {
				http.Error(w, `{"error":"shop not found"}`, http.StatusNotFound)
				return
			}
			resp := shopResponse{
				VNum:       shop.VNum,
				KeeperVNum: shop.KeeperVNum,
				BuyTypes:   shop.BuyTypes,
				SellTypes:  shop.SellTypes,
				ProfitBuy:  shop.ProfitBuy,
				ProfitSell: shop.ProfitSell,
				KeeperName: shop.KeeperName,
				RoomVNum:   shop.RoomVNum,
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				slog.Warn("admin shop encode failed", "error", err)
			}

		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}
