package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func seedForCastRoll(t *testing.T, accept func(int) bool) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		if accept(dprng.New(seed).Number(0, 101)) {
			return seed
		}
	}
	t.Fatal("could not find cast-roll seed")
	return 0
}

func prepareCaster(s *Session, class int, spellName string, proficiency int) {
	s.player.Class = class
	s.player.Stats.Int = 10
	s.player.Stats.Wis = 10
	s.player.SetLevel(1)
	s.player.SetMana(100)
	s.player.SetSkill(spellName, proficiency)
}

func captureCastDraws(t *testing.T) *int {
	t.Helper()
	draws := 0
	original := castNumber
	castNumber = func(minVal, maxVal int) int {
		draws++
		return original(minVal, maxVal)
	}
	t.Cleanup(func() { castNumber = original })
	return &draws
}

func TestCastEarlyGatesAreExactAndDrawFree(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		peaceful bool
		prepare  func(*Session)
		want     string
	}{
		{
			name: "empty command",
			want: "Cast what where?\r\n",
		},
		{
			name: "unquoted spell",
			args: []string{"flame", "arrow"},
			want: "Spell names must be enclosed in the magick symbols: '\r\n",
		},
		{
			name: "missing closing quote is accepted by C tokenizer",
			args: []string{"'flame", "arrow"},
			prepare: func(s *Session) {
				prepareCaster(s, game.ClassMageUser, "flame arrow", 95)
			},
			want: "Upon who should the spell be cast?\r\n",
		},
		{
			name: "unknown spell",
			args: []string{"'bogusspell'"},
			want: "Cast what?!?\r\n",
		},
		{
			name: "class level gate",
			args: []string{"'cure", "light'"},
			prepare: func(s *Session) {
				prepareCaster(s, game.ClassMageUser, "cure light", 95)
			},
			want: "You do not know that spell!\r\n",
		},
		{
			name: "zero proficiency gate",
			args: []string{"'flame", "arrow'"},
			prepare: func(s *Session) {
				prepareCaster(s, game.ClassMageUser, "flame arrow", 0)
			},
			want: "You are unfamiliar with that spell.\r\n",
		},
		{
			name: "peaceful violent gate",
			args: []string{"'flame", "arrow'"},
			prepare: func(s *Session) {
				prepareCaster(s, game.ClassMageUser, "flame arrow", 95)
			},
			peaceful: true,
			want:     "This room just has such a peaceful, easy feeling..\r\n",
		},
		{
			name: "empty violent target",
			args: []string{"'flame", "arrow'"},
			prepare: func(s *Session) {
				prepareCaster(s, game.ClassMageUser, "flame arrow", 95)
			},
			want: "Upon who should the spell be cast?\r\n",
		},
		{
			name: "violent self by name",
			args: []string{"'flame", "arrow'", "Caster"},
			prepare: func(s *Session) {
				prepareCaster(s, game.ClassMageUser, "flame arrow", 95)
			},
			want: "You shouldn't cast that on yourself -- could be bad for your health!\r\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var manager *Manager
			if test.peaceful {
				manager = makeGateTestManager(t, true)
			} else {
				manager = makeTestManager(t)
			}
			session := makeTestSession(t, manager, "Caster", 1001, true)
			registerInWorld(t, session)
			session.player.SetMana(100)
			if test.prepare != nil {
				test.prepare(session)
			}
			manaBefore := session.player.GetMana()
			draws := captureCastDraws(t)

			if err := cmdCast(session, test.args); err != nil {
				t.Fatalf("cmdCast: %v", err)
			}
			if got := readSessionText(t, session); got != test.want {
				t.Errorf("output = %q, want %q", got, test.want)
			}
			if *draws != 0 {
				t.Errorf("cast draws = %d, want 0", *draws)
			}
			if got := session.player.GetMana(); got != manaBefore {
				t.Errorf("mana = %d, want unchanged %d", got, manaBefore)
			}
			if got := session.player.GetWaitState(); got != 0 {
				t.Errorf("wait = %d, want 0", got)
			}
		})
	}
}

