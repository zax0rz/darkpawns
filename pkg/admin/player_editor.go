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
	"github.com/zax0rz/darkpawns/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

// OfflinePlayerReserver holds the same name reservation as character entry.
// A missing provider disables edits instead of racing a login.
type OfflinePlayerReserver interface{ ReserveOfflinePlayer(string) (func(), bool) }

type OfflinePlayerRenameReserver interface {
	ReserveOfflineRename(oldName, newName string) (func(), bool)
}

type PlayerRenameObserver interface{ PlayerRenamed(oldName, newName string) }

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

type playerFlagField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	MinLevel    int    `json:"min_level"`
	Enabled     bool   `json:"enabled"`
	Editable    bool   `json:"editable"`
}

type playerFlagSpec struct {
	Key, Label, Description string
	Bit                     int
	MinLevel                int
	Inverted                bool
	ColorPair               bool
}

// These are the PC BINARY fields from the do_set field table
// (src/act.wizard.c:2547-2597), plus CRYO from src/constants.c's player_bits[]
// table. CRYO has no do_set field; its HIGOD toggle is an explicit admin
// extension. DELETED is read-only here and changed by the admin delete action.
var playerFlagSpecs = []playerFlagSpec{
	{"brief", "Brief descriptions", "Shows shorter room descriptions.", game.PrfBrief, game.LVL_GOD, false, false},
	{"invstart", "Start invisible", "Starts the character unseen.", game.PlrInvstart, game.LVL_GOD, false, false},
	{"can_be_summoned", "Can be summoned", "Allows other characters to summon this character.", game.PrfSummonable, game.LVL_GRGOD, false, false},
	{"nohassle", "No hassle", "Turns off many interruptions for this character.", game.PrfNohassle, game.LVL_GRGOD, false, false},
	{"frozen", "Frozen", "Stops this character from entering the game.", game.PlrFrozen, game.LVL_GRGOD, false, false},
	{"outlaw", "Outlaw", "Marks this character as an outlaw.", game.PlrOutlaw, game.LVL_GOD, false, false},
	{"roomflag", "Show room flags", "Shows room details in room descriptions.", game.PrfRoomFlags, game.LVL_GRGOD, false, false},
	{"siteok", "Site-cleared", "Allows this character to enter from a normally blocked site.", game.PlrSiteok, game.LVL_GRGOD, false, false},
	{"nowizlist", "Hidden from staff list", "Hides this character from the staff list.", game.PlrNowizlist, game.LVL_GOD, false, false},
	{"quest", "Questing", "Marks this character as taking part in a quest.", game.PrfQuest, game.LVL_GOD, false, false},
	{"color", "Color", "Shows colored text for this character.", game.PrfColor1, game.LVL_GOD, false, true},
	{"nodelete", "Protected from deletion", "Stops the player from deleting this character in game.", game.PlrNODELETE, game.LVL_GOD, false, false},
	{"chosen", "Chosen", "Allows this character to use extra staff actions.", game.PlrChosen, game.LVL_GRGOD, false, false},
	{"cryo", "Cryo-saved", "Marks this character for cryogenic storage.", game.PlrCRYO, game.LVL_GRGOD, false, false},
}

