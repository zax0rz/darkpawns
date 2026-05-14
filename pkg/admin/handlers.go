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
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// processStartTime captures when the server booted — used for uptime calculation.
var processStartTime = time.Now()

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

// validateStringField checks that a pointer-to-string field, if set,
// is not empty, doesn't exceed maxLen, and contains no control characters.
func validateStringField(s *string, maxLen int) error {
	if s == nil {
		return nil // not set, skip
	}
	if *s == "" {
		return fmt.Errorf("value must not be empty")
	}
	if len(*s) > maxLen {
		return fmt.Errorf("value exceeds maximum length of %d", maxLen)
	}
	// Check for control characters (except \n, \t, \r)
	for i := 0; i < len(*s); i++ {
		c := (*s)[i]
		if c < 0x20 && c != '\n' && c != '\t' && c != '\r' {
			return fmt.Errorf("value contains invalid control characters")
		}
	}
	return nil
}

// handleZones returns all zones as a JSON array.
func handleZones(world *game.World) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		zones := world.GetAllZones()
		result := make([]zoneResponse, 0, len(zones))
		for _, z := range zones {
			result = append(result, zoneResponse{
				Number:    z.Number,
				Name:      z.Name,
				TopRoom:   z.TopRoom,
				Lifespan:  z.Lifespan,
				ResetMode: z.ResetMode,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Warn("admin zones encode failed", "error", err)
		}
	}
}