func TestResolveCastTargetOrderAndEmptyDefaults(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Caster Room", Zone: 1},
			{VNum: 1002, Name: "World Room", Zone: 1},
		},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target",
			ShortDesc: "a distant target",
			Level:     1,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 10},
		}},
		Objs: []parser.Obj{
			{VNum: 6001, Keywords: "inventory focus", ShortDesc: "an inventory focus"},
			{VNum: 6002, Keywords: "equipped focus", ShortDesc: "an equipped focus"},
			{VNum: 6003, Keywords: "room focus", ShortDesc: "a room focus"},
			{VNum: 6004, Keywords: "world focus", ShortDesc: "a world focus"},
		},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })
	manager := newTestManager(t, world, nil)
	caster := makeTestSession(t, manager, "Caster", 1001, true)
	roomTarget := makeTestSession(t, manager, "Target", 1001, true)
	registerInWorld(t, caster)
	registerInWorld(t, roomTarget)
	worldTarget, err := world.SpawnMob(5000, 1002)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	roomAndWorld := &spells.SpellInfo{Routines: spells.SpellRoutines{
		Targets: spells.TarCharRoom | spells.TarCharWorld,
	}}
	target, prompt := resolveCastTarget(caster, roomAndWorld, "target")
	if prompt != "" {
		t.Fatalf("room/world prompt = %q, want empty", prompt)
	}
	if !target.found || target.character != roomTarget.player {
		t.Fatalf("room/world target = %#v, want in-room player", target.character)
	}

	nonviolent := &spells.SpellInfo{Routines: spells.SpellRoutines{Targets: spells.TarCharRoom}}
	target, prompt = resolveCastTarget(caster, nonviolent, "")
	if prompt != "" || !target.found || target.character != caster.player {
		t.Fatalf("empty nonviolent target = %#v, prompt %q; want caster", target.character, prompt)
	}

	violent := &spells.SpellInfo{Routines: spells.SpellRoutines{
		Targets: spells.TarCharRoom,
		Violent: true,
	}}
	target, prompt = resolveCastTarget(caster, violent, "")
	if target.found || prompt != "Upon who should the spell be cast?\r\n" {
		t.Fatalf("empty violent target found=%v prompt=%q", target.found, prompt)
	}

	worldTarget.SetRoom(1001)
	caster.player.SetFighting(worldTarget.GetName())
	fightVictim := &spells.SpellInfo{Routines: spells.SpellRoutines{
		Targets: spells.TarFightVict,
		Violent: true,
	}}
	target, prompt = resolveCastTarget(caster, fightVictim, "")
	if prompt != "" || !target.found || target.character != worldTarget {
		t.Fatalf("fighting target = %#v, prompt %q; want mob opponent", target.character, prompt)
	}

	caster.player.SetHolyLight(true)
	inventoryObject, err := world.SpawnObject(6001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject inventory: %v", err)
	}
	if err := caster.player.Inventory.AddItem(inventoryObject); err != nil {
		t.Fatalf("AddItem inventory: %v", err)
	}
	equippedObject, err := world.SpawnObject(6002, 1001)
	if err != nil {
		t.Fatalf("SpawnObject equipped: %v", err)
	}
	if err := caster.player.Equipment.SetSlot(game.SlotHead, equippedObject); err != nil {
		t.Fatalf("SetSlot equipped: %v", err)
	}
	roomObject, err := world.SpawnObject(6003, 1001)
	if err != nil {
		t.Fatalf("SpawnObject room: %v", err)
	}
	world.AddItemToRoom(roomObject, 1001)
	worldObject, err := world.SpawnObject(6004, 1002)
	if err != nil {
		t.Fatalf("SpawnObject world: %v", err)
	}

	objectScopes := []struct {
		name   string
		flag   spells.TargetFlags
		query  string
		object *game.ObjectInstance
	}{
		{"inventory", spells.TarObjInv, "inventory", inventoryObject},
		{"equipment", spells.TarObjEquip, "equipped", equippedObject},
		{"room", spells.TarObjRoom, "room", roomObject},
		{"world", spells.TarObjWorld, "world", worldObject},
	}
	for _, scope := range objectScopes {
		info := &spells.SpellInfo{Routines: spells.SpellRoutines{Targets: scope.flag}}
		target, prompt = resolveCastTarget(caster, info, scope.query)
		if prompt != "" || !target.found || target.object != scope.object {
			t.Errorf("%s object target = %#v, prompt %q; want %#v", scope.name, target.object, prompt, scope.object)
		}
	}

	allObjectScopes := &spells.SpellInfo{Routines: spells.SpellRoutines{Targets: spells.TarObjInv |
		spells.TarObjEquip | spells.TarObjRoom | spells.TarObjWorld}}
	target, prompt = resolveCastTarget(caster, allObjectScopes, "focus")
	if prompt != "" || !target.found || target.object != inventoryObject {
		t.Fatalf("object scope order target = %#v, prompt %q; want inventory object", target.object, prompt)
	}
}

