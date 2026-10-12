package session

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

type actorPositionFixture struct {
	world    *game.World
	manager  *Manager
	actor    *game.MobInstance
	observer *game.Player
	roomOut  strings.Builder
}

func newActorPositionFixture(t *testing.T, position int) *actorPositionFixture {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1000, Name: "Void"}, {VNum: 1001, Name: "Hall"}},
		Mobs: []parser.Mob{{
			VNum: 99001, Keywords: "guard trainee", ShortDesc: "a guard trainee",
			Sex: 0, Position: position, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	manager.SetCommandExecFunc()
	observer := game.NewPlayer(1, "Observer", 1001)
	observer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	actor, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	actor.SetPosition(position)
	f := &actorPositionFixture{world: w, manager: manager, actor: actor, observer: observer}
	w.MessageSink = func(name string, message []byte) {
		if name == observer.GetName() {
			f.roomOut.Write(message)
		}
	}
	return f
}

func TestNPCForceUsesSharedExecutorAndActualBody(t *testing.T) {
	f := newActorPositionFixture(t, combat.PosStanding)
	caster := makeCommandTestSession(t, f.manager, "Forcegod", LVL_GOD, 1001)
	if err := f.world.AddPlayer(caster.player); err != nil {
		t.Fatal(err)
	}
	var draws int
	previous := characterCommandNumber
	characterCommandNumber = func(min, max int) int {
		draws++
		return 1
	}
	t.Cleanup(func() { characterCommandNumber = previous })

	if err := cmdForce(caster, []string{"guard", "sit"}); err != nil {
		t.Fatalf("force returned error: %v", err)
	}
	if got := readMsgText(t, caster); got != "Okay.\r\n" {
		t.Fatalf("caster bytes = %q, want acknowledgement", got)
	}
	if got := f.actor.GetPosition(); got != combat.PosSitting {
		t.Fatalf("forced NPC position = %d, want sitting", got)
	}
	if err := cmdForce(caster, []string{"guard", "stand"}); err != nil {
		t.Fatalf("force stand returned error: %v", err)
	}
	if got := readMsgText(t, caster); got != "Okay.\r\n" {
		t.Fatalf("stand caster bytes = %q, want acknowledgement", got)
	}
	if got := f.actor.GetPosition(); got != combat.PosStanding {
		t.Fatalf("forced NPC position = %d, want standing", got)
	}
	if got, want := f.roomOut.String(), "A guard trainee sits down.\r\nA guard trainee clambers to its feet.\r\n"; got != want {
		t.Fatalf("room bytes = %q", got)
	}
	if draws != 2 {
		t.Fatalf("command interpreter draws = %d, want one per direct interpreter call", draws)
	}
}

func TestNPCForcePositionGateAndMountedStandRemainExplicit(t *testing.T) {
	for _, test := range []struct {
		name     string
		command  string
		position int
		mounted  bool
	}{
		{name: "sit below minimum", command: "sit", position: combat.PosSleeping},
		{name: "mounted stand needs unported dismount", command: "stand", position: combat.PosStanding, mounted: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			caster := makeCommandTestSession(t, f.manager, "Forcegod", LVL_GOD, 1001)
			if err := f.world.AddPlayer(caster.player); err != nil {
				t.Fatal(err)
			}
			if test.mounted {
				f.actor.SetAffected(game.AffMount)
			}
			if err := cmdForce(caster, []string{"guard", test.command}); err != nil {
				t.Fatalf("force returned error: %v", err)
			}
			if got := readMsgText(t, caster); got != "Okay.\r\n" {
				t.Fatalf("caster bytes = %q, want acknowledgement", got)
			}
			if got := f.actor.GetPosition(); got != test.position {
				t.Fatalf("forced NPC position = %d, want unchanged %d", got, test.position)
			}
			if got := f.roomOut.String(); got != "" {
				t.Fatalf("room bytes = %q, want none", got)
			}
			if got := f.actor.IsAffected(game.AffMount); got != test.mounted {
				t.Fatalf("AFF_MOUNT = %v, want %v", got, test.mounted)
			}
		})
	}
}

