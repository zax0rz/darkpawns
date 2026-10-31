package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// teleportRefusalWorld builds the NPC-on-NPC spell_teleport refusal setup: a
// level-30 caster whose NPC victim is one room over in level, a combat engine
// wired like the session manager wires it, and a caster-side sink standing in
// for a switched descriptor.
func teleportRefusalWorld(t *testing.T) (*World, *MobInstance, *MobInstance, *strings.Builder) {
	t.Helper()
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Pit"})
	// Spawn through the world so the bodies carry their world reference —
	// MobInstance.SendMessage's switched-descriptor routing needs it.
	for _, proto := range []*parser.Mob{
		{VNum: 3001, Keywords: "gil", ShortDesc: "Gil", Level: 30},
		{VNum: 3002, Keywords: "stalker", ShortDesc: "a stalker", Level: 30},
	} {
		w.mu.Lock()
		w.mobs[proto.VNum] = proto
		w.mu.Unlock()
	}
	caster, err := w.SpawnMob(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob caster: %v", err)
	}
	victim, err := w.SpawnMob(3002, 1001)
	if err != nil {
		t.Fatalf("SpawnMob victim: %v", err)
	}
	caster.Runtime.NameOverride = "gil"
	victim.Runtime.NameOverride = "stalker"
	victim.SetHealth(514)

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
	combat.SetCallbacks(w.WireCombatCallbacks())
	return w, caster, victim, room
}

// TestSpellTeleportSaveRefusalCounterattacks pins spells.c:183-187: an NPC
// victim that makes its save refuses the teleport with the caster-only line
// and counterattacks through hit(victim, ch, TYPE_UNDEFINED) — the victim's
// first swing consumes hit()'s draws and enrolls both bodies at the damage
// point.
func TestSpellTeleportSaveRefusalCounterattacks(t *testing.T) {
	w, caster, victim, room := teleportRefusalWorld(t)
	casterSink := &strings.Builder{}
	w.MobileMessageSink = func(body *MobInstance, msg []byte) {
		if body == caster {
			casterSink.Write(msg)
		}
	}
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == caster {
			return &Player{Name: "Switchgod"}
		}
		return nil
	}

	// magSavingThrow draws from the process dprng stream (not the combat
	// roller): ResetStream(2)'s first Number(0,99) is 85, above the warrior
	// table's 27 at level 30, so the victim makes the save.
	dprng.ResetStream(2)
	combat.SetRoller(fixedNumberRoller{value: 50})
	spells.CastFromSpecial(caster, victim, spells.SpellTeleport, 30, w)

	if got := casterSink.String(); !strings.HasPrefix(got, "The magic words fail to form properly.\r\n") {
		t.Fatalf("caster bytes = %q, want the save-refusal line first", got)
	}
	if !strings.Contains(room.String(), "stalker") && victim.GetFightingBody() == nil {
		t.Fatal("save refusal did not counterattack: victim never enrolled")
	}
	if victim.GetFightingBody() != caster {
		t.Fatalf("victim fighting = %v, want the caster (hit(victim, ch))", victim.GetFightingBody())
	}
	if caster.GetFightingBody() != victim {
		t.Fatalf("caster fighting = %v, want the victim (damage() enrolment)", caster.GetFightingBody())
	}
}

// TestSpellTeleportLevelRefusalNeverTeleports pins spells.c:189-194: when the
// save fails and a below-immortal caster's NPC victim is within two levels,
// the spell refuses with "The magic fails utterly!", counterattacks, and the
// victim never leaves the room.
func TestSpellTeleportLevelRefusalNeverTeleports(t *testing.T) {
	w, caster, victim, _ := teleportRefusalWorld(t)
	casterSink := &strings.Builder{}
	w.MobileMessageSink = func(body *MobInstance, msg []byte) {
		if body == caster {
			casterSink.Write(msg)
		}
	}
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == caster {
			return &Player{Name: "Switchgod"}
		}
		return nil
	}

	// ResetStream(5)'s first Number(0,99) is 13, at or below the save's 27:
	// the save fails and the level gate (30-30 < 3) refuses.
	dprng.ResetStream(5)
	combat.SetRoller(fixedNumberRoller{value: 5})
	roomBefore := victim.GetRoomVNum()
	spells.CastFromSpecial(caster, victim, spells.SpellTeleport, 30, w)

	if got := casterSink.String(); !strings.HasPrefix(got, "The magic fails utterly!\r\n") {
		t.Fatalf("caster bytes = %q, want the level-refusal line first", got)
	}
	if victim.GetRoomVNum() != roomBefore {
		t.Fatalf("level-refused victim teleported to room %d, want %d", victim.GetRoomVNum(), roomBefore)
	}
	if victim.GetFightingBody() != caster {
		t.Fatal("level refusal did not counterattack")
	}
}

