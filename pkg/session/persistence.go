package session

import (
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// playerRecordForSave keeps session-owned saved fields beside the game.Player
// conversion. C stores olc_zone in player_special_data_saved, while the Go
// runtime keeps it on Session because OLC admission is descriptor-scoped today.
// The database record is the persistence boundary between those two shapes.
func (s *Session) playerRecordForSave(p *game.Player) (*db.PlayerRecord, error) {
	record, err := db.PlayerToRecord(p, nil)
	if err != nil {
		return nil, err
	}
	record.OlcZone = s.olcZone
	return record, nil
}
