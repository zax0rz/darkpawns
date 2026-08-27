package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func cmdHeal(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Heal whom?")
		return nil
	}
	targetName := args[0]
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	targetSess.player.Health = targetSess.player.MaxHealth
	targetSess.player.Mana = targetSess.player.MaxMana
	targetSess.player.Move = targetSess.player.MaxMove
	slog.Warn("wizard heal", "by", s.player.Name, "target", targetSess.player.Name)
	s.Send(fmt.Sprintf("You heal %s.", targetSess.player.Name))
	targetSess.Send(fmt.Sprintf("%s has healed you!", s.player.Name))
	return nil
}

// ---------------------------------------------------------------------------
// restore — fully restore target (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdRestore(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	// C do_restore (act.wizard.c) no-arg → "Whom do you wish to restore?";
	// cmdRestore otherwise delegates the with-target path to cmdHeal.
	if len(args) == 0 {
		s.Send("Whom do you wish to restore?\r\n")
		return nil
	}
	return cmdHeal(s, args)
}

// ---------------------------------------------------------------------------
// set — set a player field (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdSet(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 3 {
		s.Send("Usage: set <player> <field> <value>")
		return nil
	}
	targetName := args[0]
	field := args[1]
	value := strings.Join(args[2:], " ")
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	field = strings.ToLower(field)

	// C do_set cases 29-31 (act.wizard.c:2977-2993): drunk/hunger/thirst accept
	// "off" (condition → -1) or a number clamped to [0,48], with C's ack bytes.
	// These are MISC-type fields, so they bypass the numeric gate below. Field
	// names map to conditions: drunk→CondDrunk, hunger→CondFull, thirst→
	// CondThirst (GET_COND(vict, l-29); DRUNK=0, FULL=1, THIRST=2).
	switch field {
	case "drunk", "hunger", "thirst":
		cond := map[string]int{
			"drunk":  game.CondDrunk,
			"hunger": game.CondFull,
			"thirst": game.CondThirst,
		}[field]
		if value == "off" {
			targetSess.player.SetCondition(cond, -1)
			s.Send(fmt.Sprintf("%s's %s now off.\r\n", targetSess.player.Name, field))
			return nil
		}
		condVal, convErr := strconv.Atoi(value)
		if convErr != nil {
			s.Send("Must be 'off' or a value from 0 to 48.\r\n")
			return nil
		}
		condVal = clamp(condVal, 0, 48)
		targetSess.player.SetCondition(cond, condVal)
		s.Send(fmt.Sprintf("%s's %s set to %d.\r\n", targetSess.player.Name, field, condVal))
		return nil
	}

	// C do_set field 32 (act.wizard.c:2577,2886) toggles PLR_OUTLAW for a
	// player at LVL_GOD. The command itself remains behind Go's existing
	// LVL_GRGOD dispatcher gate; this branch preserves C's binary parsing and
	// acknowledgement for the God-skillset spell vehicle.
	if field == "outlaw" {
		normalized := strings.ToLower(strings.TrimSpace(value))
		var enabled bool
		switch normalized {
		case "on", "yes":
			enabled = true
		case "off", "no":
		default:
			s.Send("Value must be on or off.\r\n")
			return nil
		}
		targetSess.player.SetPlrFlag(game.PlrOutlaw, enabled)
		s.Send("Okay.\r\n")
		return nil
	}

	// C's do_set has no position field.  Its field-table walk reaches the
	// sentinel and the switch's default arm returns this exact response; do
	// not turn an unsupported field into a Go-only usage hint (R1/R2/R4).
	if !isSetNumericField(field) {
		s.Send("Can't set that!\r\n")
		return nil
	}

	// Validate numeric value before assignment
	val, err := strconv.Atoi(value)
	if err != nil {
		s.Send("Invalid numeric value.")
		return nil
	}

	// Validate field bounds
	switch field {
	case "str", "stradd", "sta", "int", "wil", "wis", "dex", "con", "cha":
		maxStat := 25
		if targetSess.player.Level < LVL_GRGOD {
			maxStat = 18
		}
		val = clamp(val, 0, maxStat)
	case "level":
		val = clamp(val, 0, 61)
	case "hp", "mana", "move":
		if val > 10000 && targetSess.player.Level < 60 {
			return fmt.Errorf("cannot set %s above 10000 for non-immortals", field)
		}
	}

	switch field {
	case "level":
		targetSess.player.Level = val
		s.Send(fmt.Sprintf("Level set to %d.", val))
	case "gold":
		targetSess.player.Gold = val
		s.Send(fmt.Sprintf("Gold set to %d.", val))
	case "alignment":
		targetSess.player.Alignment = val
		s.Send(fmt.Sprintf("Alignment set to %d.", val))
	case "align":
		targetSess.player.Alignment = val
		s.Send(fmt.Sprintf("%s's align set to %d.\r\n", targetSess.player.Name, val))
	case "str":
		targetSess.player.Stats.Str = val
		targetSess.player.Strength = val
		s.Send(fmt.Sprintf("Strength set to %d.", val))
	case "stradd":
		if val > 0 {
			targetSess.player.Stats.Str = 18
			targetSess.player.Strength = 18
		}
		s.Send(fmt.Sprintf("%s's stradd set to %d.\r\n", targetSess.player.Name, val))
	case "con":
		targetSess.player.Stats.Con = val
		s.Send(fmt.Sprintf("%s's con set to %d.\r\n", targetSess.player.Name, val))
	case "sta":
		targetSess.player.Stats.Con = val
		s.Send(fmt.Sprintf("Constitution set to %d.", val))
	case "dex":
		targetSess.player.Stats.Dex = val
		s.Send(fmt.Sprintf("%s's dex set to %d.\r\n", targetSess.player.Name, val))
	case "int":
		targetSess.player.Stats.Int = val
		s.Send(fmt.Sprintf("%s's int set to %d.\r\n", targetSess.player.Name, val))
	case "wis":
		targetSess.player.Stats.Wis = val
		s.Send(fmt.Sprintf("%s's wis set to %d.\r\n", targetSess.player.Name, val))
	case "wil":
		targetSess.player.Stats.Wis = val
		s.Send(fmt.Sprintf("Wisdom set to %d.", val))
	case "cha":
		targetSess.player.Stats.Cha = val
		s.Send(fmt.Sprintf("%s's cha set to %d.\r\n", targetSess.player.Name, val))
	case "hp":
		targetSess.player.MaxHealth = val
		targetSess.player.Health = val
		s.Send(fmt.Sprintf("Hit points set to %d.", val))
	case "mana":
		targetSess.player.MaxMana = val
		targetSess.player.Mana = val
		s.Send(fmt.Sprintf("Mana set to %d.", val))
	case "move":
		targetSess.player.MaxMove = val
		targetSess.player.Move = val
		s.Send(fmt.Sprintf("Move points set to %d.", val))
	}
	slog.Warn("wizard set", "by", s.player.Name, "target", targetName, "field", field, "value", value)
	return nil
}

