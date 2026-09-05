package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdHitNoArgumentMessage(t *testing.T) {
	m := makeGateTestManager(t, false)
	s := makeGateSession(t, m, 1, "Hero", 20)

	if err := cmdHit(s, nil); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, s), "Hit who?\r\n"; got != want {
		t.Errorf("cmdHit no-argument message = %q, want %q", got, want)
	}
}

func TestCombatMessageConsumesPulseInterruptionPrefix(t *testing.T) {
	m := makeGateTestManager(t, false)
	s := makeGateSession(t, m, 1, "Hero", 20)
	m.WireCombatCallbacks()
	m.SetCombatMessageFunc()

	s.promptInvalidated.Store(true)
	m.combatEngine.Callbacks.Broadcast(1001, "A combat line.", "")
	if got, want := readSendText(t, s), "\r\nA combat line."; got != want {
		t.Fatalf("combat interruption prefix = %q, want %q", got, want)
	}

	s.player.SendMessage("The next text line.\r\n")
	if got, want := readSendText(t, s), "The next text line.\r\n"; got != want {
		t.Fatalf("ordinary message after combat prefix = %q, want %q", got, want)
	}
}

func TestCmdHitNotFoundMessage(t *testing.T) {
	m := makeGateTestManager(t, false)
	s := makeGateSession(t, m, 1, "Hero", 20)

	if err := cmdHit(s, []string{"nobody"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, s), "They don't seem to be here.\r\n"; got != want {
		t.Errorf("cmdHit not-found message = %q, want %q", got, want)
	}
}

func TestCmdHitSelfUsesActForActorAndRoom(t *testing.T) {
	m := makeGateTestManager(t, false)
	actor := makeGateSession(t, m, 1, "Hero", 20)
	observer := makeGateSession(t, m, 2, "Observer", 20)

	if err := cmdHit(actor, []string{"self"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, actor), "You hit yourself...OUCH!.\r\n"; got != want {
		t.Errorf("self-hit actor message = %q, want %q", got, want)
	}
	if got, want := readSendText(t, observer), "Hero hits himself, and says OUCH!\r\n"; got != want {
		t.Errorf("self-hit room message = %q, want %q", got, want)
	}
}

func TestCmdHitCharmedMasterUsesActPronouns(t *testing.T) {
	m := makeGateTestManager(t, false)
	actor := makeGateSession(t, m, 1, "Thrall", 20)
	master := makeGateSession(t, m, 2, "Master", 20)
	actor.player.AddAffect(&engine.Affect{Flags: engine.AFFCharm})
	actor.player.SetFollowing(master.player.GetName())

	if err := cmdHit(actor, []string{"master"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, actor), "Master is just such a good friend, you simply can't hit him.\r\n"; got != want {
		t.Errorf("charm-friend message = %q, want %q", got, want)
	}
}

func TestCmdHitNonStandingDoesBestItCan(t *testing.T) {
	m := makeGateTestManager(t, false)
	if _, err := m.world.SpawnMob(5000, 1001); err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Hero", 20)
	s.player.SetPosition(combat.PosFighting)

	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, s), "You do the best you can!\r\n"; got != want {
		t.Errorf("non-standing hit message = %q, want %q", got, want)
	}
}

func TestCmdHitCurrentOpponentDoesBestItCan(t *testing.T) {
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Hero", 20)
	if err := m.combatEngine.StartCombat(s.player, mob); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, s), "You do the best you can!\r\n"; got != want {
		t.Errorf("current-opponent hit message = %q, want %q", got, want)
	}
}

func TestCmdHitShopkeeperGate(t *testing.T) {
	m := makeGateTestManager(t, false)
	m.world.SetShopManager(m.shopManager)
	m.shopManager.AddShop(&game.Shop{KeeperVNum: 5000, KeeperName: "Test Shop", RoomVNum: 1001})
	if _, err := m.world.SpawnMob(5000, 1001); err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Hero", 20)

	if err := cmdHit(s, []string{"target"}); err != nil {
		t.Fatalf("cmdHit returned error: %v", err)
	}
	if got, want := readSendText(t, s), "Ha ha... Don't think so.\r\n"; got != want {
		t.Errorf("shopkeeper gate message = %q, want %q", got, want)
	}
	if m.combatEngine.IsFighting(s.player.GetName()) {
		t.Error("shopkeeper gate should not leave the attacker fighting")
	}
}

func TestCmdHitResolvesFirstSwingSynchronouslyWithoutInventedAttackLine(t *testing.T) {
	m := makeGateTestManager(t, false)
	defer m.combatEngine.Stop()
	m.WireCombatCallbacks()
	m.SetCombatMessageFunc()
	if _, err := m.world.SpawnMob(5000, 1001); err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	s := makeGateSession(t, m, 1, "Hero", 20)

	roller := combat.NewScriptedRoller([]int{1, 2, 99}) // natural-one miss, second C-ordered TYPE_HIT variant
	combat.WithRoller(roller, func() {
		if err := cmdHit(s, []string{"target"}); err != nil {
			t.Fatalf("cmdHit returned error: %v", err)
		}
	})

	if got, want := readSendText(t, s), "You try to hit a test target who easily avoids the blow."; got != want {
		t.Fatalf("synchronous first-swing message = %q, want %q", got, want)
	}
	if got := s.player.GetWaitState(); got != 3*engine.PULSE_VIOLENCE {
		t.Fatalf("post-hit wait state = %d, want %d (3 rounds → 3*PULSE_VIOLENCE pulses)", got, 3*engine.PULSE_VIOLENCE)
	}
	if got := roller.Index; got != 2 {
		t.Fatalf("first-swing draws = %d, want 2 (to-hit + message selection)", got)
	}
}

func TestCmdKillMortalDelegatesBeforeArgumentParsing(t *testing.T) {
	m := makeGateTestManager(t, false)
	s := makeGateSession(t, m, 1, "Hero", 20)

	if err := cmdKill(s, nil); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}
	if got, want := readSendText(t, s), "Hit who?\r\n"; got != want {
		t.Errorf("mortal cmdKill no-argument message = %q, want %q", got, want)
	}
}

