package olc

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDraftStoreAppliesOrderedOperationsAndReturnsEffectiveCopy(t *testing.T) {
	store := NewDraftStore()
	original := parser.Room{VNum: 1001, Name: "old", Exits: map[string]parser.Exit{}}
	if _, err := store.Open("builder-1", KindRoom, 1001, original); err != nil {
		t.Fatal(err)
	}
	operations := []Operation{
		{Kind: OpSetRoomName, Text: strings.Repeat("x", MaxRoomName+10)},
		{Kind: OpSetRoomSector, Value: 15},
	}
	draft, err := store.Patch("builder-1", operations)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(draft.Working.Name), MaxRoomName-1; got != want {
		t.Fatalf("effective room name length = %d, want %d", got, want)
	}
	if draft.Working.Sector != 15 {
		t.Fatalf("effective sector = %d, want 15", draft.Working.Sector)
	}
	if got := draft.Diff(original); len(got) != 2 || got[0] != "name" || got[1] != "sector" {
		t.Fatalf("draft diff = %#v, want name/sector", got)
	}
	if operations[0].Room != nil {
		t.Fatal("patch retained an input room reference")
	}

	if _, ok := store.Commit("builder-1"); !ok {
		t.Fatal("commit did not consume draft")
	}
	if _, ok := store.Get("builder-1"); ok {
		t.Fatal("draft survived commit")
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
