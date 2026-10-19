package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/crypto/bcrypt"
)

// OfflinePlayerReserver holds the same name reservation as character entry.
// A missing provider disables edits instead of racing a login.
type OfflinePlayerReserver interface{ ReserveOfflinePlayer(string) (func(), bool) }

type playerField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Group string `json:"group"`
}

var playerFields = []playerField{
	{"level", "Level", 1, game.LVL_IMPL, "Character"},
	{"exp", "Experience", 0, 2000000000, "Character"},
	{"health", "Health", -10, 1000000, "Vitals"},
	{"max_health", "Maximum health", 1, 1000000, "Vitals"},
	{"mana", "Mana", 0, 1000000, "Vitals"},
	{"max_mana", "Maximum mana", 0, 1000000, "Vitals"},
	{"move", "Movement", 0, 1000000, "Vitals"},
	{"max_move", "Maximum movement", 0, 1000000, "Vitals"},
	{"str", "Strength", 1, 25, "Attributes"},
	{"str_add", "Exceptional strength", 0, 100, "Attributes"},
	{"int", "Intelligence", 1, 25, "Attributes"},
	{"wis", "Wisdom", 1, 25, "Attributes"},
	{"dex", "Dexterity", 1, 25, "Attributes"},
	{"con", "Constitution", 1, 25, "Attributes"},
	{"cha", "Charisma", 1, 25, "Attributes"},
}

