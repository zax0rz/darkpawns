package admin

import (
	"context"
	"fmt"
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

type webOLCOwner struct {
	identity string
	name     string
}

func (o webOLCOwner) Identity() string    { return o.identity }
func (o webOLCOwner) DisplayName() string { return o.name }
func (o webOLCOwner) Frontend() olc.Frontend {
	return olc.FrontendWeb
}

type roomDraftPathInput struct {
	VNum string `path:"vnum" doc:"Room VNUM."`
}

type roomDraftOutput struct {
	Body roomDraftBody
}

type roomDraftBody struct {
	Kind                  string      `json:"kind"`
	VNum                  int         `json:"vnum"`
	Room                  parser.Room `json:"room"`
	Dirty                 []string    `json:"dirty"`
	LeaseExpiresAt        time.Time   `json:"lease_expires_at"`
	LeaseRemainingSeconds int64       `json:"lease_remaining_seconds"`
}

type roomPatchInput struct {
	VNum string               `path:"vnum" doc:"Room VNUM."`
	Body []roomPatchOperation `json:"-"`
}

type roomPatchOperation struct {
	Kind       string `json:"kind" doc:"Semantic OLC operation."`
	Text       string `json:"text,omitempty"`
	Keywords   string `json:"keywords,omitempty"`
	Direction  string `json:"direction,omitempty"`
	Value      int    `json:"value,omitempty"`
	Bit        int    `json:"bit,omitempty"`
	Index      int    `json:"index,omitempty"`
	SourceVNum int    `json:"source_vnum,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

type olcConflictError struct {
	Message  string `json:"error"`
	Holder   string `json:"holder"`
	Frontend string `json:"frontend"`
	Idle     string `json:"idle"`
	IdleSecs int64  `json:"idle_seconds"`
}

func (e *olcConflictError) Error() string  { return e.Message }
func (e *olcConflictError) GetStatus() int { return http.StatusConflict }

type webOLCError struct {
	Message string `json:"error"`
}

func (e *webOLCError) Error() string  { return e.Message }
func (e *webOLCError) GetStatus() int { return http.StatusBadRequest }

func registerOLCRoomWrites(api huma.API, world *game.World, database *db.DB, writes OLCWriteStateProvider, presence OLCPresenceProvider, auditLogger *audit.AuditLogger, drafts *olc.DraftStore) {
	gate := olcGate(world, database)

	huma.Register(api, huma.Operation{
		OperationID: "open-olc-room-draft",
		Method:      http.MethodPost,
		Path:        "/admin/olc/room/{vnum}",
		Summary:     "Open a room OLC draft",
		Description: "Claims a room lease and opens or recovers the player's in-memory REDIT working copy.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *roomDraftPathInput) (*roomDraftOutput, error) {
		vnum, err := parseRoomVNum(in.VNum)
		if err != nil {
			return nil, err
		}
		owner, err := webOwner(ctx)
		if err != nil {
			return nil, err
		}
		if writes == nil || drafts == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
		}
		room, exists := world.SnapshotRoom(vnum)
		if !exists {
			return nil, huma.NewError(http.StatusNotFound, "room not found")
		}
		_, ok := olc.ZoneForVNum(world.GetAllZones(), vnum)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "no zone covers VNUM")
		}
		hadClaim := hasOLCClaim(writes, olc.KindRoom, vnum, owner)
		if holder, claimed := writes.ClaimOLC(olc.KindRoom, vnum, owner, olc.DefaultClaimTTL); !claimed {
			return nil, roomClaimConflict(writes, olc.KindRoom, vnum, holder)
		}
		draft, err := drafts.Open(owner.Identity(), olc.KindRoom, vnum, room)
		if err != nil {
			if !hadClaim {
				writes.ReleaseOLC(olc.KindRoom, vnum, owner)
			}
			return nil, huma.NewError(http.StatusConflict, err.Error())
		}
		if !hadClaim && presence != nil {
			presence.EmitOLCPresence(owner.DisplayName(), true)
		}
		return &roomDraftOutput{Body: makeRoomDraftBody(draft, claimEntry(writes, olc.KindRoom, vnum))}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-olc-room-draft",
		Method:      http.MethodGet,
		Path:        "/admin/olc/room/{vnum}/draft",
		Summary:     "Get a room OLC draft",
		Description: "Returns the effective room working copy, changed field set, and remaining lease.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *roomDraftPathInput) (*roomDraftOutput, error) {
		return accessRoomDraft(ctx, in, writes, drafts)
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-olc-room-draft",
		Method:      http.MethodPatch,
		Path:        "/admin/olc/room/{vnum}/draft",
		Summary:     "Patch a room OLC draft",
		Description: "Applies an ordered REDIT operation list and returns the effective working copy.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *roomPatchInput) (*roomDraftOutput, error) {
		vnum, err := parseRoomVNum(in.VNum)
		if err != nil {
			return nil, err
		}
		owner, err := webOwner(ctx)
		if err != nil {
			return nil, err
		}
		if writes == nil || drafts == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
		}
		if !writes.RenewOLC(olc.KindRoom, vnum, owner, olc.DefaultClaimTTL) {
			return nil, roomLeaseExpired(writes, olc.KindRoom, vnum)
		}
		operations := make([]olc.Operation, 0, len(in.Body))
		for _, input := range in.Body {
			operation, err := roomOperation(world, vnum, input)
			if err != nil {
				return nil, err
			}
			operations = append(operations, operation)
		}
		draft, err := drafts.Patch(owner.Identity(), operations)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, err.Error())
		}
		return &roomDraftOutput{Body: makeRoomDraftBody(draft, claimEntry(writes, olc.KindRoom, vnum))}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "commit-olc-room-draft",
		Method:      http.MethodPost,
		Path:        "/admin/olc/room/{vnum}/draft/commit",
		Summary:     "Commit a room OLC draft",
		Description: "Commits the effective room to memory, marks its zone dirty, releases the lease, and consumes the draft.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *roomDraftPathInput) (*roomDraftOutput, error) {
		vnum, err := parseRoomVNum(in.VNum)
		if err != nil {
			return nil, err
		}
		owner, err := webOwner(ctx)
		if err != nil {
			return nil, err
		}
		if writes == nil || drafts == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
		}
		var committed olc.Draft
		committed, err = drafts.CommitWith(owner.Identity(), func(draft olc.Draft) error {
			if draft.Kind != olc.KindRoom || draft.VNum != vnum {
				return huma.NewError(http.StatusConflict, "draft does not match room")
			}
			if !writes.RenewOLC(olc.KindRoom, vnum, owner, olc.DefaultClaimTTL) {
				return roomLeaseExpired(writes, olc.KindRoom, vnum)
			}
			zone, ok := olc.ZoneForVNum(world.GetAllZones(), vnum)
			if !ok {
				return huma.NewError(http.StatusNotFound, "no zone covers VNUM")
			}
			lock := olc.ZoneSaveLock(zone.Number)
			lock.Lock()
			ok = olc.CommitRoom(olc.RoomCommitInput{
				Draft:      draft,
				Actor:      owner.DisplayName(),
				IPAddress:  clientIPFrom(ctx),
				Commit:     world.CommitEditedRoom,
				LiveScript: func() (parser.Room, bool) { return world.SnapshotRoom(vnum) },
				MarkDirty:  func() { writes.MarkOLCDirty(olc.KindRoom, zone.Number) },
				Audit: func(event olc.AuditEvent) {
					if auditLogger != nil {
						auditLogger.Log(audit.AuditEvent{
							EventType: "olc",
							User:      event.User,
							IPAddress: event.IPAddress,
							Action:    event.Action,
							Details:   event.Details,
							Success:   event.Success,
						})
					}
				},
			})
			lock.Unlock()
			if !ok {
				return huma.NewError(http.StatusConflict, "room commit failed")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		writes.ReleaseOLC(olc.KindRoom, vnum, owner)
		if presence != nil {
			presence.EmitOLCPresence(owner.DisplayName(), false)
		}
		return &roomDraftOutput{Body: makeRoomDraftBody(committed, olc.ClaimEntry{})}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "discard-olc-room-draft",
		Method:        http.MethodDelete,
		Path:          "/admin/olc/room/{vnum}/draft",
		Summary:       "Discard a room OLC draft",
		Description:   "Discards the server-side draft, releases the lease, and leaves the world untouched.",
		Middlewares:   huma.Middlewares{gate},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *roomDraftPathInput) (*struct{}, error) {
		vnum, err := parseRoomVNum(in.VNum)
		if err != nil {
			return nil, err
		}
		owner, err := webOwner(ctx)
		if err != nil {
			return nil, err
		}
		if writes == nil || drafts == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
		}
		draft, ok := drafts.Get(owner.Identity())
		if !ok || draft.Kind != olc.KindRoom || draft.VNum != vnum {
			return nil, huma.NewError(http.StatusNotFound, "OLC draft not found")
		}
		drafts.Discard(owner.Identity())
		writes.ReleaseOLC(olc.KindRoom, vnum, owner)
		if presence != nil {
			presence.EmitOLCPresence(owner.DisplayName(), false)
		}
		return &struct{}{}, nil
	})
}

func accessRoomDraft(ctx context.Context, in *roomDraftPathInput, writes OLCWriteStateProvider, drafts *olc.DraftStore) (*roomDraftOutput, error) {
	vnum, err := parseRoomVNum(in.VNum)
	if err != nil {
		return nil, err
	}
	owner, err := webOwner(ctx)
	if err != nil {
		return nil, err
	}
	if writes == nil || drafts == nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
	}
	if !writes.RenewOLC(olc.KindRoom, vnum, owner, olc.DefaultClaimTTL) {
		return nil, roomLeaseExpired(writes, olc.KindRoom, vnum)
	}
	draft, ok := drafts.Get(owner.Identity())
	if !ok || draft.Kind != olc.KindRoom || draft.VNum != vnum {
		return nil, huma.NewError(http.StatusNotFound, "OLC draft not found")
	}
	return &roomDraftOutput{Body: makeRoomDraftBody(draft, claimEntry(writes, olc.KindRoom, vnum))}, nil
}

func roomOperation(world *game.World, vnum int, input roomPatchOperation) (olc.Operation, error) {
	op := olc.Operation{Text: input.Text, Value: input.Value, Bit: input.Bit, Index: input.Index, Direction: input.Direction}
	if input.Enabled != nil {
		if *input.Enabled {
			op.Value = 1
		} else {
			op.Value = 0
		}
	}
	switch input.Kind {
	case "set_room_name":
		op.Kind = olc.OpSetRoomName
	case "set_room_description":
		op.Kind = olc.OpSetRoomDescription
	case "set_room_flag":
		op.Kind = olc.OpSetRoomFlag
	case "set_room_sector":
		op.Kind = olc.OpSetRoomSector
	case "set_script_name":
		op.Kind = olc.OpSetRoomScriptName
		op.SetScriptName = func(name string) bool {
			room, ok := world.SnapshotRoom(vnum)
			return ok && world.SetRoomScript(vnum, room.ScriptFunctions, name)
		}
	case "set_script_flag":
		op.Kind = olc.OpSetRoomScriptFlag
		op.SetScriptFlag = func(bit int, enabled bool) bool {
			room, ok := world.SnapshotRoom(vnum)
			if !ok {
				return false
			}
			flags := room.ScriptFunctions
			if enabled {
				flags |= 1 << uint(bit)
			} else {
				flags &^= 1 << uint(bit)
			}
			return world.SetRoomScript(vnum, flags, room.ScriptName)
		}
	case "ensure_exit":
		op.Kind = olc.OpEnsureExit
	case "set_exit_target":
		op.Kind = olc.OpSetExitTarget
		op.RoomExists = func(vnum int) bool {
			_, ok := world.SnapshotRoom(vnum)
			return ok
		}
	case "set_exit_description":
		op.Kind = olc.OpSetExitDescription
	case "set_exit_keywords":
		op.Kind = olc.OpSetExitKeywords
	case "set_exit_key":
		op.Kind = olc.OpSetExitKey
	case "set_exit_door_flags":
		op.Kind = olc.OpSetExitDoorFlags
	case "purge_exit":
		op.Kind = olc.OpPurgeExit
	case "add_extra_description":
		op.Kind = olc.OpAddExtraDescription
		op.Index = -1
		op.Extra = &parser.ExtraDesc{Keywords: input.Keywords, Description: input.Text}
	case "remove_extra_description":
		op.Kind = olc.OpRemoveExtraDescription
	case "set_extra_keyword":
		op.Kind = olc.OpSetExtraKeywords
		op.Text = input.Keywords
		if input.Keywords == "" {
			op.Text = input.Text
		}
	case "set_extra_description":
		op.Kind = olc.OpSetExtraDescription
	case "copy_room":
		op.Kind = olc.OpCopyRoom
		source, ok := world.SnapshotRoom(input.SourceVNum)
		if !ok {
			return olc.Operation{}, huma.NewError(http.StatusBadRequest, "source room not found")
		}
		op.Source = &source
	default:
		return olc.Operation{}, &webOLCError{Message: fmt.Sprintf("unknown room operation %q", input.Kind)}
	}
	return op, nil
}

func makeRoomDraftBody(draft olc.Draft, entry olc.ClaimEntry) roomDraftBody {
	body := roomDraftBody{
		Kind:  string(draft.Kind),
		VNum:  draft.VNum,
		Room:  draft.Effective(),
		Dirty: draft.Diff(draft.Snapshot),
	}
	if !entry.ExpiresAt.IsZero() {
		body.LeaseExpiresAt = entry.ExpiresAt
		remaining := time.Until(entry.ExpiresAt)
		if remaining > 0 {
			body.LeaseRemainingSeconds = int64(remaining / time.Second)
			if remaining%time.Second != 0 {
				body.LeaseRemainingSeconds++
			}
		}
	}
	return body
}

func parseRoomVNum(raw string) (int, error) {
	vnum, err := strconv.Atoi(raw)
	if err != nil {
		return 0, huma.NewError(http.StatusBadRequest, "invalid vnum")
	}
	return vnum, nil
}

func webOwner(ctx context.Context) (webOLCOwner, error) {
	claims, ok := auth.GetClaimsFromContext(ctx)
	if !ok || claims == nil {
		return webOLCOwner{}, huma.NewError(http.StatusUnauthorized, "unauthorized")
	}
	return webOLCOwner{identity: claims.PlayerName, name: claims.PlayerName}, nil
}

func claimEntry(writes OLCWriteStateProvider, kind olc.Kind, vnum int) olc.ClaimEntry {
	for _, entry := range writes.GetOLCClaims() {
		if entry.Kind == kind && entry.Number == vnum {
			return entry
		}
	}
	return olc.ClaimEntry{}
}

func hasOLCClaim(writes OLCWriteStateProvider, kind olc.Kind, vnum int, owner olc.Owner) bool {
	for _, entry := range writes.GetOLCClaims() {
		if entry.Kind == kind && entry.Number == vnum && entry.OwnerIdentity == owner.Identity() && entry.OwnerFrontend == owner.Frontend() {
			return true
		}
	}
	return false
}

func roomClaimConflict(writes OLCWriteStateProvider, kind olc.Kind, vnum int, holder olc.Owner) error {
	entry := claimEntry(writes, kind, vnum)
	if entry.OwnerDisplayName == "" && holder != nil {
		entry.OwnerDisplayName = holder.DisplayName()
		entry.OwnerFrontend = holder.Frontend()
	}
	idle := time.Since(entry.ClaimedAt)
	if idle < 0 || entry.ClaimedAt.IsZero() {
		idle = 0
	}
	return &olcConflictError{
		Message:  "room is already claimed",
		Holder:   entry.OwnerDisplayName,
		Frontend: string(entry.OwnerFrontend),
		Idle:     idle.Round(time.Second).String(),
		IdleSecs: int64(idle / time.Second),
	}
}

func roomLeaseExpired(writes OLCWriteStateProvider, kind olc.Kind, vnum int) error {
	entry := claimEntry(writes, kind, vnum)
	if entry.OwnerDisplayName != "" {
		return roomClaimConflict(writes, kind, vnum, nil)
	}
	return &olcConflictError{Message: "draft lease expired; reopen the draft"}
}
