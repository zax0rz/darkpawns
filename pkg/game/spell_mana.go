package game

import (
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// One mana formula for the whole port, as C has one: mag_manacost
// (src/spell_parser.c:257-266) reads the spello() table (src/spell_parser.c:
// 1148-1162) and is called by both cast (src/spell_parser.c:1071) and the
// practice catalog (src/spec_procs.c:185). The table and formula lived in
// pkg/session until the practice catalog needed them; they moved here whole
// so cast and practice cannot drift onto different costs. C's skillo()
// (src/spell_parser.c:1180) defines skills through spello with zero mana, and
// spell_info[] is zero-initialized above the defined range, so a number with
// no entry here costs 0 — mag_manacost returns MAX(0, 0).

// SpellManaData holds one spell's mana parameters from the original spello()
// table. MinLevel carries the spello() per-class minimum-level defaults,
// which ClassSkillMinLevel normally supersedes (it mirrors init_spell_levels
// plus spello's LVL_IMMORT default, src/spell_parser.c:1153-1154).
type SpellManaData struct {
	SpellNum   int
	Name       string
	ManaMax    int
	ManaMin    int
	ManaChange int
	MinLevel   [12]int // 12 classes, indexed by Class constant
}

// SpellManaTable maps spell number -> spell data (the port's mirror of C's
// spell_info[] mana fields, keyed by the C spell number). SpellNum remains
// explicit in each record because it names the C number the key must equal;
// the map key is the lookup identity.
var SpellManaTable = map[int]*SpellManaData{
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

// ManaCost computes the mana cost for a spell at a given caster level and
// class. Formula (src/spell_parser.c:261-263): MAX(mana_max -
// (mana_change * (caster_level - min_level)), mana_min)
func ManaCost(spellNum int, class int, casterLevel int) int {
	sd := SpellManaTable[spellNum]
	if sd == nil {
		return 0
	}
	minLvl := ClassSkillMinLevel(class, sd.SpellNum)
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