func isSetNumericField(field string) bool {
	switch field {
	case "level", "gold", "alignment", "align", "str", "stradd", "sta", "int", "wil", "wis", "dex", "con", "cha", "hp", "mana", "move":
		return true
	default:
		return false
	}
}

// clamp restricts v to the [min, max] range.
func cmdSwitch(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}

	// M-16 toggle: if already switched, return to original body
	if s.isSwitched {
		return cmdReturn(s, args)
	}

	if len(args) == 0 {
		s.Send("Switch with who?\r\n")
		return nil
	}
	targetName := strings.ToLower(args[0])
	roomVNum := s.player.GetRoom()

	// Store original wizard state for permission gating and return
	origLevel := s.player.Level
	origPlayer := s.player

	// Save wizard state before switching
	if err := game.SavePlayer(origPlayer); err != nil {
		slog.Error("cmdSwitch: failed to save wizard state checkpoint",
			"wizard", origPlayer.Name, "error", err)
	} else {
		slog.Info("switch: saved wizard state checkpoint", "wizard", origPlayer.Name)
	}

	// Look for a mob in the room
	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			s.switchedOriginal = origPlayer
			s.switchedOriginalLevel = origLevel
			s.switchedMob = mob
			s.isSwitched = true
			s.switchedStartTime = time.Now()
			slog.Info(
				"switch: wizard switched into mob",
				"wizard", origPlayer.Name,
				"wizard_level", origLevel,
				"target_mob", mob.GetShortDesc(),
			)
			s.Send(fmt.Sprintf("You switch into %s.\r\n", mob.GetShortDesc()))
			return nil
		}
	}

	// Look for a player in the room
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.ToLower(p.GetName()) == targetName {
			if p.Level >= s.player.Level {
				s.Send("Fuuuuuuuuu!\r\n")
				return nil
			}
			// Save target player state before switching
			if err := game.SavePlayer(p); err != nil {
				slog.Error("cmdSwitch: failed to save target player checkpoint",
					"target", p.Name, "error", err)
			} else {
				slog.Info("switch: saved target player state checkpoint", "target", p.Name)
			}

			s.switchedOriginal = origPlayer
			s.switchedOriginalLevel = origLevel
			s.switchedPlayer = p
			s.isSwitched = true
			s.switchedStartTime = time.Now()
			slog.Info(
				"switch: wizard switched into player",
				"wizard", origPlayer.Name,
				"wizard_level", origLevel,
				"target_player", p.GetName(),
				"target_level", p.Level,
			)
			s.Send(fmt.Sprintf("You switch into %s.\r\n", p.GetName()))
			return nil
		}
	}
	s.Send("No one here by that name.\r\n")
	return nil
}