func TestLuaNPCCommandUsesSharedExecutorOnce(t *testing.T) {
	f := newActorPositionFixture(t, combat.PosStanding)
	var draws int
	previous := characterCommandNumber
	characterCommandNumber = func(min, max int) int {
		draws++
		return 1
	}
	t.Cleanup(func() { characterCommandNumber = previous })

	game.NewWorldScriptableAdapter(f.world).Command(scripting.CharRef{NPC: true, ID: f.actor.ID}, "sit")
	if got := f.actor.GetPosition(); got != combat.PosSitting {
		t.Fatalf("Lua-command NPC position = %d, want sitting", got)
	}
	if got := f.roomOut.String(); got != "A guard trainee sits down.\r\n" {
		t.Fatalf("room bytes = %q", got)
	}
	if draws != 1 {
		t.Fatalf("command interpreter draws = %d, want one leading hide draw", draws)
	}
}

func TestLegacyVNumMobCommandBridgeRetainsGetAndDrop(t *testing.T) {
	const mobVNum, objVNum = 991201, 991202
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Bridge Hall"}},
		Mobs:  []parser.Mob{{VNum: mobVNum, Keywords: "worker", ShortDesc: "a worker", Position: combat.PosStanding}},
		Objs:  []parser.Obj{{VNum: objVNum, Keywords: "token", ShortDesc: "a token"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	m.SetCommandExecFunc()
	mob, err := w.SpawnMob(mobVNum, 1001)
	if err != nil {
		t.Fatal(err)
	}
	item, err := w.SpawnObject(objVNum, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoomFront(item, 1001); err != nil {
		t.Fatal(err)
	}
	bridge := game.NewWorldScriptableAdapter(w)

	bridge.ExecuteMobCommand(mobVNum, "get all")
	if got := mob.InventorySnapshot(); len(got) != 1 || got[0] != item {
		t.Fatalf("legacy get inventory = %v, want the room token", got)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 0 {
		t.Fatalf("legacy get left room items = %v, want empty", got)
	}

	bridge.ExecuteMobCommand(mobVNum, "drop all")
	if got := mob.InventorySnapshot(); len(got) != 0 {
		t.Fatalf("legacy drop inventory = %v, want empty", got)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 1 || got[0] != item {
		t.Fatalf("legacy drop room items = %v, want the token", got)
	}
}

func TestNPCFloorObjectOnCmdReceivesExactRawCommandWithRunScript(t *testing.T) {
	const (
		actorVNum  = 991211
		objectVNum = 991212
	)
	const wantWord = "say  hello  "
	const wantPunctuation = "'  hello  "
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Script Hall"}},
		Mobs: []parser.Mob{{
			VNum: actorVNum, Keywords: "actor", ShortDesc: "an actor", Position: combat.PosStanding,
		}},
		Objs: []parser.Obj{{
			VNum: objectVNum, Keywords: "script token", ShortDesc: "a script token",
			ScriptName: "exact_command.lua", LuaFunctions: 1 << 1, // OS_ONCMD
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	actor, err := w.SpawnMob(actorVNum, 1001)
	if err != nil {
		t.Fatal(err)
	}
	item, err := w.SpawnObject(objectVNum, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoomFront(item, 1001); err != nil {
		t.Fatal(err)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 1 || got[0] != item || item.Prototype.LuaFunctions != 1<<1 {
		t.Fatalf("floor-script fixture items=%v script-flags=%d", got, item.Prototype.LuaFunctions)
	}
	// The C bridge interprets the trigger result numerically, so return 1 as
	// the existing Lua scripts do for a handled command.
	script := "function oncmd() return 1 end\n"
	dir := t.TempDir()
	objScripts := filepath.Join(dir, "obj")
	if err := os.MkdirAll(objScripts, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(objScripts, "exact_command.lua"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	engine := scripting.NewEngine(dir, game.NewWorldScriptableAdapter(w))
	t.Cleanup(engine.Close)
	previousEngine := game.ScriptEngine
	game.ScriptEngine = engine
	t.Cleanup(func() { game.ScriptEngine = previousEngine })

	ctx := characterCommandContext{actor: actor, manager: m}
	// Run the configured Lua file through the real scripting engine first.
	if got := runCharacterSpecials(ctx, "say", "  hello  "); got != game.SpecialHandled {
		t.Fatalf("real object RunScript result = %v, want handled", got)
	}

	// Then inspect the exact ScriptContext argument delivered to that same
	// room-object oncmd site for both C's word and punctuation command forms.
	var gotArguments []string
	game.ScriptEngine = actorOnCmdArgumentRecorder{arguments: &gotArguments}
	for _, test := range []struct{ command, raw, want string }{
		{command: "say", raw: "  hello  ", want: wantWord},
		{command: "'", raw: "  hello  ", want: wantPunctuation},
	} {
		if got := test.command + test.raw; got != test.want {
			t.Fatalf("test setup command = %q, want %q", got, test.want)
		}
		if got := runCharacterSpecials(ctx, test.command, test.raw); got != game.SpecialHandled {
			t.Errorf("RunScript oncmd argument %q result = %v, want handled (exact C bytes)", test.want, got)
		}
	}
	if !reflect.DeepEqual(gotArguments, []string{wantWord, wantPunctuation}) {
		t.Fatalf("room-object oncmd arguments = %#v, want exact C text %#v", gotArguments, []string{wantWord, wantPunctuation})
	}
}

type actorOnCmdArgumentRecorder struct{ arguments *[]string }

func (actorOnCmdArgumentRecorder) ForgetFailures() {}
func (r actorOnCmdArgumentRecorder) RunScript(ctx *game.ScriptContext, _, _ string) (bool, error) {
	*r.arguments = append(*r.arguments, ctx.Argument)
	return true, nil
}

func TestNPCSharedExecutorWalksCOrderedSpecialStages(t *testing.T) {
	const (
		actorVNum = 991101
		peerVNum  = 991102
		wornVNum  = 991103
		carryVNum = 991104
		floorVNum = 991105
	)
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Order Hall"}},
		Mobs: []parser.Mob{
			{VNum: actorVNum, Keywords: "actor", ShortDesc: "an actor", Position: combat.PosStanding},
			{VNum: peerVNum, Keywords: "peer", ShortDesc: "a peer", Position: combat.PosStanding},
		},
		Objs: []parser.Obj{
			{VNum: wornVNum, Keywords: "worn token", ShortDesc: "a worn token"},
			{VNum: carryVNum, Keywords: "carried token", ShortDesc: "a carried token"},
			{VNum: floorVNum, Keywords: "floor token", ShortDesc: "a floor token"},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	actor, err := w.SpawnMob(actorVNum, 1001)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := w.SpawnMob(peerVNum, 1001)
	if err != nil {
		t.Fatal(err)
	}

	var calls []string
	installActorRoomSpec(t, 1001, "npc_order_room", func(_ *game.World, _ game.CharacterSpecialContext, _ *game.MobInstance, _, _ string) game.SpecialResult {
		calls = append(calls, "room")
		return game.SpecialDeclined
	})
	installActorMobSpec(t, peerVNum, "npc_order_mobile", func(_ *game.World, _ game.CharacterSpecialContext, _ *game.MobInstance, _, _ string) game.SpecialResult {
		calls = append(calls, "mobile")
		return game.SpecialDeclined
	})
	installActorObjSpec(t, wornVNum, "npc_order_worn", func(_ *game.World, _ game.CharacterSpecialContext, _ *game.ObjectInstance, _, _ string) game.SpecialResult {
		calls = append(calls, "worn")
		return game.SpecialDeclined
	})
	installActorObjSpec(t, carryVNum, "npc_order_carried", func(_ *game.World, _ game.CharacterSpecialContext, _ *game.ObjectInstance, _, _ string) game.SpecialResult {
		calls = append(calls, "carried")
		return game.SpecialDeclined
	})
	installActorObjSpec(t, floorVNum, "npc_order_floor", func(_ *game.World, _ game.CharacterSpecialContext, _ *game.ObjectInstance, _, _ string) game.SpecialResult {
		calls = append(calls, "floor")
		return game.SpecialDeclined
	})

	worn, err := w.SpawnObject(wornVNum, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObject(worn, game.LocEquippedMob(actor.ID, game.EquipmentSlot(5))); err != nil {
		t.Fatalf("equip worn object: %v", err)
	}
	carried, err := w.SpawnObject(carryVNum, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToMobInventoryFront(carried, actor); err != nil {
		t.Fatalf("carry object: %v", err)
	}
	floor, err := w.SpawnObject(floorVNum, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoomFront(floor, 1001); err != nil {
		t.Fatalf("place floor object: %v", err)
	}

	result := dispatchCharacterCommand(characterCommandContext{actor: actor, manager: m}, "sit", func(int, int) int { return 1 })
	if result != game.CharacterCommandHandled {
		t.Fatalf("dispatch result = %v, want handled", result)
	}
	if got, want := calls, []string{"room", "worn", "carried", "mobile", "floor"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("special calls = %v, want %v", got, want)
	}
	if got := actor.GetPosition(); got != combat.PosSitting {
		t.Fatalf("actor position = %d, want sitting", got)
	}
	_ = peer
}

func TestNPCSharedExecutorBlocksLegacyOnlyAssignedSpecial(t *testing.T) {
	installRoomSpec(t, 1001, "legacy_only_npc_room", func(*game.World, *game.Player, *game.MobInstance, string, string) bool { return false })
	f := newActorPositionFixture(t, combat.PosStanding)
	previous, had := game.RoomSpecAssign[1001]
	game.RoomSpecAssign[1001] = "legacy_only_npc_room"
	t.Cleanup(func() {
		if had {
			game.RoomSpecAssign[1001] = previous
		} else {
			delete(game.RoomSpecAssign, 1001)
		}
	})
	result := dispatchCharacterCommand(characterCommandContext{actor: f.actor, manager: f.manager}, "sit", func(int, int) int { return 1 })
	if result != game.CharacterCommandUnsupported {
		t.Fatalf("legacy-only room special result = %v, want unsupported", result)
	}
	if got := f.actor.GetPosition(); got != combat.PosStanding {
		t.Fatalf("actor position = %d, want unchanged standing", got)
	}
}

func installActorRoomSpec(t *testing.T, vnum int, name string, fn game.ActorSpecFunc) {
	t.Helper()
	previous, had := game.RoomSpecAssign[vnum]
	game.RoomSpecAssign[vnum] = name
	game.RegisterActorSpec(name, fn)
	t.Cleanup(func() {
		if had {
			game.RoomSpecAssign[vnum] = previous
		} else {
			delete(game.RoomSpecAssign, vnum)
		}
		delete(game.SpecRegistry, name)
	})
}

func installActorMobSpec(t *testing.T, vnum int, name string, fn game.ActorSpecFunc) {
	t.Helper()
	previous, had := game.MobSpecAssign[vnum]
	game.MobSpecAssign[vnum] = name
	game.RegisterActorSpec(name, fn)
	t.Cleanup(func() {
		if had {
			game.MobSpecAssign[vnum] = previous
		} else {
			delete(game.MobSpecAssign, vnum)
		}
		delete(game.SpecRegistry, name)
	})
}

func installActorObjSpec(t *testing.T, vnum int, name string, fn game.ActorObjectSpecFunc) {
	t.Helper()
	previous, had := game.ObjSpecAssign[vnum]
	game.ObjSpecAssign[vnum] = name
	game.RegisterActorObjectSpec(name, fn)
	t.Cleanup(func() {
		if had {
			game.ObjSpecAssign[vnum] = previous
		} else {
			delete(game.ObjSpecAssign, vnum)
		}
		delete(game.SpecRegistry, name)
	})
}

func callRegisteredActorCommand(t *testing.T, f *actorPositionFixture, name string, private *strings.Builder) game.CharacterCommandResult {
	t.Helper()
	entry, ok := cmdRegistry.Lookup(name)
	if !ok || entry.ActorHandler == nil {
		t.Fatalf("%s has no registered actor callback", name)
	}
	playerHandler := entry.Handler
	result := entry.ActorHandler(game.CharacterCommandContext{
		Actor: f.actor, World: f.world,
		PrivateOutput: func(line string) { private.WriteString(line) },
	}, name, nil)
	if reflect.ValueOf(entry.Handler).Pointer() != reflect.ValueOf(playerHandler).Pointer() {
		t.Fatalf("%s actor callback changed the player handler", name)
	}
	return result
}

func TestRegisteredActorSitAndStandRoomBytesAndState(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		position     int
		wantPosition int
		wantPrivate  string
		wantRoom     string
	}{
		{"sit standing", "sit", combat.PosStanding, combat.PosSitting, "You sit down.\r\n", "A guard trainee sits down.\r\n"},
		{"sit resting", "sit", combat.PosResting, combat.PosSitting, "You stop resting, and sit up.\r\n", "A guard trainee stops resting.\r\n"},
		{"stand sitting", "stand", combat.PosSitting, combat.PosStanding, "You stand up.\r\n", "A guard trainee clambers to its feet.\r\n"},
		{"stand resting", "stand", combat.PosResting, combat.PosStanding, "You stop resting, and stand up.\r\n", "A guard trainee stops resting, and clambers on its feet.\r\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			var private strings.Builder
			if got := callRegisteredActorCommand(t, f, test.command, &private); got != game.CharacterCommandHandled {
				t.Fatalf("actor callback result = %v, want handled", got)
			}
			if got := private.String(); got != test.wantPrivate {
				t.Errorf("private bytes = %q, want %q", got, test.wantPrivate)
			}
			if got := f.roomOut.String(); got != test.wantRoom {
				t.Errorf("room bytes = %q, want %q", got, test.wantRoom)
			}
			if got := f.actor.GetPosition(); got != test.wantPosition {
				t.Errorf("actor position = %d, want %d", got, test.wantPosition)
			}
		})
	}
}

func TestRegisteredActorPositionRefusalsAndMountedDistinction(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		position    int
		mounted     bool
		wantResult  game.CharacterCommandResult
		wantPrivate string
	}{
		{"stand already standing", "stand", combat.PosStanding, false, game.CharacterCommandHandled, "You are already standing.\r\n"},
		{"stand fighting", "stand", combat.PosFighting, false, game.CharacterCommandHandled, "Do you not consider fighting as standing?\r\n"},
		{"sit already sitting", "sit", combat.PosSitting, false, game.CharacterCommandHandled, "You're sitting already.\r\n"},
		{"sit fighting", "sit", combat.PosFighting, false, game.CharacterCommandHandled, "Sit down while fighting? are you MAD?\r\n"},
		{"sit mounted", "sit", combat.PosStanding, true, game.CharacterCommandHandled, "You can't rest while mounted.\r\n"},
		{"stand mounted dismount unported", "stand", combat.PosStanding, true, game.CharacterCommandUnsupported, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			if test.mounted {
				f.actor.SetAffected(game.AffMount)
			}
			var private strings.Builder
			got := callRegisteredActorCommand(t, f, test.command, &private)
			if got != test.wantResult {
				t.Fatalf("actor callback result = %v, want %v", got, test.wantResult)
			}
			if got := private.String(); got != test.wantPrivate {
				t.Errorf("private bytes = %q, want %q", got, test.wantPrivate)
			}
			if got := f.roomOut.String(); got != "" {
				t.Errorf("room bytes = %q, want none", got)
			}
			if got := f.actor.GetPosition(); got != test.position {
				t.Errorf("actor position = %d, want unchanged %d", got, test.position)
			}
			if got := f.actor.IsAffected(game.AffMount); got != test.mounted {
				t.Errorf("AFF_MOUNT = %v, want unchanged %v", got, test.mounted)
			}
		})
	}
}

func TestRegisteredActorRoomVisibilityMatchesSitAndStandActFlags(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		position int
		wantRoom string
	}{
		{"standing sit uses visible substitution without hiding recipient", "sit", combat.PosStanding, "Someone sits down.\r\n"},
		{"resting sit suppresses invisible actor", "sit", combat.PosResting, ""},
		{"default sit suppresses invisible actor", "sit", combat.PosMortally, ""},
		{"stand suppresses invisible actor", "stand", combat.PosSitting, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			f.actor.SetAffected(game.AffInvisible)
			var private strings.Builder
			if got := callRegisteredActorCommand(t, f, test.command, &private); got != game.CharacterCommandHandled {
				t.Fatalf("actor callback result = %v, want handled", got)
			}
			if got := f.roomOut.String(); got != test.wantRoom {
				t.Errorf("room bytes = %q, want %q", got, test.wantRoom)
			}
		})
	}
}

func TestRegisteredActorCallbacksWithoutPrivateDescriptorOnlyShowRoomAct(t *testing.T) {
	f := newActorPositionFixture(t, combat.PosStanding)
	entry, ok := cmdRegistry.Lookup("sit")
	if !ok || entry.ActorHandler == nil {
		t.Fatal("sit has no registered actor callback")
	}
	result := entry.ActorHandler(game.CharacterCommandContext{Actor: f.actor, World: f.world}, "sit", nil)
	if result != game.CharacterCommandHandled {
		t.Fatalf("actor callback result = %v, want handled", result)
	}
	if got := f.roomOut.String(); got != "A guard trainee sits down.\r\n" {
		t.Fatalf("room bytes = %q", got)
	}
	if got := f.actor.GetPosition(); got != combat.PosSitting {
		t.Fatalf("actor position = %d, want sitting", got)
	}
}