// handleZoneByID returns a single zone by its number.
// Route pattern: /admin/zones/{id}
// handleZoneByIDOrReset handles GET/PUT /admin/zones/{number} and POST /admin/zones/{number}/reset.
func handleZoneByIDOrReset(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract zone number from URL: /admin/zones/123 or /admin/zones/123/reset
		idStr := strings.TrimPrefix(r.URL.Path, "/admin/zones/")
		if idStr == "" {
			http.Redirect(w, r, "/admin/zones", http.StatusMovedPermanently)
			return
		}

		// Handle /admin/zones/{number}/reset
		if strings.HasSuffix(idStr, "/reset") {
			handleZoneResetTrigger(world, auditLogger).ServeHTTP(w, r)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"invalid zone number"}`, http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodPut {
			handleZoneUpdate(world, auditLogger).ServeHTTP(w, r)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		zone, ok := world.GetZone(id)
		if !ok {
			http.Error(w, `{"error":"zone not found"}`, http.StatusNotFound)
			return
		}

		resp := zoneResponse{
			Number:    zone.Number,
			Name:      zone.Name,
			TopRoom:   zone.TopRoom,
			Lifespan:  zone.Lifespan,
			ResetMode: zone.ResetMode,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin zone encode failed", "error", err)
		}
	}
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
	VNum       int     `json:"vnum"`
	Keywords   string  `json:"keywords"`
	ShortDesc  string  `json:"short_desc"`
	LongDesc   string  `json:"long_desc"`
	TypeFlag   int     `json:"type_flag"`
	Weight     int     `json:"weight"`
	Cost       int     `json:"cost"`
	ExtraFlags [4]int  `json:"extra_flags"`
	WearFlags  [4]int  `json:"wear_flags"`
	Values     [4]int  `json:"values"`
	ScriptName string  `json:"script_name"`
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

// handleMobs returns all mob prototypes.
func handleMobs(world *game.World) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		mobs := world.GetAllMobPrototypes()
		result := make([]mobResponse, 0, len(mobs))
		for _, m := range mobs {
			result = append(result, mobResponse{
				VNum:        m.VNum,
				Keywords:    m.Keywords,
				ShortDesc:   m.ShortDesc,
				LongDesc:    m.LongDesc,
				Level:       m.Level,
				Alignment:   m.Alignment,
				AC:          m.AC,
				HP:          m.HP.String(),
				Gold:        m.Gold,
				Exp:         m.Exp,
				Position:    m.Position,
				DefaultPos:  m.DefaultPos,
				Sex:         m.Sex,
				Race:        m.Race,
				ActionFlags: m.ActionFlags,
				AffectFlags: m.AffectFlags,
				ScriptName:  m.ScriptName,
				Str:         m.Str,
				Int:         m.Int,
				Wis:         m.Wis,
				Dex:         m.Dex,
				Con:         m.Con,
				Cha:         m.Cha,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Warn("admin mobs encode failed", "error", err)
		}
	}
}

// handleMobByVnum returns a single mob prototype by VNum.
func handleMobByVnum(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			handleMobUpdate(world, auditLogger).ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/mobs/")
		if vnumStr == "" {
			http.Redirect(w, r, "/admin/mobs", http.StatusMovedPermanently)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		mob, ok := world.GetMobPrototype(vnum)
		if !ok {
			http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
			return
		}

		resp := mobResponse{
			VNum:        mob.VNum,
			Keywords:    mob.Keywords,
			ShortDesc:   mob.ShortDesc,
			LongDesc:    mob.LongDesc,
			Level:       mob.Level,
			Alignment:   mob.Alignment,
			AC:          mob.AC,
			HP:          mob.HP.String(),
			Gold:        mob.Gold,
			Exp:         mob.Exp,
			Position:    mob.Position,
			DefaultPos:  mob.DefaultPos,
			Sex:         mob.Sex,
			Race:        mob.Race,
			ActionFlags: mob.ActionFlags,
			AffectFlags: mob.AffectFlags,
			ScriptName:  mob.ScriptName,
			Str:         mob.Str,
			Int:         mob.Int,
			Wis:         mob.Wis,
			Dex:         mob.Dex,
			Con:         mob.Con,
			Cha:         mob.Cha,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin mob encode failed", "error", err)
		}
	}
}

// handleObjects returns all object prototypes.
func handleObjects(world *game.World) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		objs := world.GetAllObjPrototypes()
		result := make([]objResponse, 0, len(objs))
		for _, o := range objs {
			result = append(result, objResponse{
				VNum:       o.VNum,
				Keywords:   o.Keywords,
				ShortDesc:  o.ShortDesc,
				LongDesc:   o.LongDesc,
				TypeFlag:   o.TypeFlag,
				Weight:     o.Weight,
				Cost:       o.Cost,
				ExtraFlags: o.ExtraFlags,
				WearFlags:  o.WearFlags,
				Values:     o.Values,
				ScriptName: o.ScriptName,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Warn("admin objects encode failed", "error", err)
		}
	}
}

// handleObjectByVnum returns a single object prototype by VNum.
func handleObjectByVnum(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			handleObjectUpdate(world, auditLogger).ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/objects/")
		if vnumStr == "" {
			http.Redirect(w, r, "/admin/objects", http.StatusMovedPermanently)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		obj, ok := world.GetObjPrototype(vnum)
		if !ok {
			http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
			return
		}

		resp := objResponse{
			VNum:       obj.VNum,
			Keywords:   obj.Keywords,
			ShortDesc:  obj.ShortDesc,
			LongDesc:   obj.LongDesc,
			TypeFlag:   obj.TypeFlag,
			Weight:     obj.Weight,
			Cost:       obj.Cost,
			ExtraFlags: obj.ExtraFlags,
			WearFlags:  obj.WearFlags,
			Values:     obj.Values,
			ScriptName: obj.ScriptName,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin object encode failed", "error", err)
		}
	}
}

// handleRoomByVnum returns a single room by VNum.
func handleRoomByVnum(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			handleRoomUpdate(world, auditLogger).ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/rooms/")
		if vnumStr == "" {
			http.Error(w, `{"error":"vnum required"}`, http.StatusBadRequest)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		room := world.GetRoomInWorld(vnum)
		if room == nil {
			http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
			return
		}

		resp := roomResponse{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Zone:        room.Zone,
			Sector:      room.Sector,
			Flags:       room.Flags,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin room encode failed", "error", err)
		}
	}
}

// roomUpdateRequest is the JSON body for room update requests.
type roomUpdateRequest struct {
	Name        *string              `json:"name"`
	Description *string              `json:"description"`
	Flags       *[]string            `json:"flags"`
	Sector      *int                 `json:"sector"`
	ExtraDescs  *[]parser.ExtraDesc  `json:"extra_descs"`
}

// mobUpdateRequest is the JSON body for mob update requests.
type mobUpdateRequest struct {
	ShortDesc    *string   `json:"short_desc"`
	LongDesc     *string   `json:"long_desc"`
	Keywords     *string   `json:"keywords"`
	Level        *int      `json:"level"`
	AC           *int      `json:"ac"`
	HPNumDice    *int      `json:"hp_num_dice"`
	HPSizeDice   *int      `json:"hp_size_dice"`
	HPAdd        *int      `json:"hp_add"`
	Gold         *int      `json:"gold"`
	Exp          *int      `json:"exp"`
	Alignment    *int      `json:"alignment"`
	ActionFlags  *[]string `json:"action_flags"`
	AffectFlags  *[]string `json:"affect_flags"`
	Str          *int      `json:"str"`
	Int          *int      `json:"int"`
	Wis          *int      `json:"wis"`
	Dex          *int      `json:"dex"`
	Con          *int      `json:"con"`
	Cha          *int      `json:"cha"`
	THAC0        *int      `json:"thac0"`
	DamageNumDice *int     `json:"damage_num_dice"`
	DamageSizeDice *int    `json:"damage_size_dice"`
	DamageAdd    *int      `json:"damage_add"`
	Position     *int      `json:"position"`
	DefaultPos   *int      `json:"default_pos"`
	Sex          *int      `json:"sex"`
	Race         *int      `json:"race"`
}

// objUpdateRequest is the JSON body for object update requests.
type objUpdateRequest struct {
	ShortDesc  *string             `json:"short_desc"`
	LongDesc   *string             `json:"long_desc"`
	Keywords   *string             `json:"keywords"`
	TypeFlag   *int                `json:"type_flag"`
	Weight     *int                `json:"weight"`
	Cost       *int                `json:"cost"`
	Values     *[4]int             `json:"values"`
	WearFlags  *[4]int             `json:"wear_flags"`
	ExtraFlags *[4]int             `json:"extra_flags"`
	Affects    *[]parser.ObjAffect `json:"affects"`
	ExtraDescs *[]parser.ExtraDesc `json:"extra_descs"`
}

// handleRoomUpdate handles PUT /admin/rooms/{vnum}.
func handleRoomUpdate(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/rooms/")
		if vnumStr == "" {
			http.Error(w, `{"error":"vnum required"}`, http.StatusBadRequest)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		var req roomUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Validate string fields
		if err := validateStringField(req.Name, 256); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		if err := validateStringField(req.Description, 8192); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		// Apply partial updates
		updated := false
		if req.Name != nil {
			if !world.SetRoomName(vnum, *req.Name) {
				http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Description != nil {
			if !world.SetRoomDescription(vnum, *req.Description) {
				http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Flags != nil {
			if !world.SetRoomFlags(vnum, *req.Flags) {
				http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Sector != nil {
			if !world.SetRoomSector(vnum, *req.Sector) {
				http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.ExtraDescs != nil {
			if !world.SetRoomExtraDescs(vnum, *req.ExtraDescs) {
				http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}

		if !updated {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		// Audit log
		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: auth.GetIPFromRequest(r),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_room_update",
				Details:   fmt.Sprintf("updated room %d", vnum),
				Success:   true,
			})
		}

		// Return updated room
		room := world.GetRoomInWorld(vnum)
		if room == nil {
			http.Error(w, `{"error":"room not found after update"}`, http.StatusInternalServerError)
			return
		}

		resp := roomResponse{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Zone:        room.Zone,
			Sector:      room.Sector,
			Flags:       room.Flags,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin room update encode failed", "error", err)
		}
	}
}

// handleMobUpdate handles PUT /admin/mobs/{vnum}.
func handleMobUpdate(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/mobs/")
		if vnumStr == "" {
			http.Error(w, `{"error":"vnum required"}`, http.StatusBadRequest)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		var req mobUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Validate string fields
		if err := validateStringField(req.ShortDesc, 256); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		if err := validateStringField(req.LongDesc, 8192); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		// Apply partial updates
		updated := false
		if req.ShortDesc != nil {
			if !world.SetMobShortDesc(vnum, *req.ShortDesc) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.LongDesc != nil {
			if !world.SetMobLongDesc(vnum, *req.LongDesc) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Level != nil {
			if !world.SetMobLevel(vnum, *req.Level) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.AC != nil {
			if !world.SetMobAC(vnum, *req.AC) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.HPNumDice != nil && req.HPSizeDice != nil && req.HPAdd != nil {
			if !world.SetMobHP(vnum, *req.HPNumDice, *req.HPSizeDice, *req.HPAdd) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Gold != nil {
			if !world.SetMobGold(vnum, *req.Gold) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Exp != nil {
			if !world.SetMobExp(vnum, *req.Exp) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Alignment != nil {
			if !world.SetMobAlignment(vnum, *req.Alignment) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Keywords != nil {
			if !world.SetMobKeywords(vnum, *req.Keywords) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.ActionFlags != nil {
			if !world.SetMobActionFlags(vnum, *req.ActionFlags) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.AffectFlags != nil {
			if !world.SetMobAffectFlags(vnum, *req.AffectFlags) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Str != nil {
			if !world.SetMobStr(vnum, *req.Str) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Int != nil {
			if !world.SetMobInt(vnum, *req.Int) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Wis != nil {
			if !world.SetMobWis(vnum, *req.Wis) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Dex != nil {
			if !world.SetMobDex(vnum, *req.Dex) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Con != nil {
			if !world.SetMobCon(vnum, *req.Con) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Cha != nil {
			if !world.SetMobCha(vnum, *req.Cha) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.THAC0 != nil {
			if !world.SetMobTHAC0(vnum, *req.THAC0) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.DamageNumDice != nil && req.DamageSizeDice != nil && req.DamageAdd != nil {
			if !world.SetMobDamage(vnum, *req.DamageNumDice, *req.DamageSizeDice, *req.DamageAdd) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Position != nil {
			if !world.SetMobPosition(vnum, *req.Position) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.DefaultPos != nil {
			if !world.SetMobDefaultPos(vnum, *req.DefaultPos) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Sex != nil {
			if !world.SetMobSex(vnum, *req.Sex) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Race != nil {
			if !world.SetMobRace(vnum, *req.Race) {
				http.Error(w, `{"error":"mob not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}

		if !updated {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		// Audit log
		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: auth.GetIPFromRequest(r),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_mob_update",
				Details:   fmt.Sprintf("updated mob %d", vnum),
				Success:   true,
			})
		}

		// Return updated mob
		mob, ok := world.GetMobPrototype(vnum)
		if !ok {
			http.Error(w, `{"error":"mob not found after update"}`, http.StatusInternalServerError)
			return
		}

		resp := mobResponse{
			VNum:        mob.VNum,
			Keywords:    mob.Keywords,
			ShortDesc:   mob.ShortDesc,
			LongDesc:    mob.LongDesc,
			Level:       mob.Level,
			Alignment:   mob.Alignment,
			AC:          mob.AC,
			HP:          mob.HP.String(),
			Gold:        mob.Gold,
			Exp:         mob.Exp,
			Position:    mob.Position,
			DefaultPos:  mob.DefaultPos,
			Sex:         mob.Sex,
			Race:        mob.Race,
			ActionFlags: mob.ActionFlags,
			AffectFlags: mob.AffectFlags,
			ScriptName:  mob.ScriptName,
			Str:         mob.Str,
			Int:         mob.Int,
			Wis:         mob.Wis,
			Dex:         mob.Dex,
			Con:         mob.Con,
			Cha:         mob.Cha,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin mob update encode failed", "error", err)
		}
	}
}