// ---------------------------------------------------------------------------
// return — return to own body (LVL_IMMORT)
// ---------------------------------------------------------------------------
// cmdReturn returns the wizard to their own body after a switch.
// Expected behavior (from original C):
// - Detach the wizard's session from the switched character
// - Re-attach to the wizard's original character
func cmdReturn(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if !s.isSwitched || s.switchedOriginal == nil {
		s.Send("You aren't switched.\r\n")
		return nil
	}

	duration := time.Since(s.switchedStartTime)
	if s.switchedMob != nil {
		slog.Info(
			"return: wizard returned from mob",
			"wizard", s.switchedOriginal.Name,
			"from_mob", s.switchedMob.GetShortDesc(),
			"duration", duration,
		)
	} else if s.switchedPlayer != nil {
		slog.Info(
			"return: wizard returned from player",
			"wizard", s.switchedOriginal.Name,
			"from_player", s.switchedPlayer.GetName(),
			"duration", duration,
		)
	}

	// Save target character state to persist any changes made while switched
	if s.switchedPlayer != nil {
		if err := game.SavePlayer(s.switchedPlayer); err != nil {
			slog.Error("cmdReturn: failed to save switched player state",
				"player", s.switchedPlayer.Name, "error", err)
		} else {
			slog.Info("return: saved switched player state", "player", s.switchedPlayer.Name)
		}
	}

	// Save wizard state before restoring
	if err := game.SavePlayer(s.switchedOriginal); err != nil {
		slog.Error("cmdReturn: failed to save wizard state before restore",
			"wizard", s.switchedOriginal.Name, "error", err)
	} else {
		slog.Info("return: saved wizard state before restore", "wizard", s.switchedOriginal.Name)
	}

	s.player = s.switchedOriginal
	s.isSwitched = false
	s.switchedOriginal = nil
	s.switchedOriginalLevel = 0
	s.switchedMob = nil
	s.switchedPlayer = nil
	s.Send("You return to your own body.\r\n")
	return nil
}

// ---------------------------------------------------------------------------
// invis — toggle invisibility (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdInvis(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	// Toggle invisibility
	if s.player.Flags&game.PLR_INVISIBLE != 0 {
		s.player.Flags &^= game.PLR_INVISIBLE
		s.Send("You are now visible.")
	} else {
		s.player.Flags |= game.PLR_INVISIBLE
		s.Send("You are now invisible.")
	}
	return nil
}

