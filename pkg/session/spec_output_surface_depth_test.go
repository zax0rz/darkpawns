package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func addSpecOutputPeer(t *testing.T, w *game.World, name string) *game.Player {
	t.Helper()
	peer := game.NewPlayer(2, name, specProbeRoom)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	return peer
}

func captureSpecOutput(w *game.World) map[string]string {
	messages := make(map[string]string)
	w.MessageSink = func(name string, message []byte) { messages[name] += string(message) }
	return messages
}

func TestSpecRechargerOutputThroughCharacterDispatch(t *testing.T) {
	m, s := specProbeSession(t, []parser.Mob{{VNum: 8, Keywords: "recharger", ShortDesc: "the recharger", DefaultPos: combat.PosStanding}})
	if _, err := m.world.SpawnMob(8, specProbeRoom); err != nil {
		t.Fatalf("SpawnMob recharger: %v", err)
	}
	addSpecOutputPeer(t, m.world, "Specpeer")
	messages := captureSpecOutput(m.world)
	reset := func() {
		clear(messages)
	}
	issue := func(command string, args ...string) {
		t.Helper()
		if err := ExecuteCommand(s, command, args); err != nil {
			t.Fatalf("ExecuteCommand %s %v: %v", command, args, err)
		}
	}

	listOutput := "The recharger sighs loudly.\r\n" +
		"The recharger says 'I recharge wands and staves, of course!'\r\n" +
		"The recharger says 'My price is 100 coins per spell level per charge..'\r\n" +
		"The recharger says 'So a wand casting heal at level 25 would cost 2500 coins per charge.'\r\n" +
		"The recharger says 'If it had 10 max charges at first, it would have 9 max after charging.\r\n" +
		"The recharger says 'That means that an item could have 1 charge maximum, with 2 charges remaining (after a recharge)'\r\n" +
		"The recharger says 'To recharge an item type: recharge <staff or wand>.'\r\n"
	issue("list")
	if got, want := messages[s.player.GetName()], listOutput+"The recharger stares expectantly at you.\r\n"; got != want {
		t.Fatalf("list actor bytes = %q, want %q", got, want)
	}
	if got, want := messages["Specpeer"], listOutput+"The recharger stares expectantly at Specprobe.\r\n"; got != want {
		t.Fatalf("list peer bytes = %q, want %q", got, want)
	}

	s.player.SetPosition(combat.PosResting)
	reset()
	issue("help")
	if got, want := messages[s.player.GetName()], listOutput+"The recharger stares expectantly at you.\r\n"; got != want {
		t.Fatalf("help actor bytes = %q, want %q", got, want)
	}

	s.player.SetPosition(combat.PosStanding)
	reset()
	issue("recharge")
	if got, want := messages[s.player.GetName()], "The recharger tells you, 'You don't have that!'\r\nThe recharger whaps you upside the head.\r\n"; got != want {
		t.Fatalf("missing-item actor bytes = %q, want %q", got, want)
	}
	if got, want := messages["Specpeer"], "The recharger whaps Specprobe upside the head.\r\n"; got != want {
		t.Fatalf("missing-item peer bytes = %q, want %q", got, want)
	}

	addObject := func(vnum int, keyword, shortDesc string, itemType int, values [4]int) *game.ObjectInstance {
		t.Helper()
		item := game.NewObjectInstance(&parser.Obj{VNum: vnum, Keywords: keyword, ShortDesc: shortDesc, TypeFlag: itemType, Values: values}, -1)
		if err := s.player.Inventory.AddItem(item); err != nil {
			t.Fatalf("AddItem %s: %v", keyword, err)
		}
		return item
	}
	rock := addObject(991101, "rock", "a rock", game.ITEM_LIGHT, [4]int{})
	reset()
	issue("recharge", "rock")
	if got, want := messages[s.player.GetName()], "The recharger tells you, 'Ummm... does that look like a wand or staff to you?'\r\nThe recharger shakes his head in disgust.\r\n"; got != want {
		t.Fatalf("wrong-type actor bytes = %q, want %q", got, want)
	}
	if got, want := messages["Specpeer"], "The recharger shakes his head in disgust.\r\n"; got != want {
		t.Fatalf("wrong-type peer bytes = %q, want %q", got, want)
	}
	s.player.Inventory.RemoveItem(rock)

	wand := addObject(991102, "wand", "a recharge wand", game.ITEM_WAND, [4]int{5, 10, 3, 0})
	s.player.SetGold(0)
	reset()
	issue("recharge", "wand")
	if got, want := messages[s.player.GetName()], "The recharger tells you, 'You don't have enough gold!'\r\n"; got != want {
		t.Fatalf("insufficient-gold actor bytes = %q, want %q", got, want)
	}
	if got := messages["Specpeer"]; got != "" {
		t.Fatalf("insufficient-gold peer bytes = %q, want empty", got)
	}

	s.player.SetGold(500)
	reset()
	issue("recharge", "wand")
	if got, want := messages[s.player.GetName()], "The recharger waves his hands around and chants strange things.\r\nThe item now has 4 charges remaining.\r\n"; got != want {
		t.Fatalf("success actor bytes = %q, want %q", got, want)
	}
	if got, want := messages["Specpeer"], "The recharger waves his hands around and chants strange things.\r\n"; got != want {
		t.Fatalf("success peer bytes = %q, want %q", got, want)
	}
	if wand.GetValue(1) != 9 || wand.GetValue(2) != 4 {
		t.Fatalf("success values = max %d/current %d, want max 9/current 4", wand.GetValue(1), wand.GetValue(2))
	}

	full := addObject(991103, "staff", "a full staff", game.ITEM_STAFF, [4]int{5, 10, 10, 0})
	reset()
	issue("recharge", "staff")
	if got, want := messages[s.player.GetName()], "The recharger tells you, 'You don't have enough gold!'\r\n"; got != want {
		t.Fatalf("funds precede full check = %q, want %q", got, want)
	}
	if s.player.GetGold() != 0 || full.GetValue(1) != 10 || full.GetValue(2) != 10 {
		t.Fatal("refused recharge mutated money or charges")
	}
	s.player.SetGold(500)
	reset()
	issue("recharge", "staff")
	if got, want := messages[s.player.GetName()], "The recharger tells you, 'The item does not need recharging, stop wasting my time!'\r\n"; got != want {
		t.Fatalf("full-item actor bytes = %q, want %q", got, want)
	}
	if got := messages["Specpeer"]; got != "" {
		t.Fatalf("full-item peer bytes = %q, want empty", got)
	}
}

