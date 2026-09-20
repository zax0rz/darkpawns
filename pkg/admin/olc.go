package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// OLCReadStateProvider exposes copied OLC state to the admin read surface.
// The session manager owns the registry and dirty list; admin never receives
// their locks or live owner references.
type OLCReadStateProvider interface {
	GetOLCClaims() []olc.ClaimEntry
	GetOLCDirtyZones() []olc.DirtyEntry
}

// OLCWriteStateProvider is the narrow write seam between Huma and the
// session-owned registry/dirty list. Admin never receives either live map.
type OLCWriteStateProvider interface {
	OLCReadStateProvider
	ClaimOLC(kind olc.Kind, number int, owner olc.Owner, ttl time.Duration) (olc.Owner, bool)
	RenewOLC(kind olc.Kind, number int, owner olc.Owner, ttl time.Duration) bool
	ReleaseOLC(kind olc.Kind, number int, owner olc.Owner)
	MarkOLCDirty(kind olc.Kind, zone int)
}

// OLCPresenceProvider emits the world emote for a web claim when the player
// is online. It intentionally has no PlrWriting operation.
type OLCPresenceProvider interface {
	EmitOLCPresence(playerName string, start bool)
}

type olcPreviewInput struct {
	Kind string `path:"kind" doc:"OLC kind: room, mob, obj, shop, or zone."`
	VNum int    `path:"vnum" doc:"Prototype VNUM, shop VNUM, or room VNUM for a zone command view."`
}

type olcPreviewOutput struct {
	Body olcPreviewResponse
}

// olcPreviewResponse is a typed union. Exactly one entity field is populated;
// zone previews use the room VNUM in the URL and expose the same stale-carry
// command view that the telnet zedit setup presents.
type olcPreviewResponse struct {
	Kind       string          `json:"kind"`
	VNum       int             `json:"vnum"`
	ZoneNumber int             `json:"zone_number"`
	Room       *parser.Room    `json:"room,omitempty"`
	Mob        *parser.Mob     `json:"mob,omitempty"`
	Object     *parser.Obj     `json:"object,omitempty"`
	Shop       *game.Shop      `json:"shop,omitempty"`
	Zone       *olcZonePreview `json:"zone,omitempty"`
}

type olcZonePreview struct {
	Number    int                  `json:"number"`
	Name      string               `json:"name"`
	TopRoom   int                  `json:"top_room"`
	Lifespan  int                  `json:"lifespan"`
	ResetMode int                  `json:"reset_mode"`
	Commands  []parser.ZoneCommand `json:"commands"`
}

type olcHeldOutput struct {
	Body []olc.ClaimEntry
}

type olcPendingOutput struct {
	Body []olc.DirtyEntry
}

// registerOLC registers the P3 read operations. They are mounted individually
// by newRouter so no /admin/olc/* path-prefix route or drift allowlist entry is
// needed.
func registerOLC(api huma.API, world *game.World, database *db.DB, state OLCReadStateProvider, writes OLCWriteStateProvider, presence OLCPresenceProvider, auditLogger *audit.AuditLogger, drafts *olc.DraftStore) {
	gate := olcGate(world, database)

	huma.Register(api, huma.Operation{
		OperationID: "preview-olc-resource",
		Method:      http.MethodGet,
		Path:        "/admin/olc/{kind}/{vnum}/preview",
		Summary:     "Preview an OLC resource",
		Description: "Read-only working-copy preview of a room, mob, object, shop, or room-scoped zone reset view.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *olcPreviewInput) (*olcPreviewOutput, error) {
		preview, err := olcPreview(world, in.Kind, in.VNum)
		if err != nil {
			return nil, err
		}
		return &olcPreviewOutput{Body: preview}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-olc-held",
		Method:      http.MethodGet,
		Path:        "/admin/olc/held",
		Summary:     "List held OLC claims",
		Description: "All active OLC claims, including owner identity, frontend, and acquisition time.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *struct{}) (*olcHeldOutput, error) {
		if state == nil {
			return &olcHeldOutput{Body: []olc.ClaimEntry{}}, nil
		}
		return &olcHeldOutput{Body: state.GetOLCClaims()}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-olc-pending",
		Method:      http.MethodGet,
		Path:        "/admin/olc/pending",
		Summary:     "List pending OLC saves",
		Description: "All editor kinds and zones marked dirty in memory and awaiting a disk save.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *struct{}) (*olcPendingOutput, error) {
		if state == nil {
			return &olcPendingOutput{Body: []olc.DirtyEntry{}}, nil
		}
		return &olcPendingOutput{Body: state.GetOLCDirtyZones()}, nil
	})

	registerOLCRoomWrites(api, world, database, writes, presence, auditLogger, drafts)
}