func TestCastMissingTargetPreservesOkayIncantationEdge(t *testing.T) {
	manager := makeTestManager(t)
	session := makeTestSession(t, manager, "Caster", 1001, true)
	registerInWorld(t, session)
	prepareCaster(session, game.ClassMageUser, "flame arrow", 95)
	draws := captureCastDraws(t)

	if err := cmdCast(session, []string{"'flame", "arrow'", "missing"}); err != nil {
		t.Fatalf("cmdCast: %v", err)
	}
	if got := readSessionText(t, session); got != "Okay.\r\n" {
		t.Errorf("first output = %q, want Okay", got)
	}
	if got := readSessionText(t, session); got != "Cannot find the target of your spell!\r\n" {
		t.Errorf("second output = %q", got)
	}
	if *draws != 0 {
		t.Errorf("cast draws = %d, want 0", *draws)
	}
	if got := session.player.GetMana(); got != 100 {
		t.Errorf("mana = %d, want 100", got)
	}
	if got := session.player.GetWaitState(); got != 0 {
		t.Errorf("wait = %d, want 0", got)
	}
}

func TestCastArgumentParsingMatchesCQuoteAndWordRules(t *testing.T) {
	spell, target, errMessage := parseCastArguments([]string{"'strength'", "Casttarget", "extra"})
	if errMessage != "" {
		t.Fatalf("parse error = %q, want none", errMessage)
	}
	if spell != "strength" || target != "casttarget" {
		t.Fatalf("parsed spell/target = %q/%q, want strength/casttarget", spell, target)
	}

	spell, target, errMessage = parseCastArguments([]string{"'flame", "arrow"})
	if errMessage != "" || spell != "flame arrow" || target != "" {
		t.Fatalf("unclosed quote parse = %q/%q/%q, want flame arrow/empty/no error", spell, target, errMessage)
	}
}

func TestWillEntryUsesPowerSurface(t *testing.T) {
	if _, ok := cmdRegistry.Lookup("will"); !ok {
		t.Fatal("will command is not registered")
	}

	for _, class := range []int{game.ClassPsionic, game.ClassMystic} {
		t.Run(game.ClassNames[class], func(t *testing.T) {
			s := makeTestSession(t, makeTestManager(t), "Caster", 1001, true)
			s.player.Class = class

			if err := cmdWill(s, nil); err != nil {
				t.Fatalf("cmdWill empty: %v", err)
			}
			if got := readSessionText(t, s); got != "Will what?\r\n" {
				t.Fatalf("empty output = %q", got)
			}

			if err := cmdWill(s, []string{"flame", "arrow"}); err != nil {
				t.Fatalf("cmdWill unquoted: %v", err)
			}
			if got := readSessionText(t, s); got != "Psionic powers must be enclosed in the symbols: '\r\n" {
				t.Fatalf("unquoted output = %q", got)
			}

			if err := cmdWill(s, []string{"'frobnicate'"}); err != nil {
				t.Fatalf("cmdWill unknown: %v", err)
			}
			if got := readSessionText(t, s); got != "Will what?!?\r\n" {
				t.Fatalf("unknown output = %q", got)
			}

			prepareCaster(s, class, "mind poke", 95)
			if err := cmdWill(s, []string{"'mind", "poke'"}); err != nil {
				t.Fatalf("cmdWill known power: %v", err)
			}
			if got := readSessionText(t, s); got != "Upon who should the power be willed?\r\n" {
				t.Fatalf("known power output = %q", got)
			}
		})
	}
}

