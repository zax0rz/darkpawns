package session

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

const maxCastSpell = 130 // C spells.h MAX_SPELLS

// castNumber is a test seam for the single do_cast concentration roll.
// Production always uses the process-wide deterministic stream.
var castNumber = dprng.Number

type castTarget struct {
	character interface{}
	object    interface{}
	found     bool
}

func parseCastArguments(args []string) (spellName, targetName, errorMessage string) {
	return parseCastArgumentsForPower(args, false)
}

func parseCastArgumentsForPower(args []string, power bool) (spellName, targetName, errorMessage string) {
	spellName, targetName, _, errorMessage = parseCastArgumentsForPowerWithArgument(args, power)
	return spellName, targetName, errorMessage
}

func parseCastArgumentsForPowerWithArgument(args []string, power bool) (spellName, targetName, spellArgument, errorMessage string) {
	input := strings.Join(args, " ")
	if strings.TrimSpace(input) == "" {
		if power {
			return "", "", "", "Will what?\r\n"
		}
		return "", "", "", "Cast what where?\r\n"
	}
	opening := strings.IndexByte(input, '\'')
	if opening < 0 {
		if power {
			return "", "", "", "Psionic powers must be enclosed in the symbols: '\r\n"
		}
		return "", "", "", "Spell names must be enclosed in the magick symbols: '\r\n"
	}
	remaining := input[opening+1:]
	closing := strings.IndexByte(remaining, '\'')
	if closing < 0 {
		// C's do_cast receives the command interpreter's remainder, which
		// retains the separating space before the opening quote. strtok() sees
		// that space as its first token and accepts the rest of the input as
		// the spell name when there is no closing quote. The target is empty.
		if spellName := strings.TrimSpace(remaining); spellName != "" {
			return spellName, "", "", ""
		}
		if power {
			return "", "", "", "Psionic powers must be enclosed in the symbols: '\r\n"
		}
		return "", "", "", "Spell names must be enclosed in the magick symbols: '\r\n"
	}
	spellArgument = remaining[closing+1:]
	targetName, _ = game.OneArgument(spellArgument)
	return strings.TrimSpace(remaining[:closing]), targetName, spellArgument, ""
}

func resolveCastTarget(s *Session, info *spells.SpellInfo, targetName string) (castTarget, string) {
	return resolveCastTargetForCommand(s, info, targetName, false)
}

func resolveCastTargetForCommand(s *Session, info *spells.SpellInfo, targetName string, power bool) (castTarget, string) {
	if info.HasTarget(spells.TarIgnore) {
		return castTarget{found: true}, ""
	}

	if targetName != "" {
		if info.HasTarget(spells.TarCharRoom) {
			if target, ok := s.manager.world.ResolveCharInRoom(s.player, targetName); ok {
				return castTarget{character: target.Combatant, found: true}, ""
			}
		}
		if info.HasTarget(spells.TarCharWorld) {
			if target, ok := s.manager.world.ResolveCharWorld(s.player, targetName); ok {
				return castTarget{character: target.Combatant, found: true}, ""
			}
		}
		if info.HasTarget(spells.TarObjInv) {
			if target, ok := s.manager.world.ResolveObjectInInventory(s.player, targetName); ok {
				return castTarget{object: target, found: true}, ""
			}
		}
		if info.HasTarget(spells.TarObjEquip) {
			if target, ok := s.manager.world.ResolveObjectInEquipment(s.player, targetName); ok {
				return castTarget{object: target, found: true}, ""
			}
		}
		if info.HasTarget(spells.TarObjRoom) {
			if target, ok := s.manager.world.ResolveObjectInRoom(s.player, targetName); ok {
				return castTarget{object: target, found: true}, ""
			}
		}
		if info.HasTarget(spells.TarObjWorld) {
			if target, ok := s.manager.world.ResolveObjectWorld(s.player, targetName); ok {
				return castTarget{object: target, found: true}, ""
			}
		}
		return castTarget{}, ""
	}

	fighting := s.player.GetFightingBody()
	if fighting != nil && info.HasTarget(spells.TarFightSelf) {
		return castTarget{character: s.player, found: true}, ""
	}
	if fighting != nil && info.HasTarget(spells.TarFightVict) {
		if target, ok := s.manager.world.ResolveFightingTarget(s.player); ok {
			return castTarget{character: target.Combatant, found: true}, ""
		}
	}
	if info.HasTarget(spells.TarCharRoom) && !info.IsViolent() {
		return castTarget{character: s.player, found: true}, ""
	}

	targetWord := "who"
	if info.Routines.Targets&(spells.TarObjRoom|spells.TarObjInv|spells.TarObjWorld) != 0 {
		targetWord = "what"
	}
	if power {
		return castTarget{}, fmt.Sprintf("Upon %s should the power be willed?\r\n", targetWord)
	}
	return castTarget{}, fmt.Sprintf("Upon %s should the spell be cast?\r\n", targetWord)
}

