package session

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

const maxCastSpell = 130 // C spells.h MAX_SPELLS

// castNumber is a test seam for the single do_cast concentration roll.
// Production always uses the process-wide deterministic stream.
var castNumber = dprng.Number

// spellData holds spell mana parameters from the original spello() table.
type spellData struct {
	SpellNum   int
	Name       string
	ManaMax    int
	ManaMin    int
	ManaChange int
	MinLevel   [12]int // 12 classes, indexed by Class constant
}

// spells maps spell number -> spell data.
var spellDB = map[int]*spellData{
	1:   {1, "armor", 30, 15, 3, [12]int{}},
	2:   {2, "teleport", 60, 50, 3, [12]int{}},
	3:   {3, "bless", 36, 10, 2, [12]int{}},
	4:   {4, "blindness", 35, 25, 1, [12]int{}},
	5:   {5, "burning hands", 45, 20, 5, [12]int{}},
	6:   {6, "call lightning", 68, 52, 5, [12]int{}},
	7:   {7, "charm", 75, 50, 5, [12]int{}},
	8:   {8, "chill touch", 35, 15, 5, [12]int{}},
	9:   {9, "clone", 80, 65, 5, [12]int{}},
	10:  {10, "color spray", 58, 38, 4, [12]int{}},
	11:  {11, "control weather", 75, 25, 5, [12]int{}},
	12:  {12, "create food", 35, 10, 5, [12]int{}},
	13:  {13, "create water", 35, 10, 5, [12]int{}},
	14:  {14, "cure blind", 35, 5, 5, [12]int{}},
	15:  {15, "cure critical", 70, 40, 5, [12]int{}},
	16:  {16, "cure light", 30, 10, 2, [12]int{}},
	17:  {17, "curse", 80, 50, 2, [12]int{}},
	18:  {18, "detect alignment", 20, 10, 2, [12]int{}},
	19:  {19, "detect invis", 20, 10, 2, [12]int{}},
	20:  {20, "detect magic", 20, 10, 2, [12]int{}},
	21:  {21, "detect poison", 20, 10, 2, [12]int{}},
	22:  {22, "dispel evil", 95, 65, 5, [12]int{}},
	23:  {23, "earthquake", 70, 50, 5, [12]int{}},
	24:  {24, "enchant weapon", 200, 150, 10, [12]int{}},
	25:  {25, "energy drain", 60, 45, 5, [12]int{}},
	26:  {26, "fireball", 70, 50, 2, [12]int{}},
	27:  {27, "harm", 105, 75, 5, [12]int{}},
	28:  {28, "heal", 90, 80, 3, [12]int{}},
	29:  {29, "invisible", 45, 45, 1, [12]int{}},
	30:  {30, "lightning bolt", 54, 34, 4, [12]int{}},
	31:  {31, "locate object", 25, 20, 1, [12]int{}},
	32:  {32, "flame arrow", 30, 15, 5, [12]int{}},
	33:  {33, "poison", 50, 40, 2, [12]int{}},
	34:  {34, "protect evil", 50, 50, 1, [12]int{}},
	35:  {35, "remove curse", 45, 45, 1, [12]int{}},
	36:  {36, "sanctuary", 110, 85, 2, [12]int{}},
	37:  {37, "shocking grasp", 55, 35, 5, [12]int{}},
	38:  {38, "sleep", 40, 35, 1, [12]int{}},
	39:  {39, "strength", 35, 30, 1, [12]int{}},
	40:  {40, "summon", 90, 70, 1, [12]int{}},
	41:  {41, "meteor swarm", 180, 170, 5, [12]int{}},
	42:  {42, "recall", 50, 50, 1, [12]int{}},
	43:  {43, "remove poison", 40, 30, 1, [12]int{}},
	44:  {44, "sense life", 30, 20, 1, [12]int{}},
	45:  {45, "animate dead", 120, 100, 10, [12]int{}},
	46:  {46, "dispel good", 95, 65, 5, [12]int{}},
	47:  {47, "holy shield", 90, 65, 5, [12]int{}},
	48:  {48, "group heal", 210, 150, 5, [12]int{}},
	49:  {49, "group recall", 155, 125, 5, [12]int{}},
	50:  {50, "infravision", 25, 25, 1, [12]int{}},
	51:  {51, "waterwalk", 80, 55, 1, [12]int{}},
	52:  {52, "mass heal", 130, 100, 1, [12]int{}},
	53:  {53, "fly", 100, 80, 5, [12]int{}},
	54:  {54, "calliope", 100, 50, 10, [12]int{}},
	55:  {55, "vampirism", 1, 1, 1, [12]int{}},
	56:  {56, "sobriety", 35, 20, 5, [12]int{}},
	57:  {57, "group invis", 135, 135, 1, [12]int{}},
	58:  {58, "hellfire", 200, 150, 10, [12]int{}},
	59:  {59, "enchant armor", 150, 130, 10, [12]int{}},
	60:  {60, "identify", 125, 100, 10, [12]int{}},
	61:  {61, "mindpoke", 30, 15, 5, [12]int{}},
	62:  {62, "mindblast", 70, 40, 2, [12]int{}},
	63:  {63, "chameleon", 50, 30, 5, [12]int{}},
	64:  {64, "levitate", 90, 70, 5, [12]int{}},
	65:  {65, "metalskin", 75, 60, 1, [12]int{}},
	66:  {66, "invulnerability", 85, 85, 1, [12]int{}},
	67:  {67, "vitality", 110, 100, 1, [12]int{}},
	68:  {68, "invigorate", 110, 95, 1, [12]int{}},
	69:  {69, "lesser perception", 40, 30, 1, [12]int{}},
	70:  {70, "greater perception", 65, 45, 1, [12]int{}},
	71:  {71, "mind attack", 55, 25, 1, [12]int{}},
	72:  {72, "adrenaline", 35, 30, 1, [12]int{}},
	73:  {73, "psyshield", 30, 20, 1, [12]int{}},
	74:  {74, "change density", 70, 55, 1, [12]int{}},
	75:  {75, "acid blast", 35, 20, 1, [12]int{}},
	76:  {76, "dominate", 75, 50, 5, [12]int{}},
	77:  {77, "cell adjustment", 85, 75, 1, [12]int{}},
	78:  {78, "zen", 70, 60, 4, [12]int{}},
	79:  {79, "mirror image", 150, 130, 5, [12]int{}},
	80:  {80, "mass dominate", 220, 150, 10, [12]int{}},
	81:  {81, "divine int", 290, 290, 1, [12]int{}},
	82:  {82, "mind bar", 115, 100, 1, [12]int{}},
	83:  {83, "soul leech", 60, 55, 1, [12]int{}},
	84:  {84, "mindsight", 70, 60, 1, [12]int{}},
	85:  {85, "transparency", 35, 25, 1, [12]int{}},
	86:  {86, "know alignment", 20, 20, 1, [12]int{}},
	87:  {87, "gate", 95, 95, 1, [12]int{}},
	88:  {88, "intellect", 60, 60, 1, [12]int{}},
	89:  {89, "lay hands", 90, 90, 1, [12]int{}},
	90:  {90, "mental lapse", 100, 90, 1, [12]int{}},
	91:  {91, "smokescreen", 100, 100, 1, [12]int{}},
	92:  {92, "disrupt", 175, 165, 1, [12]int{}},
	93:  {93, "disintegrate", 120, 120, 1, [12]int{}},
	94:  {94, "calliope", 100, 50, 10, [12]int{}},
	95:  {95, "protect good", 50, 50, 1, [12]int{}},
	96:  {96, "flamestrike", 105, 100, 1, [12]int{}},
	97:  {97, "haste", 140, 140, 1, [12]int{}},
	98:  {98, "slow", 80, 50, 2, [12]int{}},
	99:  {99, "dream travel", 60, 45, 1, [12]int{}},
	100: {100, "psiblast", 180, 150, 10, [12]int{}},
	101: {101, "call of chaos", 90, 70, 1, [12]int{}},
	102: {102, "water breathe", 92, 58, 6, [12]int{}},
	105: {105, "conjure elemental", 165, 145, 1, [12]int{}},
}

