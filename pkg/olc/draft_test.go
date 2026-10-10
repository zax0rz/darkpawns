package olc

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneCommandsForRoomUsesStickyRoomCarry(t *testing.T) {
	commands := []parser.ZoneCommand{
		{Command: "M", Arg3: 1001},
		{Command: "G", Arg1: 3001},
		{Command: "E", Arg1: 3002},
		{Command: "O", Arg3: 1002},
		{Command: "P", Arg1: 3003, Arg3: 3004},
		{Command: "D", Arg1: 1001},
		{Command: "R", Arg1: 1001},
	}
	filtered := ZoneCommandsForRoom(commands, 1001)
	if len(filtered) != 5 {
		t.Fatalf("filtered commands = %+v, want M/G/E/D/R", filtered)
	}
	for index, want := range []string{"M", "G", "E", "D", "R"} {
		if filtered[index].Command != want {
			t.Fatalf("filtered[%d] = %q, want %q", index, filtered[index].Command, want)
		}
	}
}

func TestDraftSurvivesExpiredClaim(t *testing.T) {
	registry := NewRegistry()
	owner := testOwner{id: "builder-1", name: "Builder", frontend: FrontendWeb}
	store := NewDraftStore()
	if _, err := store.Open(owner.Identity(), KindRoom, 1001, parser.Room{VNum: 1001}); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Claim(KindRoom, 1001, owner, 5*time.Millisecond); !ok {
		t.Fatal("claim refused")
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok := registry.Holder(KindRoom, 1001); ok {
		t.Fatal("expired claim still held")
	}
	if _, ok := store.Get(owner.Identity()); !ok {
		t.Fatal("draft was discarded with lease")
	}
	if _, ok := registry.Claim(KindRoom, 1001, owner, DefaultClaimTTL); !ok {
		t.Fatal("re-open could not reclaim expired room")
	}
}

func TestEntityDraftSurvivesLeaseExpiryForEveryKind(t *testing.T) {
	tests := []struct {
		name  string
		kind  Kind
		vnum  int
		value EntityValue
	}{
		{"mob", KindMob, 2001, EntityValue{Mob: parser.Mob{VNum: 2001}}},
		{"object", KindObject, 3001, EntityValue{Object: parser.Obj{VNum: 3001}}},
		{"shop", KindShop, 4001, EntityValue{Shop: parser.ShopProto{VNum: 4001}}},
		{"zone", KindZone, 1001, EntityValue{Zone: ZoneDraft{Zone: parser.Zone{Number: 1}, RoomVNum: 1001}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			store := NewEntityDraftStore()
			owner := testOwner{id: test.name, name: test.name, frontend: FrontendWeb}
			if _, err := store.Open(owner.Identity(), test.kind, test.vnum, test.value); err != nil {
				t.Fatal(err)
			}
			if _, ok := registry.Claim(test.kind, test.vnum, owner, 5*time.Millisecond); !ok {
				t.Fatal("claim refused")
			}
			time.Sleep(15 * time.Millisecond)
			if _, ok := registry.Holder(test.kind, test.vnum); ok {
				t.Fatal("expired claim still held")
			}
			if _, ok := store.Get(owner.Identity()); !ok {
				t.Fatal("draft was discarded with lease")
			}
			if _, ok := registry.Claim(test.kind, test.vnum, owner, DefaultClaimTTL); !ok {
				t.Fatal("re-open could not reclaim expired claim")
			}
		})
	}
}

func TestCommitRoomAuditsFieldsWithoutValues(t *testing.T) {
	events := make([]AuditEvent, 0, 1)
	draft := Draft{
		Kind:     KindRoom,
		VNum:     1001,
		Snapshot: parser.Room{VNum: 1001, Name: "old", Exits: map[string]parser.Exit{}},
		Working:  parser.Room{VNum: 1001, Name: "new-secret-name", Exits: map[string]parser.Exit{}},
	}
	committed := false
	if !CommitRoom(RoomCommitInput{
		Draft: draft,
		Actor: "Builder",
		Commit: func(room parser.Room) bool {
			committed = room.Name == "new-secret-name"
			return committed
		},
		MarkDirty: func() {},
		Audit:     func(event AuditEvent) { events = append(events, event) },
	}) {
		t.Fatal("commit failed")
	}
	if len(events) != 1 || events[0].Details != "fields=name" {
		t.Fatalf("audit event = %#v", events)
	}
	if strings.Contains(events[0].Details, "secret") {
		t.Fatalf("audit event leaked a field value: %#v", events[0])
	}
}

func TestCommitRoomPreservesLiveScript(t *testing.T) {
	draft := Draft{
		Kind:     KindRoom,
		VNum:     1001,
		Snapshot: parser.Room{VNum: 1001, Name: "old", ScriptName: "old.lua", Exits: map[string]parser.Exit{}},
		Working:  parser.Room{VNum: 1001, Name: "new", ScriptName: "stale.lua", Exits: map[string]parser.Exit{}},
	}
	var committed parser.Room
	if !CommitRoom(RoomCommitInput{
		Draft: draft,
		LiveScript: func() (parser.Room, bool) {
			return parser.Room{VNum: 1001, ScriptName: "live.lua", ScriptFunctions: 4}, true
		},
		Commit: func(room parser.Room) bool {
			committed = room
			return true
		},
	}) {
		t.Fatal("commit failed")
	}
	if committed.ScriptName != "live.lua" || committed.ScriptFunctions != 4 {
		t.Fatalf("committed script = %q/%d, want live.lua/4", committed.ScriptName, committed.ScriptFunctions)
	}
}
