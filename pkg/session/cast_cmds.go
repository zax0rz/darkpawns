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

// spellDB maps spell number -> spell data. SpellNum remains explicit because
// manaCost uses it to resolve the class minimum level; the map key is the
// lookup identity.
var spellDB = map[int]*spellData{
	spells.SpellArmor:          {SpellNum: spells.SpellArmor, Name: "armor", ManaMax: 30, ManaMin: 15, ManaChange: 3, MinLevel: [12]int{}},
	spells.SpellTeleport:       {SpellNum: spells.SpellTeleport, Name: "teleport", ManaMax: 60, ManaMin: 50, ManaChange: 3, MinLevel: [12]int{}},
	spells.SpellBless:          {SpellNum: spells.SpellBless, Name: "bless", ManaMax: 36, ManaMin: 10, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellBlindness:      {SpellNum: spells.SpellBlindness, Name: "blindness", ManaMax: 35, ManaMin: 25, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellBurningHands:   {SpellNum: spells.SpellBurningHands, Name: "burning hands", ManaMax: 45, ManaMin: 20, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCallLightning:  {SpellNum: spells.SpellCallLightning, Name: "call lightning", ManaMax: 68, ManaMin: 52, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCharm:          {SpellNum: spells.SpellCharm, Name: "charm", ManaMax: 75, ManaMin: 50, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellChillTouch:     {SpellNum: spells.SpellChillTouch, Name: "chill touch", ManaMax: 35, ManaMin: 15, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellClone:          {SpellNum: spells.SpellClone, Name: "clone", ManaMax: 80, ManaMin: 65, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellColorSpray:     {SpellNum: spells.SpellColorSpray, Name: "color spray", ManaMax: 58, ManaMin: 38, ManaChange: 4, MinLevel: [12]int{}},
	spells.SpellControlWeather: {SpellNum: spells.SpellControlWeather, Name: "control weather", ManaMax: 75, ManaMin: 25, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCreateFood:     {SpellNum: spells.SpellCreateFood, Name: "create food", ManaMax: 35, ManaMin: 10, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCreateWater:    {SpellNum: spells.SpellCreateWater, Name: "create water", ManaMax: 35, ManaMin: 10, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCureBlind:      {SpellNum: spells.SpellCureBlind, Name: "cure blind", ManaMax: 35, ManaMin: 5, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCureCritic:     {SpellNum: spells.SpellCureCritic, Name: "cure critical", ManaMax: 70, ManaMin: 40, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCureLight:      {SpellNum: spells.SpellCureLight, Name: "cure light", ManaMax: 30, ManaMin: 10, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellCurse:          {SpellNum: spells.SpellCurse, Name: "curse", ManaMax: 80, ManaMin: 50, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellDetectAlign:    {SpellNum: spells.SpellDetectAlign, Name: "detect alignment", ManaMax: 20, ManaMin: 10, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellDetectInvis:    {SpellNum: spells.SpellDetectInvis, Name: "detect invis", ManaMax: 20, ManaMin: 10, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellDetectMagic:    {SpellNum: spells.SpellDetectMagic, Name: "detect magic", ManaMax: 20, ManaMin: 10, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellDetectPoison:   {SpellNum: spells.SpellDetectPoison, Name: "detect poison", ManaMax: 20, ManaMin: 10, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellDispelEvil:     {SpellNum: spells.SpellDispelEvil, Name: "dispel evil", ManaMax: 95, ManaMin: 65, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellEarthquake:     {SpellNum: spells.SpellEarthquake, Name: "earthquake", ManaMax: 70, ManaMin: 50, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellEnchantWeapon:  {SpellNum: spells.SpellEnchantWeapon, Name: "enchant weapon", ManaMax: 200, ManaMin: 150, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellEnergyDrain:    {SpellNum: spells.SpellEnergyDrain, Name: "energy drain", ManaMax: 60, ManaMin: 45, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellFireball:       {SpellNum: spells.SpellFireball, Name: "fireball", ManaMax: 70, ManaMin: 50, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellHarm:           {SpellNum: spells.SpellHarm, Name: "harm", ManaMax: 105, ManaMin: 75, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellHeal:           {SpellNum: spells.SpellHeal, Name: "heal", ManaMax: 90, ManaMin: 80, ManaChange: 3, MinLevel: [12]int{}},
	spells.SpellInvisible:      {SpellNum: spells.SpellInvisible, Name: "invisible", ManaMax: 45, ManaMin: 45, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellLightningBolt:  {SpellNum: spells.SpellLightningBolt, Name: "lightning bolt", ManaMax: 54, ManaMin: 34, ManaChange: 4, MinLevel: [12]int{}},
	spells.SpellLocateObject:   {SpellNum: spells.SpellLocateObject, Name: "locate object", ManaMax: 25, ManaMin: 20, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMagicMissile:   {SpellNum: spells.SpellMagicMissile, Name: "flame arrow", ManaMax: 30, ManaMin: 15, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellPoison:         {SpellNum: spells.SpellPoison, Name: "poison", ManaMax: 50, ManaMin: 40, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellProtFromEvil:   {SpellNum: spells.SpellProtFromEvil, Name: "protect evil", ManaMax: 50, ManaMin: 50, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellRemoveCurse:    {SpellNum: spells.SpellRemoveCurse, Name: "remove curse", ManaMax: 45, ManaMin: 45, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSanctuary:      {SpellNum: spells.SpellSanctuary, Name: "sanctuary", ManaMax: 110, ManaMin: 85, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellShockingGrasp:  {SpellNum: spells.SpellShockingGrasp, Name: "shocking grasp", ManaMax: 55, ManaMin: 35, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellSleep:          {SpellNum: spells.SpellSleep, Name: "sleep", ManaMax: 40, ManaMin: 35, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellStrength:       {SpellNum: spells.SpellStrength, Name: "strength", ManaMax: 35, ManaMin: 30, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSummon:         {SpellNum: spells.SpellSummon, Name: "summon", ManaMax: 90, ManaMin: 70, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMeteorSwarm:    {SpellNum: spells.SpellMeteorSwarm, Name: "meteor swarm", ManaMax: 180, ManaMin: 170, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellWordOfRecall:   {SpellNum: spells.SpellWordOfRecall, Name: "recall", ManaMax: 50, ManaMin: 50, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellRemovePoison:   {SpellNum: spells.SpellRemovePoison, Name: "remove poison", ManaMax: 40, ManaMin: 30, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSenseLife:      {SpellNum: spells.SpellSenseLife, Name: "sense life", ManaMax: 30, ManaMin: 20, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellAnimateDead:    {SpellNum: spells.SpellAnimateDead, Name: "animate dead", ManaMax: 120, ManaMin: 100, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellDispelGood:     {SpellNum: spells.SpellDispelGood, Name: "dispel good", ManaMax: 95, ManaMin: 65, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellHolyShield:     {SpellNum: spells.SpellHolyShield, Name: "holy shield", ManaMax: 90, ManaMin: 65, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellGroupHeal:      {SpellNum: spells.SpellGroupHeal, Name: "group heal", ManaMax: 210, ManaMin: 150, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellGroupRecall:    {SpellNum: spells.SpellGroupRecall, Name: "group recall", ManaMax: 155, ManaMin: 125, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellInfravision:    {SpellNum: spells.SpellInfravision, Name: "infravision", ManaMax: 25, ManaMin: 25, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellWaterwalk:      {SpellNum: spells.SpellWaterwalk, Name: "waterwalk", ManaMax: 80, ManaMin: 55, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMassHeal:       {SpellNum: spells.SpellMassHeal, Name: "mass heal", ManaMax: 130, ManaMin: 100, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellFly:            {SpellNum: spells.SpellFly, Name: "fly", ManaMax: 100, ManaMin: 80, ManaChange: 5, MinLevel: [12]int{}},
	// Preserve the pre-keying 54/calliope record; C identifies 54 as lycanthropy.
	spells.SpellLycanthropy:      {SpellNum: spells.SpellLycanthropy, Name: "calliope", ManaMax: 100, ManaMin: 50, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellVampirism:        {SpellNum: spells.SpellVampirism, Name: "vampirism", ManaMax: 1, ManaMin: 1, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSobriety:         {SpellNum: spells.SpellSobriety, Name: "sobriety", ManaMax: 35, ManaMin: 20, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellGroupInvis:       {SpellNum: spells.SpellGroupInvis, Name: "group invis", ManaMax: 135, ManaMin: 135, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellHellfire:         {SpellNum: spells.SpellHellfire, Name: "hellfire", ManaMax: 200, ManaMin: 150, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellEnchantArmor:     {SpellNum: spells.SpellEnchantArmor, Name: "enchant armor", ManaMax: 150, ManaMin: 130, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellIdentify:         {SpellNum: spells.SpellIdentify, Name: "identify", ManaMax: 125, ManaMin: 100, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellMindPoke:         {SpellNum: spells.SpellMindPoke, Name: "mindpoke", ManaMax: 30, ManaMin: 15, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellMindBlast:        {SpellNum: spells.SpellMindBlast, Name: "mindblast", ManaMax: 70, ManaMin: 40, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellChameleon:        {SpellNum: spells.SpellChameleon, Name: "chameleon", ManaMax: 50, ManaMin: 30, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellLevitate:         {SpellNum: spells.SpellLevitate, Name: "levitate", ManaMax: 90, ManaMin: 70, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellMetalskin:        {SpellNum: spells.SpellMetalskin, Name: "metalskin", ManaMax: 75, ManaMin: 60, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellInvulnerability:  {SpellNum: spells.SpellInvulnerability, Name: "invulnerability", ManaMax: 85, ManaMin: 85, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellVitality:         {SpellNum: spells.SpellVitality, Name: "vitality", ManaMax: 110, ManaMin: 100, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellInvigorate:       {SpellNum: spells.SpellInvigorate, Name: "invigorate", ManaMax: 110, ManaMin: 95, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellLessPercept:      {SpellNum: spells.SpellLessPercept, Name: "lesser perception", ManaMax: 40, ManaMin: 30, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellGreatPercept:     {SpellNum: spells.SpellGreatPercept, Name: "greater perception", ManaMax: 65, ManaMin: 45, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMindAttack:       {SpellNum: spells.SpellMindAttack, Name: "mind attack", ManaMax: 55, ManaMin: 25, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellAdrenaline:       {SpellNum: spells.SpellAdrenaline, Name: "adrenaline", ManaMax: 35, ManaMin: 30, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellPsyshield:        {SpellNum: spells.SpellPsyshield, Name: "psyshield", ManaMax: 30, ManaMin: 20, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellChangeDensity:    {SpellNum: spells.SpellChangeDensity, Name: "change density", ManaMax: 70, ManaMin: 55, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellAcidBlast:        {SpellNum: spells.SpellAcidBlast, Name: "acid blast", ManaMax: 35, ManaMin: 20, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellDominate:         {SpellNum: spells.SpellDominate, Name: "dominate", ManaMax: 75, ManaMin: 50, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellCellAdjustment:   {SpellNum: spells.SpellCellAdjustment, Name: "cell adjustment", ManaMax: 85, ManaMin: 75, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellZen:              {SpellNum: spells.SpellZen, Name: "zen", ManaMax: 70, ManaMin: 60, ManaChange: 4, MinLevel: [12]int{}},
	spells.SpellMirrorImage:      {SpellNum: spells.SpellMirrorImage, Name: "mirror image", ManaMax: 150, ManaMin: 130, ManaChange: 5, MinLevel: [12]int{}},
	spells.SpellMassDominate:     {SpellNum: spells.SpellMassDominate, Name: "mass dominate", ManaMax: 220, ManaMin: 150, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellDivineInt:        {SpellNum: spells.SpellDivineInt, Name: "divine int", ManaMax: 290, ManaMin: 290, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMindBar:          {SpellNum: spells.SpellMindBar, Name: "mind bar", ManaMax: 115, ManaMin: 100, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSoulLeech:        {SpellNum: spells.SpellSoulLeech, Name: "soul leech", ManaMax: 60, ManaMin: 55, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMindsight:        {SpellNum: spells.SpellMindsight, Name: "mindsight", ManaMax: 70, ManaMin: 60, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellTransparency:     {SpellNum: spells.SpellTransparency, Name: "transparency", ManaMax: 35, ManaMin: 25, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellKnowAlign:        {SpellNum: spells.SpellKnowAlign, Name: "know alignment", ManaMax: 20, ManaMin: 20, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellGate:             {SpellNum: spells.SpellGate, Name: "gate", ManaMax: 95, ManaMin: 95, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellIntellect:        {SpellNum: spells.SpellIntellect, Name: "intellect", ManaMax: 60, ManaMin: 60, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellLayHands:         {SpellNum: spells.SpellLayHands, Name: "lay hands", ManaMax: 90, ManaMin: 90, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellMentalLapse:      {SpellNum: spells.SpellMentalLapse, Name: "mental lapse", ManaMax: 100, ManaMin: 90, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSmokescreen:      {SpellNum: spells.SpellSmokescreen, Name: "smokescreen", ManaMax: 100, ManaMin: 100, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellDisrupt:          {SpellNum: spells.SpellDisrupt, Name: "disrupt", ManaMax: 175, ManaMin: 165, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellDisintegrate:     {SpellNum: spells.SpellDisintegrate, Name: "disintegrate", ManaMax: 120, ManaMin: 120, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellCalliope:         {SpellNum: spells.SpellCalliope, Name: "calliope", ManaMax: 100, ManaMin: 50, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellProtFromGood:     {SpellNum: spells.SpellProtFromGood, Name: "protect good", ManaMax: 50, ManaMin: 50, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellFlamestrike:      {SpellNum: spells.SpellFlamestrike, Name: "flamestrike", ManaMax: 105, ManaMin: 100, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellHaste:            {SpellNum: spells.SpellHaste, Name: "haste", ManaMax: 140, ManaMin: 140, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellSlow:             {SpellNum: spells.SpellSlow, Name: "slow", ManaMax: 80, ManaMin: 50, ManaChange: 2, MinLevel: [12]int{}},
	spells.SpellDreamTravel:      {SpellNum: spells.SpellDreamTravel, Name: "dream travel", ManaMax: 60, ManaMin: 45, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellPsiblast:         {SpellNum: spells.SpellPsiblast, Name: "psiblast", ManaMax: 180, ManaMin: 150, ManaChange: 10, MinLevel: [12]int{}},
	spells.SpellCoC:              {SpellNum: spells.SpellCoC, Name: "call of chaos", ManaMax: 90, ManaMin: 70, ManaChange: 1, MinLevel: [12]int{}},
	spells.SpellWaterBreathe:     {SpellNum: spells.SpellWaterBreathe, Name: "water breathe", ManaMax: 92, ManaMin: 58, ManaChange: 6, MinLevel: [12]int{}},
	spells.SpellConjureElemental: {SpellNum: spells.SpellConjureElemental, Name: "conjure elemental", ManaMax: 165, ManaMin: 145, ManaChange: 1, MinLevel: [12]int{}},
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
