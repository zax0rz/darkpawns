//lint:file-ignore U1000 Wizard command handlers share generated command-table wiring.
package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/admin"
	"github.com/zax0rz/darkpawns/pkg/game"
)

type shutdownPlan struct {
	broadcast string
	marker    string
}

func shutdownPlanFor(option string) (shutdownPlan, bool) {
	switch strings.ToLower(option) {
	case "":
		return shutdownPlan{broadcast: "Shutting down.\r\n"}, true
	case "reboot":
		return shutdownPlan{
			broadcast: "Rebooting.. come back in a minute or two.\r\n",
			marker:    ".fastboot",
		}, true
	case "die":
		return shutdownPlan{
			broadcast: "Shutting down for maintenance.\r\n",
			marker:    ".killscript",
		}, true
	case "pause":
		return shutdownPlan{
			broadcast: "Shutting down for maintenance.\r\n",
			marker:    "pause",
		}, true
	default:
		return shutdownPlan{}, false
	}
}

func cmdShutdown(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	option := ""
	if len(args) > 0 {
		option = strings.ToLower(args[0])
	}
	plan, ok := shutdownPlanFor(option)
	if !ok {
		s.Send("Unknown shutdown option.\r\n")
		return nil
	}

	s.manager.SendToAll(plan.broadcast)
	s.manager.forceAllSave(s)
	s.manager.RequestShutdown(plan.marker)
	return nil
}

// forceAllSave mirrors do_shutdown's do_force(ch, "all save") continuation.
// The command itself sends the C force notice and invokes the existing Go save
// path for lower-level playing sessions; the process shutdown is then handed
// to main through Manager.RequestShutdown.
func (m *Manager) forceAllSave(caster *Session) {
	if caster == nil || caster.player == nil {
		return
	}
	caster.Send("Okay.\r\n")

	m.mu.RLock()
	targets := make([]*Session, 0, len(m.sessions))
	for _, target := range m.sessions {
		if target == nil || target.player == nil || !target.authenticated || target == caster {
			continue
		}
		if target.player.Level >= caster.player.Level {
			continue
		}
		targets = append(targets, target)
	}
	m.mu.RUnlock()

	for _, target := range targets {
		target.Send(fmt.Sprintf("%s has forced you to 'all save'.\r\n", caster.player.Name))
		m.world.ExecSave(target.player)
	}
}

// ---------------------------------------------------------------------------
// snoop — spy on player input (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdSnoop(s *Session, args []string) error {
	// The LVL_GOD and POS_DEAD gates live in the authoritative command table;
	// C's do_snoop itself starts at the descriptor check and emits no second
	// authorization branch.
	if s == nil || s.manager == nil || s.player == nil || s.manager.world == nil {
		return nil
	}

	if len(args) == 0 {
		if stopSnooping(s) {
			s.Send("You stop snooping.\r\n")
		} else {
			s.Send("You aren't snooping anyone.\r\n")
		}
		return nil
	}

	target, ok := s.manager.world.ResolveCharWorld(s.player, args[0])
	if !ok {
		s.Send("No such person around.\r\n")
		return nil
	}
	if target.Player == nil {
		s.Send("There's no link.. nothing to snoop.\r\n")
		return nil
	}
	targetSession := findSessionForPlayer(s.manager, target.Player)
	if targetSession == nil {
		s.Send("There's no link.. nothing to snoop.\r\n")
		return nil
	}

	s.manager.snoopMu.Lock()
	switch {
	case targetSession == s:
		if s.snooping != nil {
			s.snooping.snoopBy = nil
			s.snooping = nil
		}
		s.manager.snoopMu.Unlock()
		s.Send("You stop snooping.\r\n")
		return nil
	case targetSession.snoopBy != nil:
		s.manager.snoopMu.Unlock()
		s.Send("Busy already. \r\n")
		return nil
	case targetSession.snooping == s:
		s.manager.snoopMu.Unlock()
		s.Send("Don't be stupid.\r\n")
		return nil
	case getEffectiveLevel(targetSession) >= getEffectiveLevel(s):
		s.manager.snoopMu.Unlock()
		s.Send("You can't.\r\n")
		return nil
	default:
		if s.snooping != nil {
			s.snooping.snoopBy = nil
		}
		s.snooping = targetSession
		targetSession.snoopBy = s
	}
	s.manager.snoopMu.Unlock()
	s.Send("Okay.\r\n")
	return nil
}

