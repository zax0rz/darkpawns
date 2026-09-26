package session

import (
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
	record, err := db.PlayerToRecord(p, nil)
	if err != nil {
		return nil, err
	}
	record.OlcZone = s.olcZone
	if p.GetFlags()&(1<<uint(game.PlrLoadroom)) == 0 {
		record.RoomVNum = loadRoom
	}
	return record, nil
}
