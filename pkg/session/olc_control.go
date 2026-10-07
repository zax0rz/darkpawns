package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

// cmdOlc is do_olc's SCMD_OLC_SAVEINFO route (src/interpreter.c:590;
// src/olc.c:85-99). Editor zone authorization and claim gates do not apply.
func cmdOlc(s *Session, args []string) error {
	if s.isSwitched && s.switchedMob != nil {
		return nil
	}
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return fmt.Errorf("not logged in")
	}
	s.olcSaveInfo()
	return nil
}

// These are C save_msg, not editor names (src/olc.c:311).
func olcSaveLabel(kind olc.Kind) string {
	switch kind {
	case olc.KindRoom:
		return "Rooms"
	case olc.KindObject:
		return "Objects"
	case olc.KindZone:
		return "Zone info"
	case olc.KindMob:
		return "Mobiles"
	case olc.KindShop:
		return "Shops"
	default:
		return ""
	}
}

func (s *Session) olcSaveInfo() {
	entries := olcSaveList.Ordered()
	if len(entries) == 0 {
		s.reditSendDirect("The database is up to date.\r\n")
		return
	}
	s.reditSendDirect("The following OLC components need saving:-\r\n")
	for _, entry := range entries {
		s.reditSendDirect(fmt.Sprintf(" - %s for zone %d.\r\n", olcSaveLabel(entry.Kind), entry.Zone))
	}
}