func stopSnooping(s *Session) bool {
	s.manager.snoopMu.Lock()
	defer s.manager.snoopMu.Unlock()
	if s.snooping == nil {
		return false
	}
	s.snooping.snoopBy = nil
	s.snooping = nil
	return true
}

// ---------------------------------------------------------------------------
// reload — refresh cached text/help data (LVL_IMPL-1)
// ---------------------------------------------------------------------------
func cmdReload(s *Session, args []string) error {
	option := ""
	if len(args) > 0 {
		option = strings.ToLower(args[0])
	}
	if option == "" {
		s.Send("Unknown reload option.\r\n")
		return nil
	}
	if option == "all" || strings.HasPrefix(option, "*") {
		reloadCachedText(s, "wizlist", "immlist", "news", "credits", "motd", "imotd", "info", "policies", "handbook", "future")
		reloadHelpScreen(s)
	} else {
		switch option {
		case "wizlist", "immlist", "news", "credits", "motd", "imotd", "info", "handbook", "future":
			reloadCachedText(s, option)
		case "policy":
			reloadCachedText(s, "policies")
		case "help":
			reloadHelpScreen(s)
		case "background":
			// Background text is read on demand by the Go port.
		case "xhelp":
			reloadHelpTable(s)
		default:
			s.Send("Unknown reload option.\r\n")
			return nil
		}
	}
	s.Send("Okay.\r\n")
	return nil
}

// cmdStat — inspect a character, room, or object (LVL_IMMORT)
func cmdWizlock(s *Session, args []string) error {
	if s.manager == nil {
		s.Send("Cannot access manager state.")
		return nil
	}

	s.manager.wizlockMutex.Lock()
	defer s.manager.wizlockMutex.Unlock()

	when := "currently"
	if len(args) > 0 {
		val := cAtoi(args[0])
		if val < 0 || val > s.player.Level {
			s.Send("Invalid wizlock value.\r\n")
			return nil
		}
		s.manager.wizlockLevel = val
		s.manager.wizlocked = val != 0
		when = "now"
	}

	level := s.manager.wizlockLevel
	switch level {
	case 0:
		s.Send(fmt.Sprintf("The game is %s completely open.\r\n", when))
	case 1:
		s.Send(fmt.Sprintf("The game is %s closed to new players.\r\n", when))
	default:
		s.Send(fmt.Sprintf("Only level %d and above may enter the game %s.\r\n", level, when))
	}
	return nil
}

// cmdDc — disconnect a player (LVL_IMMORT+1)
func cmdDc(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT+1) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: DC <connection number> (type USERS for a list)\r\n")
		return nil
	}
	numToDC, ok := parseDCNumber(args[0])
	if !ok || numToDC == 0 {
		s.Send("Usage: DC <connection number> (type USERS for a list)\r\n")
		return nil
	}

	var target *Session
	s.manager.mu.RLock()
	for _, candidate := range s.manager.sessions {
		if candidate.connectionNumber == numToDC {
			target = candidate
			break
		}
	}
	s.manager.mu.RUnlock()
	if target == nil {
		s.Send("No such connection.\r\n")
		return nil
	}
	if target.player != nil && target.player.GetLevel() >= s.player.GetLevel() {
		s.Send("Umm.. maybe that's not such a good idea...\r\n")
		return nil
	}
	target.Close()
	s.Send(fmt.Sprintf("Connection #%d closed.\r\n", numToDC))
	return nil
}

// parseDCNumber mirrors the decimal-prefix and optional-sign behavior of C's
// atoi used by do_dc. A zero result is handled by cmdDc as the usage branch.
func parseDCNumber(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	index := 0
	sign := 1
	if value[0] == '+' || value[0] == '-' {
		if value[0] == '-' {
			sign = -1
		}
		index++
	}
	start := index
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	if index == start {
		return 0, false
	}
	parsed, err := strconv.Atoi(value[start:index])
	if err != nil {
		return 0, false
	}
	return sign * parsed, true
}

