package spells

import (
	"strings"
	"testing"
)

// mockSayWorld implements the roomIterable interface used by sendToRoom.
type mockSayWorld struct {
	players []interface{}
	mobs    []interface{}
}

func (w *mockSayWorld) ForEachPlayerInRoomInterface(roomVNum int, fn func(interface{})) {
	for _, p := range w.players {
		fn(p)
	}
}

func (w *mockSayWorld) ForEachMobInRoomInterface(roomVNum int, fn func(interface{})) {
	for _, m := range w.mobs {
		fn(m)
	}
}

func newSayChar(name string, class int, room int) *mockSpellsChar {
	return &mockSpellsChar{
		name:     name,
		class:    class,
		roomVNum: room,
	}
}

func TestSaySpell_RoomMessageNamesCaster(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	observer := newSayChar("Bob", 0, 100)
	world := &mockSayWorld{players: []interface{}{caster, observer}}

	SaySpell(caster, 1, nil, nil, world)

	if len(observer.messages) == 0 {
		t.Fatal("observer received no message")
	}
	msg := observer.messages[0]
	if !strings.Contains(msg, "Alice") {
		t.Errorf("room message should name caster Alice, got: %s", msg)
	}
	if !strings.Contains(msg, "holy ward") {
		t.Errorf("room message should show real spell name to same-class observer, got: %s", msg)
	}
}

func TestSaySpell_RoomMessageObfuscatesForDifferentClass(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	observer := newSayChar("Bob", 9, 100)
	world := &mockSayWorld{players: []interface{}{caster, observer}}

	SaySpell(caster, 1, nil, nil, world)

	if len(observer.messages) == 0 {
		t.Fatal("observer received no message")
	}
	msg := observer.messages[0]
	if !strings.Contains(msg, "Alice") {
		t.Errorf("room message should name caster Alice, got: %s", msg)
	}
	if strings.Contains(msg, "holy ward") {
		t.Errorf("room message should obfuscate spell for different-class observer, got: %s", msg)
	}
}

func TestGetSpellNameUsesDarkPawnsCatalog(t *testing.T) {
	tests := map[int]string{
		1:   "holy ward",
		7:   "charm person",
		16:  "cure light",
		32:  "flame arrow",
		50:  "infravision",
		134: "kick",
	}
	for num, want := range tests {
		if got := GetSpellName(num); got != want {
			t.Errorf("GetSpellName(%d) = %q, want %q", num, got, want)
		}
	}
	if got := GetSpellName(0); got != "" {
		t.Errorf("GetSpellName(0 reserved) = %q, want empty", got)
	}
	if got := GetSpellName(207); got != "" {
		t.Errorf("GetSpellName(C sentinel) = %q, want empty", got)
	}
	if got := GetSpellName(9999); got != "" {
		t.Errorf("GetSpellName(out of range) = %q, want empty", got)
	}
}

func TestObfuscateSpellName_PoisonMatchesOracle(t *testing.T) {
	if got := ObfuscateSpellName("poison"); got != "saugab" {
		t.Fatalf("poison incantation = %q, want %q", got, "saugab")
	}
}

func TestObfuscateSpellNameMatchesCCompoundAndCharacterPass(t *testing.T) {
	// The C loop in src/spell_parser.c:282-289 does not break after a
	// substitution. This is the source-audited set of spell catalog entries
	// whose output differed when Go restarted the table after each match.
	tests := map[string]string{
		"clone":                    "qrabo",
		"color spray":              "qahif gsfil",
		"create food":              "qfoi yaam",
		"create water":             "qfoi xif",
		"detect poison":            "oculoinfra saugab",
		"energy drain":             "unsoftl mfiub",
		"fireball":                 "yufovirr",
		"poison":                   "saugab",
		"protection from evil":     "sfahkaminauab yfaw ozur",
		"remove curse":             "canduswazo judifgo",
		"shocking grasp":           "gpaqtubt tfigs",
		"infravision":              "ubyfizuguab",
		"hellfire":                 "porryufo",
		"globe of invulnerability": "travo ay ubzerbofivuruhl",
		"invigorate":               "ubzutafi",
		"greater perception":       "tfoif dudaqoshuab",
		"adrenaline boost":         "imfunsirubo vaagh",
		"divine intervention":      "muzubo ubhofzunshuab",
		"smokescreen":              "gwatogqfouns",
		"disintegration":           "marubhotfihuab",
		"protection from good":     "sfahkaminauab yfaw taam",
		"flame strike":             "yriwo ghfuto",
		"dream travel":             "mfoiw hgruzor",
		"waterbreathe":             "xifvfoihpo",
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ObfuscateSpellName(name); got != want {
				t.Errorf("ObfuscateSpellName(%q) = %q, want C output %q", name, got, want)
			}
		})
	}
}

func TestSaySpell_TargetMessageDeliveredToTarget(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	target := newSayChar("Bob", 0, 100)
	world := &mockSayWorld{players: []interface{}{caster, target}}

	SaySpell(caster, 1, target, nil, world)

	foundTargetMsg := false
	for _, msg := range target.messages {
		if strings.Contains(msg, "stares at you") {
			foundTargetMsg = true
			break
		}
	}
	if !foundTargetMsg {
		t.Errorf("target should receive a 'stares at you' message, got: %v", target.messages)
	}
	if len(target.messages) != 1 {
		t.Errorf("target received room message plus TO_VICT duplicate: %v", target.messages)
	}
	if strings.Contains(target.messages[0], "$") || !strings.HasSuffix(target.messages[0], "\r\n") {
		t.Errorf("target message has unresolved act token or missing CRLF: %q", target.messages[0])
	}
}

func TestSaySpell_TargetMessageNotDeliveredToCaster(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	target := newSayChar("Bob", 0, 100)
	world := &mockSayWorld{players: []interface{}{caster, target}}

	SaySpell(caster, 1, target, nil, world)

	for _, msg := range caster.messages {
		if strings.Contains(msg, "stares at you") {
			t.Errorf("caster should not receive target-specific message, got: %s", msg)
		}
	}
}
