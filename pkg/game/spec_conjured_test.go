package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func conjuredTestMob(t *testing.T, w *World, vnum int, room int) *MobInstance {
	t.Helper()
	mob := newSpecProcTestMob(t, w, room, 10)
	mob.VNum = vnum
	mob.Prototype.VNum = vnum
	mob.Prototype.ShortDesc = map[int]string{
		81: "an earth elemental",
		85: "a Dominion Angel",
	}[vnum]
	return mob
}

func TestSpecConjured_EntryAndCharmGates(t *testing.T) {
	w, actor, lastMsg := newSpecProcTestWorld(t)
	mob := conjuredTestMob(t, w, 81, actor.GetRoom())
	lastMsg()
	mob.SetAffected(affCharm)

	if got := specConjured(w, actor, mob, "say", ""); got {
		t.Fatal("charmed conjured mob should fall through")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("charmed conjured output = %q, want empty", got)
	}

	if got := specConjured(w, actor, mob, "", ""); got {
		t.Fatal("runtime-charmed conjured mob should fall through")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("runtime charm gate output = %q, want empty", got)
	}
	if got := len(w.GetMobsInRoom(actor.GetRoom())); got != 1 {
		t.Fatalf("runtime charm gate mob count = %d, want 1", got)
	}
}

func TestSpecConjured_FizzleAudienceAndExtraction(t *testing.T) {
	w, master, lastMsg := newSpecProcTestWorld(t)
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	peer := NewPlayer(2, "Peer", master.GetRoom())
	peer.SetLevel(5)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer(peer): %v", err)
	}
	mob := conjuredTestMob(t, w, 81, master.GetRoom())
	mob.SetFollowing(master.GetName())
	lastMsg()
	clearConjuredMessages(messages)

	if got := specConjured(w, master, mob, "say", ""); !got {
		t.Fatal("uncharmed fizzle should be handled")
	}
	masterOutput := messages[master.GetName()]
	peerOutput := messages[peer.GetName()]
	if masterOutput != "You lose control and an earth elemental fizzles away!\r\n"+
		"An earth elemental returns to its own plane of existence.\r\n" {
		t.Fatalf("master fizzle output = %q, want direct and room notices", masterOutput)
	}
	if peerOutput != "An earth elemental returns to its own plane of existence.\r\n" {
		t.Fatalf("peer fizzle output = %q, want room plane notice only", peerOutput)
	}
	if got := len(w.GetMobsInRoom(master.GetRoom())); got != 0 {
		t.Fatalf("fizzle room mob count = %d, want 0", got)
	}
	if got := len(w.GetAllMobs()); got != 0 {
		t.Fatalf("fizzle active mob count = %d, want 0", got)
	}
	if strings.Contains(masterOutput+peerOutput, "corpse") {
		t.Fatalf("fizzle output invented corpse text: master=%q peer=%q", masterOutput, peerOutput)
	}
}

func TestSpecConjured_DefaultSpeechAudienceAndExtraction(t *testing.T) {
	w, actor, lastMsg := newSpecProcTestWorld(t)
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	peer := NewPlayer(2, "Peer", actor.GetRoom())
	peer.SetLevel(5)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer(peer): %v", err)
	}
	mob := conjuredTestMob(t, w, 85, actor.GetRoom())
	lastMsg()
	clearConjuredMessages(messages)

	if got := specConjured(w, actor, mob, "look", ""); !got {
		t.Fatal("default conjured branch should be handled")
	}
	want := "A Dominion Angel states, 'My work here is done.'\r\n" +
		"A Dominion Angel disappears in a flash of white light!\r\n"
	if messages[actor.GetName()] != want || messages[peer.GetName()] != want {
		t.Fatalf("default audience output = actor %q peer %q, want %q each", messages[actor.GetName()], messages[peer.GetName()], want)
	}
	if got := len(w.GetMobsInRoom(actor.GetRoom())); got != 0 {
		t.Fatalf("default room mob count = %d, want 0", got)
	}
}

func TestSpecConjured_AutonomousRegisteredDispatch(t *testing.T) {
	w, actor, lastMsg := newSpecProcTestWorld(t)
	mob := conjuredTestMob(t, w, 85, actor.GetRoom())
	mob.Prototype.ActionFlags = []string{"SPEC"}
	oldName, hadName := MobSpecAssign[mob.GetVNum()]
	MobSpecAssign[mob.GetVNum()] = "conjured"
	t.Cleanup(func() {
		if hadName {
			MobSpecAssign[mob.GetVNum()] = oldName
		} else {
			delete(MobSpecAssign, mob.GetVNum())
		}
	})
	lastMsg()

	w.MobileActivityForMob(mob)

	want := "A Dominion Angel states, 'My work here is done.'\r\n" +
		"A Dominion Angel disappears in a flash of white light!\r\n"
	if got := lastMsg(); got != want {
		t.Fatalf("autonomous dispatch output = %q, want %q", got, want)
	}
	if got := len(w.GetAllMobs()); got != 0 {
		t.Fatalf("autonomous dispatch active mob count = %d, want 0", got)
	}
}

func clearConjuredMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}