// cmdHome — teleport to home room (LVL_IMMORT)
func cmdDate(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	now := time.Now()
	// C dispatches date with SCMD_DATE, so do_date ignores its argument;
	// uptime is a separate command-table entry using SCMD_UPTIME.
	s.Send(fmt.Sprintf("Current machine time: %s", now.Format("Mon Jan 2 15:04:05 2006")))
	return nil
}

// cmdUptime — standalone "uptime" command.
// Source: src/interpreter.c do_date/SCMD_UPTIME, gated at LVL_IMMORT.
// C (act.wizard.c) prints "Up since <boot_time>: N day(s), H:MM".
func cmdUptime(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	sendUptime(s, time.Now())
	return nil
}

// sendUptime prints the server uptime in C's day(s)/H:MM format relative to bootTime.
func sendUptime(s *Session, now time.Time) {
	// admin.processStartTime is the server boot timestamp (set at package init).
	boot := admin.ProcessStartTime()
	elapsed := now.Sub(boot)
	days := int(elapsed.Hours()) / 24
	hours := int(elapsed.Hours()) % 24
	minutes := int(elapsed.Minutes()) % 60
	// C (act.wizard.c:1825) uses real singular/plural: "1 day" vs "3 days".
	dayWord := "days"
	if days == 1 {
		dayWord = "day"
	}
	s.Send(fmt.Sprintf("Up since %s: %d %s, %d:%02d", boot.Format("Mon Jan 2 15:04:05 2006"), days, dayWord, hours, minutes))
}

// cmdLast — show last login info for a player (LVL_GOD-1).
// Source: act.wizard.c:1834-1854; the command-table gate is
// LVL_GOD-1/POS_DEAD (interpreter.c:535).
func cmdLast(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD-1) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("For whom do you wish to search?\r\n")
		return nil
	}
	target, _ := game.OneArgument(strings.Join(args, " "))
	if s.manager == nil || !s.manager.hasDB {
		s.Send("There is no such player.\r\n")
		return nil
	}
	rec, err := s.manager.db.GetPlayer(target)
	if err != nil || rec == nil {
		s.Send("There is no such player.\r\n")
		return nil
	}
	s.Send(fmt.Sprintf("[%d] [%2d] %-12s : Level %d\r\n", rec.ID, rec.Level, rec.Name, rec.Level))
	return nil
}

// wizutilSubcmd represents a wizutil sub-command.
type wizutilSubcmd int

const (
	wizutilReroll wizutilSubcmd = iota
	wizutilPardon
	wizutilNotitle
	wizutilSquelch
	wizutilFreeze
	wizutilThaw
	wizutilUnaffect
)

var wizutilNames = map[wizutilSubcmd]string{
	wizutilReroll:   "reroll",
	wizutilPardon:   "pardon",
	wizutilNotitle:  "notitle",
	wizutilSquelch:  "squelch",
	wizutilFreeze:   "freeze",
	wizutilThaw:     "thaw",
	wizutilUnaffect: "unaffect",
}

// cmdWizutil — player utility commands (LVL_IMMORT)
func cmdWizutil(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: reroll|pardon|notitle|squelch|freeze|thaw|unaffect <player>")
		return nil
	}
	subName := strings.ToLower(args[0])
	targetName := args[1]

	var subcmd wizutilSubcmd
	found := false
	for k, v := range wizutilNames {
		if strings.HasPrefix(v, subName) {
			subcmd = k
			found = true
			break
		}
	}
	if !found {
		s.Send("Unknown sub-command. Options: reroll, pardon, notitle, squelch, freeze, thaw, unaffect")
		return nil
	}

	return wizutilDispatch(s, subcmd, targetName)
}