func checkCastSpellContract(s *Session, info *spells.SpellInfo, target castTarget) bool {
	power := castUsesPower(s.player)
	if s.player.GetWis() == 0 || s.player.GetInt() == 0 {
		s.Send("You're not smart enough to cast!\r\n")
		return false
	}
	if !checkCastPosition(s, info) {
		return false
	}
	if s.player.IsAffected(game.AffCharm) && s.player.GetFollowing() != "" && castTargetName(target) != "" && strings.EqualFold(s.player.GetFollowing(), castTargetName(target)) {
		s.Send("You are afraid you might hurt your master!\r\n")
		return false
	}
	if target.character != s.player && info.HasTarget(spells.TarSelfOnly) {
		s.Send(castMessage(power, "You can only cast this spell upon yourself!\r\n", "You can only will this power upon yourself!\r\n"))
		return false
	}
	if target.character == s.player && info.HasTarget(spells.TarNotSelf) {
		s.Send(castMessage(power, "You cannot cast this spell upon yourself!\r\n", "You cannot will this power upon yourself!\r\n"))
		return false
	}
	if info.HasRoutine(spells.RoutineGroups) && !s.player.InGroup {
		s.Send(castMessage(power, "You can't cast this spell if you're not in a group!\r\n", "You cannot use this power if you are not in a group!\r\n"))
		return false
	}
	return true
}

func castTargetName(target castTarget) string {
	if named, ok := target.character.(interface{ GetName() string }); ok {
		return named.GetName()
	}
	return ""
}

func castUsesPower(player *game.Player) bool {
	if player == nil {
		return false
	}
	class := player.GetClass()
	return class == game.ClassPsionic || class == game.ClassMystic
}

func castMessage(power bool, normal, powered string) string {
	if power {
		return powered
	}
	return normal
}

// cmdCast handles the "cast <spell> [target]" command.
// Implements do_cast from cast.c / spell_parser.c.
func cmdCast(s *Session, args []string) error {
	return cmdCastCommand(s, args, "cast")
}

func cmdWill(s *Session, args []string) error {
	return cmdCastCommand(s, args, "will")
}