func TestCmdKillImplementorMessages(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no argument", want: "Kill who?\r\n"},
		{name: "not found", args: []string{"nobody"}, want: "They aren't here.\r\n"},
		{name: "self", args: []string{"self"}, want: "Your mother would be so sad.. :(\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeGateTestManager(t, false)
			s := makeGateSession(t, m, 1, "Impl", 40)

			if err := cmdKill(s, tt.args); err != nil {
				t.Fatalf("cmdKill returned error: %v", err)
			}
			if got := readSendText(t, s); got != tt.want {
				t.Errorf("cmdKill message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCmdKillEqualLevelPreservesTrailingSpace(t *testing.T) {
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetLevel(40)
	s := makeGateSession(t, m, 1, "Impl", 40)

	if err := cmdKill(s, []string{"target"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}
	if got, want := readSendText(t, s), "No can do, buddy.. \r\n"; got != want {
		t.Errorf("equal-level kill message = %q, want %q", got, want)
	}
}

func TestCmdKillUsesActForChopTrio(t *testing.T) {
	m := makeGateTestManager(t, false)
	killer := makeGateSession(t, m, 1, "Killer", 40)
	victim := makeGateSession(t, m, 2, "Victim", 10)
	observer := makeGateSession(t, m, 3, "Observer", 20)

	if err := cmdKill(killer, []string{"victim"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}
	if got, want := readSendText(t, killer), "You chop him to pieces!  Ah!  The blood!\r\n"; got != want {
		t.Errorf("killer chop message = %q, want %q", got, want)
	}
	if got, want := readSendText(t, victim), "Killer chops you to pieces!\r\n"; got != want {
		t.Errorf("victim chop message = %q, want %q", got, want)
	}
	if got, want := readSendText(t, observer), "Killer brutally slays Victim!\r\n"; got != want {
		t.Errorf("room chop message = %q, want %q", got, want)
	}
	// C raw_kill's death_cry (fight.c:558-577) reaches the room after the
	// chop trio — the killer and every observer hear the freeze line.
	if got, want := readSendText(t, killer), "Your blood freezes as you hear Victim's death cry.\r\n"; got != want {
		t.Errorf("killer death cry = %q, want %q", got, want)
	}
	if got, want := readSendText(t, observer), "Your blood freezes as you hear Victim's death cry.\r\n"; got != want {
		t.Errorf("observer death cry = %q, want %q", got, want)
	}
	// The chop line is the victim's ENTIRE death transcript: C defers the
	// extraction (and the return to the menu) to the next heartbeat, so no
	// invented resurrection bytes may reach them (R4), and nothing moves
	// them out of the room synchronously.
	select {
	case msg := <-victim.send:
		t.Errorf("victim received invented post-death bytes: %s", string(msg))
	default:
	}
	if victim.player.GetRoom() != 1001 {
		t.Errorf("victim moved to room %d; C defers extraction to the next heartbeat", victim.player.GetRoom())
	}
}

func TestCmdKillReturnsConnectedVictimToMenuOnHeartbeat(t *testing.T) {
	m := makeGateTestManager(t, false)
	killer := makeGateSession(t, m, 1, "Killer", 40)
	victim := makeGateSession(t, m, 2, "Victim", 10)

	if err := cmdKill(killer, []string{"victim"}); err != nil {
		t.Fatalf("cmdKill returned error: %v", err)
	}
	_ = drainMsg(t, killer)
	_ = drainMsg(t, victim)

	if _, ok := m.world.GetPlayer("Victim"); !ok {
		t.Fatal("victim was extracted synchronously; C defers extraction to heartbeat")
	}
	m.ExtractPendingChars()

	if _, ok := m.world.GetPlayer("Victim"); ok {
		t.Fatal("victim remains in world after deferred extraction")
	}
	_, prompt := unmarshalCharCreate(t, drainMsg(t, victim))
	if prompt.Stage != "menu" || prompt.Prompt != menuText {
		t.Fatalf("post-death prompt = (%q, %q), want menu text", prompt.Stage, prompt.Prompt)
	}
}

func TestCmdAssistMobHelpeeUsesPersAndHitGates(t *testing.T) {
	m := makeGateTestManager(t, false)
	mob, err := m.world.SpawnMob(5000, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	helper := makeGateSession(t, m, 1, "Helper", 1)
	helpee := makeGateSession(t, m, 2, "Helpee", 20)
	observer := makeGateSession(t, m, 3, "Observer", 20)
	if err := m.combatEngine.StartCombat(helpee.player, mob); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}
	if err := cmdAssist(helper, []string{"target"}); err != nil {
		t.Fatalf("cmdAssist returned error: %v", err)
	}
	if got, want := readSendText(t, helper), "You join the fight!\r\n"; got != want {
		t.Errorf("helper join output = %q, want %q", got, want)
	}
	if got, want := readSendText(t, helper), "You are not experienced enough to attack Helpee!\r\n"; got != want {
		t.Errorf("helper hit-gate output = %q, want %q", got, want)
	}
	for _, session := range []*Session{helpee, observer} {
		if got, want := readSendText(t, session), "Helper assists a test target.\r\n"; got != want {
			t.Errorf("room output = %q, want %q", got, want)
		}
	}
	if m.combatEngine.IsFighting(helper.player.Name) {
		t.Error("low-level helper should not be enrolled after hit() gate")
	}
}
