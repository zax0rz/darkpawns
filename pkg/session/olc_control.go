package session

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"

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
	first, _ := game.OneArgument(strings.Join(args, " "))
	if strings.HasPrefix(first, "save") {
		s.olcSaveAll()
	} else {
		s.olcSaveInfo()
	}
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

// olcSaveAll ports src/olc.c:313-344. DP-1399 bounds its failing-head
// retry: preserve that marker and all later entries, with no final producer.
func (s *Session) olcSaveAll() {
	if len(olcSaveList.Ordered()) == 0 {
		s.reditSendDirect("The database is up to date.\r\n")
		return
	}
	for {
		entries := olcSaveList.Ordered()
		if len(entries) == 0 {
			break
		}
		entry := entries[0]
		// C acknowledges before invoking the writer. No save/list/world lock
		// is held during delivery; each writer owns the existing zone lock.
		s.reditSendDirect(fmt.Sprintf("%s saved for zone %d.\r\n", olcSaveLabel(entry.Kind), entry.Zone))
		if err := saveOLCEntry(s.manager.world, entry); err != nil {
			logOLCSaveAllFailure(s.manager.world, entry, err)
			slog.Error("olc disk save failed", "player", s.playerName, "kind", entry.Kind, "zone", entry.Zone, "error", err)
			return
		}
	}
	// src/olc.c:343-344: CMP/builder/file TRUE, with no invisibility term.
	game.MudLog(fmt.Sprintf("OLC: %s saves all", s.player.GetName()), game.MudlogComplete, game.LVL_IMMORT, true)
}

// saveOLCEntry uses the same commit/snapshot/write/marker lock as telnet
// and web editors. Refresh zone bounds under that lock; never route through
// the admin save-all-kinds endpoint or apply its active-claim gate.
func saveOLCEntry(world *game.World, entry olc.DirtyEntry) error {
	lock := zoneSaveLock(entry.Zone)
	lock.Lock()
	defer lock.Unlock()
	if !olcSaveList.Dirty(entry.Kind, entry.Zone) {
		return nil // another saver already removed this head
	}
	zone, ok := world.SnapshotZone(entry.Zone)
	if !ok {
		return fmt.Errorf("zone %d not found", entry.Zone)
	}
	var writer func(*game.World, *parser.Zone) error
	switch entry.Kind {
	case olc.KindRoom:
		writer = saveReditZoneLocked
	case olc.KindObject:
		writer = saveOeditZoneLocked
	case olc.KindZone:
		writer = saveZeditZoneLocked
	case olc.KindMob:
		writer = saveMeditZoneLocked
	case olc.KindShop:
		writer = saveSeditZoneLocked
	default:
		return fmt.Errorf("unknown OLC kind %q", entry.Kind)
	}
	return writer(world, &zone)
}

// Reuse only the already-proven common fopen/atomic-open parent-obstruction
// boundary. The approved retry divergence does not widen error classification.
// No writer, zone, world or save-list lock survives into MudLog delivery.
func logOLCSaveAllFailure(world *game.World, entry olc.DirtyEntry, err error) {
	var extension, payload string
	switch entry.Kind {
	case olc.KindRoom: // src/redit.c:291-294
		extension, payload = "wld", "SYSERR: OLC: Cannot open room file!"
	case olc.KindObject: // src/oedit.c:347-350
		extension, payload = "obj", "SYSERR: OLC: Cannot open objects file!"
	case olc.KindZone: // src/zedit.c:368-372
		extension, payload = "zon", fmt.Sprintf("SYSERR: OLC: zedit_save_to_disk:  Can't write zone %d.", entry.Zone)
	case olc.KindMob: // src/medit.c:349-352
		extension, payload = "mob", "SYSERR: OLC: Cannot open mob file!"
	case olc.KindShop: // src/sedit.c:481-483
		extension, payload = "shp", "SYSERR: OLC: Cannot open shop file!"
	default:
		return
	}
	logOLCOpenParentFailure(world, extension, payload, err)
}
