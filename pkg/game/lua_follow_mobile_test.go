package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestLuaFollowMobileLeaderResolvesKeywordInFollowerRoom(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Follower room"},
			{VNum: 1002, Name: "Remote room"},
		},
		Mobs: []parser.Mob{
			{VNum: 9001, Keywords: "follower", ShortDesc: "a mobile follower"},
			{VNum: 9002, Keywords: "guard", ShortDesc: "a remote guard"},
			{VNum: 9003, Keywords: "guard", ShortDesc: "the local guard"},
			{VNum: 9004, Keywords: "captain", ShortDesc: "a captain"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	observer := NewPlayer(1, "Observer", 1001)
	observer.SetPosition(8)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	follower, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	localLeader, err := w.SpawnMob(9003, 1001)
	if err != nil {
		t.Fatal(err)
	}
	newLeader, err := w.SpawnMob(9004, 1001)
	if err != nil {
		t.Fatal(err)
	}
	remoteLeader, err := w.SpawnMob(9002, 1002)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "follow.lua"), []byte(`function oncmd()
if argument == "remote" or argument == "same" then
follow(inworld("mob", 9002), 0)
elseif argument == "switch" then
follow(inworld("mob", 9004), 0)
elseif argument == "charmed" or argument == "cycle" then
follow(inworld("mob", 9003), 0)
elseif argument == "self" then
follow(me, 0)
else
return FALSE
end
return TRUE
end
`), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewWorldScriptableAdapter(w)
	engine := scripting.NewEngine(dir, adapter)
	t.Cleanup(engine.Close)
	chRef := scripting.CharRef{ID: observer.ID}
	meRef := scripting.CharRef{NPC: true, ID: follower.GetID()}
	ctx := &scripting.ScriptContext{World: adapter, ChRef: &chRef, MeRef: &meRef, RoomVNum: 1001}

	output := captureMovementOutput(w)
	runFollow := func(action string) {
		t.Helper()
		ctx.Argument = action
		handled, err := engine.RunScript(ctx, "follow.lua", "oncmd")
		if err != nil || !handled {
			t.Fatalf("Lua follow (%s): handled=%t err=%v", action, handled, err)
		}
	}

	runFollow("remote")
	if got := w.FollowingBody(follower); got != localLeader {
		t.Fatalf("mobile follower selected body %p, want local same-keyword guard %p", got, localLeader)
	}
	if w.FollowingBody(remoteLeader) != nil {
		t.Fatal("remote guard acquired a follower")
	}
	roomOutput := output[observer.Name]
	if roomOutput == nil || !strings.Contains(roomOutput.String(), "starts to follow the local guard.") {
		t.Fatalf("Lua follow room output = %v", output)
	}

	runFollow("same")
	if got := w.FollowingBody(follower); got != localLeader || strings.Count(roomOutput.String(), "starts to follow") != 1 {
		t.Fatalf("repeat follow changed body or emitted a duplicate notice: body=%p output=%q", got, roomOutput.String())
	}

	follower.SetAffected(affGroup)
	runFollow("switch")
	if got := w.FollowingBody(follower); got != newLeader || follower.IsAffected(affGroup) || strings.Count(roomOutput.String(), "starts to follow") != 2 {
		t.Fatalf("switch follow body=%p grouped=%t output=%q", got, follower.IsAffected(affGroup), roomOutput.String())
	}

	follower.SetAffected(affCharm)
	runFollow("charmed")
	if got := w.FollowingBody(follower); got != newLeader {
		t.Fatalf("charmed follower switched masters: body=%p", got)
	}
	follower.RemoveAffected(affCharm)

	localLeader.SetFollowingBody(follower)
	runFollow("cycle")
	if got := w.FollowingBody(follower); got != newLeader {
		t.Fatalf("cyclic follow changed master: body=%p", got)
	}

	runFollow("self")
	if got := w.FollowingBody(follower); got != nil || follower.GetFollowing() != "" {
		t.Fatalf("self-follow did not stop following: body=%p name=%q", got, follower.GetFollowing())
	}
}
