package olc

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

var (
	ErrDraftAlreadyOpen = errors.New("an OLC draft is already open for this owner")
	ErrDraftNotFound    = errors.New("OLC draft not found")
)

// Draft is an owner-keyed, server-side working copy. Snapshot is the room at
// open time; Working is the only value exposed to Apply and to callers.
type Draft struct {
	OwnerIdentity string
	Kind          Kind
	VNum          int
	Snapshot      parser.Room
	Working       parser.Room
	OpenedAt      time.Time
}

// Effective returns a detached copy of the current working room.
func (d Draft) Effective() parser.Room {
	return cloneRoom(d.Working)
}

// Diff returns deterministic field names whose values differ from snapshot.
// It deliberately reports fields, never their contents, so the same result
// can be used in audit details and a future review panel.
func (d Draft) Diff(snapshot parser.Room) []string {
	working := d.Working
	fields := make([]string, 0)
	if working.Name != snapshot.Name {
		fields = append(fields, "name")
	}
	if working.Description != snapshot.Description {
		fields = append(fields, "description")
	}
	if !reflect.DeepEqual(working.Flags, snapshot.Flags) {
		fields = append(fields, "flags")
	}
	if working.Sector != snapshot.Sector {
		fields = append(fields, "sector")
	}

	directions := make(map[string]struct{}, len(working.Exits)+len(snapshot.Exits))
	for direction := range working.Exits {
		directions[direction] = struct{}{}
	}
	for direction := range snapshot.Exits {
		directions[direction] = struct{}{}
	}
	sortedDirections := make([]string, 0, len(directions))
	for direction := range directions {
		sortedDirections = append(sortedDirections, direction)
	}
	sort.Strings(sortedDirections)
	for _, direction := range sortedDirections {
		current, currentOK := working.Exits[direction]
		original, originalOK := snapshot.Exits[direction]
		if currentOK != originalOK {
			fields = append(fields, "exit."+direction)
			continue
		}
		if current.ToRoom != original.ToRoom {
			fields = append(fields, "exit."+direction+".target")
		}
		if current.Description != original.Description {
			fields = append(fields, "exit."+direction+".description")
		}
		if current.Keywords != original.Keywords {
			fields = append(fields, "exit."+direction+".keywords")
		}
		if current.Key != original.Key {
			fields = append(fields, "exit."+direction+".key")
		}
		if current.DoorState != original.DoorState || current.ExitInfo != original.ExitInfo {
			fields = append(fields, "exit."+direction+".door_flags")
		}
	}

	maxExtras := len(working.ExtraDescs)
	if len(snapshot.ExtraDescs) > maxExtras {
		maxExtras = len(snapshot.ExtraDescs)
	}
	for i := 0; i < maxExtras; i++ {
		if i >= len(working.ExtraDescs) || i >= len(snapshot.ExtraDescs) {
			fields = append(fields, fmt.Sprintf("extra.%d", i))
			continue
		}
		current := working.ExtraDescs[i]
		original := snapshot.ExtraDescs[i]
		if current.Keywords != original.Keywords {
			fields = append(fields, fmt.Sprintf("extra.%d.keywords", i))
		}
		if current.Description != original.Description {
			fields = append(fields, fmt.Sprintf("extra.%d.description", i))
		}
	}
	return fields
}

// DraftStore keeps one active draft per stable player identity. It is
// intentionally process-local: a JWT can expire while the draft remains, but
// a process restart is not a persistence boundary for P4.
type DraftStore struct {
	mu       sync.Mutex
	drafts   map[string]Draft
	entities *EntityDraftStore
}

func NewDraftStore() *DraftStore {
	return &DraftStore{drafts: make(map[string]Draft), entities: NewEntityDraftStore()}
}

// The typed entity methods keep one DraftStore lifecycle owner for all five
// editor kinds while preserving the room-shaped P4 API used by existing
// callers. The entity substore has its own mutex because a player may recover
// a non-room draft independently of a room draft created by the older API.
func (s *DraftStore) entityStore() *EntityDraftStore {
	if s.entities == nil {
		s.entities = NewEntityDraftStore()
	}
	return s.entities
}