// wizutilDispatch performs one wizutil sub-action against a named target.
// Faithful port of do_wizutil (act.wizard.c:2077). The mortal-affecting cases
// (pardon/notitle/squelch/freeze/thaw) now mirror C byte-for-byte: the right
// pre-state guards, the actual PLR flag toggles, the victim (TO_VICT) and room
// (TO_ROOM) messages, and the "(GC) ... by <god>." acknowledgements. The
// mudlog() calls are not player-facing and are intentionally not reproduced.
func wizutilDispatch(s *Session, subcmd wizutilSubcmd, targetName string) error {
	targetName, _ = game.OneArgument(targetName)
	resolved, ok := s.manager.world.ResolveCharWorld(s.player, targetName)
	if ok && resolved.Mob != nil {
		// do_wizutil resolves visible NPCs before the sub-command switch and
		// rejects them with this shared branch (act.wizard.c:2101-2103).
		s.Send("You can't do that to a mob!\r\n")
		return nil
	}

	// Keep the session fallback for test fixtures and link-state cases where a
	// player session exists without a corresponding World registration. Live
	// commands normally take the C-faithful world-resolved path above.
	targetNameForSession := targetName
	if ok && resolved.Player != nil {
		targetNameForSession = resolved.Player.Name
	}
	target := findSessionByName(s.manager, targetNameForSession)
	if target == nil || target.player == nil {
		s.Send("There is no such player.\r\n")
		return nil
	}
	if target.player.Level > s.player.Level && target.player.Level >= LVL_IMMORT {
		s.Send("Hmmm...you'd better not.\r\n")
		return nil
	}

	switch subcmd {
	case wizutilReroll:
		// C roll_real_abils() replaces both real_abils and aff_abils with a
		// fresh class/race-dependent roll before copying the new constitution
		// into GET_ORIG_CON (class.c:380-497; act.wizard.c:2101-2111).
		stats := game.RollRealAbils(target.player.Class, target.player.Race)
		target.player.Lock()
		target.player.Stats = stats
		target.player.Strength = stats.Str
		target.player.OrigCon = stats.Con
		target.player.Unlock()
		s.Send("Rerolled...\r\n")
		s.Send(fmt.Sprintf("New stats: Str %d/%d, Int %d, Wis %d, Dex %d, Con %d, Cha %d\r\n",
			stats.Str, stats.StrAdd, stats.Int, stats.Wis, stats.Dex, stats.Con, stats.Cha))
	case wizutilPardon:
		// act.wizard.c:2113 — a non-outlaw victim is rejected.
		if target.player.GetFlags()&(1<<game.PlrOutlaw) == 0 {
			s.Send("Your victim is not flagged.\r\n")
			return nil
		}
		target.player.SetPlrFlag(game.PlrOutlaw, false)
		s.Send("Pardoned.\r\n")
		target.Send("You have been pardoned by the Gods!\r\n")
	case wizutilNotitle:
		// PLR_TOG_CHK toggles and returns the NEW state; ch sees the (GC) ack.
		newState := target.player.GetFlags()&(1<<game.PlrNotitle) == 0
		target.player.SetPlrFlag(game.PlrNotitle, newState)
		s.Send(fmt.Sprintf("(GC) Notitle %s for %s by %s.\r\n", onOff(newState), target.player.Name, s.player.Name))
	case wizutilSquelch:
		// SCMD_SQUELCH toggles PLR_NOSHOUT; the command name is "mute".
		newState := target.player.GetFlags()&(1<<game.PlrNoshout) == 0
		target.player.SetPlrFlag(game.PlrNoshout, newState)
		s.Send(fmt.Sprintf("(GC) Squelch %s for %s by %s.\r\n", onOff(newState), target.player.Name, s.player.Name))
	case wizutilFreeze:
		if target == s {
			s.Send("Oh, yeah, THAT'S real smart...\r\n")
			return nil
		}
		if target.player.GetFlags()&(1<<game.PlrFrozen) != 0 {
			s.Send("Your victim is already pretty cold.\r\n")
			return nil
		}
		target.player.SetPlrFlag(game.PlrFrozen, true)
		target.player.FreezeLevel = s.player.Level // GET_FREEZE_LEV(vict) = GET_LEVEL(ch)
		target.Send("A bitter wind suddenly rises and drains every ounce of heat from your body!\r\nYou feel frozen!\r\n")
		s.Send("Frozen.\r\n")
		// act("A sudden cold wind conjured from nowhere freezes $n!", vict, TO_ROOM)
		broadcastToRoomExcept(s, target.player.GetRoom(), target.player.Name,
			fmt.Sprintf("A sudden cold wind conjured from nowhere freezes %s!\r\n", target.player.Name))
	case wizutilThaw:
		if target.player.GetFlags()&(1<<game.PlrFrozen) == 0 {
			s.Send("Sorry, your victim is not morbidly encased in ice at the moment.\r\n")
			return nil
		}
		if target.player.FreezeLevel > s.player.Level {
			s.Send(fmt.Sprintf("Sorry, a level %d God froze %s... you can't unfreeze %s.\r\n",
				target.player.FreezeLevel, target.player.Name, hmhr(target.player)))
			return nil
		}
		target.player.SetPlrFlag(game.PlrFrozen, false)
		target.Send("A fireball suddenly explodes in front of you, melting the ice!\r\nYou feel thawed.\r\n")
		s.Send("Thawed.\r\n")
		// act("A sudden fireball conjured from nowhere thaws $n!", vict, TO_ROOM)
		broadcastToRoomExcept(s, target.player.GetRoom(), target.player.Name,
			fmt.Sprintf("A sudden fireball conjured from nowhere thaws %s!\r\n", target.player.Name))
	case wizutilUnaffect:
		target.player.Lock()
		if target.player.ActiveAffects != nil {
			target.player.ActiveAffects = nil
			target.player.Unlock()
			target.Send("There is a brief flash of light! You feel slightly different.")
			s.Send("All spells removed.")
		} else {
			target.player.Unlock()
			s.Send("Your victim does not have any affections!")
		}
	}
	return nil
}