// ---------------------------------------------------------------------------
// vis — make invisible players visible (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdVis(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Vis whom?")
		return nil
	}
	targetName := args[0]
	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("There is no such player.")
		return nil
	}
	// Clear their invisible flag to make them visible to lower-level players
	if target.player.Flags&game.PLR_INVISIBLE != 0 {
		target.player.Flags &^= game.PLR_INVISIBLE
		target.Send("You have been revealed by a higher power!")
		s.Send(fmt.Sprintf("%s is now visible to mortals.", target.player.Name))
	} else {
		s.Send(fmt.Sprintf("%s is already visible.", target.player.Name))
	}
	return nil
}

// ---------------------------------------------------------------------------
// gecho — broadcast to all players (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdAdvance(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Advance who?\r\n")
		return nil
	}
	targetName := args[0]
	newLevel, err := strconv.Atoi(args[1])
	if err != nil {
		s.Send("Invalid level.")
		return nil
	}
	if newLevel < 0 || newLevel > 60 {
		s.Send("Level must be between 0 and 60.")
		return nil
	}
	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("There is no such player.")
		return nil
	}
	oldLevel := target.player.Level
	target.player.Level = newLevel
	slog.Warn("player advanced", "target", target.player.Name, "old", oldLevel, "new", newLevel, "by", s.player.Name)
	s.Send(fmt.Sprintf("%s advanced from level %d to %d.", target.player.Name, oldLevel, newLevel))
	target.Send(fmt.Sprintf("You have been advanced from level %d to %d!", oldLevel, newLevel))
	return nil
}

// ---------------------------------------------------------------------------
// skillset — set a player's skill value (LVL_GRGOD, POS_SLEEPING)
// ---------------------------------------------------------------------------
// Port of do_skillset (src/modify.c:255-341). Syntax:
//
//	skillset <name> '<skill>' <value>
//
// Every message byte matches C exactly, including the MIXED \n\r vs \r\n
// terminators (C uses \r\n for the syntax line and NOPERSON, \n\r elsewhere —
// do not normalize). The no-argument path lists every spells[] entry 4-per-line
// at %18s, skipping entries whose name starts with '!', breaking on the RAW
// table index modulo 4 (i%4==3) — exactly as C's loop, so !-entries that
// increment i but are skipped still determine column alignment. The mudlog
// (step 9) is server-side only and is intentionally NOT emitted (same policy
// as do_help's usage-file write). SET_SKILL maps to Player.SetSkill keyed by the
// canonical lowercased spells[] display name.
func cmdSkillset(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh!?!")
		return nil
	}

	// Step 1: no argument → print syntax + the full skill list.
	if len(args) == 0 {
		s.Send("Syntax: skillset <name> '<skill>' <value>\r\n")
		s.Send(skillsetSkillList())
		return nil
	}

	// Step 2: target lookup (get_char_vis scope: player sessions, then mobs).
	name := args[0]
	targetSess := findSessionByName(s.manager, name)
	if targetSess == nil || targetSess.player == nil {
		// Could be a mob — C's get_char_vis resolves mobs too. If it is a mob,
		// fall through to the NPC rejection below; otherwise NOPERSON.
		if mob := s.manager.world.GetMobByName(name); mob == nil {
			s.Send(noPersonHere)
			return nil
		}
		s.Send("You can't set NPC skills.\n\r")
		return nil
	}
	vict := targetSess.player

	// The remainder after the target name: '<skill>' <value>.
	rest := strings.Join(args[1:], " ")

	// Step 3: skip_spaces; empty rest → "Skill name expected.\n\r".
	rest = strings.TrimLeft(rest, " \t")
	if rest == "" {
		s.Send("Skill name expected.\n\r")
		return nil
	}

	// Step 4: first non-space must be '\'' else "Skill must be enclosed in: ''\n\r".
	if rest[0] != '\'' {
		s.Send("Skill must be enclosed in: ''\n\r")
		return nil
	}

	// Step 5: read to the closing '\'' (C lowercases inside the quotes).
	closeIdx := strings.IndexByte(rest[1:], '\'')
	if closeIdx < 0 {
		s.Send("Skill must be enclosed in: ''\n\r")
		return nil
	}
	quoted := strings.ToLower(rest[1 : 1+closeIdx])

	// Step 6: find_skill_num → skill number; <= 0 → "Unrecognized skill.\n\r".
	skillNum := game.FindSkillNum(quoted)
	if skillNum <= 0 {
		s.Send("Unrecognized skill.\n\r")
		return nil
	}

	// Step 7: next arg = value (whatever follows the closing quote).
	valueStr := strings.TrimSpace(rest[1+closeIdx+1:])
	if valueStr == "" {
		s.Send("Learned value expected.\n\r")
		return nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		// C's atoi returns 0 on non-numeric input (e.g. atoi("garbage")==0),
		// which passes the >=0 check and sets the skill to 0. strconv.Atoi
		// rejects the whole string, so mirror C by treating a parse failure as
		// value 0. Known divergence: C atoi("75abc")==75 (leading digits), but
		// strconv.Atoi("75abc") fails → 0 here. Edge case; left as-is (note).
		value = 0
	} else if value < 0 {
		// atoi of a negative like "-5" yields -5 → min error.
		s.Send("Minimum value for learned is 0.\n\r")
		return nil
	}
	if value > 100 {
		s.Send("Max value for learned is 100.\n\r")
		return nil
	}

	// Step 8: IS_NPC(vict) → "You can't set NPC skills.\n\r". (Unreachable for a
	// session-resolved player, but kept for parity with C's order and the mob
	// path above.)
	if vict.IsNPC() {
		s.Send("You can't set NPC skills.\n\r")
		return nil
	}

	// Step 9: mudlog — server-side only, NOT player-facing: skipped (see header).

	// Step 10: SET_SKILL(vict, skill, value). Go stores skills by name string;
	// use the canonical spells[] display name (lowercased, matching how callers
	// key GetSkill/SetSkill — see spec_procs.go practice).
	canonicalName := strings.ToLower(game.SkillCatalogName(skillNum))
	// C's spells[] display name is "pick lock", while the door command's
	// gameplay lookup uses the Go storage key pick_lock. Keep other display
	// names unchanged because skillset also accepts ordinary spell names.
	if canonicalName == "pick lock" {
		canonicalName = game.SkillPickLock
	}
	vict.SetSkill(canonicalName, value)

	// Step 11: confirmation to the actor. spells[skill] is the display name.
	s.Send(fmt.Sprintf("You change %s's %s to %d.\n\r", vict.Name, game.SkillCatalogName(skillNum), value))
	return nil
}