func TestSpecBeholderCommandOutputThroughCharacterDispatch(t *testing.T) {
	m, s := specProbeSession(t, []parser.Mob{{VNum: 11011, Keywords: "beholder", ShortDesc: "the beholder", DefaultPos: combat.PosStanding}})
	if _, err := m.world.SpawnMob(11011, specProbeRoom); err != nil {
		t.Fatalf("SpawnMob beholder: %v", err)
	}
	addSpecOutputPeer(t, m.world, "Specpeer")
	messages := captureSpecOutput(m.world)
	for _, command := range []string{"cast", "recite"} {
		clear(messages)
		if err := ExecuteCommand(s, command, []string{"invisibility"}); err != nil {
			t.Fatalf("ExecuteCommand %s: %v", command, err)
		}
		if got, want := messages[s.player.GetName()], "A ray of light shoots out from one of the beholder's eyestalks, breaking your concentration!\r\n"; got != want {
			t.Fatalf("%s actor bytes = %q, want %q", command, got, want)
		}
		if got, want := messages["Specpeer"], "A ray of light shoots out from one of the beholder's eyestalks, hitting Specprobe!\r\n"; got != want {
			t.Fatalf("%s peer bytes = %q, want %q", command, got, want)
		}
	}

	clear(messages)
	if err := ExecuteCommand(s, "look", nil); err != nil {
		t.Fatalf("ExecuteCommand look: %v", err)
	}
	if got := messages[s.player.GetName()]; !strings.Contains(got, "Probe Room") || strings.Contains(got, "ray of light") {
		t.Fatalf("ordinary command did not pass the beholder special: %q", got)
	}
}