func cmdCastCommand(s *Session, args []string, commandName string) error {
	if s.player == nil || s.player.IsNPC() {
		return nil
	}

	power := castUsesPower(s.player)
	if commandName == "cast" && s.player.GetClass() == game.ClassPsionic && s.player.GetLevel() < LVL_IMMORT {
		s.Send("Psionics 'will' things, not 'cast' them!\r\n")
		return nil
	}

	spellName, targetName, spellArgument, parseError := parseCastArgumentsForPowerWithArgument(args, power)
	if parseError != "" {
		s.Send(parseError)
		return nil
	}

	spellNum := game.FindSkillNum(spellName)
	if spellNum < 1 || spellNum > maxCastSpell {
		s.Send(castMessage(power, "Cast what?!?\r\n", "Will what?!?\r\n"))
		return nil
	}
	sd := game.SpellManaTable[spellNum]
	info := spells.GetSpellInfo(spellNum)
	if sd == nil || info == nil {
		s.Send(castMessage(power, "Cast what?!?\r\n", "Will what?!?\r\n"))
		return nil
	}

	minLevel := game.ClassSkillMinLevel(s.player.GetClass(), spellNum)
	if s.player.GetLevel() < minLevel {
		s.Send(castMessage(power, "You do not know that spell!\r\n", "You are not learned in that power!\r\n"))
		return nil
	}
	canonicalName := game.SkillStorageName(spellNum)
	proficiency := s.player.GetSkill(canonicalName)
	if proficiency == 0 {
		s.Send(castMessage(power, "You are unfamiliar with that spell.\r\n", "You are unfamiliar with that power.\r\n"))
		return nil
	}

	if info.IsViolent() {
		room := s.manager.world.GetRoomInWorld(s.player.GetRoomVNum())
		if room != nil && room.HasFlag(spells.RoomPeaceful) {
			s.Send("This room just has such a peaceful, easy feeling..\r\n")
			return nil
		}
	}

	target, prompt := resolveCastTargetForCommand(s, info, targetName, power)
	if prompt != "" {
		s.Send(prompt)
		return nil
	}
	if target.found && target.character == s.player && info.IsViolent() {
		s.Send(castMessage(power, "You shouldn't cast that on yourself -- could be bad for your health!\r\n", "Exerting that power on yourself could be harmful!\r\n"))
		return nil
	}
	if !target.found {
		s.Send("Okay.\r\n")
		spells.SaySpell(s.player, spellNum, nil, nil, s.manager.world)
		s.Send(castMessage(power, "Cannot find the target of your spell!\r\n", "Cannot find the target of your will!\r\n"))
		return nil
	}

	casterLevel := s.player.GetLevel()
	cost := game.ManaCost(spellNum, s.player.GetClass(), casterLevel)
	mortal := casterLevel < LVL_IMMORT

	// Reserve the mana here. C checks sufficiency at this point and deducts
	// after the concentration roll, so two concurrent casts both passed the
	// check and one cost bought two spells (CWE-367). Immortals skip the
	// sufficiency check but still pay, exactly as C does, so their deduction
	// stays at the success site.
	if cost > 0 && mortal && !s.player.TrySpendMana(cost) {
		s.Send(castMessage(power, "You haven't the energy to cast that spell!\r\n", "You haven't the energy to will that power!\r\n"))
		return nil
	}
	// refund releases whatever part of the reservation this path did not spend.
	// Every exit before the spell resolves must call it, or a cast that C never
	// charged for would still cost mana.
	refund := func(keep int) {
		if cost > 0 && mortal && keep < cost {
			s.player.AddMana(cost - keep)
		}
	}

	weightAdd := castWeightPenalty(s.player)
	if s.player.Level >= LVL_IMMORT {
		weightAdd = -20
	}
	// #nosec G404 — game RNG, not cryptographic
	if castNumber(0, 101+weightAdd) > proficiency {
		s.player.SetWaitState(1)
		// C routes aimed concentration failures through skill_message before
		// falling back to this generic text (spell_parser.c:1085-1095).
		if victim, ok := target.character.(combat.Combatant); !ok || victim == nil || !combat.EmitSkillMessage(0, s.player, victim, spellNum, s.player.GetRoomVNum()) {
			s.Send("You lost your concentration!\r\n")
		}
		if cost > 0 {
			if mortal {
				// C's failed concentration leaves half the cost spent; the
				// reservation already paid the full amount.
				refund(cost >> 1)
			} else {
				s.player.SpendManaUpTo(cost >> 1)
			}
		}
		if info.IsViolent() {
			if mob, ok := target.character.(*game.MobInstance); ok && mob.GetFightingBody() == nil && s.manager.combatEngine != nil {
				// C: hit(tch, ch, TYPE_UNDEFINED) on a violent fumble
				// (spell_parser.c:1101-1106) — the mob's first swing consumes
				// hit()'s draws and runs damage()'s gates before anyone
				// enrols; a plain enrolment skips the swing entirely.
				if err := s.manager.combatEngine.PerformDeferredInitialAttack(mob, s.player); err != nil {
					slog.Warn("cast-failure retaliation failed", "mob", mob.GetName(), "caster", s.player.Name, "error", err)
				}
			}
		}
		return nil
	}

	if !checkCastSpellContract(s, info, target) {
		// The spell never resolves, so C charges nothing for it.
		refund(0)
		return nil
	}

	s.Send("Okay.\r\n")
	spells.SaySpell(s.player, spellNum, target.character, target.object, s.manager.world)
	if spells.CallMagicWithArgument(s.player, target.character, target.object, spellNum, casterLevel, spells.CastSpell, s.manager.world, spellArgument) {
		s.player.SetWaitState(1)
		if cost > 0 && !mortal {
			// Mortals already paid at the sufficiency check.
			s.player.SpendManaUpTo(cost)
		}
	} else {
		// The spell did not resolve, so the reservation is returned.
		refund(0)
	}

	return nil
}

func checkCastPosition(s *Session, info *spells.SpellInfo) bool {
	if info == nil || s.player.GetPosition() >= int(info.MinPosition) {
		return true
	}
	switch s.player.GetPosition() {
	case int(spells.PosSleeping):
		s.Send("You dream about great magical powers.\r\n")
	case int(spells.PosResting):
		s.Send("You cannot concentrate while resting.\r\n")
	case int(spells.PosSitting):
		s.Send("You can't do this sitting!\r\n")
	case int(spells.PosFighting):
		s.Send("Impossible!  You can't concentrate enough!\r\n")
	default:
		s.Send("You can't do much of anything like this!\r\n")
	}
	return false
}

func castWeightPenalty(ch *game.Player) int {
	carried := ch.CarriedWeight()
	if carried == 0 {
		return 0
	}
	ratio := ch.MaxCarryWeight() / carried
	switch ratio {
	case 1:
		return 10
	case 2:
		return 7
	case 3:
		return 5
	default:
		return 0
	}
}

func init() {
	// Register the cast command with aliases
	registerCommand("cast", wrapArgs(cmdCast), "Cast a spell.")
	registerCommand("will", wrapArgs(cmdWill), "Will a psionic or mystic power.")
}