func (s *DraftStore) OpenEntity(owner string, kind Kind, vnum int, snapshot EntityValue) (EntityDraft, error) {
	return s.entityStore().Open(owner, kind, vnum, snapshot)
}

func (s *DraftStore) GetEntity(owner string) (EntityDraft, bool) {
	return s.entityStore().Get(owner)
}

func (s *DraftStore) PatchEntity(owner string, operations []Operation) (EntityDraft, error) {
	return s.entityStore().Patch(owner, operations)
}

func (s *DraftStore) CommitEntityWith(owner string, commit func(EntityDraft) error) (EntityDraft, error) {
	return s.entityStore().CommitWith(owner, commit)
}

func (s *DraftStore) DiscardEntity(owner string) bool {
	return s.entityStore().Discard(owner)
}

// Open creates a draft or returns the owner's existing draft for the same
// resource. Re-open is what lets a re-authenticated owner recover work after a
// lease lapse without replacing the effective working copy.
func (s *DraftStore) Open(owner string, kind Kind, vnum int, snapshot parser.Room) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner == "" {
		return Draft{}, errors.New("draft owner identity is required")
	}
	if existing, ok := s.drafts[owner]; ok {
		if existing.Kind != kind || existing.VNum != vnum {
			return Draft{}, ErrDraftAlreadyOpen
		}
		return cloneDraft(existing), nil
	}
	now := time.Now()
	draft := Draft{
		OwnerIdentity: owner,
		Kind:          kind,
		VNum:          vnum,
		Snapshot:      cloneRoom(snapshot),
		Working:       cloneRoom(snapshot),
		OpenedAt:      now,
	}
	s.drafts[owner] = draft
	return cloneDraft(draft), nil
}

// Get returns a detached draft snapshot keyed only by stable player identity.
func (s *DraftStore) Get(owner string) (Draft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return Draft{}, false
	}
	return cloneDraft(draft), true
}

// Patch applies ordered operations to the stored working copy and returns the
// effective result. The input operations are never echoed or retained.
func (s *DraftStore) Patch(owner string, operations []Operation) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return Draft{}, ErrDraftNotFound
	}
	working := cloneRoom(draft.Working)
	for i := range operations {
		operation := operations[i]
		operation.Room = &working
		if err := Apply(operation); err != nil {
			return cloneDraft(draft), err
		}
	}
	draft.Working = working
	s.drafts[owner] = draft
	return cloneDraft(draft), nil
}

// Commit removes and returns the detached effective draft. World mutation is
// deliberately separate: the caller holds the zone save lock and invokes the
// shared commit primitive before consuming this value.
func (s *DraftStore) Commit(owner string) (Draft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return Draft{}, false
	}
	delete(s.drafts, owner)
	return cloneDraft(draft), true
}

// CommitWith performs an atomic store-side consume around a caller-supplied
// world commit. An error leaves the draft available for correction or retry.
func (s *DraftStore) CommitWith(owner string, commit func(Draft) error) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return Draft{}, ErrDraftNotFound
	}
	if err := commit(cloneDraft(draft)); err != nil {
		return cloneDraft(draft), err
	}
	delete(s.drafts, owner)
	return cloneDraft(draft), nil
}

// Discard drops the draft without touching the world.
func (s *DraftStore) Discard(owner string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.drafts[owner]; !ok {
		return false
	}
	delete(s.drafts, owner)
	return true
}

func cloneDraft(draft Draft) Draft {
	draft.Snapshot = cloneRoom(draft.Snapshot)
	draft.Working = cloneRoom(draft.Working)
	return draft
}

func cloneRoom(room parser.Room) parser.Room {
	clone := room
	clone.Flags = append([]string(nil), room.Flags...)
	clone.ExtraDescs = append([]parser.ExtraDesc(nil), room.ExtraDescs...)
	clone.Exits = make(map[string]parser.Exit, len(room.Exits))
	for direction, exit := range room.Exits {
		clone.Exits[direction] = exit
	}
	return clone
}