// broadcastToRoomExcept sends msg to every player in roomVNum except the named
// player — the TO_ROOM recipient set for act(..., TO_ROOM) where the named
// player is the act's ch (excluded). Used by freeze/thaw room messages.
func broadcastToRoomExcept(s *Session, roomVNum int, excludeName, msg string) {
	if s.manager != nil {
		s.manager.BroadcastToRoom(roomVNum, []byte(msg), excludeName)
	}
}

// hmhr mirrors C's HMHR macro (utils.h:507): him/her/it by sex. Used only in
// thaw's freeze-level guard (act.wizard.c:2162), which a single-God harness
// never reaches; included for faithfulness.
func hmhr(p *game.Player) string {
	switch p.Sex {
	case game.SexFemale:
		return "her"
	case game.SexMale:
		return "him"
	default:
		return "it"
	}
}

// wizutilAuthed mirrors do_wizutil's inner authority guard (act.wizard.c:2083): a caller may
// run a wizutil sub-action if they are level >= LVL_IMMORT OR carry the PLR_CHOSEN flag. The
// low-table-gate commands (pardon, mute) rely on this inner guard; the higher-gated wrappers
// (freeze/thaw/notitle at LVL_GRGOD/LVL_GOD) satisfy it via level alone but call this for parity.
func wizutilAuthed(s *Session) bool {
	if checkLevel(s, LVL_IMMORT) {
		return true
	}
	return s.player != nil && s.player.GetFlags()&(1<<uint(game.PlrChosen)) != 0
}

// cmdReroll — standalone "reroll <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_REROLL, gated at LVL_GRGOD.
func cmdReroll(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes, but for whom?!?\r\n")
		return nil
	}
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	return wizutilDispatch(s, wizutilReroll, targetName)
}

// cmdUnaffect — standalone "unaffect <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_UNAFFECT, gated at LVL_GOD.
func cmdUnaffect(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: unaffect <player>")
		return nil
	}
	return wizutilDispatch(s, wizutilUnaffect, args[0])
}

// cmdFreeze — standalone "freeze <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_FREEZE, gated at LVL_FREEZE (=LVL_GRGOD=38).
func cmdFreeze(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes, but for whom?!?\r\n")
		return nil
	}
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	return wizutilDispatch(s, wizutilFreeze, targetName)
}

// cmdThaw — standalone "thaw <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_THAW, gated at LVL_FREEZE (=LVL_GRGOD=38).
func cmdThaw(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes, but for whom?!?\r\n")
		return nil
	}
	return wizutilDispatch(s, wizutilThaw, args[0])
}

