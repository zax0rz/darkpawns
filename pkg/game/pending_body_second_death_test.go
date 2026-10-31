package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// secondDeathWorld builds C's do_kill second-death setup: an implementor with
// a lethal weapon, a 1-HP mobile, and a combat engine wired like the session
// manager wires it (DeathFunc -> World.HandleDeath, broadcasts to the room).
func secondDeathWorld(t *testing.T) (*World, *Player, *MobInstance, *strings.Builder, map[string]string) {
	t.Helper()
	w, messages := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Pit"})
	god := NewPlayer(1, "Twodiegod", 1001)
	god.SetLevel(34)
	// Ordinary mortal-range stats so hit()'s to-hit is not swamped by the
	// (stat-13) thaco penalties a zero-initialized test player would take
	// (src/fight.c:1808-1812); +10 hitroll leaves only a natural 1 missing.
	god.Stats.Str, god.Stats.StrAdd = 18, 100
	god.Stats.Dex, god.Stats.Int, god.Stats.Wis, god.Stats.Con, god.Stats.Cha = 13, 13, 13, 13, 13
	god.Hitroll = 10
	if err := w.AddPlayer(god); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	weapon := &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      1,
			Keywords:  "sword test",
			ShortDesc: "a test sword",
			TypeFlag:  5,                        // ITEM_WEAPON
			WearFlags: [4]int{1 << 13, 0, 0, 0}, // ITEM_WEAR_WIELD
			Values:    [4]int{0, 20, 20, 3},     // 20d20, val3=3 (slash)
		},
	}
	equipWeapon(t, god, weapon)

	mob := NewMob(&parser.Mob{VNum: 3001, Keywords: "fish", ShortDesc: "a fish", Exp: 0}, 1001)
	mob.Runtime.NameOverride = "fish"
	mob.SetHealth(1)
	w.mu.Lock()
	w.activeMobs[mob.ID] = mob
	w.mu.Unlock()

	room := &strings.Builder{}
	ce := combat.NewCombatEngine()
	ce.DeathFunc = w.HandleDeath
	ce.SetBroadcastFunc(func(_ int, message string, _ []combat.Combatant) {
		fmt.Fprint(room, message)
	})
	w.SetCombatEngine(ce)
	t.Cleanup(ce.Stop)

	oldCB := combat.GetCallbacks()
	oldRoller := combat.GetRoller()
	t.Cleanup(func() { combat.SetCallbacks(oldCB); combat.SetRoller(oldRoller) })
	cb := w.WireCombatCallbacks()
	// The session manager wires cb.Broadcast to room delivery (manager.go
	// SetCombatBroadcastFunc); mirror the world-side equivalent so death_cry's
	// TO_ROOM act reaches the room's occupants in this fixture too.
	cb.Broadcast = func(roomVNum int, message string, _ []combat.Combatant) {
		for _, c := range w.GetAllCharsInRoom(roomVNum) {
			if p, ok := c.(*Player); ok && p != nil {
				w.MessageSink(p.GetName(), []byte(message))
			}
		}
	}
	combat.SetCallbacks(cb)
	return w, god, mob, room, messages
}

func corpsesInRoom(w *World, room int) int {
	n := 0
	for _, item := range w.GetItemsInRoom(room) {
		if item.IsCorpse {
			n++
		}
	}
	return n
}

// TestPendingBodySecondDeathRunsFullDeathBranch pins C's second-death path:
// do_kill's raw_kill flags a body without setting POS_DEAD
// (src/act.offensive.c:155-161), so in a non-peaceful room a later swing's
// damage() gates pass, the damage applies, and the lethal swing runs the full
// death branch a second time (src/fight.c:1634-1691): the kill-count bump,
// the exp line, die_with_killer -> raw_kill's second death cry and a second,
// empty corpse (src/fight.c:610-622, 258-427). The second extract_char
// returns early on the already-set flag (src/handler.c:1197), so the body is
// still queued exactly once and freed once by the extraction pass.
func TestPendingBodySecondDeathRunsFullDeathBranch(t *testing.T) {
	// A seed whose first to-hit roll is not a natural 1; the assertions below
	// pin the death branch, so a miss seed would fail loudly instead of
	// silently passing.
	// Save the real roller first: the helper's own cleanup restores whatever
	// roller is current when it runs, which would otherwise be this seeded one.
	origRoller := combat.GetRoller()
	t.Cleanup(func() { combat.SetRoller(origRoller) })
	combat.SetRoller(combat.NewSeededRoller(3))
	w, god, mob, room, messages := secondDeathWorld(t)

	w.Instakill(mob, god, combat.TYPE_SLASH)
	if mob.IsAlive() || !mob.HasMobFlag(MobFlagExtract) {
		t.Fatal("instakill did not flag the body")
	}
	if mob.GetPosition() <= combat.PosDead {
		t.Fatalf("do_kill's raw_kill must not set POS_DEAD, got %d", mob.GetPosition())
	}
	if got := corpsesInRoom(w, 1001); got != 1 {
		t.Fatalf("first kill left %d corpses, want 1", got)
	}
	if god.Kills != 0 {
		t.Fatalf("raw_kill must not count a kill, kills=%d", god.Kills)
	}

	swing, ok := w.combatEngine.(interface {
		PerformPendingBodySwing(attacker, defender combat.Combatant) error
	})
	if !ok {
		t.Fatal("engine lacks PerformPendingBodySwing")
	}
	if err := swing.PerformPendingBodySwing(god, mob); err != nil {
		t.Fatalf("pending-body swing: %v", err)
	}

	if got := mob.GetHP(); got > -11 {
		t.Fatalf("lethal swing left HP %d; the damage never applied to the flagged body", got)
	}
	if god.Kills != 1 {
		t.Fatalf("kills = %d, want 1: only damage()'s second death branch counts", god.Kills)
	}
	if got := corpsesInRoom(w, 1001); got != 2 {
		t.Fatalf("second death left %d corpses, want 2", got)
	}
	if !strings.Contains(room.String(), "dead!  R.I.P.") {
		t.Fatalf("room bytes missing the POS_DEAD line: %q", room.String())
	}
	if !strings.Contains(messages["Twodiegod"], "Your blood freezes as you hear fish's death cry.") {
		t.Fatalf("bytes missing the second death cry (TO_ROOM act, src/fight.c:516-517): %q", messages["Twodiegod"])
	}
	// die_with_killer enrolls then stops both sides: the killer's exchange
	// ends with no dangling pair (fight.c:612-616).
	if god.GetFightingBody() != nil {
		t.Fatal("killer still fighting after die_with_killer")
	}
	// The body is queued once despite two raw_kills (src/handler.c:1197).
	w.ExtractPendingPlayers()
	if got := w.GetMobsInRoom(1001); len(got) != 0 {
		t.Fatalf("extraction pass left %d bodies", len(got))
	}
}

