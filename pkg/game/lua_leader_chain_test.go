package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestLuaLeaderChainBeyondFormerLimit(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Leader room"}},
		Mobs:  []parser.Mob{{VNum: 9001, Keywords: "healer probe", ShortDesc: "the healer"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	root := NewPlayer(1, "Chainroot", 1001)
	root.SetPosition(8)
	if err := w.AddPlayer(root); err != nil {
		t.Fatal(err)
	}
	leaders := make([]*Player, 40)
	for i := range leaders {
		name := fmt.Sprintf("Leader%02d", i)
		if i == len(leaders)-1 {
			name = "DeepLeader"
		}
		leaders[i] = NewPlayer(i+2, name, 1001)
		leaders[i].SetPosition(8)
		if err := w.AddPlayer(leaders[i]); err != nil {
			t.Fatal(err)
		}
	}

	link := func(follower, leader *Player) {
		t.Helper()
		if CircleFollow(w, follower, leader) {
			t.Fatalf("linking %s to %s would create a cycle", follower.Name, leader.Name)
		}
		AddFollowerQuiet(follower, leader)
	}
	link(root, leaders[0])
	for i := 0; i < len(leaders)-1; i++ {
		link(leaders[i], leaders[i+1])
	}

	healer, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "leader.lua"), []byte(`function oncmd()
  local c = ch
  for i = 1, 40 do c = c.leader end
  say(c.name)
  return TRUE
end
`), 0o600); err != nil {
		t.Fatal(err)
	}

	adapter := NewWorldScriptableAdapter(w)
	engine := scripting.NewEngine(dir, adapter)
	t.Cleanup(engine.Close)
	chRef := scripting.CharRef{ID: root.ID}
	meRef := scripting.CharRef{NPC: true, ID: healer.GetID()}
	ctx := &scripting.ScriptContext{World: adapter, ChRef: &chRef, MeRef: &meRef, RoomVNum: 1001, OwnerType: "mob"}
	output := captureMovementOutput(w)
	handled, err := engine.RunScript(ctx, "leader.lua", "oncmd")
	if err != nil || !handled {
		t.Fatalf("RunScript: handled=%t err=%v", handled, err)
	}
	if got := output[root.Name].String(); !strings.Contains(got, "DeepLeader") {
		t.Fatalf("deep leader output = %q, want final leader %q", got, leaders[len(leaders)-1].Name)
	}
}
