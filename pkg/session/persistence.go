package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// playerRecordForSave keeps session-owned saved fields beside the game.Player
// conversion. C stores olc_zone in player_special_data_saved, while the Go
// runtime keeps it on Session because OLC admission is descriptor-scoped today.
// The database record is the persistence boundary between those two shapes.
//
// loadRoom is C's save_char parameter (db.c:2366): the room the record carries
// for a character without PLR_LOADROOM. Extraction saves pass the in-memory
// load room (handler.c:1162), entry saves pass NOWHERE (interpreter.c:2186).
// A PLR_LOADROOM-flagged character always carries its chosen room instead —
// save_char skips the override (db.c:2387).
func (s *Session) playerRecordForSave(p *game.Player, loadRoom int) (*db.PlayerRecord, error) {
	p.AccountCharacterSave()
	record, err := db.PlayerToRecord(p, nil)
	if err != nil {
		return nil, err
	}
	if s.manager != nil {
		s.manager.mu.RLock()
		record.OlcZone = s.olcZone
		s.manager.mu.RUnlock()
	} else {
		record.OlcZone = s.olcZone
	}
	if p.GetFlags()&(1<<uint(game.PlrLoadroom)) == 0 {
		record.RoomVNum = loadRoom
	}
	return record, nil
}

// creationRecord materializes C's saved phase without undoing constructor
// defaults on the live character. The Go constructor pre-applies do_start's
// final preferences, practices and orig_con; C applies them only after its
// advance_level save (class.c:574-591). This changes the durable snapshot,
// never the initial live state or the RNG stream.
func (s *Session) creationRecord(live *game.Player, advanced bool, entering bool) (*db.PlayerRecord, error) {
	r, err := s.playerRecordForSave(live, game.LoadRoomNowhere)
	if err != nil {
		return nil, err
	}
	p, err := db.RecordToPlayer(r, s.manager.world)
	if err != nil {
		return nil, err
	}
	p.OrigCon = 0
	p.Practices = 0
	if advanced {
		p.Practices = live.Practices - 2
		// C do_start gives its kit before advance_level (class.c:499-533),
		// setting PLR_CRASH through obj_to_char (handler.c:569-571).
		// Go gives the same live kit after entry; preserve C's saved flag
		// without moving those object operations or changing their draws.
		p.MarkCrashNeeded()
	}
	for _, bit := range []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove, game.PrfAutoexit} {
		p.SetPlrFlag(bit, false)
	}
	p.AutoExit = false
	p.WimpLevel = 0
	if live.Level < game.LVL_IMMORT {
		p.Health, p.MaxHealth = 0, 0
		if entering {
			p.Health = 1 // reset_char (db.c:2957-2958).
		}
		if advanced {
			p.MaxHealth = live.MaxHealth
		}
		p.Hunger, p.Thirst, p.Drunk = 24, 24, 24
		p.Conditions[game.CondFull], p.Conditions[game.CondThirst], p.Conditions[game.CondDrunk] = 24, 24, 24
	}
	// This is a scalar phase projection of the same save, not a second
	// char_to_store boundary: retain the live accounting timestamp.
	olcZone := r.OlcZone
	r, err = db.PlayerToRecord(p, nil)
	if err == nil {
		r.OlcZone = olcZone
	}
	return r, err
}

func (s *Session) saveCreationRecord(advanced bool) error {
	if !s.manager.hasDB {
		return nil
	}
	r, err := s.creationRecord(s.player, advanced, true)
	if err != nil {
		return err
	}
	return s.manager.db.SavePlayer(r)
}

func (s *Session) storedPlayer(name string) (*game.Player, *db.PlayerRecord, error) {
	if !s.manager.hasDB {
		return nil, nil, fmt.Errorf("player store unavailable")
	}
	r, err := s.manager.db.GetPlayer(name)
	if err != nil || r == nil {
		return nil, r, err
	}
	p, err := db.RecordToPlayer(r, s.manager.world)
	return p, r, err
}