// cmdPardon — standalone "pardon <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_PARDON, gated at level 1 (do_wizutil's inner
// LVL_IMMORT||PLR_CHOSEN guard applies; the table level is the C default of 1).
func cmdPardon(s *Session, args []string) error {
	if !wizutilAuthed(s) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes, but for whom?!?\r\n")
		return nil
	}
	// C do_wizutil receives the complete argument remainder and applies
	// one_argument, which skips leading fill words before resolving the target
	// (interpreter.c:1265-1283; act.wizard.c:2088). Preserve that boundary
	// instead of selecting args[0] in the command wrapper.
	return wizutilDispatch(s, wizutilPardon, strings.Join(args, " "))
}

// cmdNotitle — standalone "notitle <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_NOTITLE, gated at LVL_GOD.
func cmdNotitle(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes, but for whom?!?\r\n")
		return nil
	}
	// C do_wizutil receives the whole argument string and applies
	// one_argument, which skips leading fill words and ignores the remainder
	// (interpreter.c:1265; act.wizard.c:2082). Rejoin the tokenized command so
	// wizutilDispatch can preserve that boundary.
	return wizutilDispatch(s, wizutilNotitle, strings.Join(args, " "))
}

// cmdMute — standalone "mute <player>" command.
// Source: src/interpreter.c do_wizutil/SCMD_SQUELCH, table level 1 (do_wizutil's inner
// LVL_IMMORT||PLR_CHOSEN guard applies). C toggles PLR_NOSHOUT; the dispatch case
// (wizutilSquelch) carries the behavior. NOTE: pkg/command/admin_commands.go also defines a
// "mute" (duration-based moderation), but AdminCommands is never instantiated/wired at
// startup (NewAdminCommands has no caller), so it never registers — this C-faithful command
// is the one that actually runs. See DP-1225.
func cmdMute(s *Session, args []string) error {
	if !wizutilAuthed(s) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Yes, but for whom?!?\r\n")
		return nil
	}
	return wizutilDispatch(s, wizutilSquelch, args[0])
}

// cmdShow — show system info (LVL_IMMORT)
func cmdTick(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}

	slog.Warn("wizard tick forced", "by", s.playerName)
	game.WeatherAndTime(true, s.manager.SendToOutdoor)
	s.manager.world.AffectUpdate()
	s.manager.world.PointUpdate()
	// TODO(port): C calls hunt_items() after point_update(); no Go equivalent
	// exists yet. Add it here in the same order when that subsystem is ported.
	return nil
}

// cmdBroadcast — broadcast a message to all playing characters (LVL_GOD)
// Ported from act.wizard.c:do_broadcast(). Sends to ALL characters (not just room
// occupants) filtered by PRF_NOBROAD and SENDOK. Uses perform_act for substitution.
// Format: broadcast <message>
func cmdBroadcast(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Broadcast what?")
		return nil
	}
	msg := "[Broadcast] " + strings.Join(args, " ")

	if len(msg) > 500 {
		s.Send("Maximum broadcast length is 500 characters.")
		return nil
	}

	// Send to all playing sessions (equivalent to checking !d->connected in C)
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil || !sess.authenticated {
			continue
		}
		// Check PRF_NOBROAD equivalent: if the session has a "nobroad" preference, skip
		sess.player.RLock()
		nb := sess.player.NoBroadcast
		sess.player.RUnlock()
		if nb {
			continue
		}
		sess.Send(msg)
	}
	s.manager.mu.RUnlock()

	slog.Warn("wizard broadcast", "by", s.playerName, "message", msg)
	return nil
}

// cmdNewbie — give newbie equipment to a player (LVL_IMMORT)
// Original: act.wizard.c do_newbie() — gives starter items: tunic, bread, skin, club
func cmdNewbie(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Whom do you wish to newbie?")
		return nil
	}
	targetName := args[0]
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	slog.Warn("wizard newbie", "by", s.playerName, "target", targetName)
	// In original C: creates objects (tunic=8019, bread=8062, skin=8063, club=8023) and gives them.
	// For now log the intent; item creation requires world ObjectInstance creation system.
	s.Send("Newbied.")
	return nil
}
