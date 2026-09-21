package olc

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ZoneDraft is the room-scoped ZEDIT value. RoomVNum is part of the draft
// identity because the same zone can be edited through several stale-carry
// command views, just as it can through telnet.
type ZoneDraft struct {
	Zone     parser.Zone
	RoomVNum int
}

// EntityValue is the typed, detached value held by the non-room draft store.
// Exactly one member is meaningful for a given Kind.
type EntityValue struct {
	Mob    parser.Mob
	Object parser.Obj
	Shop   parser.ShopProto
	Zone   ZoneDraft
}

// EntityDraft is the common lifecycle envelope for mob, object, shop, and
// zone drafts. Its values are copied on every store boundary; callers never
// receive a live store reference.
type EntityDraft struct {
	OwnerIdentity string
	Kind          Kind
	VNum          int
	Snapshot      EntityValue
	Working       EntityValue
	OpenedAt      time.Time
}

func (d EntityDraft) Effective() EntityValue { return cloneEntityValue(d.Kind, d.Working) }

// Diff reports changed field names only. Audit consumers deliberately never
// receive draft contents.
func (d EntityDraft) Diff() []string {
	var fields []string
	switch d.Kind {
	case KindMob:
		if d.Working.Mob.Keywords != d.Snapshot.Mob.Keywords {
			fields = append(fields, "keywords")
		}
		if !reflect.DeepEqual(d.Working.Mob, d.Snapshot.Mob) {
			if d.Working.Mob.ShortDesc != d.Snapshot.Mob.ShortDesc {
				fields = append(fields, "short_description")
			}
			if d.Working.Mob.LongDesc != d.Snapshot.Mob.LongDesc {
				fields = append(fields, "long_description")
			}
			if d.Working.Mob.DetailedDesc != d.Snapshot.Mob.DetailedDesc {
				fields = append(fields, "details")
			}
			if d.Working.Mob.Level != d.Snapshot.Mob.Level {
				fields = append(fields, "level")
			}
			if d.Working.Mob.Exp != d.Snapshot.Mob.Exp {
				fields = append(fields, "exp")
			}
			if len(fields) == 0 {
				fields = append(fields, "mob")
			}
		}
	case KindObject:
		if !reflect.DeepEqual(d.Working.Object, d.Snapshot.Object) {
			fields = append(fields, "object")
		}
	case KindShop:
		if !reflect.DeepEqual(d.Working.Shop, d.Snapshot.Shop) {
			fields = append(fields, "shop")
		}
	case KindZone:
		if d.Working.Zone.Zone.Name != d.Snapshot.Zone.Zone.Name {
			fields = append(fields, "name")
		}
		if d.Working.Zone.Zone.TopRoom != d.Snapshot.Zone.Zone.TopRoom {
			fields = append(fields, "top_room")
		}
		if d.Working.Zone.Zone.Lifespan != d.Snapshot.Zone.Zone.Lifespan {
			fields = append(fields, "lifespan")
		}
		if d.Working.Zone.Zone.ResetMode != d.Snapshot.Zone.Zone.ResetMode {
			fields = append(fields, "reset_mode")
		}
		if !reflect.DeepEqual(d.Working.Zone.Zone.Commands, d.Snapshot.Zone.Zone.Commands) {
			fields = append(fields, "commands")
		}
	}
	sort.Strings(fields)
	return fields
}

// EntityDraftStore owns one recoverable draft per player identity. Lease
// expiry belongs to Registry; this store intentionally retains a draft until
// commit or discard, so a reconnect can recover unfinished work.
type EntityDraftStore struct {
	mu     sync.Mutex
	drafts map[string]EntityDraft
}

func NewEntityDraftStore() *EntityDraftStore {
	return &EntityDraftStore{drafts: make(map[string]EntityDraft)}
}

func (s *EntityDraftStore) Open(owner string, kind Kind, vnum int, snapshot EntityValue) (EntityDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.drafts[owner]; ok {
		if existing.Kind == kind && existing.VNum == vnum {
			return cloneEntityDraft(existing), nil
		}
		return EntityDraft{}, ErrDraftAlreadyOpen
	}
	draft := EntityDraft{
		OwnerIdentity: owner,
		Kind:          kind,
		VNum:          vnum,
		Snapshot:      cloneEntityValue(kind, snapshot),
		Working:       cloneEntityValue(kind, snapshot),
		OpenedAt:      time.Now(),
	}
	s.drafts[owner] = draft
	return cloneEntityDraft(draft), nil
}

