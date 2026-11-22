package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mickeyTestWorld builds rooms 1001 (origin) and 11369 (MICKSROOM,
// src/spec_procs.c:1266), with a Mickey proto (7969, src/spec_procs.c:1267)
// and a Mallory proto, plus a victim and an observer capturing bytes.
func mickeyTestWorld(t *testing.T, peaceful bool) (*World, map[string]string, *Player, *Player) {
	t.Helper()
	originFlags := []string{}
	if peaceful {
		originFlags = []string{"peaceful"}
	}
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Origin", Zone: 1, Flags: originFlags},
			{VNum: 11369, Name: "Micksroom", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 7969, Keywords: "mickey", ShortDesc: "Mickey", LongDesc: "Mickey is here.", Level: 20, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 100}, Alignment: 0, Race: 1, Int: 11, Wis: 11, Sex: 1, Position: combat.PosStanding, DefaultPos: combat.PosStanding},
			{VNum: 99001, Keywords: "mallory", ShortDesc: "Mallory", LongDesc: "Mallory is here.", Level: 20, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 100}, Alignment: 0, Race: 1, Int: 11, Wis: 11, Sex: 1, Position: combat.PosStanding, DefaultPos: combat.PosStanding},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	// Watcher first so Victim is newest (C char_to_room prepends,
	// src/spec_procs.c:1310) and the approach scan lands on Victim.
	observer := NewPlayer(2, "Watcher", 1001)
	observer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	out["Victim"] = ""
	out["Watcher"] = ""
	return w, out, victim, observer
}

// seedForNumber110 finds a stream seed whose first number(1,10) is want.
func seedForNumber110(t *testing.T, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(1, 10) == want {
			return seed
		}
	}
	t.Fatalf("no seed in 1..4999 draws %d first", want)
	return 0
}

func spawnMickeyOrMallory(t *testing.T, w *World, vnum, room int) *MobInstance {
	t.Helper()
	mob, err := w.SpawnMobQuiet(vnum, room)
	if err != nil {
		t.Fatalf("SpawnMobQuiet %d: %v", vnum, err)
	}
	mob.SetLevel(20)
	mob.SetPosition(combat.PosStanding)
	mob.CurrentHP = 200
	mob.MaxHP = 200
	mob.SetHealth(200)
	return mob
}

// TestSpecMickey_FightingBarks pins SPECIAL(mickey)'s fighting barks
// (src/spec_procs.c:1282-1306): cases 1-5 to TO_ROOM, case 5 falls into
// default break, rolls 6-10 silent, returns TRUE.
func TestSpecMickey_FightingBarks(t *testing.T) {
	want := map[int]string{
		1: "Mickey shouts, 'I'll always love you Mal, no matter what!'\r\n",
		2: "Mickey asks, 'Do you believe in fate?'\r\n",
		3: "Mickey says, 'You're not centered.'\r\n",
		4: "Mickey shouts, 'When they come and ask you who did this, tell them it was Mickey and Mallory Knox!'\r\n",
		5: "Mickey states, 'It's not nice to point.'\r\n",
	}
	for pick, line := range want {
		t.Run(string(rune('0'+pick)), func(t *testing.T) {
			w, out, victim, _ := mickeyTestWorld(t, false)
			mickey := spawnMickeyOrMallory(t, w, 7969, 1001)
			mickey.SetFightingBody(victim)
			seed := seedForNumber110(t, pick)
			dprng.ResetStream(seed)
			if !specMickey(w, nil, mickey, "", "") {
				t.Fatal("specMickey should return true while fighting")
			}
			if got := out["Watcher"]; got != line {
				t.Fatalf("watcher bytes = %q, want %q", got, line)
			}
		})
	}
	// Rolls 6-10 print nothing (C default break) but still return TRUE and
	// consume exactly one number(1,10) draw.
	w, out, victim, _ := mickeyTestWorld(t, false)
	mickey := spawnMickeyOrMallory(t, w, 7969, 1001)
	mickey.SetFightingBody(victim)
	dprng.ResetStream(seedForNumber110(t, 7))
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("specMickey should return true on the silent default roll")
	}
	if got := out["Watcher"]; got != "" {
		t.Fatalf("silent default emitted %q", got)
	}
}

