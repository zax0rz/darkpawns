package olc

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

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
	Draft     Draft
	Actor     string
	IPAddress string
	Commit    func(parser.Room) bool
	MarkDirty func()
	Audit     func(AuditEvent)
}

// CommitRoom applies a room working copy to the world and marks its zone
// dirty. Callers must hold ZoneSaveLock(input.Draft.Working.Zone) around this
// function. No descriptor or frontend concerns enter this primitive.
func CommitRoom(input RoomCommitInput) bool {
	fields := strings.Join(input.Draft.Diff(input.Draft.Snapshot), ",")
	success := input.Commit != nil && input.Commit(input.Draft.Effective())
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