// TestPendingBodyPeacefulRefusalStillSkipsEnrollment keeps the other half of
// the lane pinned: a body killed in a PEACEFUL room is refused by damage()'s
// peaceful gate (src/fight.c:1336-1342) BEFORE set_fighting, so nothing is
// enrolled, no damage lands, and no second corpse appears. The relaxation
// that admits the non-peaceful second death must not widen this refusal.
func TestPendingBodyPeacefulRefusalStillSkipsEnrollment(t *testing.T) {
	// Save the real roller first: the helper's own cleanup restores whatever
	// roller is current when it runs, which would otherwise be this seeded one.
	origRoller := combat.GetRoller()
	t.Cleanup(func() { combat.SetRoller(origRoller) })
	combat.SetRoller(combat.NewSeededRoller(3))
	w, god, mob, _, _ := secondDeathWorld(t)
	w.mu.Lock()
	w.rooms[1001].Flags = []string{"peaceful"} // ROOM_PEACEFUL (src/structs.h:60)
	w.mu.Unlock()

	w.Instakill(mob, god, combat.TYPE_SLASH)
	before := mob.GetHP()

	swing, ok := w.combatEngine.(interface {
		PerformPendingBodySwing(attacker, defender combat.Combatant) error
	})
	if !ok {
		t.Fatal("engine lacks PerformPendingBodySwing")
	}
	if err := swing.PerformPendingBodySwing(god, mob); err != nil {
		t.Fatalf("pending-body swing: %v", err)
	}

	if mob.GetHP() != before {
		t.Fatalf("peaceful-room swing changed HP: %d -> %d", before, mob.GetHP())
	}
	if god.GetFightingBody() != nil || mob.GetFightingBody() != nil {
		t.Fatal("peaceful refusal enrolled a side; C enrolls only after the gate passes")
	}
	if got := corpsesInRoom(w, 1001); got != 1 {
		t.Fatalf("peaceful refusal built %d extra corpses", got-1)
	}
	if god.Kills != 0 {
		t.Fatalf("peaceful refusal counted a kill: %d", god.Kills)
	}
}

// TestPendingAttackerSwingsWithoutEnrolling pins C's hit() for a FLAGGED
// attacker (src/fight.c:1763-1779 has no extraction guard): a body do_kill's
// raw_kill left standing still draws and swings when a spell refusal makes it
// attack the caster (src/spells.c:581-591 — identify on a just-killed mob,
// Luna's release-review finding). The swing must not re-enroll the body: it is
// extracted at the next heartbeat before any violence pass.
func TestPendingAttackerSwingsWithoutEnrolling(t *testing.T) {
	w, god, mob, room, _ := secondDeathWorld(t)

	// do_kill's raw_kill: flagged, still standing, not enrolled.
	w.Instakill(mob, god, combat.TYPE_SLASH)
	if mob.IsAlive() || !mob.HasMobFlag(MobFlagExtract) {
		t.Fatal("instakill did not flag the body")
	}
	if mob.GetFightingBody() != nil {
		t.Fatal("flagged body starts enrolled")
	}

	swing, ok := w.combatEngine.(interface {
		PerformPendingBodySwing(attacker, defender combat.Combatant) error
	})
	if !ok {
		t.Fatal("engine lacks PerformPendingBodySwing")
	}
	if err := swing.PerformPendingBodySwing(mob, god); err != nil {
		t.Fatalf("pending-attacker swing: %v", err)
	}

	// The swing ran: the flagged attacker's message reached the room (the
	// implementor's damage is zeroed, so the bytes are the proof the draws
	// flowed), and the ATTACKER was not re-enrolled by its own swing — it is
	// extracted before any violence pass. The defender's enrolment against
	// the flagged body IS C's set_fighting(victim, ch) (fight.c:1441-1445);
	// the extraction pass stops it, as extract_char_final does
	// (src/handler.c:1147-1151).
	if !strings.Contains(room.String(), "fish") && !strings.Contains(room.String(), "testmob") {
		t.Fatalf("pending attacker's swing produced no room bytes: %q", room.String())
	}
	if mob.GetFightingBody() != nil {
		t.Fatal("pending attacker was re-enrolled by its swing")
	}
}
