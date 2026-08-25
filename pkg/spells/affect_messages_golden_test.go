package spells

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// affectMsgChar records messages and satisfies every interface magAffectsApply
// and its message helpers require for PC-shaped actors.
type affectMsgChar struct {
	name      string
	sex       int
	level     int
	class     int
	room      int
	alignment int
	npc       bool
	mobFlags  uint64
	position  int
	affects   []*engine.Affect
	messages  []string
}

func (m *affectMsgChar) SendMessage(msg string)   { m.messages = append(m.messages, msg) }
func (m *affectMsgChar) GetName() string          { return m.name }
func (m *affectMsgChar) GetSex() int              { return m.sex }
func (m *affectMsgChar) GetLevel() int            { return m.level }
func (m *affectMsgChar) GetClass() int            { return m.class }
func (m *affectMsgChar) GetRoomVNum() int         { return m.room }
func (m *affectMsgChar) GetAlignment() int        { return m.alignment }
func (m *affectMsgChar) IsNPC() bool              { return m.npc }
func (m *affectMsgChar) HasMobFlag(f uint64) bool { return m.mobFlags&f != 0 }
func (m *affectMsgChar) GetPosition() int         { return m.position }
func (m *affectMsgChar) SetPosition(pos int)      { m.position = pos }
func (m *affectMsgChar) AddAffect(a *engine.Affect) {
	if a != nil {
		m.affects = append(m.affects, a)
	}
}

func (m *affectMsgChar) JoinAffect(a *engine.Affect, addDuration, addMagnitude bool) {
	m.AddAffect(a)
}

// affectMsgWorld is a roomIterable world that captures room-audience sends.
type affectMsgWorld struct {
	rooms map[int][]interface{}
}

func (w *affectMsgWorld) ForEachPlayerInRoomInterface(room int, fn func(interface{})) {
	for _, p := range w.rooms[room] {
		fn(p)
	}
}

func (w *affectMsgWorld) ForEachMobInRoomInterface(room int, fn func(interface{})) {}

func newAffectMsgWorld(chars ...*affectMsgChar) *affectMsgWorld {
	w := &affectMsgWorld{rooms: map[int][]interface{}{}}
	for _, c := range chars {
		w.rooms[c.room] = append(w.rooms[c.room], c)
	}
	return w
}

func newAffectMale(name string, room int) *affectMsgChar {
	return &affectMsgChar{name: name, sex: 0, level: 20, class: 0, room: room, position: 8}
}

