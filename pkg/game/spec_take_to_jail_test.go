package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecTakeToJail_EntryGatesAndOutlawWarning(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 20)
	_ = lastMsg()
	player.SetPlrFlag(PlrOutlaw, true)

	if specTakeToJail(w, player, guard, "look", "") {
		t.Fatal("take_to_jail should reject command dispatch")
	}
	guard.SetPosition(combat.PosSleeping)
	if specTakeToJail(w, nil, guard, "", "") {
		t.Fatal("take_to_jail should reject a sleeping mob")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("gated call emitted %q", got)
	}

	guard.SetPosition(combat.PosStanding)
	engine := &cityguardTestCombatEngine{}
	w.SetCombatEngine(engine)
	specTakeToJail(w, nil, guard, "", "")

	got := lastMsg()
	if !strings.Contains(got, "A test mob says 'We don't like OUTLAWS like you in this city!'") {
		t.Fatalf("outlaw warning = %q", got)
	}
	if strings.Contains(got, ", 'We don't like") {
		t.Fatal("take_to_jail warning retained cityguard's comma")
	}
	if got, want := len(engine.starts), 1; got != want {
		t.Fatalf("combat starts = %d, want %d", got, want)
	}
	if got, want := engine.starts[0][1], player.GetName(); got != want {
		t.Fatalf("combat target = %q, want %q", got, want)
	}
}

func TestSpecTakeToJail_ReturnContractAndProtectionDelegation(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 20)
	_ = lastMsg()

	if specTakeToJail(w, nil, guard, "", "") {
		t.Fatal("take_to_jail should fall through without an eligible target")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("fallthrough emitted %q", got)
	}

	protected := NewPlayer(2, "Protected", player.GetRoomVNum())
	protected.SetAlignment(100)
	if err := w.AddPlayer(protected); err != nil {
		t.Fatalf("AddPlayer protected: %v", err)
	}
	attacker := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	{
		p := *attacker.Proto()
		p.Alignment = -500
		attacker.SetProto(&p)
	}
	attacker.SetFightingBody(protected)
	engine := &cityguardTestCombatEngine{}
	w.SetCombatEngine(engine)
	specTakeToJail(w, nil, guard, "", "")

	if got := lastMsg(); !strings.Contains(got, "You just pissed me off") {
		t.Fatalf("protection warning = %q", got)
	}
	if got, want := len(engine.starts), 1; got != want {
		t.Fatalf("protection combat starts = %d, want %d", got, want)
	}
}

func TestSpecTakeToJailAlignmentInterventionOutput(t *testing.T) {
	for _, test := range []struct {
		name              string
		defenderAlignment int
		wantOutput        string
		wantCombat        bool
	}{
		{
			name:              "good-aligned target",
			defenderAlignment: 0,
			wantOutput:        "A city guard says, 'You just pissed me off, Evil!'\r\n",
			wantCombat:        true,
		},
		{name: "evil-aligned target", defenderAlignment: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewWorld(&parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "City Street"}},
				Mobs: []parser.Mob{
					{
						VNum:        8001,
						Keywords:    "city guard",
						ShortDesc:   "a city guard",
						LongDesc:    "A city guard is here.\r\n",
						Level:       20,
						Race:        1,
						Alignment:   0,
						HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
						ActionFlags: []string{"spec"},
					},
					{
						VNum:      99001,
						Keywords:  "fighter",
						ShortDesc: "a fighter",
						LongDesc:  "A fighter is here.\r\n",
						Level:     10,
						Race:      1,
						Alignment: test.defenderAlignment,
						HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
					},
				},
			})
			if err != nil {
				t.Fatalf("NewWorld: %v", err)
			}
			t.Cleanup(func() { w.StopAITicker() })
			if name, assigned := GetMobSpecName(8001); !assigned || name != "take_to_jail" {
				t.Fatalf("mob 8001 special = %q, assigned %v; want take_to_jail", name, assigned)
			}

			outputs := make(map[string]string)
			w.MessageSink = func(name string, message []byte) { outputs[name] += string(message) }
			evil := NewPlayer(1, "Evil", 1001)
			evil.SetAlignment(-100)
			evil.SetPosition(combat.PosFighting)
			witness := NewPlayer(2, "Witness", 1001)
			if err := w.AddPlayer(evil); err != nil {
				t.Fatalf("AddPlayer evil: %v", err)
			}
			if err := w.AddPlayer(witness); err != nil {
				t.Fatalf("AddPlayer witness: %v", err)
			}
			guard, err := w.SpawnMob(8001, 1001)
			if err != nil {
				t.Fatalf("SpawnMob guard: %v", err)
			}
			defender, err := w.SpawnMob(99001, 1001)
			if err != nil {
				t.Fatalf("SpawnMob defender: %v", err)
			}
			evil.SetFightingBody(defender)
			defender.SetFightingBody(evil)
			defender.SetPosition(combat.PosFighting)

			engine := &cityguardTestCombatEngine{}
			w.SetCombatEngine(engine)
			w.MobileActivity()

			for _, recipient := range []string{"Evil", "Witness"} {
				if got := outputs[recipient]; got != test.wantOutput {
					t.Errorf("%s bytes = %q, want %q", recipient, got, test.wantOutput)
				}
			}
			if got := len(engine.starts); (got == 1) != test.wantCombat {
				t.Errorf("combat starts = %d, want combat %v", got, test.wantCombat)
			}
			if test.wantCombat && engine.starts[0] != [2]string{guard.GetName(), evil.GetName()} {
				t.Errorf("combat pair = %v, want guard against evil player", engine.starts[0])
			}
		})
	}
}

