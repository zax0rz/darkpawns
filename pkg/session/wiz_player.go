package session

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
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
	if !checkLevel(s, LVL_GOD-1) {
		s.Send("Huh?!?")
		return nil
	}
	// C do_restore uses one_argument: fill words are skipped and trailing input
	// is ignored. Its visible lookup includes both players and NPCs.
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName == "" {
		s.Send("Whom do you wish to restore?\r\n")
		return nil
	}

	targetSess := findSessionByName(s.manager, targetName)
	var targetMob *game.MobInstance
	if targetSess == nil && s.manager != nil && s.manager.world != nil {
		targetMob = s.manager.world.GetMobByName(targetName)
	}
	if targetSess == nil && targetMob == nil {
		s.Send("No-one by that name here.\r\n")
		return nil
	}

	if targetSess != nil && targetSess.player != nil {
		target := targetSess.player
		target.SetHealth(target.GetMaxHP())
		target.SetMana(target.GetMaxMana())
		target.SetMove(target.GetMaxMove())
		if target.GetHP() > 0 && target.GetPosition() <= combat.PosStunned {
			target.SetPosition(combat.PosStanding)
		}

		if getEffectiveLevel(s) >= LVL_GRGOD && target.GetLevel() >= LVL_IMMORT {
			if target.SkillManager != nil {
				for _, skill := range target.SkillManager.GetAllSkills() {
					target.SetSkill(skill.Name, 100)
				}
			}
			if target.GetLevel() >= LVL_GRGOD {
				target.Lock()
				target.Stats.StrAdd = 100
				target.Stats.Int = 25
				target.Stats.Wis = 25
				target.Stats.Dex = 25
				target.Stats.Str = 25
				target.Stats.Con = 25
				target.Stats.Cha = 25
				target.Strength = 25
				target.Unlock()
			}
		}
	} else if targetMob != nil {
		targetMob.SetHealth(targetMob.GetMaxHP())
		targetMob.SetMana(targetMob.GetMaxMana())
		targetMob.SetMove(targetMob.GetMaxMove())
		if targetMob.GetHP() > 0 && targetMob.GetPosition() <= combat.PosStunned {
			targetMob.SetPosition(combat.PosStanding)
		}
	}

	s.Send("Okay.\r\n")
	if targetSess != nil && targetSess.player != nil {
		targetSess.Send(fmt.Sprintf("The hand of %s touches you, healing your wounds and leaving you refreshed!\r\n", s.player.Name))
	}
	return nil
}

