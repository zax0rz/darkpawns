package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// crash_load.go — DP-1404. C stores a rent code in the crash file's header and
// Crash_load selects one of seven entry mudlogs from it (src/objsave.c:459-663).
// The port's carrier is object_saves.kind (pkg/db/object_save.go): the two
// values it already wrote — 1 for a crash save and 2 for a rent — coincide with
// C's RENT_CRASH and RENT_RENTED, and the load-time header rewrite to
// RENT_CRASH is already db.ObjectSaveLoaded. Only RENT_CRYO was missing, and
// only the PLR_NODELETE quit writes it (src/act.other.c:164-165), so DP-1404 is
// a 1:1 carry on the existing kind, not a new column.

// C's rent codes (src/structs.h:587-591). RENT_UNDEF, RENT_FORCED and
// RENT_TIMEDOUT are never written under free_rent = YES (src/config.c:106), and
// a file this codebase wrote always holds one of the three below; the arms that
// select the others are excluded-valid-play (Zach's D7 rulings, 2026-10-09).
const (
	rentCrash  = 1 // RENT_CRASH
	rentRented = 2 // RENT_RENTED
	rentCryo   = 3 // RENT_CRYO
)

// objectSaveReader reads the crash-file identity. GetObjectSave is implemented
// by the SQLite store and is deliberately not part of the minimal GameStore
// interface (the server's concrete store is a *db.DB; tested by
// cmdShow's rent report, which uses the same assertion).
type objectSaveReader interface {
	GetObjectSave(string) (*db.ObjectSave, error)
}

// crashLoadEntry ports Crash_load's entry-visible half (src/objsave.c:459-663):
// read the stored rent code, emit the entry mudlog it selects, then rewrite the
// header to RENT_CRASH. It runs at C's position in entry — after reset_char and
// the CON gate (src/interpreter.c:2174-2184) and before the entry save
// (src/interpreter.c:2186) — and holds no lock when it calls MudLog. The
// object load itself stays where the port already does it (the store record),
// which is C's order only in that both precede the header rewrite; the producer
// is an independent immortal-facing channel, so no player byte depends on that
// ordering.
//
// A store that does not implement objectSaveReader keeps the previous
// behaviour: rewrite only, no producer.
func (s *Session) crashLoadEntry() error {
	if !s.manager.hasDB || s.player == nil {
		return nil
	}
	name := s.player.GetName()
	if reader, ok := s.manager.db.(objectSaveReader); ok {
		snapshot, err := reader.GetObjectSave(name)
		if err != nil {
			return err
		}
		if payload := crashLoadEntryLine(name, snapshot); payload != "" {
			// All four live arms are NRM / MAX(LVL_IMMORT, invis) / file TRUE
			// (src/objsave.c:489, :519, :524, :528).
			game.MudLog(payload, game.MudlogNormal, max(game.LVL_IMMORT, s.player.GetInvisLevel()), true)
		}
	}
	return db.ObjectSaveLoaded(s.manager.db, name)
}

// crashLoadEntryLine is Crash_load's switch on rent.rentcode
// (src/objsave.c:516-541). A nil snapshot is C's failed fopen: the crash file
// does not exist, which is every new character's first entry
// (src/objsave.c:474-489). The RENT_FORCED/RENT_TIMEDOUT arm (:534) and the
// default arm (:539) emit nothing: under free rent no Go exit path writes those
// codes, and both inventory rows are excluded-valid-play. Likewise the
// corrupt-cost arm (:504) is unreachable here without a corrupt file.
func crashLoadEntryLine(name string, snapshot *db.ObjectSave) string {
	if snapshot == nil {
		return fmt.Sprintf("%s entering game with no equipment.", name)
	}
	switch snapshot.Kind {
	case rentRented:
		return fmt.Sprintf("%s un-renting and entering game.", name)
	case rentCrash:
		return fmt.Sprintf("%s retrieving crash-saved items and entering game.", name)
	case rentCryo:
		return fmt.Sprintf("%s un-cryo'ing and entering game.", name)
	}
	return ""
}