type savedPlayerItem struct {
	Index  int    `json:"index"`
	VNum   int    `json:"vnum"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Locate int    `json:"locate"`
	Parent int    `json:"parent"`
}
type savedPlayerResponse struct {
	Name        string            `json:"name"`
	LastSavedAt *time.Time        `json:"last_saved_at"`
	Revision    string            `json:"revision"`
	Values      map[string]int    `json:"values"`
	Fields      []playerField     `json:"fields"`
	Inventory   []savedPlayerItem `json:"inventory"`
	Equipment   []savedPlayerItem `json:"equipment"`
	Slots       []playerSlot      `json:"slots"`
}
type (
	playerSlot struct {
		Locate int    `json:"locate"`
		Name   string `json:"name"`
	}
	savedPlayerOutput struct{ Body savedPlayerResponse }
	savedPlayerInput  struct {
		Name string `path:"name"`
	}
	editSavedPlayerInput struct {
		Name string `path:"name"`
		Body struct {
			Revision string         `json:"revision" minLength:"64" maxLength:"64"`
			Values   map[string]int `json:"values" doc:"Changed numeric fields from the field catalog"`
		}
	}
)

type resetSavedPasswordInput struct {
	Name string `path:"name"`
	Body struct {
		Revision string `json:"revision" minLength:"64" maxLength:"64"`
		Password string `json:"password" minLength:"8" maxLength:"72" writeOnly:"true"`
	}
}
type editPlayerItemsInput struct {
	Name string `path:"name"`
	Body struct {
		Revision string `json:"revision" minLength:"64" maxLength:"64"`
		Action   string `json:"action" enum:"grant,remove,move"`
		Source   string `json:"source" enum:"inventory,equipment"`
		Index    int    `json:"index" minimum:"0"`
		VNum     int    `json:"vnum" minimum:"0"`
		Locate   int    `json:"locate" minimum:"0" maximum:"22" doc:"0 for inventory; 1-22 for an equipment slot"`
	}
}

func playerRevision(r *db.PlayerRecord) string {
	raw, _ := json.Marshal(r) // PlayerRecord contains only JSON-compatible fields.
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func playerValues(r *db.PlayerRecord) map[string]*int {
	return map[string]*int{"level": &r.Level, "exp": &r.Exp, "health": &r.Health, "max_health": &r.MaxHealth, "mana": &r.Mana, "max_mana": &r.MaxMana, "move": &r.Move, "max_move": &r.MaxMove, "str": &r.StatStr, "str_add": &r.StatStrAdd, "int": &r.StatInt, "wis": &r.StatWis, "dex": &r.StatDex, "con": &r.StatCon, "cha": &r.StatCha}
}

func registerPlayerEditor(api huma.API, world *game.World, database *db.DB, sessions any, logger *audit.AuditLogger) {
	gate := playerEditorGate(database)
	reserver, _ := sessions.(OfflinePlayerReserver)
	// Reserve before reading the character, and keep it through commit. Even a
	// descriptor waiting at the menu owns the identity and blocks this editor.
	load := func(ctx context.Context, name, revision string) (*db.PlayerRecord, func(), error) {
		if reserver == nil {
			return nil, nil, huma.Error503ServiceUnavailable("Offline editing is unavailable on this server")
		}
		release, ok := reserver.ReserveOfflinePlayer(name)
		if !ok {
			return nil, nil, huma.Error409Conflict("Player is connected or entering the game. Disconnect fully, then reload.")
		}
		r, err := database.GetPlayer(name)
		if err != nil {
			release()
			return nil, nil, huma.Error500InternalServerError("Could not read saved character")
		}
		if r == nil || game.CharacterDataDeleted(r.CharacterData) {
			release()
			return nil, nil, huma.Error404NotFound("Saved character not found")
		}
		level := fileEditLevel(ctx)
		if level < game.LVL_IMPL && r.Level >= level {
			release()
			return nil, nil, huma.Error403Forbidden("You can edit only characters below your level")
		}
		if revision != "" && revision != playerRevision(r) {
			release()
			return nil, nil, huma.Error409Conflict("Character changed. Reload before saving.")
		}
		return r, release, nil
	}
	save := func(ctx context.Context, updated, original *db.PlayerRecord, password *string, objects bool, action string) (*savedPlayerOutput, error) {
		// Build the response before writing so malformed legacy equipment cannot
		// turn a successful commit into an error response.
		if password != nil {
			updated.Password = *password
			updated.FailedLoginAttempts = 0
			updated.LockedUntil = nil
		}
		out, err := savedPlayerView(world, updated)
		if err != nil {
			return nil, err
		}
		if err := database.AdminSavePlayer(updated, original, password, objects); err != nil {
			if errors.Is(err, db.ErrPlayerRecordChanged) {
				return nil, huma.Error409Conflict("Character changed. Reload before saving.")
			}
			slog.Error("admin saved character edit failed", "player", original.Name, "error", err)
			return nil, huma.Error500InternalServerError("Could not save character; no changes were applied")
		}
		auditAdmin(ctx, logger, action, fmt.Sprintf("saved character %s", original.Name), true)
		if password != nil {
			updated.Password = *password
			updated.FailedLoginAttempts = 0
			updated.LockedUntil = nil
		}
		return out, nil
	}
	type listInput struct {
		Query string `query:"q" maxLength:"80"`
	}
	type listOutput struct{ Body []string }
	huma.Register(api, huma.Operation{OperationID: "list-saved-players", Method: http.MethodGet, Path: "/admin/characters", Summary: "Search saved characters (HIGOD)", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *listInput) (*listOutput, error) {
		names, err := database.ListPlayerNames()
		if err != nil {
			return nil, huma.Error500InternalServerError("Could not list characters")
		}
		found := make([]string, 0)
		for _, name := range names {
			if strings.Contains(strings.ToLower(name), strings.ToLower(in.Query)) {
				found = append(found, name)
			}
		}
		return &listOutput{Body: found}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "get-saved-player", Method: http.MethodGet, Path: "/admin/characters/{name}", Summary: "Read a disconnected saved character (HIGOD)", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *savedPlayerInput) (*savedPlayerOutput, error) {
		r, release, err := load(ctx, in.Name, "")
		if err != nil {
			return nil, err
		}
		defer release()
		return savedPlayerView(world, r)
	})
	huma.Register(api, huma.Operation{OperationID: "edit-saved-player", Method: http.MethodPatch, Path: "/admin/characters/{name}", Summary: "Update disconnected character stats with revision check", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *editSavedPlayerInput) (*savedPlayerOutput, error) {
		r, release, err := load(ctx, in.Name, in.Body.Revision)
		if err != nil {
			return nil, err
		}
		defer release()
		updated := *r
		values := playerValues(&updated)
		for key, value := range in.Body.Values {
			field, ok := values[key]
			if !ok {
				return nil, huma.Error400BadRequest("Unknown character field: " + key)
			}
			for _, spec := range playerFields {
				if spec.Key == key && (value < spec.Min || value > spec.Max) {
					return nil, huma.Error400BadRequest("Out of range: " + spec.Label)
				}
			}
			*field = value
		}
		if updated.Level > fileEditLevel(ctx) {
			return nil, huma.Error403Forbidden("Cannot grant a level above your own")
		}
		if updated.Health > updated.MaxHealth || updated.Mana > updated.MaxMana || updated.Move > updated.MaxMove {
			return nil, huma.Error400BadRequest("Current vitals cannot exceed their maximums")
		}
		if _, changed := in.Body.Values["str"]; changed {
			updated.Strength = updated.StatStr
		}
		return save(ctx, &updated, r, nil, false, "admin_character_stats")
	})
	huma.Register(api, huma.Operation{OperationID: "reset-saved-player-password", Method: http.MethodPost, Path: "/admin/characters/{name}/password", Summary: "Set a disconnected character password and clear login lockout", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *resetSavedPasswordInput) (*savedPlayerOutput, error) {
		if len(in.Body.Password) < 8 || len(in.Body.Password) > 72 {
			return nil, huma.Error400BadRequest("Use a password of 8 to 72 bytes")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Body.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, huma.Error400BadRequest("Could not hash password")
		}
		r, release, err := load(ctx, in.Name, in.Body.Revision)
		if err != nil {
			return nil, err
		}
		defer release()
		updated := *r
		secret := string(hash)
		return save(ctx, &updated, r, &secret, false, "admin_character_password")
	})
	huma.Register(api, huma.Operation{OperationID: "edit-saved-player-items", Method: http.MethodPost, Path: "/admin/characters/{name}/items", Summary: "Grant, remove or relocate saved objects with revision check", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *editPlayerItemsInput) (*savedPlayerOutput, error) {
		r, release, err := load(ctx, in.Name, in.Body.Revision)
		if err != nil {
			return nil, err
		}
		defer release()
		updated := *r
		if err := editSavedItems(world, &updated, in); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return save(ctx, &updated, r, nil, true, "admin_character_items")
	})
}

func playerEditorGate(database *db.DB) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		ctx.SetHeader("Cache-Control", "no-store")
		if !hasAdminCSRFHeader(ctx.Header) {
			writeOLCError(ctx, 401, olcMiddlewareError{Error: "unauthorized"})
			return
		}
		claims, err := claimsFromAmbient(ctx.Header)
		if err != nil {
			writeOLCError(ctx, 401, olcMiddlewareError{Error: "unauthorized"})
			return
		}
		if database == nil {
			writeOLCError(ctx, 503, olcMiddlewareError{Error: "Player storage unavailable"})
			return
		}
		r, err := database.GetPlayer(claims.PlayerName)
		if err != nil || r == nil || game.CharacterDataDeleted(r.CharacterData) {
			writeOLCError(ctx, 401, olcMiddlewareError{Error: "unauthorized"})
			return
		}
		if !claims.HasRole("admin") || r.Level < game.LVL_HIGOD {
			writeOLCError(ctx, 403, olcMiddlewareError{Error: "Player editing requires HIGOD"})
			return
		}
		c := auth.SetClaimsOnContext(ctx.Context(), claims)
		next(huma.WithContext(ctx, context.WithValue(c, fileEditLevelKey{}, r.Level)))
	}
}

func decodeEditorItems(raw []byte) ([]game.SaveItemData, error) {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return []game.SaveItemData{}, nil
	}
	var items []game.SaveItemData
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("legacy or invalid object data requires migration before editing")
	}
	for i, item := range items {
		if item.ContainerIndex < 0 || item.ContainerIndex > i {
			return nil, fmt.Errorf("invalid saved container reference")
		}
	}
	return items, nil
}

func savedPlayerView(world *game.World, r *db.PlayerRecord) (*savedPlayerOutput, error) {
	var lastSaved *time.Time
	if r.LastLogon > 0 {
		timestamp := time.Unix(r.LastLogon, 0).UTC()
		lastSaved = &timestamp
	}
	out := savedPlayerResponse{Name: r.Name, LastSavedAt: lastSaved, Revision: playerRevision(r), Fields: playerFields, Values: map[string]int{}, Inventory: []savedPlayerItem{}, Equipment: []savedPlayerItem{}, Slots: []playerSlot{}}
	for key, value := range playerValues(r) {
		out.Values[key] = *value
	}
	for pos := 0; pos < 22; pos++ {
		slot, _ := game.CWearPosToSlot(pos)
		out.Slots = append(out.Slots, playerSlot{pos + 1, slot.String()})
	}
	for _, equipped := range []bool{false, true} {
		raw := r.Inventory
		if equipped {
			raw = r.Equipment
		}
		items, err := decodeEditorItems(raw)
		if err != nil {
			return nil, huma.Error409Conflict(err.Error())
		}
		for i, item := range items {
			name := "Unknown object"
			if proto, ok := world.GetObjPrototype(item.VNum); ok {
				name = proto.ShortDesc
			}
			row := savedPlayerItem{i, item.VNum, name, max(1, item.Count), item.Locate, item.ContainerIndex}
			if equipped {
				out.Equipment = append(out.Equipment, row)
			} else {
				out.Inventory = append(out.Inventory, row)
			}
		}
	}
	return &savedPlayerOutput{Body: out}, nil
}

// splitSavedTree preserves parent indices and instance state for both the
// selected subtree and the remaining siblings. Children travel with a bag.
func splitSavedTree(items []game.SaveItemData, index int) ([]game.SaveItemData, []game.SaveItemData) {
	selected := map[int]bool{index + 1: true}
	keep, tree := make([]game.SaveItemData, 0), make([]game.SaveItemData, 0)
	keepIndices, treeIndices := map[int]int{}, map[int]int{}
	for i, item := range items {
		if selected[item.ContainerIndex] {
			selected[i+1] = true
		}
		if selected[i+1] {
			treeIndices[i+1] = len(tree) + 1
			item.ContainerIndex = treeIndices[item.ContainerIndex]
			if i == index {
				item.ContainerIndex = 0
				item.ContainerVNum = 0
			}
			tree = append(tree, item)
		} else {
			keepIndices[i+1] = len(keep) + 1
			item.ContainerIndex = keepIndices[item.ContainerIndex]
			keep = append(keep, item)
		}
	}
	return keep, tree
}

func editSavedItems(world *game.World, r *db.PlayerRecord, in *editPlayerItemsInput) error {
	inv, err := decodeEditorItems(r.Inventory)
	if err != nil {
		return err
	}
	eq, err := decodeEditorItems(r.Equipment)
	if err != nil {
		return err
	}
	var tree []game.SaveItemData
	if in.Body.Action == "grant" {
		if _, ok := world.GetObjPrototype(in.Body.VNum); !ok {
			return fmt.Errorf("object prototype not found")
		}
		tree = []game.SaveItemData{{VNum: in.Body.VNum, Count: 1}}
	} else {
		source := &inv
		if in.Body.Source == "equipment" {
			source = &eq
		}
		if in.Body.Index < 0 || in.Body.Index >= len(*source) {
			return fmt.Errorf("saved object not found")
		}
		if in.Body.Action == "move" && (*source)[in.Body.Index].Count > 1 && in.Body.Locate > 0 {
			return fmt.Errorf("a stacked object cannot occupy one equipment slot")
		}
		*source, tree = splitSavedTree(*source, in.Body.Index)
	}
	if in.Body.Action != "remove" {
		if in.Body.Locate < 0 || in.Body.Locate > 22 {
			return fmt.Errorf("unknown equipment slot")
		}
		target := &inv
		if in.Body.Locate > 0 {
			target = &eq
			for _, item := range eq {
				if item.ContainerIndex == 0 && item.Locate == in.Body.Locate {
					return fmt.Errorf("slot occupied; move its item to inventory first")
				}
			}
		}
		if len(inv)+len(eq)+len(tree) > 2000 {
			return fmt.Errorf("saved object limit reached")
		}
		offset := len(*target)
		tree[0].Locate = in.Body.Locate
		for i := range tree {
			if tree[i].ContainerIndex > 0 {
				tree[i].ContainerIndex += offset
			}
		}
		*target = append(*target, tree...)
	}
	r.Inventory, err = json.Marshal(inv)
	if err != nil {
		return err
	}
	r.Equipment, err = json.Marshal(eq)
	return err
}