// TestMagAffectsMessages_SelfTarget proves the to_vict bytes C mag_affects
// sends when the drinker/caster is also the victim (quaff/recite-self shape:
// to_self never fires). Source of truth: src/magic.c per-spell to_vict strings.
func TestMagAffectsMessages_SelfTarget(t *testing.T) {
	cases := []struct {
		name     string
		spellNum int
		level    int
		want     []string
	}{
		{"chill touch", SpellChillTouch, 20, []string{"You feel your strength wither!\r\n"}},
		{"bless", SpellBless, 20, []string{"You feel righteous.\r\n"}},
		{"armor", SpellArmor, 20, []string{"You feel someone protecting you.\r\n"}},
		{"detect invis", SpellDetectInvis, 20, []string{"Your eyes tingle.\r\n"}},
		{"detect magic", SpellDetectMagic, 20, []string{"Your eyes tingle.\r\n"}},
		{"detect align", SpellDetectAlign, 20, []string{"Your eyes tingle.\r\n"}},
		{"know align", SpellKnowAlign, 20, []string{"Like a physical blow, emotions of others wash over you.\r\n"}},
		{"slow", SpellSlow, 20, []string{"You feel the world speed up around you.\r\n"}},
		{"dream travel", SpellDreamTravel, 20, []string{"You feel the power of the Dream Lords surround you.\r\n"}},
		{"water breathe", SpellWaterBreathe, 20, []string{"You feel your breath become colder.\r\n"}},
		{"invisible", SpellInvisible, 20, []string{"You vanish.\r\n"}},
		{"transparency", SpellTransparency, 20, []string{"Your skin turns transparent.\r\n"}},
		{"fly", SpellFly, 20, []string{"Your feet rise off the ground!\r\n"}},
		{"levitate", SpellLevitate, 20, []string{"Your feet rise off the ground!\r\n"}},
		{"strength", SpellStrength, 20, []string{"You feel stronger!\r\n"}},
		{"adrenaline", SpellAdrenaline, 20, []string{"You feel stronger!\r\n"}},
		{"sense life", SpellSenseLife, 20, []string{"Your feel your awareness improve.\r\n"}},
		{"waterwalk", SpellWaterwalk, 20, []string{"You feel webbing between your toes.\r\n"}},
		{"change density", SpellChangeDensity, 20, []string{"Your molecular density shifts.\r\n"}},
		{"chameleon", SpellChameleon, 20, []string{"You blend into the surroundings.\r\n"}},
		{"metalskin", SpellMetalskin, 20, []string{"Your skin turns metallic!\r\n"}},
		{"invulnerability", SpellInvulnerability, 20, []string{"A globe of protection appears around you!\r\n"}},
		{"psyshield", SpellPsyshield, 20, []string{"You feel a shield of energy form around you.\r\n"}},
		{"great percept", SpellGreatPercept, 20, []string{"Your eyes glow briefly.\r\n"}},
		{"less percept", SpellLessPercept, 20, []string{"Your eyes glow briefly.\r\n"}},
		{"intellect", SpellIntellect, 20, []string{"Your head clears and you realize some of the secrets of life!\r\n"}},
		{"mind bar", SpellMindBar, 20, []string{"Suddenly, your mind numbs and you feel somewhat impaired.\r\n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			self := newAffectMale("Mordecai", 8162)
			magAffectsApply(tc.level, self, self, tc.spellNum, false, 0, nil)
			if !reflect.DeepEqual(self.messages, tc.want) {
				t.Errorf("self-cast %s messages = %q, want %q", tc.name, self.messages, tc.want)
			}
		})
	}
}

// TestMagAffectsMessages_CastOnOther proves the to_self dispatch gate: when
// ch != victim the caster receives the to_self line with $M/$S resolved from
// the victim, and the victim still receives to_vict (magic.c:1414-1419).
func TestMagAffectsMessages_CastOnOther(t *testing.T) {
	cases := []struct {
		name       string
		spellNum   int
		wantCaster []string
		wantVictim []string
	}{
		{"chill touch", SpellChillTouch, []string{"Summoning the forces of magick, you press your icy hand against him.\r\n"}, []string{"You feel your strength wither!\r\n"}},
		{"bless", SpellBless, []string{"You bestow the blessing of your gods on him.\r\n"}, []string{"You feel righteous.\r\n"}},
		{"armor", SpellArmor, []string{"The magick protects him.\r\n"}, []string{"You feel someone protecting you.\r\n"}},
		{"detect invis", SpellDetectInvis, []string{"A streak of yellow light courses from your hand, washing over him!\r\n"}, []string{"Your eyes tingle.\r\n"}},
		{"detect magic", SpellDetectMagic, []string{"A streak blue light courses from your fingertips, washing over him!\r\n"}, []string{"Your eyes tingle.\r\n"}},
		{"infravision", SpellInfravision, []string{"With a light touch, you bestow the magick into his eyes.\r\n", "Barnaby's eyes glow red.\r\n"}, []string{"Your eyes glow red.\r\n"}},
		{"slow", SpellSlow, []string{"You send the forces of time against his!\r\n"}, []string{"You feel the world speed up around you.\r\n"}},
		{"fly", SpellFly, []string{"Like a falling feather, your magick floats toward him.\r\n", "Barnaby's feet rise off the ground!\r\n"}, []string{"Your feet rise off the ground!\r\n"}},
		{"strength", SpellStrength, []string{"Grabbing him, you feel a strong flow of magick course between you.\r\n"}, []string{"You feel stronger!\r\n"}},
		{"waterwalk", SpellWaterwalk, []string{"Your magic makes him light footed.\r\n"}, []string{"You feel webbing between your toes.\r\n"}},
		{"change density", SpellChangeDensity, []string{"You shift his molecular density.\r\n"}, []string{"Your molecular density shifts.\r\n"}},
		{"mind bar", SpellMindBar, []string{"You place a mental bar across his mind.\r\n"}, []string{"Suddenly, your mind numbs and you feel somewhat impaired.\r\n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caster := newAffectMale("Castero", 8162)
			victim := newAffectMale("Barnaby", 8162)
			world := newAffectMsgWorld(caster, victim)
			magAffectsApply(20, caster, victim, tc.spellNum, false, 0, world)
			if !reflect.DeepEqual(caster.messages, tc.wantCaster) {
				t.Errorf("cast-on-other %s caster messages = %q, want %q", tc.name, caster.messages, tc.wantCaster)
			}
			if !reflect.DeepEqual(victim.messages, tc.wantVictim) {
				t.Errorf("cast-on-other %s victim messages = %q, want %q", tc.name, victim.messages, tc.wantVictim)
			}
		})
	}
}