func TestPsionicCastRejectsLiteralCast(t *testing.T) {
	s := makeTestSession(t, makeTestManager(t), "Psion", 1001, true)
	s.player.Class = game.ClassPsionic
	if err := cmdCast(s, nil); err != nil {
		t.Fatalf("cmdCast: %v", err)
	}
	if got := readSessionText(t, s); got != "Psionics 'will' things, not 'cast' them!\r\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestCastSpellContractSmartnessGate(t *testing.T) {
	s := makeTestSession(t, makeTestManager(t), "Caster", 1001, true)
	prepareCaster(s, game.ClassCleric, "cure light", 100)
	s.player.Stats.Int = 0

	original := castNumber
	castNumber = func(minVal, maxVal int) int { return minVal }
	t.Cleanup(func() { castNumber = original })

	if err := cmdCast(s, []string{"'cure", "light'"}); err != nil {
		t.Fatalf("cmdCast: %v", err)
	}
	if got := readSessionText(t, s); got != "You're not smart enough to cast!\r\n" {
		t.Fatalf("output = %q", got)
	}
	if got := s.player.GetMana(); got != 100 {
		t.Errorf("mana = %d, want unchanged 100", got)
	}
	if got := s.player.GetWaitState(); got != 0 {
		t.Errorf("wait = %d, want unchanged 0", got)
	}
}