// TestSpecMickey_EntryGates pins the autonomous gate
// (src/spec_procs.c:1274): sleeping, command, player-trigger and negative
// HP all return false with no output and no approach.
func TestSpecMickey_EntryGates(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *World, mickey *MobInstance, victim *Player) (ch *Player, cmd string)
	}{
		{"sleeping", func(w *World, mickey *MobInstance, victim *Player) (*Player, string) {
			mickey.SetPosition(combat.PosSleeping)
			return nil, ""
		}},
		{"command", func(w *World, mickey *MobInstance, victim *Player) (*Player, string) {
			return nil, "look"
		}},
		{"player trigger", func(w *World, mickey *MobInstance, victim *Player) (*Player, string) {
			return victim, ""
		}},
		{"negative hp", func(w *World, mickey *MobInstance, victim *Player) (*Player, string) {
			mickey.CurrentHP = -1
			return nil, ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, out, victim, _ := mickeyTestWorld(t, false)
			mickey := spawnMickeyOrMallory(t, w, 7969, 1001)
			ch, cmd := tc.setup(w, mickey, victim)
			if specMickey(w, ch, mickey, cmd, "") {
				t.Fatal("specMickey should return false at the gate")
			}
			if got := out["Watcher"]; got != "" {
				t.Fatalf("gate emitted %q", got)
			}
		})
	}
}

// TestSpecMickey_ApproachBeats pins the three non-combat beats
// (src/spec_procs.c:1310-1370): wake + kill line, "And you wanna know
// why?", then the non-peaceful natural-born-killer line plus hit. Each
// beat returns TRUE and records the victim ID in GET_ALIGNMENT.
func TestSpecMickey_ApproachBeats(t *testing.T) {
	w, out, victim, _ := mickeyTestWorld(t, false)
	mickey := spawnMickeyOrMallory(t, w, 7969, 1001)

	// Beat 0: do_wake (already awake, so only the say line reaches the
	// room) + kill line to TO_ROOM.
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 0 should return true")
	}
	want0 := "Mickey says, 'Victim, I'm gonna kill you.'\r\n"
	if got := out["Watcher"]; got != want0 {
		t.Fatalf("beat 0 watcher = %q, want %q", got, want0)
	}
	if got := mickey.GetAlignment(); got != victim.GetID() {
		t.Fatalf("alignment = %d, want victim id %d", got, victim.GetID())
	}

	// Beat 1: asks "And you wanna know why?".
	out["Watcher"] = ""
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 1 should return true")
	}
	want1 := "Mickey asks, 'And you wanna know why?'\r\n"
	if got := out["Watcher"]; got != want1 {
		t.Fatalf("beat 1 watcher = %q, want %q", got, want1)
	}

	// Beat 2 non-peaceful: natural-born-killer line plus hit opener.
	engine := &mickeyTestCombatEngine{}
	w.SetCombatEngine(engine)
	out["Watcher"] = ""
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 2 should return true")
	}
	want2 := "Mickey says, 'Shit Victim, I'm a natural born killer.'.\r\n"
	if got := out["Watcher"]; got != want2 {
		t.Fatalf("beat 2 watcher = %q, want %q", got, want2)
	}
	if len(engine.initialAttacks) != 1 {
		t.Fatalf("initial attacks = %d, want 1 hit at %q", len(engine.initialAttacks), victim.GetName())
	}
}

// TestSpecMickey_ApproachPeaceful pins the PEACEFUL exit
// (src/spec_procs.c:1351-1358): "I'll tell you later" + vanishes, Mickey
// relocates to MICKSROOM 11369, no hit.
func TestSpecMickey_ApproachPeaceful(t *testing.T) {
	w, out, victim, _ := mickeyTestWorld(t, true)
	mickey := spawnMickeyOrMallory(t, w, 7969, 1001)
	// Walk beats 0 and 1.
	out["Watcher"] = ""
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 0 should return true")
	}
	out["Watcher"] = ""
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 1 should return true")
	}
	engine := &mickeyTestCombatEngine{}
	w.SetCombatEngine(engine)
	out["Watcher"] = ""
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 2 should return true")
	}
	want := "Mickey says, 'I'll tell you later, Victim'.\r\nMickey walks out the door and vanishes.\r\n"
	if got := out["Watcher"]; got != want {
		t.Fatalf("peaceful watcher = %q, want %q", got, want)
	}
	if got := mickey.GetRoomVNum(); got != 11369 {
		t.Fatalf("mickey room = %d, want 11369", got)
	}
	if len(engine.initialAttacks) != 0 {
		t.Fatal("peaceful beat must not hit")
	}
	_ = victim
}