type fixedNumberRoller struct{ value int }

func (r fixedNumberRoller) Number(_, _ int) int     { return r.value }
func (r fixedNumberRoller) Dice(num, sides int) int { return num * 1 }
func (r fixedNumberRoller) IntN(int) int            { return 0 }

// TestCanSeeIgnoresAFFHide pins C's CAN_SEE (src/utils.h:521-530): AFF_HIDE
// is not part of the predicate — a hidden body remains visible and
// targetable. The room listing keeps its own explicit hide filter
// (src/act.informative.c:637-643); only that surface suppresses hidden
// bodies. Taught by the teleport_victim vehicle: the dune stalker carries
// AFF_HIDE in its prototype and the port could not force, stat or hit it.
func TestCanSeeIgnoresAFFHide(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Pit"})
	observer := NewPlayer(1, "Watcher", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	mob := NewMob(&parser.Mob{VNum: 3001, Keywords: "stalker", ShortDesc: "a stalker"}, 1001)
	mob.SetAffected(affHide)
	w.mu.Lock()
	w.activeMobs[mob.GetID()] = mob
	w.mu.Unlock()

	if !canSee(observer, mob) {
		t.Fatal("canSee suppressed an AFF_HIDE body; C's CAN_SEE does not test AFF_HIDE")
	}
	if _, ok := w.resolveCharInRoomAt(observer, 1001, "stalker"); !ok {
		t.Fatal("hidden body is not targetable; C get_char_room_vis finds it (src/handler.c:1277-1300)")
	}
}

// TestNPCSocialCharLineReachesSwitchedBody pins do_action's char-directed
// lines for a mobile (src/act.social.c:35-73): scoff is a no-char_found
// social, so the special's named argument takes the no-arg branch and the
// actor's own "You scoff at the idea." goes to send_to_char — delivered to a
// switched body's wizard through MobInstance.SendMessage's descriptor
// routing, silent for a descriptorless mobile.
func TestNPCSocialCharLineReachesSwitchedBody(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Pit"})
	w.mu.Lock()
	w.mobs[3001] = &parser.Mob{VNum: 3001, Keywords: "gil", ShortDesc: "Gil", Level: 30}
	w.mu.Unlock()
	mob, err := w.SpawnMob(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	var mobileOutput strings.Builder
	w.MobileMessageSink = func(body *MobInstance, msg []byte) {
		if body == mob {
			mobileOutput.Write(msg)
		}
	}
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == mob {
			return &Player{Name: "Switchgod"}
		}
		return nil
	}

	w.NPCCommandDirect(mob, "scoff", "stalker")

	if got, want := mobileOutput.String(), "You scoff at the idea.\r\n"; got != want {
		t.Fatalf("switched-body char line = %q, want %q", got, want)
	}
}

// TestMobSendMessageSilentWithoutDescriptor pins the other half: a
// descriptorless mobile's SendMessage stays a no-op (C send_to_char with
// ch->desc == NULL), so engine noise never reaches nobody.
func TestMobSendMessageSilentWithoutDescriptor(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Pit"})
	w.mu.Lock()
	w.mobs[3001] = &parser.Mob{VNum: 3001, Keywords: "gil", ShortDesc: "Gil", Level: 30}
	w.mu.Unlock()
	mob, err := w.SpawnMob(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	var delivered int
	w.MobileMessageSink = func(_ *MobInstance, _ []byte) { delivered++ }
	// No MobileDescriptorOwner: the body has no descriptor.
	mob.SendMessage("You should not see this.\r\n")
	if delivered != 0 {
		t.Fatalf("descriptorless mobile delivered %d messages", delivered)
	}
}

// TestSpellCounterattackSeamContract is the loud half of the retaliation
// wiring: the production world MUST satisfy the counterattack seam, or every
// spell save-refusal hit() silently vanishes again (the npcRetaliate class
// bug this contract guards against).
var _ interface {
	ExecuteSpellCounterattack(victim, caster interface{})
} = (*World)(nil)