func TestZenMasterOutputThroughCombatSpecialDispatch(t *testing.T) {
	tests := []struct {
		name      string
		seed      uint32
		actorLine string
		peerLine  string
	}{
		{
			name:      "arm-touch recall branch",
			seed:      71,
			actorLine: "The zen master touches you lightly on the arm.\r\n",
			peerLine:  "The zen master touches Specprobe on the arm.\r\n",
		},
		{
			name:      "violence thought teleport branch",
			seed:      3,
			actorLine: "The zen master says, 'You have violence, but not thought.'\r\n",
			peerLine:  "The zen master says, 'You have violence, but not thought.'\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, s := specProbeSession(t, []parser.Mob{{
				VNum: 18219, Keywords: "zen master", ShortDesc: "the zen master",
				DefaultPos: combat.PosStanding, Level: 1,
			}})
			m.combatEngine.Stop()
			mob, err := m.world.SpawnMob(18219, specProbeRoom)
			if err != nil {
				t.Fatalf("SpawnMob zen master: %v", err)
			}
			mob.SetMobFlag(game.MobFlagSpec)
			mob.SetMaxHP(100000)
			mob.CurrentHP = 100000
			s.player.SetMaxHP(100000)
			s.player.SetHP(100000)
			addSpecOutputPeer(t, m.world, "Specpeer")
			messages := captureSpecOutput(m.world)

			m.WireCombatCallbacks()
			m.SetCombatBroadcastFunc()
			if err := m.SetCombatMessageFunc(); err != nil {
				t.Fatalf("SetCombatMessageFunc: %v", err)
			}
			m.SetMobSpecialFunc()
			dprng.ResetStream(tt.seed)
			t.Cleanup(func() { dprng.ResetStream(1) })
			if err := m.combatEngine.StartCombat(mob, s.player); err != nil {
				t.Fatalf("StartCombat: %v", err)
			}
			m.combatEngine.PerformRound()

			if got := messages[s.player.GetName()]; !strings.HasPrefix(got, tt.actorLine) {
				t.Fatalf("actor bytes = %q, want prefix %q", got, tt.actorLine)
			}
			if got := messages["Specpeer"]; !strings.HasPrefix(got, tt.peerLine) {
				t.Fatalf("peer bytes = %q, want prefix %q", got, tt.peerLine)
			}
		})
	}
}

func TestDoDisarmEntryOutputThroughCharacterDispatch(t *testing.T) {
	m, s := specProbeSession(t, nil)
	peer := addSpecOutputPeer(t, m.world, "Specpeer")
	_ = peer
	disarm := func(args ...string) string {
		t.Helper()
		if err := ExecuteCommand(s, "disarm", args); err != nil {
			t.Fatalf("ExecuteCommand disarm %v: %v", args, err)
		}
		return readMsgText(t, s)
	}

	if got, want := disarm(), "You'd better leave all the martial arts to fighters.\r\n"; got != want {
		t.Fatalf("no-skill bytes = %q, want %q", got, want)
	}
	s.player.SetSkill(game.SkillDisarm, 100)
	if got, want := disarm(), "Disarm who?\r\n"; got != want {
		t.Fatalf("no-argument bytes = %q, want %q", got, want)
	}
	if got, want := disarm(s.player.GetName()), "Just try removing your weapon instead.\r\n"; got != want {
		t.Fatalf("self-target bytes = %q, want %q", got, want)
	}

	weapon := game.NewObjectInstance(&parser.Obj{
		VNum: 991104, Keywords: "sword", ShortDesc: "a test sword", TypeFlag: game.ITEM_WEAPON,
		WearFlags: [4]int{1 << 13}, Values: [4]int{1, 5, 0, 0},
	}, -1)
	if err := peer.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("AddItem peer weapon: %v", err)
	}
	if err := peer.Equipment.Equip(weapon, peer.Inventory); err != nil {
		t.Fatalf("Equip peer weapon: %v", err)
	}
	if got, want := disarm(peer.GetName()), "You can't disarm them if you aren't fighting them!\r\n"; got != want {
		t.Fatalf("not-fighting bytes = %q, want %q", got, want)
	}
}