func (s *EntityDraftStore) Get(owner string) (EntityDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return EntityDraft{}, false
	}
	return cloneEntityDraft(draft), true
}

func (s *EntityDraftStore) Patch(owner string, operations []Operation) (EntityDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return EntityDraft{}, ErrDraftNotFound
	}
	working := cloneEntityValue(draft.Kind, draft.Working)
	for i := range operations {
		operation := operations[i]
		switch draft.Kind {
		case KindMob:
			operation.Mob = &working.Mob
		case KindObject:
			operation.Obj = &working.Object
		case KindShop:
			operation.Shop = &working.Shop
		case KindZone:
			operation.Zone = &working.Zone.Zone
		default:
			return EntityDraft{}, fmt.Errorf("kind %q is not an entity draft", draft.Kind)
		}
		if err := Apply(operation); err != nil {
			return EntityDraft{}, err
		}
	}
	draft.Working = working
	s.drafts[owner] = draft
	return cloneEntityDraft(draft), nil
}

func (s *EntityDraftStore) Commit(owner string) (EntityDraft, bool) {
	var committed EntityDraft
	s.mu.Lock()
	defer s.mu.Unlock()
	committed, ok := s.drafts[owner]
	if !ok {
		return EntityDraft{}, false
	}
	delete(s.drafts, owner)
	return cloneEntityDraft(committed), true
}

func (s *EntityDraftStore) CommitWith(owner string, commit func(EntityDraft) error) (EntityDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.drafts[owner]
	if !ok {
		return EntityDraft{}, ErrDraftNotFound
	}
	copyDraft := cloneEntityDraft(draft)
	if err := commit(copyDraft); err != nil {
		return EntityDraft{}, err
	}
	delete(s.drafts, owner)
	return copyDraft, nil
}

func (s *EntityDraftStore) Discard(owner string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.drafts[owner]; !ok {
		return false
	}
	delete(s.drafts, owner)
	return true
}

// ZoneCommandsForRoom is the shared setup-side stale-carry filter used by
// telnet ZEDIT and web drafts. It intentionally starts at -1; the save-side
// world commit has its separate C-compatible -2 pass.
func ZoneCommandsForRoom(commands []parser.ZoneCommand, roomVNum int) []parser.ZoneCommand {
	filtered := make([]parser.ZoneCommand, 0)
	cmdRoom := -1
	for _, command := range commands {
		if room, hasRoom := ZoneCommandRoom(command); hasRoom {
			cmdRoom = room
		}
		if cmdRoom == roomVNum {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

// ZoneCommandRoom mirrors zedit.c's switch. E/G/P deliberately have no case:
// their command room is the sticky room carried from the previous room-bearing
// command. This is the C quirk that makes the setup filter faithful.
func ZoneCommandRoom(command parser.ZoneCommand) (int, bool) {
	switch command.Command {
	case "M", "O":
		return command.Arg3, true
	case "D", "R", "L":
		return command.Arg1, true
	default:
		return 0, false
	}
}

func cloneEntityDraft(draft EntityDraft) EntityDraft {
	draft.Snapshot = cloneEntityValue(draft.Kind, draft.Snapshot)
	draft.Working = cloneEntityValue(draft.Kind, draft.Working)
	return draft
}

func cloneEntityValue(kind Kind, value EntityValue) EntityValue {
	switch kind {
	case KindMob:
		value.Mob.ActionFlags = append([]string(nil), value.Mob.ActionFlags...)
		value.Mob.AffectFlags = append([]string(nil), value.Mob.AffectFlags...)
	case KindObject:
		value.Object.Affects = append([]parser.ObjAffect(nil), value.Object.Affects...)
		value.Object.ExtraDescs = append([]parser.ExtraDesc(nil), value.Object.ExtraDescs...)
	case KindShop:
		value.Shop.Products = append([]int(nil), value.Shop.Products...)
		value.Shop.BuyTypes = append([]int(nil), value.Shop.BuyTypes...)
		value.Shop.BuyWords = append([]string(nil), value.Shop.BuyWords...)
		value.Shop.Rooms = append([]int(nil), value.Shop.Rooms...)
	case KindZone:
		value.Zone.Zone.Commands = append([]parser.ZoneCommand(nil), value.Zone.Zone.Commands...)
	}
	return value
}