// manaCost computes the mana cost for a spell at a given caster level and class.
// Formula: MAX(mana_max - (mana_change * (caster_level - min_level)), mana_min)
func manaCost(sd *spellData, casterLevel int, class int) int {
	minLvl := game.ClassSkillMinLevel(class, sd.SpellNum)
	if minLvl == 999 {
		minLvl = sd.MinLevel[class]
	}
	// Don't allow negative scaling; if caster below min level, use max cost
	diff := casterLevel - minLvl
	if diff < 0 {
		diff = 0
	}
	cost := sd.ManaMax - (sd.ManaChange * diff)
	if cost < sd.ManaMin {
		cost = sd.ManaMin
	}
	return cost
}

type castTarget struct {
	character interface{}
	object    interface{}
	found     bool
}

func parseCastArguments(args []string) (spellName, targetName, errorMessage string) {
	return parseCastArgumentsForPower(args, false)
}

func parseCastArgumentsForPower(args []string, power bool) (spellName, targetName, errorMessage string) {
	input := strings.Join(args, " ")
	if strings.TrimSpace(input) == "" {
		if power {
			return "", "", "Will what?\r\n"
		}
		return "", "", "Cast what where?\r\n"
	}
	opening := strings.IndexByte(input, '\'')
	if opening < 0 {
		if power {
			return "", "", "Psionic powers must be enclosed in the symbols: '\r\n"
		}
		return "", "", "Spell names must be enclosed in the magick symbols: '\r\n"
	}
	remaining := input[opening+1:]
	closing := strings.IndexByte(remaining, '\'')
	if closing < 0 {
		// C's do_cast receives the command interpreter's remainder, which
		// retains the separating space before the opening quote. strtok() sees
		// that space as its first token and accepts the rest of the input as
		// the spell name when there is no closing quote. The target is empty.
		if spellName := strings.TrimSpace(remaining); spellName != "" {
			return spellName, "", ""
		}
		if power {
			return "", "", "Psionic powers must be enclosed in the symbols: '\r\n"
		}
		return "", "", "Spell names must be enclosed in the magick symbols: '\r\n"
	}
	targetName, _ = game.OneArgument(remaining[closing+1:])
	return strings.TrimSpace(remaining[:closing]), targetName, ""
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

	fighting := s.player.GetFighting()
	if fighting != "" && info.HasTarget(spells.TarFightSelf) {
		return castTarget{character: s.player, found: true}, ""
	}
	if fighting != "" && info.HasTarget(spells.TarFightVict) {
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

	spellName, targetName, parseError := parseCastArgumentsForPower(args, power)
	if parseError != "" {
		s.Send(parseError)
		return nil
	}

	spellNum := game.FindSkillNum(spellName)
	if spellNum < 1 || spellNum > maxCastSpell {
		s.Send(castMessage(power, "Cast what?!?\r\n", "Will what?!?\r\n"))
		return nil
	}
	sd := spellDB[spellNum]
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
	canonicalName := strings.ToLower(game.SkillCatalogName(spellNum))
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
	cost := manaCost(sd, casterLevel, s.player.GetClass())

	if cost > 0 && s.player.GetMana() < cost && s.player.GetLevel() < LVL_IMMORT {
		s.Send(castMessage(power, "You haven't the energy to cast that spell!\r\n", "You haven't the energy to will that power!\r\n"))
		return nil
	}

	weightAdd := castWeightPenalty(s.player)
	if s.player.Level >= LVL_IMMORT {
		weightAdd = -20
	}
	// #nosec G404 — game RNG, not cryptographic
	if castNumber(0, 101+weightAdd) > proficiency {
		s.player.SetWaitState(1)
		s.Send("You lost your concentration!\r\n")
		if cost > 0 {
			s.player.SetMana(max(0, s.player.GetMana()-(cost>>1)))
		}
		if info.IsViolent() {
			if mob, ok := target.character.(*game.MobInstance); ok && mob.GetFighting() == "" && s.manager.combatEngine != nil {
				if err := s.manager.combatEngine.StartCombat(mob, s.player); err != nil {
					slog.Warn("cast-failure retaliation failed", "mob", mob.GetName(), "caster", s.player.Name, "error", err)
				}
			}
		}
		return nil
	}

	if !checkCastSpellContract(s, info, target) {
		return nil
	}

	s.Send("Okay.\r\n")
	spells.SaySpell(s.player, spellNum, target.character, target.object, s.manager.world)
	if spells.CallMagic(s.player, target.character, target.object, spellNum, casterLevel, spells.CastSpell, s.manager.world) {
		s.player.SetWaitState(1)
		if cost > 0 {
			s.player.SetMana(max(0, s.player.GetMana()-cost))
		}
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