// TestSpecMickey_VictimChangeResets pins the talk reset on a new victim
// (src/spec_procs.c:1316-1322): after beat 0 for Victim, a different
// player restarts at beat 0 rather than advancing to beat 1.
func TestSpecMickey_VictimChangeResets(t *testing.T) {
	w, out, victim, _ := mickeyTestWorld(t, false)
	mickey := spawnMickeyOrMallory(t, w, 7969, 1001)
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("beat 0 should return true")
	}
	// Move the first victim out and bring in a second one.
	victim.SetRoom(11369)
	second := NewPlayer(99, "Second", 1001)
	second.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer second: %v", err)
	}
	out["Watcher"] = ""
	if !specMickey(w, nil, mickey, "", "") {
		t.Fatal("new-victim beat should return true")
	}
	want := "Mickey says, 'Second, I'm gonna kill you.'\r\n"
	if got := out["Watcher"]; got != want {
		t.Fatalf("reset watcher = %q, want %q (beat 0 for the new victim)", got, want)
	}
	if got := mickey.GetAlignment(); got != second.GetID() {
		t.Fatalf("alignment = %d, want second id %d", got, second.GetID())
	}
}

type mickeyTestCombatEngine struct {
	initialAttacks [][2]string
}

func (e *mickeyTestCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	return nil
}

