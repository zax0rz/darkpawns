package olc

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

type MobCommitInput struct {
	Draft      EntityDraft
	Actor      string
	IPAddress  string
	Commit     func(parser.Mob) bool
	LiveScript func() (parser.Mob, bool)
	MarkDirty  func()
	Audit      func(AuditEvent)
}

type ObjectCommitInput struct {
	Draft      EntityDraft
	Actor      string
	IPAddress  string
	Commit     func(parser.Obj) bool
	LiveScript func() (parser.Obj, bool)
	MarkDirty  func()
	Audit      func(AuditEvent)
}

type ShopCommitInput struct {
	Draft     EntityDraft
	Actor     string
	IPAddress string
	Commit    func(parser.ShopProto) bool
	MarkDirty func()
	Audit     func(AuditEvent)
}

type ZoneCommitInput struct {
	Draft     EntityDraft
	Actor     string
	IPAddress string
	Commit    func(int, parser.Zone) bool
	MarkDirty func()
	Audit     func(AuditEvent)
}

// AuditEvent is the transport-neutral event emitted by the shared OLC memory
// commit. Frontends adapt it to pkg/audit without making this layer know about
// descriptors, HTTP, or session state.
type AuditEvent struct {
	User      string
	IPAddress string
	Action    string
	Details   string
	Success   bool
}

// RoomCommitInput supplies the lock-owned side effects that differ between
// game and admin packages. CommitRoom itself owns the ordering and the audit
// payload so both frontends report the same changed-field-only details.
type RoomCommitInput struct {
	Draft      Draft
	Actor      string
	IPAddress  string
	Commit     func(parser.Room) bool
	LiveScript func() (parser.Room, bool)
	MarkDirty  func()
	Audit      func(AuditEvent)
}

// CommitRoom applies a room working copy to the world and marks its zone
// dirty. Callers must hold ZoneSaveLock(input.Draft.Working.Zone) around this
// function. No descriptor or frontend concerns enter this primitive.
func CommitRoom(input RoomCommitInput) bool {
	draft := input.Draft
	if input.LiveScript != nil {
		if live, ok := input.LiveScript(); ok {
			// C's REDIT copy shares the live script storage. Re-read those
			// fields immediately before replacing the whole room so a live
			// script edit cannot be clobbered by the stale draft copy.
			draft.Working.ScriptName = live.ScriptName
			draft.Working.ScriptFunctions = live.ScriptFunctions
		}
	}
	fields := strings.Join(draft.Diff(draft.Snapshot), ",")
	success := input.Commit != nil && input.Commit(draft.Effective())
	if success && input.MarkDirty != nil {
		input.MarkDirty()
	}
	if input.Audit != nil {
		input.Audit(AuditEvent{
			User:      input.Actor,
			IPAddress: input.IPAddress,
			Action:    "olc_room_commit",
			Details:   "fields=" + fields,
			Success:   success,
		})
	}
	return success
}

func CommitMob(input MobCommitInput) bool {
	draft := input.Draft
	if input.LiveScript != nil {
		if live, ok := input.LiveScript(); ok {
			// C's MEDIT copy shares the live script storage. Re-read those
			// fields immediately before replacing the whole prototype so a
			// live script edit cannot be clobbered by the stale draft copy.
			draft.Working.Mob.ScriptName = live.ScriptName
			draft.Working.Mob.LuaFunctions = live.LuaFunctions
		}
	}
	return commitEntity("mob", draft, input.Actor, input.IPAddress,
		func() bool { return input.Commit != nil && input.Commit(draft.Effective().Mob) },
		input.MarkDirty, input.Audit)
}

func CommitObj(input ObjectCommitInput) bool {
	draft := input.Draft
	if input.LiveScript != nil {
		if live, ok := input.LiveScript(); ok {
			// C's OEDIT copy shares the live script storage. Keep the same
			// shallow-copy behavior across a whole-prototype commit.
			draft.Working.Object.ScriptName = live.ScriptName
			draft.Working.Object.LuaFunctions = live.LuaFunctions
		}
	}
	return commitEntity("object", draft, input.Actor, input.IPAddress,
		func() bool { return input.Commit != nil && input.Commit(draft.Effective().Object) },
		input.MarkDirty, input.Audit)
}

func CommitShop(input ShopCommitInput) bool {
	return commitEntity("shop", input.Draft, input.Actor, input.IPAddress,
		func() bool { return input.Commit != nil && input.Commit(input.Draft.Effective().Shop) },
		input.MarkDirty, input.Audit)
}

func CommitZone(input ZoneCommitInput) bool {
	working := input.Draft.Effective().Zone
	return commitEntity("zone", input.Draft, input.Actor, input.IPAddress,
		func() bool { return input.Commit != nil && input.Commit(working.RoomVNum, working.Zone) },
		input.MarkDirty, input.Audit)
}

func commitEntity(kind string, draft EntityDraft, actor, ip string, commit func() bool, markDirty func(), audit func(AuditEvent)) bool {
	fields := strings.Join(draft.Diff(), ",")
	success := commit()
	if success && markDirty != nil {
		markDirty()
	}
	if audit != nil {
		audit(AuditEvent{
			User:      actor,
			IPAddress: ip,
			Action:    "olc_" + kind + "_commit",
			Details:   "fields=" + fields,
			Success:   success,
		})
	}
	return success
}