// TestMagAffectsMessages_HasteToSelfQuirk pins the C quirk at magic.c:1046:
// haste's only message is a to_self line, so the caster sees "You feel your
// movement quicken!" when ch != victim and the victim sees nothing at all.
func TestMagAffectsMessages_HasteToSelfQuirk(t *testing.T) {
	caster := newAffectMale("Castero", 8162)
	victim := newAffectMale("Barnaby", 8162)
	world := newAffectMsgWorld(caster, victim)
	magAffectsApply(20, caster, victim, SpellHaste, false, 0, world)
	if !reflect.DeepEqual(caster.messages, []string{"You feel your movement quicken!\r\n"}) {
		t.Errorf("haste caster messages = %q, want the single to_self line", caster.messages)
	}
	if len(victim.messages) != 0 {
		t.Errorf("haste victim messages = %q, want none (C sends no to_vict for haste)", victim.messages)
	}

	self := newAffectMale("Mordecai", 8162)
	magAffectsApply(20, self, self, SpellHaste, false, 0, nil)
	if len(self.messages) != 0 {
		t.Errorf("self-haste messages = %q, want none (to_self is ch != victim only)", self.messages)
	}
}

// TestMagAffectsMessages_RoomAudience proves to_room bytes with $n and $m
// resolved from the victim, delivered to everyone in the room except the
// victim (C act(..., victim, ..., TO_ROOM)).
func TestMagAffectsMessages_RoomAudience(t *testing.T) {
	cases := []struct {
		name     string
		spellNum int
		want     string
	}{
		{"invisible", SpellInvisible, "Barnaby slowly fades out of existence.\r\n"},
		{"transparency", SpellTransparency, "Barnaby slowly fades out of existence.\r\n"},
		{"sanctuary neutral", SpellSanctuary, "Barnaby is surrounded by a white aura.\r\n"},
		{"fly", SpellFly, "Barnaby's feet rise off the ground!\r\n"},
		{"levitate", SpellLevitate, "Barnaby's feet rise off the ground!\r\n"},
		{"prot evil", SpellProtFromEvil, "A stream of silver light surges from Barnaby's fingertips, covering him!\r\n"},
		{"prot good", SpellProtFromGood, "A stream of silver light surges from Barnaby's fingertips, covering him!\r\n"},
		{"great percept", SpellGreatPercept, "Barnaby's eyes glow briefly.\r\n"},
		{"less percept", SpellLessPercept, "Barnaby's eyes glow briefly.\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caster := newAffectMale("Castero", 8162)
			victim := newAffectMale("Barnaby", 8162)
			observer := newAffectMale("Observe", 8162)
			world := newAffectMsgWorld(caster, victim, observer)
			magAffectsApply(20, caster, victim, tc.spellNum, false, 0, world)
			for _, msg := range observer.messages {
				if msg == tc.want {
					return
				}
			}
			t.Errorf("%s observer messages = %q, want to contain %q", tc.name, observer.messages, tc.want)
			// The victim is excluded from TO_ROOM.
			for _, msg := range victim.messages {
				if msg == tc.want {
					t.Errorf("%s victim also received room line %q", tc.name, msg)
				}
			}
		})
	}
}