func olcPreview(world *game.World, kind string, vnum int) (olcPreviewResponse, error) {
	entityKind, ok := olcKindForPath(kind)
	if !ok {
		return olcPreviewResponse{}, huma.NewError(http.StatusBadRequest, "unknown OLC kind")
	}
	zone, ok := olc.ZoneForVNum(world.GetAllZones(), vnum)
	if !ok {
		return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "no zone covers VNUM")
	}

	preview := olcPreviewResponse{
		Kind:       kind,
		VNum:       vnum,
		ZoneNumber: zone.Number,
	}
	switch entityKind {
	case olc.KindRoom:
		room, exists := world.SnapshotRoom(vnum)
		if !exists {
			return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "room not found")
		}
		preview.Room = &room
	case olc.KindMob:
		mob, exists := world.SnapshotMob(vnum)
		if !exists {
			return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "mob not found")
		}
		preview.Mob = &mob
	case olc.KindObject:
		obj, exists := world.SnapshotObj(vnum)
		if !exists {
			return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "object not found")
		}
		preview.Object = &obj
	case olc.KindShop:
		shop, exists := world.SnapshotShop(vnum)
		if !exists {
			return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "shop not found")
		}
		preview.Shop = &shop
	case olc.KindZone:
		if _, exists := world.SnapshotRoom(vnum); !exists {
			return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "room not found")
		}
		zoneCopy, exists := world.SnapshotZone(zone.Number)
		if !exists {
			return olcPreviewResponse{}, huma.NewError(http.StatusNotFound, "zone not found")
		}
		preview.Zone = &olcZonePreview{
			Number:    zoneCopy.Number,
			Name:      zoneCopy.Name,
			TopRoom:   zoneCopy.TopRoom,
			Lifespan:  zoneCopy.Lifespan,
			ResetMode: zoneCopy.ResetMode,
			Commands:  zoneCommandsForRoom(zoneCopy.Commands, vnum),
		}
	}
	return preview, nil
}

func olcKindForPath(kind string) (olc.Kind, bool) {
	switch kind {
	case "room":
		return olc.KindRoom, true
	case "mob":
		return olc.KindMob, true
	case "obj", "object":
		return olc.KindObject, true
	case "shop":
		return olc.KindShop, true
	case "zone":
		return olc.KindZone, true
	default:
		return "", false
	}
}

// zoneCommandsForRoom mirrors zeditSetupCommands, including the C stale-carry
// rule that G/E/P/* commands inherit the previous room-bearing command.
func zoneCommandsForRoom(commands []parser.ZoneCommand, roomVNum int) []parser.ZoneCommand {
	filtered := make([]parser.ZoneCommand, 0)
	cmdRoom := -1
	for _, command := range commands {
		if room, hasRoom := game.ZoneCommandRoom(command); hasRoom {
			cmdRoom = room
		}
		if cmdRoom == roomVNum {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

type olcMiddlewareError struct {
	Error    string `json:"error"`
	Required string `json:"required,omitempty"`
	Zone     *int   `json:"zone,omitempty"`
}

// olcGate is operation middleware rather than a path-prefix authorization
// wrapper so Huma has already selected the operation and exposed its path
// parameters. The level/zone rule itself remains exclusively in pkg/olc.
func olcGate(world *game.World, database *db.DB) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		claims, ok := auth.GetClaimsFromContext(ctx.Context())
		if !ok {
			var err error
			claims, err = claimsFromBearerHeader(ctx.Header("Authorization"))
			if err != nil {
				writeOLCError(ctx, http.StatusUnauthorized, olcMiddlewareError{Error: "unauthorized"})
				return
			}
		}
		if !claims.HasRole("builder") {
			writeOLCError(ctx, http.StatusForbidden, olcMiddlewareError{Error: "forbidden", Required: "builder"})
			return
		}
		if database == nil || world == nil {
			writeOLCError(ctx, http.StatusServiceUnavailable, olcMiddlewareError{Error: "OLC authorization unavailable"})
			return
		}

		record, err := database.GetPlayer(claims.PlayerName)
		if err != nil || record == nil {
			writeOLCError(ctx, http.StatusUnauthorized, olcMiddlewareError{Error: "unauthorized"})
			return
		}

		zoneNumber := record.OlcZone
		if rawVNum := ctx.Param("vnum"); rawVNum != "" {
			vnum, err := strconv.Atoi(rawVNum)
			if err != nil {
				writeOLCError(ctx, http.StatusBadRequest, olcMiddlewareError{Error: "invalid vnum"})
				return
			}
			zone, ok := olc.ZoneForVNum(world.GetAllZones(), vnum)
			if !ok {
				writeOLCError(ctx, http.StatusNotFound, olcMiddlewareError{Error: "no zone covers VNUM"})
				return
			}
			zoneNumber = zone.Number
		} else if zone, ok := olc.ZoneForVNum(world.GetAllZones(), record.OlcZone*100); ok {
			zoneNumber = zone.Number
		}
		if !olc.Authorized(record.Level, record.OlcZone, zoneNumber) {
			zone := zoneNumber
			writeOLCError(ctx, http.StatusForbidden, olcMiddlewareError{
				Error: "not authorized for OLC zone",
				Zone:  &zone,
			})
			return
		}
		// The exact route mount may be used without requireRole in tests or by
		// another caller. Preserve the validated identity for the operation
		// handler whenever this gate had to parse the bearer header itself.
		ctx = huma.WithContext(ctx, auth.SetClaimsOnContext(ctx.Context(), claims))
		next(ctx)
	}
}

func writeOLCError(ctx huma.Context, status int, body olcMiddlewareError) {
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetStatus(status)
	if err := json.NewEncoder(ctx.BodyWriter()).Encode(body); err != nil {
		slog.Warn("OLC middleware error response failed", "error", err)
	}
}