func TestSpecTakeToJailSubdueStateAndAudience(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Street"},
		{VNum: 8118, Name: "The Main Holding Cell", Description: "A plain cell."},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	transcript := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { transcript[name] += string(msg) }
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(5)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	mountProto := &parser.Mob{
		VNum:        99002,
		Keywords:    "horse pony",
		ShortDesc:   "a horse",
		ActionFlags: []string{"mountable"},
	}
	w.mu.Lock()
	w.mobs[mountProto.VNum] = mountProto
	w.mu.Unlock()
	mount, err := w.spawnMob(mountProto.VNum, 1001)
	if err != nil {
		t.Fatalf("spawn mount: %v", err)
	}
	victim.MountName = mount.GetName()
	victim.SetAffect(affMounted, true)
	mount.SetAffected(affMounted)
	mount.SetMountRider(victim.Name)
	mount.SetFollowingBody(victim)
	guardProto := &parser.Mob{
		VNum:      99001,
		Keywords:  "gateguard",
		ShortDesc: "Hans the gateguard",
		Level:     20,
		HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
	}
	w.mu.Lock()
	w.mobs[guardProto.VNum] = guardProto
	w.mu.Unlock()
	guard, err := w.spawnMob(guardProto.VNum, 1001)
	if err != nil {
		t.Fatalf("spawn guard: %v", err)
	}
	victim.SetFightingBody(guard)
	guard.SetFightingBody(victim)
	guard.SetHunting(victim.GetName())

	callbacks := w.WireCombatCallbacks()
	if !callbacks.JailGuardSubdue(guard, victim) {
		t.Fatal("jail guard callback should subdue the victim")
	}

	want := "Hans the gateguard grabs you by the collar and quickly beats you into submission.\r\n" +
		"Jerking you to your feet, he carts you off to jail...\r\n" +
		"The Main Holding Cell\r\nA plain cell.\r\n[ Exits: None! ]\r\n"
	if got := transcript[victim.GetName()]; got != want {
		t.Fatalf("victim transcript = %q, want %q", got, want)
	}
	if got := victim.GetRoomVNum(); got != 8118 {
		t.Fatalf("victim room = %d, want 8118", got)
	}
	if got := victim.GetHP(); got != 1 {
		t.Fatalf("victim hp = %d, want 1", got)
	}
	if got := victim.JailTimer; got != 2 {
		t.Fatalf("jail timer = %d, want 2", got)
	}
	if victim.IsFighting() || guard.IsFighting() {
		t.Fatal("jail subdue should clear reciprocal combat")
	}
	if guard.GetHunting() != "" {
		t.Fatalf("guard hunting target = %q, want empty", guard.GetHunting())
	}
	if victim.IsAffected(affMounted) || victim.GetMountName() != "" || mount.IsAffected(affMounted) {
		t.Fatalf("jail subdue left mounted affects/cache: rider=%t name=%q mount=%t", victim.IsAffected(affMounted), victim.GetMountName(), mount.IsAffected(affMounted))
	}
	if mount.GetMountRider() != victim.Name || mount.GetFollowing() != victim.Name || w.FollowingBody(mount) != victim {
		t.Fatalf("jail unmount changed retained master/follower links: rider=%q following=%q body=%p", mount.GetMountRider(), mount.GetFollowing(), w.FollowingBody(mount))
	}
}

func TestTakeToJailBreedCallerAndRoomOrder(t *testing.T) {
	for _, outlawFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(outlawFirst), func(t *testing.T) {
			w, player, output := newSpecProcTestWorld(t)
			guard := newSpecProcTestMob(t, w, 1001, 20)
			victim := newSpecProcTestMob(t, w, 1001, 1)
			proto := *victim.Proto()
			proto.ShortDesc = "the vampire"
			victim.SetProto(&proto)
			victim.SetAffected(affVampire)
			player.SetPlrFlag(PlrOutlaw, true)
			if outlawFirst {
				if err := w.PlayerTransfer(player, 1001); err != nil {
					t.Fatal(err)
				}
			}
			engine := &cityguardTestCombatEngine{}
			w.SetCombatEngine(engine)
			output()
			handled := specTakeToJail(w, nil, guard, "", "")
			got := output()
			want := victim.GetName()
			if outlawFirst {
				want = player.GetName()
				if strings.Contains(got, "nightbreed") || !strings.Contains(got, "OUTLAWS") {
					t.Fatalf("first outlaw did not keep priority: %q", got)
				}
			} else {
				if !handled || !strings.Contains(got, "Die, nightbreed!!") || strings.Contains(got, "OUTLAWS") {
					t.Fatalf("earlier breed handoff did not consume before later outlaw: handled=%t output=%q", handled, got)
				}
			}
			if len(engine.starts) != 1 || engine.starts[0][1] != want {
				t.Fatalf("selected combat starts=%v, want only %s", engine.starts, want)
			}
		})
	}
}

func TestTakeToJailBreedCallerEntryGates(t *testing.T) {
	for _, sleeping := range []bool{false, true} {
		w, _, output := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, 1001, 20)
		victim := newSpecProcTestMob(t, w, 1001, 1)
		victim.SetAffected(affVampire)
		engine := &cityguardTestCombatEngine{}
		w.SetCombatEngine(engine)
		cmd := "look"
		if sleeping {
			guard.SetPosition(combat.PosSleeping)
			cmd = ""
		}
		output()
		if specTakeToJail(w, nil, guard, cmd, "") || len(engine.starts) != 0 || output() != "" {
			t.Fatal("entry gate reached breed helper")
		}
	}
}