type savedPlayerItem struct {
	Index  int    `json:"index"`
	VNum   int    `json:"vnum"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Locate int    `json:"locate"`
	Parent int    `json:"parent"`
	NoRent bool   `json:"no_rent"`
}
type savedPlayerResponse struct {
	Name        string            `json:"name"`
	LastSavedAt *time.Time        `json:"last_saved_at"`
	Revision    string            `json:"revision"`
	Values      map[string]int    `json:"values"`
	Fields      []playerField     `json:"fields"`
	Flags       []playerFlagField `json:"flags"`
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
	editSavedPlayerFlagsInput struct {
		Name string `path:"name"`
		Body struct {
			Revision string          `json:"revision" minLength:"64" maxLength:"64"`
			Flags    map[string]bool `json:"flags"`
		}
	}
	renameSavedPlayerInput struct {
		Name string `path:"name"`
		Body struct {
			Revision string `json:"revision" minLength:"64" maxLength:"64"`
			NewName  string `json:"new_name" minLength:"2" maxLength:"80"`
		}
	}
	deleteSavedPlayerInput struct {
		Name string `path:"name"`
		Body struct {
			Revision    string `json:"revision" minLength:"64" maxLength:"64"`
			ConfirmName string `json:"confirm_name,omitempty" maxLength:"80"`
		}
	}
	deleteSavedPlayerResponse struct {
		Name           string `json:"name"`
		MarkedDeleted  bool   `json:"marked_deleted"`
		AliasesRemoved bool   `json:"aliases_removed"`
		ClanCleared    bool   `json:"clan_cleared"`
	}
	deleteSavedPlayerOutput struct{ Body deleteSavedPlayerResponse }
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

func playerFlagCatalog(raw []byte, actorLevel, targetLevel int) ([]playerFlagField, error) {
	flags, err := game.CharacterDataFlags(raw)
	if err != nil {
		return nil, err
	}
	fields := make([]playerFlagField, 0, len(playerFlagSpecs)+1)
	for _, spec := range playerFlagSpecs {
		enabled := flags&(uint64(1)<<uint(spec.Bit)) != 0
		if spec.ColorPair {
			enabled = enabled && flags&(uint64(1)<<uint(game.PrfColor2)) != 0
		}
		fields = append(fields, playerFlagField{
			Key: spec.Key, Label: spec.Label, Description: spec.Description,
			MinLevel: spec.MinLevel, Enabled: enabled,
			Editable: actorLevel >= spec.MinLevel && (actorLevel >= game.LVL_IMPL || targetLevel < actorLevel),
		})
	}
	fields = append(fields, playerFlagField{
		Key: "deleted", Label: "Deleted", Description: "Use Delete on the Identity tab to remove this character and free the name.",
		MinLevel: game.LVL_GRGOD, Enabled: flags&(uint64(1)<<uint(game.PlrDeleted)) != 0,
		Editable: false,
	})
	return fields, nil
}

func playerFlagSpecByKey(key string) (playerFlagSpec, bool) {
	for _, spec := range playerFlagSpecs {
		if spec.Key == key {
			return spec, true
		}
	}
	return playerFlagSpec{}, false
}

func registerPlayerEditor(api huma.API, world *game.World, database *db.DB, sessions any, logger *audit.AuditLogger, externalRenameObserver PlayerRenameObserver) {
	gate := playerEditorGate(database)
	reserver, _ := sessions.(OfflinePlayerReserver)
	renameReserver, _ := sessions.(OfflinePlayerRenameReserver)
	renameObserver, _ := sessions.(PlayerRenameObserver)
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
		out, err := savedPlayerView(world, updated, fileEditLevel(ctx))
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
	huma.Register(api, huma.Operation{OperationID: "list-saved-players", Method: http.MethodGet, Path: "/admin/characters", Summary: "Find saved characters", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *listInput) (*listOutput, error) {
		names, err := database.ListActivePlayerNames()
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
	huma.Register(api, huma.Operation{OperationID: "get-saved-player", Method: http.MethodGet, Path: "/admin/characters/{name}", Summary: "Open a disconnected character", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *savedPlayerInput) (*savedPlayerOutput, error) {
		r, release, err := load(ctx, in.Name, "")
		if err != nil {
			return nil, err
		}
		defer release()
		return savedPlayerView(world, r, fileEditLevel(ctx))
	})
	huma.Register(api, huma.Operation{OperationID: "edit-saved-player", Method: http.MethodPatch, Path: "/admin/characters/{name}", Summary: "Update a disconnected character's details", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *editSavedPlayerInput) (*savedPlayerOutput, error) {
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
	huma.Register(api, huma.Operation{OperationID: "edit-saved-player-flags", Method: http.MethodPatch, Path: "/admin/characters/{name}/flags", Summary: "Update disconnected character flags with level gates", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *editSavedPlayerFlagsInput) (*savedPlayerOutput, error) {
		r, release, err := load(ctx, in.Name, in.Body.Revision)
		if err != nil {
			return nil, err
		}
		defer release()
		actorLevel := fileEditLevel(ctx)
		flags, err := game.CharacterDataFlags(r.CharacterData)
		if err != nil {
			return nil, huma.Error409Conflict("Character flags are malformed; no changes were applied")
		}
		for key, enabled := range in.Body.Flags {
			spec, ok := playerFlagSpecByKey(key)
			if !ok {
				return nil, huma.Error400BadRequest("Unknown or read-only character flag: " + key)
			}
			if actorLevel < spec.MinLevel || (actorLevel < game.LVL_IMPL && r.Level >= actorLevel) {
				return nil, huma.Error403Forbidden("Your staff level cannot change " + spec.Label)
			}
			mask := uint64(1) << uint(spec.Bit)
			if enabled {
				flags |= mask
			} else {
				flags &^= mask
			}
			if spec.ColorPair {
				color2 := uint64(1) << uint(game.PrfColor2)
				if enabled {
					flags |= color2
				} else {
					flags &^= color2
				}
			}
		}
		updated := *r
		updated.CharacterData, err = game.SetCharacterDataFlags(r.CharacterData, flags)
		if err != nil {
			return nil, huma.Error409Conflict("Character flags are malformed; no changes were applied")
		}
		return save(ctx, &updated, r, nil, false, "admin_character_flags")
	})
	huma.Register(api, huma.Operation{OperationID: "rename-saved-player", Method: http.MethodPost, Path: "/admin/characters/{name}/rename", Summary: "Rename a disconnected character and its references", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *renameSavedPlayerInput) (*savedPlayerOutput, error) {
		if renameReserver == nil {
			return nil, huma.Error503ServiceUnavailable("Offline rename is unavailable on this server")
		}
		newName := strings.TrimLeft(in.Body.NewName, " \t\n\r\v\f")
		if !validation.IsCharacterCreationNameSyntax(newName) {
			return nil, huma.Error400BadRequest("Use a 2 to 20 letter name allowed at character creation")
		}
		if validation.IsReservedPlayerName(newName) {
			return nil, huma.Error400BadRequest("That name cannot be used")
		}
		if !game.ValidNameNoActive(newName) {
			return nil, huma.Error400BadRequest("That name cannot be used")
		}
		release, ok := renameReserver.ReserveOfflineRename(in.Name, newName)
		if !ok {
			return nil, huma.Error409Conflict("Player is connected, linkdead, entering, or writing. Disconnect fully, then reload.")
		}
		defer release()
		original, err := database.GetPlayer(in.Name)
		if err != nil {
			return nil, huma.Error500InternalServerError("Could not read saved character")
		}
		if original == nil || game.CharacterDataDeleted(original.CharacterData) {
			return nil, huma.Error404NotFound("Saved character not found")
		}
		actorLevel := fileEditLevel(ctx)
		if actorLevel < game.LVL_IMPL && original.Level >= actorLevel {
			return nil, huma.Error403Forbidden("You can edit only characters below your level")
		}
		if in.Body.Revision != playerRevision(original) {
			return nil, huma.Error409Conflict("Character changed. Reload before renaming.")
		}
		if newName == original.Name {
			return savedPlayerView(world, original, actorLevel)
		}
		if collision, err := database.GetPlayer(newName); err != nil {
			return nil, huma.Error500InternalServerError("Could not check the requested name")
		} else if collision != nil && collision.ID != original.ID {
			return nil, huma.Error409Conflict("That name already belongs to another character")
		}
		updated := *original
		updated.Name = newName
		updated.CharacterData, err = game.RenameCharacterData(original.CharacterData, newName)
		if err != nil {
			return nil, huma.Error409Conflict("Character data is malformed; no changes were applied")
		}
		out, err := savedPlayerView(world, &updated, actorLevel)
		if err != nil {
			return nil, err
		}
		if err := game.ValidateAliasRename(original.Name, newName); err != nil {
			return nil, huma.Error409Conflict("Could not move the character's alias file; no changes were applied")
		}
		prepareRename := func(snapshot []byte) error {
			return database.PrepareCharacterRename(original.Name, newName, snapshot)
		}
		if world != nil && world.Boards != nil {
			_, err = world.Boards.RenameAuthorWithPrepare(original.Name, newName, prepareRename)
			if err != nil {
				if recoveryErr := recoverPendingCharacterRenames(world, database); recoveryErr != nil {
					slog.Error("recover character references after board rename failure", "player", original.Name, "error", recoveryErr)
				}
				return nil, huma.Error409Conflict("Could not update the character's board headings; no changes were applied")
			}
		} else if err := prepareRename(nil); err != nil {
			return nil, huma.Error500InternalServerError("Could not prepare character rename; no changes were applied")
		}
		if _, err := game.RenameAliases(original.Name, newName); err != nil {
			if recoveryErr := recoverPendingCharacterRenames(world, database); recoveryErr != nil {
				slog.Error("recover character references after alias rename failure", "player", original.Name, "error", recoveryErr)
			}
			return nil, huma.Error409Conflict("Could not move the character's alias file; no changes were applied")
		}
		if err := database.RenamePlayerIfCurrent(&updated, original); err != nil {
			if recoveryErr := recoverPendingCharacterRenames(world, database); recoveryErr != nil {
				slog.Error("recover character references after database rename failure", "player", original.Name, "error", recoveryErr)
			}
			switch {
			case errors.Is(err, db.ErrPlayerRecordChanged):
				return nil, huma.Error409Conflict("Character changed. Reload before renaming.")
			case errors.Is(err, db.ErrPlayerNameTaken), errors.Is(err, db.ErrRenameReferenceConflict):
				return nil, huma.Error409Conflict("That name or one of its saved references is already in use")
			default:
				slog.Error("admin saved character rename failed", "player", original.Name, "new_name", newName, "error", err)
				return nil, huma.Error500InternalServerError("Could not rename character; no changes were applied")
			}
		}
		if renameObserver != nil {
			renameObserver.PlayerRenamed(original.Name, newName)
		}
		if externalRenameObserver != nil {
			externalRenameObserver.PlayerRenamed(original.Name, newName)
		}
		auditAdmin(ctx, logger, "admin_character_rename", fmt.Sprintf("renamed character %s to %s", original.Name, newName), true)
		return out, nil
	})
	huma.Register(api, huma.Operation{OperationID: "delete-saved-player", Method: http.MethodPost, Path: "/admin/characters/{name}/delete", Summary: "Delete an offline character from the admin console", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *deleteSavedPlayerInput) (*deleteSavedPlayerOutput, error) {
		claims, ok := auth.GetClaimsFromContext(ctx)
		if ok && strings.EqualFold(claims.PlayerName, in.Name) {
			return nil, huma.Error403Forbidden("You cannot delete your own character")
		}
		if reserver == nil {
			return nil, huma.Error503ServiceUnavailable("Offline editing is unavailable on this server")
		}
		release, ok := reserver.ReserveOfflinePlayer(in.Name)
		if !ok {
			return nil, huma.Error409Conflict("Character is connected or entering the game. Disconnect fully, then reload.")
		}
		defer release()
		r, err := database.GetPlayer(in.Name)
		if err != nil {
			return nil, huma.Error500InternalServerError("Could not read saved character")
		}
		if r == nil || game.CharacterDataDeleted(r.CharacterData) {
			return nil, huma.Error404NotFound("Saved character not found")
		}
		if r.Level >= game.LVL_IMPL {
			return nil, huma.Error403Forbidden("Level 40 characters cannot be deleted")
		}
		if in.Body.Revision != playerRevision(r) {
			return nil, huma.Error409Conflict("Character changed. Reload before deleting.")
		}
		if r.Level >= game.LVL_IMMORT && strings.TrimSpace(in.Body.ConfirmName) != r.Name {
			return nil, huma.Error400BadRequest("Type the character's full name to confirm deletion")
		}
		flags, err := game.CharacterDataFlags(r.CharacterData)
		if err != nil {
			return nil, huma.Error409Conflict("Character data is malformed; no changes were applied")
		}
		clanID, clanRank, err := game.CharacterDataClan(r.CharacterData)
		if err != nil {
			return nil, huma.Error409Conflict("Character data is malformed; no changes were applied")
		}
		clearClan := clanID != 0 || clanRank != 0
		updated := *r
		flags |= uint64(1) << uint(game.PlrDeleted)
		updated.CharacterData, err = game.SetCharacterDataFlags(updated.CharacterData, flags)
		if err != nil {
			return nil, huma.Error409Conflict("Character data is malformed; no changes were applied")
		}
		if clearClan {
			updated.CharacterData, err = game.SetCharacterDataClan(updated.CharacterData, 0, 0)
			if err != nil {
				return nil, huma.Error409Conflict("Character data is malformed; no changes were applied")
			}
		}
		updated.Inventory, updated.Equipment = []byte("[]"), []byte("{}")
		if err := database.DeleteCharacterAdminIfCurrent(&updated, r); err != nil {
			if errors.Is(err, db.ErrPlayerRecordChanged) {
				return nil, huma.Error409Conflict("Character changed. Reload before deleting.")
			}
			slog.Error("admin saved character deletion failed", "player", r.Name, "error", err)
			return nil, huma.Error500InternalServerError("Could not delete character; no changes were applied")
		}
		if clearClan && world != nil && world.Clans != nil {
			world.Clans.RemoveAdminDeletedMember(clanID, clanRank, r.Level)
		}
		aliasesRemoved := true
		if err := game.DeleteAliases(r.Name); err != nil {
			aliasesRemoved = false
			slog.Error("admin delete character aliases failed", "player", r.Name, "error", err)
		}
		markedDeleted := true
		auditAdmin(ctx, logger, "admin_character_delete", fmt.Sprintf("deleted character %s (level %d)", r.Name, r.Level), true)
		return &deleteSavedPlayerOutput{Body: deleteSavedPlayerResponse{Name: r.Name, MarkedDeleted: markedDeleted, AliasesRemoved: aliasesRemoved, ClanCleared: clearClan}}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "reset-saved-player-password", Method: http.MethodPost, Path: "/admin/characters/{name}/password", Summary: "Change a disconnected character's password", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *resetSavedPasswordInput) (*savedPlayerOutput, error) {
		if len(in.Body.Password) < 8 || len(in.Body.Password) > 72 {
			return nil, huma.Error400BadRequest("Use a password from 8 to 72 characters long")
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
	huma.Register(api, huma.Operation{OperationID: "edit-saved-player-items", Method: http.MethodPost, Path: "/admin/characters/{name}/items", Summary: "Add, remove, or move a character's items", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *editPlayerItemsInput) (*savedPlayerOutput, error) {
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
	return staffAdminGate(database, "Player editing requires admin access")
}

func staffAdminGate(database *db.DB, denied string) func(huma.Context, func(huma.Context)) {
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
			writeOLCError(ctx, 403, olcMiddlewareError{Error: denied})
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

func savedPlayerView(world *game.World, r *db.PlayerRecord, actorLevel int) (*savedPlayerOutput, error) {
	var lastSaved *time.Time
	if r.LastLogon > 0 {
		timestamp := time.Unix(r.LastLogon, 0).UTC()
		lastSaved = &timestamp
	}
	out := savedPlayerResponse{Name: r.Name, LastSavedAt: lastSaved, Revision: playerRevision(r), Fields: playerFields, Values: map[string]int{}, Inventory: []savedPlayerItem{}, Equipment: []savedPlayerItem{}, Slots: []playerSlot{}}
	flags, err := playerFlagCatalog(r.CharacterData, actorLevel, r.Level)
	if err != nil {
		return nil, huma.Error409Conflict("Character flags are malformed")
	}
	out.Flags = flags
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
			noRent := false
			if proto, ok := world.GetObjPrototype(item.VNum); ok {
				name = proto.ShortDesc
				noRent = proto.ExtraFlags[0]&game.FlagNoRent != 0
			}
			row := savedPlayerItem{Index: i, VNum: item.VNum, Name: name, Count: max(1, item.Count), Locate: item.Locate, Parent: item.ContainerIndex, NoRent: noRent}
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
