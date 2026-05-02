package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

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
	32:  {32, "magic missile", 30, 15, 5, [12]int{}},
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
	46:  {46, "dispel good", 95, 65, 5, [12]int{}},
	47:  {47, "holy shield", 90, 65, 5, [12]int{}},
	48:  {48, "group heal", 210, 150, 5, [12]int{}},
	49:  {49, "group recall", 155, 125, 5, [12]int{}},
	50:  {50, "infravision", 25, 25, 1, [12]int{}},
	51:  {51, "waterwalk", 80, 55, 1, [12]int{}},
	52:  {52, "mass heal", 130, 100, 1, [12]int{}},
	53:  {53, "fly", 100, 80, 5, [12]int{}},
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
	82:  {82, "mind bar", 115, 100, 1, [12]int{}},
	94:  {94, "soul leech", 60, 55, 1, [12]int{}},
	96:  {96, "flamestrike", 105, 100, 1, [12]int{}},
	97:  {97, "haste", 140, 140, 1, [12]int{}},
	98:  {98, "slow", 80, 50, 2, [12]int{}},
	99:  {99, "dream travel", 60, 45, 1, [12]int{}},
	100: {100, "psiblast", 180, 150, 10, [12]int{}},
	101: {101, "call of chaos", 90, 70, 1, [12]int{}},
	102: {102, "water breathe", 92, 58, 6, [12]int{}},
	105: {105, "conjure elemental", 165, 145, 1, [12]int{}},
	// Additional spells from class.c references
	// mind sight
	93: {93, "mindsight", 70, 60, 1, [12]int{}},
	// mass dominate — discovered no. from class.c
	80: {80, "mass dominate", 220, 150, 10, [12]int{}},
	// calliope
	54: {54, "calliope", 100, 50, 10, [12]int{}},
	// protect good (SPELL_PROT_FROM_GOOD 45 duped with dispel good 46 — use 45)
	45: {45, "protect good", 50, 50, 1, [12]int{}},
}

// buildSpellIndex builds a case-insensitive lookup from spell name -> spellData.
var spellByName = func() map[string]*spellData {
	m := make(map[string]*spellData)
	for _, sd := range spellDB {
		m[strings.ToLower(sd.Name)] = sd
	}
	return m
}()

// manaCost computes the mana cost for a spell at a given caster level and class.
// Formula: MAX(mana_max - (mana_change * (caster_level - min_level)), mana_min)
func manaCost(sd *spellData, casterLevel int, class int) int {
	minLvl := sd.MinLevel[class]
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

// cmdCast handles the "cast <spell> [target]" command.
// Implements do_cast from cast.c / spell_parser.c.
func cmdCast(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Cast which spell?")
		return nil
	}

	fullInput := strings.Join(args, " ")

	// Parse spell name and target.
	// Support: cast <spell> and cast '<spell>' <target>
	var spellName string
	var targetName string

	if strings.HasPrefix(fullInput, "'") {
		// Quoted spell name: cast '<spell>' <target>
		endQuote := strings.Index(fullInput[1:], "'")
		if endQuote == -1 {
			s.Send("Cast which spell?")
			return nil
		}
		spellName = fullInput[1 : endQuote+1]
		targetName = strings.TrimSpace(fullInput[endQuote+2:])
	} else {
		// No quotes: cast <spell> or cast <spell> <target>
		parts := strings.SplitN(fullInput, " ", 2)
		spellName = parts[0]
		if len(parts) > 1 {
			targetName = strings.TrimSpace(parts[1])
		}
	}

	spellName = strings.ToLower(spellName)

	// Look up spell
	sd, ok := spellByName[spellName]
	if !ok {
		s.Send("You don't know any spells of that name.")
		return nil
	}

	// Check player knows the spell via SpellMap
	_, knows := s.player.SpellMap[spellName]
	if !knows {
		s.Send(fmt.Sprintf("You don't know '%s'.", sd.Name))
		return nil
	}

	// Determine caster level: minimum of player level and class min level
	class := s.player.Class
	playerLevel := s.player.Level
	casterLevel := playerLevel

	// Calculate mana cost
	cost := manaCost(sd, casterLevel, class)

	// Check mana
	if s.player.Mana < cost {
		s.Send(fmt.Sprintf("You don't have enough mana to cast '%s'. You need %d mana.", sd.Name, cost))
		return nil
	}

	// Deduct mana
	s.player.Mana -= cost

	// Determine target
	var target interface{}

	if targetName == "" || strings.EqualFold(targetName, s.player.Name) || strings.EqualFold(targetName, "self") || strings.EqualFold(targetName, "me") {
		// Self-cast
		target = s.player
	} else {
		// Find target in room — check mobs first, then players
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			s.Send("You are in a strange void.")
			return nil
		}

		// Check mobs in room
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), strings.ToLower(targetName)) ||
				strings.Contains(strings.ToLower(mob.GetName()), strings.ToLower(targetName)) {
				target = mob
				break
			}
		}

		// Check players if no mob found
		if target == nil {
			players := s.manager.world.GetPlayersInRoom(room.VNum)
			for _, p := range players {
				if strings.EqualFold(p.Name, targetName) {
					target = p
					break
				}
			}
		}

		if target == nil {
			s.Send("They aren't here.")
			// Refund mana on failed targeting
			s.player.Mana += cost
			if s.player.Mana > s.player.MaxMana {
				s.player.Mana = s.player.MaxMana
			}
			return nil
		}
	}

	// Execute the spell
	am := engine.NewAffectManager()
	spells.Cast(s.player, target, sd.SpellNum, casterLevel, am)

	// Send confirmation
	if target == s.player {
		s.Send(fmt.Sprintf("You cast '%s' on yourself.", sd.Name))
	} else {
		// Get target name for display
		targetDisplay := targetName
		if t, ok := target.(*game.Player); ok {
			targetDisplay = t.Name
		} else if t, ok := target.(interface{ GetShortDesc() string }); ok {
			targetDisplay = t.GetShortDesc()
		}
		s.Send(fmt.Sprintf("You cast '%s' on %s.", sd.Name, targetDisplay))

		// Notify target if it's a player
		if t, ok := target.(*game.Player); ok {
			if targetSession, ok := s.manager.GetSession(t.Name); ok {
				targetSession.Send(fmt.Sprintf("%s casts '%s' on you.", s.player.Name, sd.Name))
			}
		}
	}

	return nil
}

func init() {
	// Register the cast command with aliases
	cmdRegistry.Register("cast", wrapArgs(cmdCast), "Cast a spell.", 0, 0)
}