// clamp restricts v to the [min, max] range.
func cmdSwitch(s *Session, args []string) error {
	// The command-table level gate is authoritative. If the descriptor is
	// already attached to a switched body, do_switch reports this before any
	// target parsing; the interpreter's switched-NPC gate normally intercepts
	// that case first.
	if s.isSwitched {
		s.Send("You're already switched.\r\n")
		return nil
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

	// Look for a mob in the room
	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			s.switchedOriginal = origPlayer
			s.switchedOriginalLevel = origLevel
			s.switchedMob = mob
			s.isSwitched = true
			s.switchedStartTime = time.Now()
			s.Send("Okay.\r\n")
			return nil
		}
	}

	// Look for a player in the room
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.ToLower(p.GetName()) == targetName {
			if p == s.player {
				s.Send("Hee hee... we are jolly funny today, eh?\r\n")
				return nil
			}
			if findSessionByName(s.manager, p.GetName()) != nil {
				s.Send("You can't do that, the body is already in use!\r\n")
				return nil
			}
			if s.player.Level < LVL_IMPL {
				s.Send("You aren't holy enough to use a mortal's body.\r\n")
				return nil
			}

			s.switchedOriginal = origPlayer
			s.switchedOriginalLevel = origLevel
			s.switchedPlayer = p
			s.isSwitched = true
			s.switchedStartTime = time.Now()
			s.Send("Okay.\r\n")
			return nil
		}
	}
	s.Send("No such character.\r\n")
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
	// C do_return has no handler-level authorization gate and is silent unless
	// the descriptor is attached to a switched body. Its arguments are ignored.
	if !s.isSwitched || s.switchedOriginal == nil {
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

	s.player = s.switchedOriginal
	s.isSwitched = false
	s.switchedOriginal = nil
	s.switchedOriginalLevel = 0
	s.switchedMob = nil
	s.switchedPlayer = nil
	s.Send("You return to your original body.\r\n")
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
	s.manager.world.DoInvis(s.player, strings.Join(args, " "))
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
	if len(args) == 0 {
		s.Send("Advance who?\r\n")
		return nil
	}
	target, ok := s.manager.world.ResolveCharWorld(s.player, args[0])
	if !ok {
		s.Send("That player is not here.\r\n")
		return nil
	}

	actorLevel := s.player.GetLevel()
	victimLevel := target.Combatant.GetLevel()
	if actorLevel != LVL_IMPL && actorLevel <= victimLevel {
		s.Send("Maybe that's not such a great idea.\r\n")
		return nil
	}
	if target.Combatant.IsNPC() {
		s.Send("NO!  Not on NPC's.\r\n")
		return nil
	}
	newLevel, ok := parseAdvanceLevel(args[1:])
	if !ok || newLevel <= 0 {
		s.Send("That's not a level!\r\n")
		return nil
	}
	if newLevel > LVL_IMPL {
		s.Send(fmt.Sprintf("%d is the highest possible level.\r\n", LVL_IMPL))
		return nil
	}
	if newLevel > actorLevel {
		s.Send("Yeah, right.\r\n")
		return nil
	}
	victim, ok := target.Combatant.(*game.Player)
	if !ok || victim == nil {
		// The NPC branch above is exhaustive for the current CharTarget
		// implementations. Keep the C target failure text if a future target
		// type reaches this command instead of inventing a success path.
		s.Send("That player is not here.\r\n")
		return nil
	}
	if newLevel == victimLevel {
		s.Send("They are already at that level.\r\n")
		return nil
	}

	oldLevel := victimLevel
	if newLevel < victimLevel {
		// C do_advance demotes by running do_start(), which resets level/exp and
		// base hit/mana, gives another starting kit, advances once, then leaves
		// the real abilities intact. Go stores real abilities directly in Stats;
		// no separate roll is performed here, so there is nothing to restore.
		victim.SendMessage("You are momentarily enveloped by darkness!\r\n" +
			"You can feel all your power and knowledge being\r\n" +
			"drained away from you!\r\n")
		s.manager.world.GiveStartingItems(victim)
		victim.SetLevel(1)
		victim.SetExp(1)
		victim.SetMaxHP(10)
		victim.SetMaxMana(100)
		game.GiveStartingSkills(victim)
		victim.AdvanceLevel()
		victim.SetHP(victim.GetMaxHP())
		victim.SetMana(victim.GetMaxMana())
		victim.SetMove(victim.GetMaxMove())
		victim.SetCondition(game.CondThirst, 36)
		victim.SetCondition(game.CondFull, 36)
		victim.SetCondition(game.CondDrunk, 0)
		victim.WimpLevel = 5
		victim.SetPractices(victim.GetPractices() + 2)

		if newLevel != 1 {
			// This mirrors do_advance's recursive promotion after do_start().
			_ = cmdAdvance(s, args)
		}
	} else {
		promotionText := "$n makes some strange gestures.\n" +
			"A strange feeling comes upon you,\n" +
			"Like a giant hand, light comes down\n" +
			"from above, grabbing your body, that\n" +
			"begins to pulse with colored lights\n" +
			"from inside.\n\n" +
			"Your head seems to be filled with demons\n" +
			"from another plane as your body dissolves\n" +
			"to the elements of time and space itself.\n" +
			"Suddenly a silent explosion of light\n" +
			"snaps you back to reality.\n\n" +
			"You feel slightly different."
		game.Act(s.manager.world, false, s.player, victim, nil, nil,
			promotionText, "", game.ToVict)
		var levelMessages strings.Builder
		for level := victim.GetLevel(); level < newLevel; level++ {
			gain := game.ExpNeededForLevel(victim) - victim.GetExp()
			levelsGained := s.manager.world.GainExpRegardlessSilent(victim, gain)
			if levelsGained == 1 {
				levelMessages.WriteString("You rise a level!\r\n")
			} else if levelsGained > 1 {
				_, _ = fmt.Fprintf(&levelMessages, "You rise %d levels!\r\n", levelsGained)
			}
		}
		if levelMessages.Len() > 0 {
			victim.SendMessage(levelMessages.String())
		}
	}

	s.Send("Okay.\r\n")
	slog.Info("(GC) player advanced", "by", s.player.Name, "target", victim.Name, "old", oldLevel, "new", newLevel)
	if err := game.SavePlayer(victim); err != nil {
		slog.Error("advance: save player failed", "player", victim.Name, "error", err)
	}
	return nil
}

// parseAdvanceLevel mirrors C atoi() for do_advance: it accepts an optional
// sign and consumes the leading decimal digits, ignoring a trailing suffix.
func parseAdvanceLevel(args []string) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	return parseSignedIntPrefix(args[0])
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

	// Step 2: target lookup. C's get_char_vis checks visible in-room players and
	// mobs with keyword abbreviations, then exact global character names. Keep
	// that scope in the shared resolver so self/me, ordinals, NPCs, and room
	// visibility follow the same call path as other C-targeted commands.
	name := args[0]
	target, found := s.manager.world.ResolveCharWorld(s.player, name)
	if !found {
		s.Send(noPersonHere)
		return nil
	}
	if target.Mob != nil {
		s.Send("You can't set NPC skills.\n\r")
		return nil
	}
	vict := target.Player
	if vict == nil {
		// ResolveCharWorld only returns a player or mob, but preserve the C
		// failure bytes if a future combatant type reaches this command.
		s.Send(noPersonHere)
		return nil
	}

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

	// Step 7: next arg = value. C calls one_argument, so only the first token
	// is passed to atoi; atoi consumes an optional sign and leading digits and
	// returns zero when no digits are present.
	valueStr := strings.TrimSpace(rest[1+closeIdx+1:])
	if valueStr == "" {
		s.Send("Learned value expected.\n\r")
		return nil
	}
	valueToken := strings.Fields(valueStr)[0]
	value := cAtoi(valueToken)
	if value < 0 {
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
	canonicalName := game.SkillStorageName(skillNum)
	// C's spells[] display name is "pick lock", while the door command's
	// gameplay lookup uses the Go storage key pick_lock. First aid is likewise
	// displayed as C's "aid" but stored under the Go key first_aid.
	switch canonicalName {
	case "search":
		// C's spells[SKILL_DETECT] display name is "search", while
		// do_detect's gameplay lookup uses the command-facing SkillDetect key.
		canonicalName = game.SkillDetect
	case "pick lock":
		canonicalName = game.SkillPickLock
	case "aid":
		canonicalName = game.SkillFirstAid
	case "flesh alter":
		canonicalName = game.SkillFleshAlter
	case "dragon kick":
		canonicalName = game.SkillDragonKick
	case "serpent kick":
		canonicalName = game.SkillSerpentKick
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
