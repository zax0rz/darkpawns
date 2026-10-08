package scripting

import (
	"testing"
)

// canCarryStub answers canget's world query with a fixed verdict.
type canCarryStub struct {
	*scriptWorldStub
	canCarry bool
}

func (c *canCarryStub) CanCarryObject(string, int) bool { return c.canCarry }

// TestLuaCanGetValidPathPushesNil proves src/scripts.c:211-212: when
// CAN_GET_OBJ fails, C pushes nil, so a script's "if canget(obj) then" is not
// taken. The port pushed 0, which Lua treats as true, so the branch was taken
// for an object the character cannot pick up. The scripts below check the
// branch directly, and each fails if the pushed value is 0 instead of nil.
func TestLuaCanGetValidPathPushesNil(t *testing.T) {
	for _, tc := range []struct {
		name     string
		canCarry bool
		src      string
	}{
		{
			name:     "cannot carry takes the else branch",
			canCarry: false,
			src:      "function oncmd()\n  if canget({vnum = 200}) then return FALSE end\n  return TRUE\nend\n",
		},
		{
			name:     "can carry takes the then branch",
			canCarry: true,
			src:      "function oncmd()\n  if canget({vnum = 200}) then return TRUE end\n  return FALSE\nend\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeScript(t, dir, "canget.lua", tc.src)
			bridge := &shopProbeBridge{recordingBridge: newRecordingBridge()}
			engine := NewEngine(dir, &canCarryStub{scriptWorldStub: worldWith(100), canCarry: tc.canCarry})
			t.Cleanup(func() { engine.l.Close() })
			ch, me := player, player
			ctx := &ScriptContext{World: bridge, ChRef: &ch, MeRef: &me, OwnerType: "mob"}
			handled, err := engine.RunScript(ctx, "canget.lua", "oncmd")
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if !handled {
				t.Fatalf("the script's canget branch disagreed with C (canCarry=%v); logs=%q", tc.canCarry, bridge.logs)
			}
			if len(bridge.logs) != 0 {
				t.Fatalf("a valid canget call logged: %q", bridge.logs)
			}
		})
	}
}