// noPersonHere is C's NOPERSON ("No-one by that name here.\r\n" — config.c:93),
// reused verbatim (not an invented variant — see DP-1200). The British
// hyphenated "No-one" and CRLF terminator are byte-exact.
const noPersonHere = "No-one by that name here.\r\n"

// skillsetSkillList renders the no-argument skill list exactly as C's
// do_skillset (modify.c:266-279): "Skill being one of the following:\n\r",
// then every spells[] entry 4-per-line at %18s, skipping names whose first
// char is '!', breaking the line on the RAW index modulo 4 (i%4==3). A
// completed line (i%4==3) gets a trailing "\r\n"; the partial final line is
// sent AS-IS with no "\r\n", followed by a single trailing "\n\r" — matching C,
// which flushes completed lines in-loop and sends the leftover help buffer
// verbatim. Do NOT add a "\r\n" to the partial line (R1).
func skillsetSkillList() string {
	var b strings.Builder
	b.WriteString("Skill being one of the following:\n\r")
	size := spells.SkillCatalogSize()
	for i := 0; i < size; i++ {
		raw := spells.SpellRawName(i)
		if raw == "" {
			continue
		}
		if raw[0] == '\n' { // C loop terminator: *spells[i] != '\n'
			break
		}
		if raw[0] == '!' {
			continue // skip ! entries, but i still increments
		}
		fmt.Fprintf(&b, "%18s", raw)
		if i%4 == 3 {
			b.WriteString("\r\n") // completed line; C resets the help buffer here
		}
	}
	// C: if (*help) send_to_char(help, ch); — the partial final line is already
	// in the builder with NO trailing \r\n. Then the single trailing "\n\r".
	b.WriteString("\n\r")
	return b.String()
}

// ---------------------------------------------------------------------------
// reload — reload world data (LVL_GOD)
// ---------------------------------------------------------------------------
// reload — reload world data (LVL_GOD)
// Re-reads world files from disk and replaces the in-memory world.
