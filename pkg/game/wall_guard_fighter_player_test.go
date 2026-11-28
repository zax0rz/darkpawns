package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// C-ERRATA E-034: wall_guard_ns's command arm runs fighter(ch, ch, 0, NULL)
// with ch = the typing player (src/spec_procs2.c:2268-2270), so the free
// martial move and the swallowed command belong to the player, not the guard
// (src/spec_procs.c:509-535).

type wallGuardDirectCall struct {
	body     combat.Combatant
	command  string
	argument string
	subcmd   int
}

// wallGuardFighterFixture builds a room with a fighting player and the wall
// guard; the guard itself is never fighting, proving the actor is the player.
func wallGuardFighterFixture(t *testing.T) (*minorFixWorld, *Player, *MobInstance, *[]wallGuardDirectCall) {
	t.Helper()
	mf := newMinorFixWorld(t)
	p := mf.addPlayer(t, "Wallmage", 1001, 10)
	guard := mf.addMob(t, 1001, 20, "wallguard", "the wall guard")
	p.SetFightingBody(guard)
	p.SetPosition(combat.PosFighting)
	var calls []wallGuardDirectCall
	mf.w.CharacterCommandDirectExecFunc = func(ch combat.Combatant, command, argument string, subcmd int) CharacterCommandResult {
		calls = append(calls, wallGuardDirectCall{body: ch, command: command, argument: argument, subcmd: subcmd})
		return CharacterCommandHandled
	}
	return mf, p, guard, &calls
}

// drawsConsumed replays seed and reports whether running fn consumed exactly
// one number(0,10) draw (fighter's switch, src/spec_procs.c:522).
func drawsConsumed(seed uint32, fn func()) (exactlyOne bool) {
	dprng.ResetStream(seed)
	fn()
	after := dprng.Number(0, 1000000)
	dprng.ResetStream(seed)
	_ = dprng.Number(0, 10)
	want := dprng.Number(0, 1000000)
	return after == want
}

func TestWallGuardFighterRunsAsTypingPlayer(t *testing.T) {
	for _, tc := range []struct {
		roll    int
		command string
		argName bool // argument is the opponent's GET_NAME (berserk passes NULL)
	}{
		{1, "headbutt", true},
		{2, "parry", true},
		{3, "bash", true},
		{4, "berserk", false},
	} {
		t.Run(tc.command, func(t *testing.T) {
			mf, p, guard, calls := wallGuardFighterFixture(t)
			seed := minorNumberSeed(t, 0, 10, tc.roll)
			var handled bool
			if !drawsConsumed(seed, func() { handled = specWallGuardNS(mf.w, p, guard, "look", "") }) {
				t.Fatal("fighter's switch did not consume exactly one number(0,10)")
			}
			if !handled {
				t.Fatal("roll 1-4 must swallow the typed command (fighter returns TRUE)")
			}
			if len(*calls) != 1 {
				t.Fatalf("direct ACMD calls = %d, want 1", len(*calls))
			}
			call := (*calls)[0]
			if call.body != combat.Combatant(p) {
				t.Fatalf("direct ACMD body = %v, want the typing player", call.body)
			}
			if call.command != tc.command || call.subcmd != 1 {
				t.Fatalf("direct ACMD = %q subcmd %d, want %q subcmd 1", call.command, call.subcmd, tc.command)
			}
			wantArg := ""
			if tc.argName {
				wantArg = guard.GetName()
			}
			if call.argument != wantArg {
				t.Fatalf("direct ACMD argument = %q, want %q", call.argument, wantArg)
			}
		})
	}
}

func TestWallGuardFighterOtherRollsRunCommand(t *testing.T) {
	for _, roll := range []int{0, 5, 10} {
		mf, p, guard, calls := wallGuardFighterFixture(t)
		seed := minorNumberSeed(t, 0, 10, roll)
		var handled bool
		if !drawsConsumed(seed, func() { handled = specWallGuardNS(mf.w, p, guard, "look", "") }) {
			t.Fatalf("roll %d: fighter drew other than one number(0,10)", roll)
		}
		if handled {
			t.Fatalf("roll %d: command must run (fighter returns FALSE)", roll)
		}
		if len(*calls) != 0 {
			t.Fatalf("roll %d: %d direct ACMD calls, want 0", roll, len(*calls))
		}
	}
}

// TestWallGuardFighterGateArms pins fighter's early returns
// (src/spec_procs.c:513-517): no draw, no move, command runs. The last case
// is the pre-E-034 bug's shape: only the guard fighting must not produce a
// guard free move or swallow the player's command.
func TestWallGuardFighterGateArms(t *testing.T) {
	noDraw := func(t *testing.T, seed uint32, fn func() bool) {
		t.Helper()
		dprng.ResetStream(seed)
		if fn() {
			t.Fatal("gate arm must decline (command runs)")
		}
		fresh := dprng.Number(0, 1000000)
		dprng.ResetStream(seed)
		if got := dprng.Number(0, 1000000); got != fresh {
			t.Fatal("gate arm consumed a draw; C returns FALSE before number(0,10)")
		}
	}

	t.Run("player not fighting", func(t *testing.T) {
		mf := newMinorFixWorld(t)
		p := mf.addPlayer(t, "Wallmage", 1001, 10)
		guard := mf.addMob(t, 1001, 20, "wallguard", "the wall guard")
		calls := 0
		mf.w.CharacterCommandDirectExecFunc = func(combat.Combatant, string, string, int) CharacterCommandResult {
			calls++
			return CharacterCommandHandled
		}
		noDraw(t, 42, func() bool { return specWallGuardNS(mf.w, p, guard, "look", "") })
		if calls != 0 {
			t.Fatalf("%d direct ACMD calls, want 0", calls)
		}
	})

	t.Run("player fighting but not in fighting position", func(t *testing.T) {
		mf, p, guard, calls := wallGuardFighterFixture(t)
		p.SetPosition(combat.PosStanding)
		noDraw(t, 42, func() bool { return specWallGuardNS(mf.w, p, guard, "look", "") })
		if len(*calls) != 0 {
			t.Fatalf("%d direct ACMD calls, want 0", len(*calls))
		}
	})

	t.Run("player hit below zero", func(t *testing.T) {
		mf, p, guard, calls := wallGuardFighterFixture(t)
		p.SetHP(-1)
		noDraw(t, 42, func() bool { return specWallGuardNS(mf.w, p, guard, "look", "") })
		if len(*calls) != 0 {
			t.Fatalf("%d direct ACMD calls, want 0", len(*calls))
		}
	})

	t.Run("only the guard fighting", func(t *testing.T) {
		mf := newMinorFixWorld(t)
		p := mf.addPlayer(t, "Wallmage", 1001, 10)
		guard := mf.addMob(t, 1001, 20, "wallguard", "the wall guard")
		guard.SetFightingBody(p)
		guard.SetPosition(combat.PosFighting)
		calls := 0
		mf.w.CharacterCommandDirectExecFunc = func(combat.Combatant, string, string, int) CharacterCommandResult {
			calls++
			return CharacterCommandHandled
		}
		noDraw(t, 42, func() bool { return specWallGuardNS(mf.w, p, guard, "look", "") })
		if calls != 0 {
			t.Fatalf("guard-side fighter fired %d times on a player's command; E-034 gives the move to the player only", calls)
		}
	})
}