// handleObjectUpdate handles PUT /admin/objects/{vnum}.
func handleObjectUpdate(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		vnumStr := strings.TrimPrefix(r.URL.Path, "/admin/objects/")
		if vnumStr == "" {
			http.Error(w, `{"error":"vnum required"}`, http.StatusBadRequest)
			return
		}

		vnum, err := strconv.Atoi(vnumStr)
		if err != nil {
			http.Error(w, `{"error":"invalid vnum"}`, http.StatusBadRequest)
			return
		}

		var req objUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		// Validate string fields
		if err := validateStringField(req.ShortDesc, 256); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		if err := validateStringField(req.LongDesc, 8192); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		// Apply partial updates
		updated := false
		if req.ShortDesc != nil {
			if !world.SetObjShortDesc(vnum, *req.ShortDesc) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.LongDesc != nil {
			if !world.SetObjLongDesc(vnum, *req.LongDesc) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Weight != nil {
			if !world.SetObjWeight(vnum, *req.Weight) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Cost != nil {
			if !world.SetObjCost(vnum, *req.Cost) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Keywords != nil {
			if !world.SetObjKeywords(vnum, *req.Keywords) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.TypeFlag != nil {
			if !world.SetObjTypeFlag(vnum, *req.TypeFlag) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Values != nil {
			if !world.SetObjValues(vnum, *req.Values) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.WearFlags != nil {
			if !world.SetObjWearFlags(vnum, *req.WearFlags) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.ExtraFlags != nil {
			if !world.SetObjExtraFlags(vnum, *req.ExtraFlags) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.Affects != nil {
			if !world.SetObjAffects(vnum, *req.Affects) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.ExtraDescs != nil {
			if !world.SetObjExtraDescs(vnum, *req.ExtraDescs) {
				http.Error(w, `{"error":"object not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}

		if !updated {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		// Audit log
		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: auth.GetIPFromRequest(r),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_object_update",
				Details:   fmt.Sprintf("updated object %d", vnum),
				Success:   true,
			})
		}

		// Return updated object
		obj, ok := world.GetObjPrototype(vnum)
		if !ok {
			http.Error(w, `{"error":"object not found after update"}`, http.StatusInternalServerError)
			return
		}

		resp := objResponse{
			VNum:       obj.VNum,
			Keywords:   obj.Keywords,
			ShortDesc:  obj.ShortDesc,
			LongDesc:   obj.LongDesc,
			TypeFlag:   obj.TypeFlag,
			Weight:     obj.Weight,
			Cost:       obj.Cost,
			ExtraFlags: obj.ExtraFlags,
			WearFlags:  obj.WearFlags,
			Values:     obj.Values,
			ScriptName: obj.ScriptName,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin object update encode failed", "error", err)
		}
	}
}

// handleServerInfo returns server status information.
func handleServerInfo(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		resp := serverInfoResponse{
			Uptime:      time.Since(processStartTime).Round(time.Second).String(),
			RoomCount:   world.GetRoomCount(),
			PlayerCount: world.GetPlayerCount(),
			ZoneCount:   len(world.GetAllZones()),
		}

		// Log admin access
		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: auth.GetIPFromRequest(r),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_server_info",
				Details:   "viewed server info",
				Success:   true,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin server info encode failed", "error", err)
		}
	}
}

// handleLogs returns recent log entries from the in-memory buffer.
func handleLogs(logBuffer *LogBuffer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		n := 100
		if q := r.URL.Query().Get("lines"); q != "" {
			if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 {
				n = parsed
			}
		}

		entries := logBuffer.GetRecent(n)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			slog.Warn("admin logs encode failed", "error", err)
		}
	}
}

// playerResponse is the JSON shape returned by the players endpoint.
type playerResponse struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
	Room  int    `json:"room"`
}

// handlePlayers returns a list of online players.
func handlePlayers(world *game.World) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		players := world.GetAllPlayers()
		result := make([]playerResponse, 0, len(players))
		for _, p := range players {
			result = append(result, playerResponse{
				Name:  p.Name,
				Level: p.Level,
				Room:  p.RoomVNum,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Warn("admin players encode failed", "error", err)
		}
	}
}

// handleZoneReset triggers a zone reset (placeholder).
func handleZoneReset(world *game.World) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error":   "not implemented",
			"message": "Zone reset trigger is not yet wired to the zone dispatcher",
		}); err != nil {
			slog.Warn("admin zone reset encode failed", "error", err)
		}
	}
}

// handleAgents returns all agent statuses.
func handleAgents(store *AgentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		agents := store.GetAgents()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(agents); err != nil {
			slog.Warn("admin agents encode failed", "error", err)
		}
	}
}

// handleAgentStatus updates an agent's status (POST).
func handleAgentStatus(store *AgentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			AgentID string `json:"agent_id"`
			Status  string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.AgentID == "" || req.Status == "" {
			http.Error(w, `{"error":"agent_id and status are required"}`, http.StatusBadRequest)
			return
		}

		agent, ok := store.UpdateAgentStatus(req.AgentID, req.Status)
		if !ok {
			http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(agent); err != nil {
			slog.Warn("admin agent status encode failed", "error", err)
		}
	}
}

// handleFindings returns findings (GET) or creates a new finding (POST).
func handleFindings(store *AgentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			status := r.URL.Query().Get("status")
			severity := r.URL.Query().Get("severity")
			source := r.URL.Query().Get("source")

			findings := store.GetFindings(status, severity, source)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(findings); err != nil {
				slog.Warn("admin findings encode failed", "error", err)
			}

		case http.MethodPost:
			var req struct {
				Source      string `json:"source"`
				Severity    string `json:"severity"`
				Title       string `json:"title"`
				File        string `json:"file"`
				Line        int    `json:"line"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}

			if req.Source == "" || req.Severity == "" || req.Title == "" {
				http.Error(w, `{"error":"source, severity, and title are required"}`, http.StatusBadRequest)
				return
			}

			f := store.AddFinding(req.Source, req.Severity, req.Title, req.File, req.Line, req.Description)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(f); err != nil {
				slog.Warn("admin finding create encode failed", "error", err)
			}

		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
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
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Status == "" {
			http.Error(w, `{"error":"status is required"}`, http.StatusBadRequest)
			return
		}

		finding, ok := store.UpdateFindingStatus(id, req.Status)
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

// handleTriageSummaries returns triage summaries (GET) or creates a new one (POST).
func handleTriageSummaries(store *AgentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			summaries := store.GetTriageSummaries()
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(summaries); err != nil {
				slog.Warn("admin triage summaries encode failed", "error", err)
			}

		case http.MethodPost:
			var req struct {
				Date      string `json:"date"`
				Confirmed int    `json:"confirmed"`
				Rejected  int    `json:"rejected"`
				Pending   int    `json:"pending"`
				Summary   string `json:"summary"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}

			if req.Date == "" {
				http.Error(w, `{"error":"date is required"}`, http.StatusBadRequest)
				return
			}

			s := store.AddTriageSummary(req.Date, req.Summary, req.Confirmed, req.Rejected, req.Pending)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(s); err != nil {
				slog.Warn("admin triage summary create encode failed", "error", err)
			}

		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

// shopResponse is the JSON shape returned by shop endpoints.
type shopResponse struct {
	KeeperVNum int     `json:"keeper_vnum"`
	BuyTypes   []int   `json:"buy_types"`
	SellTypes  []int   `json:"sell_types"`
	ProfitBuy  float64 `json:"profit_buy"`
	ProfitSell float64 `json:"profit_sell"`
	KeeperName string  `json:"keeper_name"`
	RoomVNum   int     `json:"room_vnum"`
}

// shopUpdateRequest is the JSON body for shop update requests.
type shopUpdateRequest struct {
	BuyTypes   *[]int   `json:"buy_types"`
	SellTypes  *[]int   `json:"sell_types"`
	ProfitBuy  *float64 `json:"profit_buy"`
	ProfitSell *float64 `json:"profit_sell"`
}

// handleShops returns all shops.
func handleShops(world *game.World) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		shops := world.GetAllShops()
		resp := make([]shopResponse, 0, len(shops))
		for _, s := range shops {
			resp = append(resp, shopResponse{
				KeeperVNum: s.KeeperVNum,
				BuyTypes:   s.BuyTypes,
				SellTypes:  s.SellTypes,
				ProfitBuy:  s.ProfitBuy,
				ProfitSell: s.ProfitSell,
				KeeperName: s.KeeperName,
				RoomVNum:   s.RoomVNum,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin shops encode failed", "error", err)
		}
	}
}

// handleShopByKeeper handles GET/PUT /admin/shops/{keeper_vnum}.
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

		case http.MethodPut:
			var req shopUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}

			updated := false
			if req.BuyTypes != nil {
				if !world.SetShopBuyTypes(vnum, *req.BuyTypes) {
					http.Error(w, `{"error":"shop not found"}`, http.StatusNotFound)
					return
				}
				updated = true
			}
			if req.SellTypes != nil {
				if !world.SetShopSellTypes(vnum, *req.SellTypes) {
					http.Error(w, `{"error":"shop not found"}`, http.StatusNotFound)
					return
				}
				updated = true
			}
			if req.ProfitBuy != nil && req.ProfitSell != nil {
				if !world.SetShopProfit(vnum, *req.ProfitBuy, *req.ProfitSell) {
					http.Error(w, `{"error":"shop not found"}`, http.StatusNotFound)
					return
				}
				updated = true
			}

			if !updated {
				http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
				return
			}

			if auditLogger != nil {
				playerName := ""
				if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
					playerName = claims.PlayerName
				}
				auditLogger.Log(audit.AuditEvent{
					IPAddress: auth.GetIPFromRequest(r),
					EventType: "administration",
					User:      playerName,
					Action:    "admin_shop_update",
					Details:   fmt.Sprintf("updated shop keeper %d", vnum),
					Success:   true,
				})
			}

			shop, ok := world.GetShopByKeeper(vnum)
			if !ok {
				http.Error(w, `{"error":"shop not found after update"}`, http.StatusInternalServerError)
				return
			}
			resp := shopResponse{
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
				slog.Warn("admin shop update encode failed", "error", err)
			}

		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

// zoneUpdateRequest is the JSON body for zone update requests.
type zoneUpdateRequest struct {
	Lifespan  *int `json:"lifespan"`
	ResetMode *int `json:"reset_mode"`
}

// handleZoneUpdate handles PUT /admin/zones/{number}.
func handleZoneUpdate(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		numStr := strings.TrimPrefix(r.URL.Path, "/admin/zones/")
		if numStr == "" {
			http.Error(w, `{"error":"zone number required"}`, http.StatusBadRequest)
			return
		}

		num, err := strconv.Atoi(numStr)
		if err != nil {
			http.Error(w, `{"error":"invalid zone number"}`, http.StatusBadRequest)
			return
		}

		var req zoneUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		updated := false
		if req.Lifespan != nil {
			if !world.SetZoneLifespan(num, *req.Lifespan) {
				http.Error(w, `{"error":"zone not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}
		if req.ResetMode != nil {
			if !world.SetZoneResetMode(num, *req.ResetMode) {
				http.Error(w, `{"error":"zone not found"}`, http.StatusNotFound)
				return
			}
			updated = true
		}

		if !updated {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: auth.GetIPFromRequest(r),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_zone_update",
				Details:   fmt.Sprintf("updated zone %d", num),
				Success:   true,
			})
		}

		zone, ok := world.GetZone(num)
		if !ok {
			http.Error(w, `{"error":"zone not found after update"}`, http.StatusInternalServerError)
			return
		}

		resp := zoneResponse{
			Number:    zone.Number,
			Name:      zone.Name,
			TopRoom:   zone.TopRoom,
			Lifespan:  zone.Lifespan,
			ResetMode: zone.ResetMode,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Warn("admin zone update encode failed", "error", err)
		}
	}
}

// handleZoneResetTrigger handles POST /admin/zones/{number}/reset — triggers a manual zone reset.
func handleZoneResetTrigger(world *game.World, auditLogger *audit.AuditLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Extract zone number from path: /admin/zones/{number}/reset
		path := strings.TrimPrefix(r.URL.Path, "/admin/zones/")
		path = strings.TrimSuffix(path, "/reset")
		if path == "" {
			http.Error(w, `{"error":"zone number required"}`, http.StatusBadRequest)
			return
		}

		num, err := strconv.Atoi(path)
		if err != nil {
			http.Error(w, `{"error":"invalid zone number"}`, http.StatusBadRequest)
			return
		}

		if err := world.ResetZone(num); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(r.Context()); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: auth.GetIPFromRequest(r),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_zone_reset",
				Details:   fmt.Sprintf("manual reset zone %d", num),
				Success:   true,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "reset triggered"}); err != nil {
			slog.Warn("admin zone reset encode failed", "error", err)
		}
	}
}