// TestMagAffectsMessages_SanctuaryAlignment proves the IS_EVIL branch pair.
func TestMagAffectsMessages_SanctuaryAlignment(t *testing.T) {
	victim := newAffectMale("Barnaby", 8162)
	victim.alignment = -600
	magAffectsApply(20, victim, victim, SpellSanctuary, false, 0, nil)
	if !reflect.DeepEqual(victim.messages, []string{"A black aura momentarily surrounds you.\r\n"}) {
		t.Errorf("evil sanctuary to_vict = %q, want black aura", victim.messages)
	}
}

// TestMagAffectsMessages_SaveBranches proves the early-return bytes on the
// save-gated hostile affects (RNG is fixed by passing `saved` directly).
func TestMagAffectsMessages_SaveBranches(t *testing.T) {
	t.Run("curse saved sends NOEFFECT to the caster only", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		world := newAffectMsgWorld(caster, victim)
		magAffectsApply(20, caster, victim, SpellCurse, true, 0, world)
		if !reflect.DeepEqual(caster.messages, []string{"Nothing seems to happen.\r\n"}) {
			t.Errorf("saved curse caster = %q, want NOEFFECT", caster.messages)
		}
		if len(victim.messages) != 0 {
			t.Errorf("saved curse victim = %q, want silence", victim.messages)
		}
	})
	t.Run("curse unsaved full audiences", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		observer := newAffectMale("Observe", 8162)
		world := newAffectMsgWorld(caster, victim, observer)
		magAffectsApply(20, caster, victim, SpellCurse, false, 0, world)
		if !reflect.DeepEqual(victim.messages, []string{"You feel very uncomfortable.\r\n"}) {
			t.Errorf("curse victim = %q", victim.messages)
		}
		if !reflect.DeepEqual(observer.messages, []string{"Barnaby briefly glows red!\r\n"}) {
			t.Errorf("curse room = %q", observer.messages)
		}
		if !reflect.DeepEqual(caster.messages, []string{"A streak of red light courses from your hand!\r\n", "Barnaby briefly glows red!\r\n"}) {
			t.Errorf("curse caster = %q", caster.messages)
		}
	})
	t.Run("poison saved sends NOEFFECT to the caster", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		world := newAffectMsgWorld(caster, victim)
		magAffectsApply(20, caster, victim, SpellPoison, true, 0, world)
		if !reflect.DeepEqual(caster.messages, []string{"Nothing seems to happen.\r\n"}) {
			t.Errorf("saved poison caster = %q, want NOEFFECT", caster.messages)
		}
		if len(victim.messages) != 0 {
			t.Errorf("saved poison victim = %q, want silence", victim.messages)
		}
	})
	t.Run("poison unsaved full audiences", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		observer := newAffectMale("Observe", 8162)
		world := newAffectMsgWorld(caster, victim, observer)
		magAffectsApply(20, caster, victim, SpellPoison, false, 0, world)
		if !reflect.DeepEqual(victim.messages, []string{"You feel very sick.\r\n"}) {
			t.Errorf("poison victim = %q", victim.messages)
		}
		if !reflect.DeepEqual(observer.messages, []string{"Barnaby gets violently ill!\r\n"}) {
			t.Errorf("poison room = %q", observer.messages)
		}
		if !reflect.DeepEqual(caster.messages, []string{"Your tainted magick pulses towards him.\r\n", "Barnaby gets violently ill!\r\n"}) {
			t.Errorf("poison caster = %q", caster.messages)
		}
	})
	t.Run("blindness saved fade message", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		world := newAffectMsgWorld(caster, victim)
		magAffectsApply(20, caster, victim, SpellBlindness, true, 0, world)
		if !reflect.DeepEqual(caster.messages, []string{"Your magic fades, then dies out totally.\r\n"}) {
			t.Errorf("saved blindness caster = %q", caster.messages)
		}
	})
	t.Run("blindness unsaved full audiences", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		observer := newAffectMale("Observe", 8162)
		world := newAffectMsgWorld(caster, victim, observer)
		magAffectsApply(20, caster, victim, SpellBlindness, false, 0, world)
		if !reflect.DeepEqual(victim.messages, []string{"You have been blinded!\r\n"}) {
			t.Errorf("blindness victim = %q", victim.messages)
		}
		if !reflect.DeepEqual(observer.messages, []string{"Barnaby seems to be blinded!\r\n"}) {
			t.Errorf("blindness room = %q", observer.messages)
		}
		if !reflect.DeepEqual(caster.messages, []string{"A streak of blackness courses from your hand!\r\n", "Barnaby seems to be blinded!\r\n"}) {
			t.Errorf("blindness caster = %q", caster.messages)
		}
	})
	t.Run("sleep saved room-only shake head", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		observer := newAffectMale("Observe", 8162)
		world := newAffectMsgWorld(caster, victim, observer)
		magAffectsApply(20, caster, victim, SpellSleep, true, 0, world)
		if !reflect.DeepEqual(observer.messages, []string{"Barnaby shakes his head wearily, but then snaps out of it!\r\n"}) {
			t.Errorf("saved sleep room = %q", observer.messages)
		}
		if !reflect.DeepEqual(caster.messages, []string{"Barnaby shakes his head wearily, but then snaps out of it!\r\n"}) {
			t.Errorf("saved sleep caster = %q, want the room line (caster is in the room)", caster.messages)
		}
		if len(victim.messages) != 0 {
			t.Errorf("saved sleep victim = %q, want silence", victim.messages)
		}
	})
	t.Run("sleep unsaved sleepy lines and position drop", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		observer := newAffectMale("Observe", 8162)
		world := newAffectMsgWorld(caster, victim, observer)
		magAffectsApply(20, caster, victim, SpellSleep, false, 0, world)
		if !reflect.DeepEqual(victim.messages, []string{"You feel very sleepy...  Zzzz......\r\n"}) {
			t.Errorf("sleep victim = %q", victim.messages)
		}
		if !reflect.DeepEqual(observer.messages, []string{"Barnaby goes to sleep.\r\n"}) {
			t.Errorf("sleep room = %q", observer.messages)
		}
		if victim.position != int(PosSleeping) {
			t.Errorf("sleep position = %d, want %d", victim.position, PosSleeping)
		}
	})
	t.Run("flamestrike saved NOEFFECT", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		magAffectsApply(20, caster, victim, SpellFlameStrike, true, 0, nil)
		if !reflect.DeepEqual(caster.messages, []string{"Nothing seems to happen.\r\n"}) {
			t.Errorf("saved flamestrike caster = %q, want NOEFFECT", caster.messages)
		}
	})
	t.Run("flamestrike unsaved full audiences", func(t *testing.T) {
		caster := newAffectMale("Castero", 8162)
		victim := newAffectMale("Barnaby", 8162)
		observer := newAffectMale("Observe", 8162)
		world := newAffectMsgWorld(caster, victim, observer)
		magAffectsApply(20, caster, victim, SpellFlameStrike, false, 0, world)
		if !reflect.DeepEqual(victim.messages, []string{"A bolt of flame shoots down from the heavens and engulfs you!\r\n"}) {
			t.Errorf("flamestrike victim = %q", victim.messages)
		}
		if !reflect.DeepEqual(observer.messages, []string{"A bolt of flame shoots down from the heavens and engulfs Barnaby!\r\n"}) {
			t.Errorf("flamestrike room = %q", observer.messages)
		}
		if !reflect.DeepEqual(caster.messages, []string{"You call down a bolt of flame on him!\r\n", "A bolt of flame shoots down from the heavens and engulfs Barnaby!\r\n"}) {
			t.Errorf("flamestrike caster = %q", caster.messages)
		}
	})
}