func TestCastSpellContractUsesPowerGates(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Caster", 1001, true)
	prepareCaster(s, game.ClassMystic, "mindpoke", 100)
	master := game.NewPlayer(2, "Master", 1001)

	tests := []struct {
		name   string
		info   *spells.SpellInfo
		setup  func()
		target castTarget
		want   string
	}{
		{
			name:   "self only",
			info:   &spells.SpellInfo{Routines: spells.SpellRoutines{Targets: spells.TarSelfOnly}},
			target: castTarget{character: master, found: true},
			want:   "You can only will this power upon yourself!\r\n",
		},
		{
			name:   "not self",
			info:   &spells.SpellInfo{Routines: spells.SpellRoutines{Targets: spells.TarNotSelf}},
			target: castTarget{character: s.player, found: true},
			want:   "You cannot will this power upon yourself!\r\n",
		},
		{
			name:   "not grouped",
			info:   &spells.SpellInfo{Routines: spells.SpellRoutines{Routines: spells.RoutineGroups}},
			target: castTarget{character: s.player, found: true},
			want:   "You cannot use this power if you are not in a group!\r\n",
		},
		{
			name: "charmed master",
			setup: func() {
				s.player.SetAffect(game.AffCharm, true)
				s.player.SetFollowing(master.Name)
			},
			info:   &spells.SpellInfo{},
			target: castTarget{character: master, found: true},
			want:   "You are afraid you might hurt your master!\r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s.player.SetAffect(game.AffCharm, false)
			s.player.SetFollowing("")
			if tc.setup != nil {
				tc.setup()
			}
			if ok := checkCastSpellContract(s, tc.info, tc.target); ok {
				t.Fatal("checkCastSpellContract returned true, want false")
			}
			if got := readSessionText(t, s); got != tc.want {
				t.Fatalf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCastWeightPenaltyMatchesCThresholds(t *testing.T) {
	tests := []struct {
		name   string
		weight int
		want   int
	}{
		{name: "empty", want: 0},
		{name: "ratio at least four", weight: 70, want: 0},
		{name: "ratio three", weight: 93, want: 5},
		{name: "ratio two", weight: 140, want: 7},
		{name: "ratio one", weight: 280, want: 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := game.NewPlayer(1, "Caster", 1001)
			p.Stats.Str = 18
			p.Stats.StrAdd = 50
			p.Inventory.SetCapacity(p.GetStr(), p.GetStrAdd(), p.GetDex(), p.GetLevel())
			if tc.weight > 0 {
				weight := tc.weight
				if err := p.Inventory.AddItem(&game.ObjectInstance{WeightOverride: &weight}); err != nil {
					t.Fatalf("AddItem: %v", err)
				}
			}
			if got := castWeightPenalty(p); got != tc.want {
				t.Errorf("castWeightPenalty(%d) = %d, want %d", tc.weight, got, tc.want)
			}
		})
	}
}

func TestCastInsufficientManaDrawsNothingAndSpendsNothing(t *testing.T) {
	manager := makeTestManager(t)
	session := makeTestSession(t, manager, "Cleric", 1001, true)
	registerInWorld(t, session)
	prepareCaster(session, game.ClassCleric, "cure light", 95)
	session.player.SetMana(29)
	draws := captureCastDraws(t)

	if err := cmdCast(session, []string{"'cure", "light'"}); err != nil {
		t.Fatalf("cmdCast: %v", err)
	}
	if got := readSessionText(t, session); got != "You haven't the energy to cast that spell!\r\n" {
		t.Errorf("output = %q", got)
	}
	if *draws != 0 {
		t.Errorf("cast draws = %d, want 0", *draws)
	}
	if got := session.player.GetMana(); got != 29 {
		t.Errorf("mana = %d, want unchanged 29", got)
	}
	if got := session.player.GetWaitState(); got != 0 {
		t.Errorf("wait = %d, want 0", got)
	}
}

func TestManaCostUsesClassMinimumLevel(t *testing.T) {
	if got := manaCost(spellDB[spells.SpellMagicMissile], 1, game.ClassMageUser); got != 30 {
		t.Errorf("L1 flame arrow mana = %d, want 30", got)
	}
	if got := manaCost(spellDB[spells.SpellCureLight], 1, game.ClassCleric); got != 30 {
		t.Errorf("L1 cure light mana = %d, want 30", got)
	}
	if got := manaCost(spellDB[spells.SpellInfravision], 1, game.ClassMageUser); got != 25 {
		t.Errorf("L1 infravision mana = %d, want 25", got)
	}
}

func TestCastCureLightDrawOrderManaWaitAndFullHPCap(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Cleric", 1001, true)
	registerInWorld(t, s)
	prepareCaster(s, game.ClassCleric, "cure light", 95)
	s.player.SetHP(s.player.GetMaxHP())

	seed := seedForCastRoll(t, func(roll int) bool { return roll <= 95 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101) // concentration
	wantStream.Number(1, 8)   // cure-light die 1
	wantStream.Number(1, 8)   // cure-light die 2
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	if err := cmdCast(s, []string{"'cure", "light'"}); err != nil {
		t.Fatalf("cmdCast cure light: %v", err)
	}
	if got := readSessionText(t, s); got != "Okay.\r\n" {
		t.Errorf("first output = %q, want Okay", got)
	}
	if got := readSessionText(t, s); got != "You feel better.\r\n" {
		t.Errorf("effect output = %q", got)
	}
	if got := s.player.GetHP(); got != s.player.GetMaxHP() {
		t.Errorf("full-HP cure changed HP to %d/%d", got, s.player.GetMaxHP())
	}
	if got := s.player.GetMana(); got != 70 {
		t.Errorf("mana = %d, want 70", got)
	}
	if got := s.player.GetWaitState(); got != engine.PULSE_VIOLENCE {
		t.Errorf("wait = %d, want %d (1 round → PULSE_VIOLENCE pulses)", got, engine.PULSE_VIOLENCE)
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after concentration + 2d8", got, wantNext)
	}
}

func TestCastInfravisionConsumesOnlyConcentrationAndAccumulates(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Mage", 1001, true)
	registerInWorld(t, s)
	prepareCaster(s, game.ClassMageUser, "infravision", 95)

	seed := seedForCastRoll(t, func(roll int) bool { return roll <= 95 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101)
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	if err := cmdCast(s, []string{"'infravision'"}); err != nil {
		t.Fatalf("cmdCast infravision: %v", err)
	}
	if got := readSessionText(t, s); got != "Okay.\r\n" {
		t.Errorf("first output = %q", got)
	}
	if got := readSessionText(t, s); got != "Your eyes glow red.\r\n" {
		t.Errorf("effect output = %q", got)
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after concentration only", got, wantNext)
	}
	if len(s.player.ActiveAffects) != 1 || s.player.ActiveAffects[0].Duration != 13 {
		t.Fatalf("first infravision affect = %+v, want one duration-13 affect", s.player.ActiveAffects)
	}

	// Direct command invocation bypasses the dispatcher wait gate so the recast
	// can exercise C affect_join(accum_duration=true).
	dprng.ResetStream(seed)
	if err := cmdCast(s, []string{"'infravision'"}); err != nil {
		t.Fatalf("recast infravision: %v", err)
	}
	if got := readSessionText(t, s); got != "Okay.\r\n" {
		t.Errorf("recast first output = %q", got)
	}
	if got := readSessionText(t, s); got != "Your eyes glow red.\r\n" {
		t.Errorf("recast effect output = %q", got)
	}
	if len(s.player.ActiveAffects) != 1 || s.player.ActiveAffects[0].Duration != 26 {
		t.Fatalf("recast infravision affect = %+v, want one duration-26 affect", s.player.ActiveAffects)
	}
	if got := s.player.GetMana(); got != 50 {
		t.Errorf("mana after two casts = %d, want 50", got)
	}
}

func TestCastFlameArrowDrawOrderIncantationAndComponentMessage(t *testing.T) {
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	caster := makeGateSession(t, m, 1, "Caster", 1)
	observer := makeGateSession(t, m, 2, "Observer", 1)
	prepareCaster(caster, game.ClassMageUser, "flame arrow", 95)

	seed := seedForCastRoll(t, func(roll int) bool { return roll <= 95 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101) // concentration
	for range 4 {
		wantStream.Number(1, 3)
	}
	wantStream.Number(0, 99) // target saving throw
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	beforeHP := mob.GetHP()
	if err := cmdCast(caster, []string{"'flame", "arrow'", "target"}); err != nil {
		t.Fatalf("cmdCast flame arrow: %v", err)
	}
	if got := readSessionText(t, caster); got != "Okay.\r\n" {
		t.Errorf("first output = %q", got)
	}
	if got := readSessionText(t, caster); got != "You attempt the spell without the components..\r\n" {
		t.Errorf("component output = %q", got)
	}
	incantation := readSessionText(t, observer)
	if !strings.Contains(incantation, "Caster stares at a test target") || !strings.Contains(incantation, "'flame arrow'") {
		t.Errorf("observer incantation = %q", incantation)
	}
	if mob.GetHP() >= beforeHP {
		t.Errorf("flame arrow did not damage mob: %d -> %d", beforeHP, mob.GetHP())
	}
	if got := caster.player.GetMana(); got != 70 {
		t.Errorf("mana = %d, want 70", got)
	}
	if got := caster.player.GetWaitState(); got != engine.PULSE_VIOLENCE {
		t.Errorf("wait = %d, want %d (1 round → PULSE_VIOLENCE pulses)", got, engine.PULSE_VIOLENCE)
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after concentration + 4d3 + save", got, wantNext)
	}
}

func TestCastFailureConsumesOneDrawHalfManaAndNoIncantation(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Cleric", 1001, true)
	observer := makeTestSession(t, m, "Observer", 1001, true)
	registerInWorld(t, s)
	registerInWorld(t, observer)
	prepareCaster(s, game.ClassCleric, "cure light", 1)

	seed := seedForCastRoll(t, func(roll int) bool { return roll > 1 })
	wantStream := dprng.New(seed)
	wantStream.Number(0, 101)
	wantNext := wantStream.Number(0, 101)
	dprng.ResetStream(seed)

	if err := cmdCast(s, []string{"'cure", "light'"}); err != nil {
		t.Fatalf("cmdCast failure: %v", err)
	}
	if got := readSessionText(t, s); got != "You lost your concentration!\r\n" {
		t.Errorf("failure output = %q", got)
	}
	if got := s.player.GetMana(); got != 85 {
		t.Errorf("failure mana = %d, want 85", got)
	}
	if got := s.player.GetWaitState(); got != engine.PULSE_VIOLENCE {
		t.Errorf("failure wait = %d, want %d (1 round → PULSE_VIOLENCE pulses)", got, engine.PULSE_VIOLENCE)
	}
	select {
	case msg := <-observer.send:
		t.Fatalf("observer received incantation on failed cast: %s", msg)
	default:
	}
	if got := dprng.Number(0, 101); got != wantNext {
		t.Errorf("next RNG = %d, want %d after failure draw", got, wantNext)
	}
}