func (e *mickeyTestCombatEngine) PerformInitialAttack(attacker, defender combat.Combatant) error {
	e.initialAttacks = append(e.initialAttacks, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *mickeyTestCombatEngine) IsFighting(combat.Combatant) bool { return false }

func (e *mickeyTestCombatEngine) GetCombatTarget(combat.Combatant) (combat.Combatant, bool) {
	return nil, false
}

// TestSpecMallory_FightingBarks pins SPECIAL(mallory)'s barks
// (src/spec_procs.c:1387-1411): lowercase-i case 3 twice, case 5
// "giggles madly" (not the invented line), case 4 silent, default
// "biiitch" for rolls 6-10, all via act TO_ROOM, always returns FALSE.
func TestSpecMallory_FightingBarks(t *testing.T) {
	want := map[int]string{
		1: "Mallory asks, 'How do you like me now?'\r\n",
		2: "Mallory says, 'That was the worst head I've ever got.'\r\n",
		3: "Mallory asks, 'How sexy am i now, fucker?'\r\nMallory asks, 'How sexy am i NOW?'\r\n",
		5: "Mallory giggles madly.\r\n",
	}
	for pick, line := range want {
		t.Run(string(rune('0'+pick)), func(t *testing.T) {
			w, out, victim, _ := mickeyTestWorld(t, false)
			mallory := spawnMickeyOrMallory(t, w, 99001, 1001)
			mallory.SetFightingBody(victim)
			// No Mickey in the world: the revenge lookup fails, but the
			// bark still prints and the return is FALSE.
			dprng.ResetStream(seedForNumber110(t, pick))
			if specMallory(w, nil, mallory, "", "") {
				t.Fatal("specMallory must return false while fighting (C returns FALSE)")
			}
			if got := out["Watcher"]; got != line {
				t.Fatalf("watcher bytes = %q, want %q", got, line)
			}
		})
	}
	// Case 4 is a silent break; the default (6-10) shouts.
	w, out, victim, _ := mickeyTestWorld(t, false)
	mallory := spawnMickeyOrMallory(t, w, 99001, 1001)
	mallory.SetFightingBody(victim)
	dprng.ResetStream(seedForNumber110(t, 4))
	if specMallory(w, nil, mallory, "", "") {
		t.Fatal("case 4 must return false")
	}
	if got := out["Watcher"]; got != "" {
		t.Fatalf("case 4 emitted %q, want silence", got)
	}
	out["Watcher"] = ""
	dprng.ResetStream(seedForNumber110(t, 8))
	if specMallory(w, nil, mallory, "", "") {
		t.Fatal("default must return false")
	}
	want8 := "Mallory shouts, 'You stupid biiitch!'\r\n"
	if got := out["Watcher"]; got != want8 {
		t.Fatalf("default = %q, want %q", got, want8)
	}
}

// TestSpecMallory_StandAndRevengeGate pins the spectator-free contract:
// do_stand via npcStand (src/spec_procs.c:1385), no Mickey means no hunt
// and FALSE; a hunting Mickey is left alone.
func TestSpecMallory_StandAndRevengeGate(t *testing.T) {
	w, out, victim, _ := mickeyTestWorld(t, false)
	mallory := spawnMickeyOrMallory(t, w, 99001, 1001)
	mallory.SetFightingBody(victim)
	mallory.SetPosition(combat.PosResting)
	if specMallory(w, nil, mallory, "", "") {
		t.Fatal("stand arm must return false")
	}
	want := "Mallory stops resting, and clambers on his feet.\r\n"
	if got := out["Watcher"]; got != want {
		t.Fatalf("stand = %q, want %q", got, want)
	}
	if strings.Contains(out["Watcher"], "stands up!") {
		t.Fatalf("hand-built stand survived: %q", out["Watcher"])
	}
	if got := mallory.GetPosition(); got != combat.PosStanding {
		t.Fatalf("position = %d, want standing", got)
	}
}

// TestSpecMallory_Revenge pins set_hunting revenge
// (src/spec_procs.c:1412-1419): Mallory fighting a player sends the
// newest Mickey (7969, src/spec_procs.c:1267) after her attacker; a
// complete-syslog immortal sees "Mickey started hunting Victim"; the
// entry gates (src/spec_procs.c:1378) return false with no output.
func TestSpecMallory_Revenge(t *testing.T) {
	w, out, victim, _ := mickeyTestWorld(t, false)
	// Complete-syslog watcher for the mudlog line.
	logWatcher := NewPlayer(50, "Logwatch", 1001)
	logWatcher.Level = LVL_IMMORT
	logWatcher.SetPlrFlag(PrfLog1, true)
	logWatcher.SetPlrFlag(PrfLog2, true)
	if err := w.AddPlayer(logWatcher); err != nil {
		t.Fatalf("AddPlayer logwatch: %v", err)
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(&testSessions{logWatcher})
	t.Cleanup(func() { SetImmortalSessionProvider(old) })

	mallory := spawnMickeyOrMallory(t, w, 99001, 1001)
	mickey := spawnMickeyOrMallory(t, w, 7969, 1001)
	mallory.SetFightingBody(victim)
	out["Victim"] = ""
	dprng.ResetStream(seedForNumber110(t, 1))
	if specMallory(w, nil, mallory, "", "") {
		t.Fatal("revenge path must return false")
	}
	if !mickey.IsHunting() {
		t.Fatal("mickey should be hunting after revenge")
	}
	if got := mickey.GetHunting(); got != victim.GetName() {
		t.Fatalf("hunting = %q, want %q", got, victim.GetName())
	}
	logged := out["Logwatch"]
	wantLog := "[ " + mickey.GetName() + " started hunting " + victim.GetName() + " ]\r\n"
	foundExactHuntLine := false
	for _, line := range strings.Split(logged, "\r\n") {
		if line+"\r\n" == wantLog {
			foundExactHuntLine = true
			break
		}
	}
	if !foundExactHuntLine {
		t.Fatalf("syslog = %q, want exact C hunting line %q", logged, wantLog)
	}
	// Already hunting: a second Mallory pulse changes nothing.
	oldHunt := mickey.GetHunting()
	dprng.ResetStream(seedForNumber110(t, 1))
	if specMallory(w, nil, mallory, "", "") {
		t.Fatal("busy Mickey path must return false")
	}
	if got := mickey.GetHunting(); got != oldHunt {
		t.Fatalf("hunting changed to %q, want unchanged %q", got, oldHunt)
	}
	// Sleeping / command gates return false with no output.
	mallory.SetPosition(combat.PosSleeping)
	out["Watcher"] = ""
	if specMallory(w, nil, mallory, "", "") {
		t.Fatal("sleeping Mallory must return false")
	}
	if got := out["Watcher"]; got != "" {
		t.Fatalf("sleeping emitted %q", got)
	}
}
