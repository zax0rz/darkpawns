package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestApplyObjectFixturesPreparesDisposableWorlds(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	worldDir := filepath.Join(t.TempDir(), "world")
	if err := os.CopyFS(worldDir, os.DirFS(filepath.Join(repoRoot, "lib", "world"))); err != nil {
		t.Fatalf("copy world: %v", err)
	}
	fixture := oraclediff.ObjectFixture{ObjectVNum: 8038}
	if err := applyObjectFixtures(worldDir, []oraclediff.ObjectFixture{fixture}); err != nil {
		t.Fatalf("apply fixtures: %v", err)
	}
	parsed, err := parser.ParseWorld(worldDir)
	if err != nil {
		t.Fatalf("parse disposable world: %v", err)
	}
	foundObject := false
	for _, obj := range parsed.Objs {
		if obj.VNum == 8038 {
			foundObject = true
			if got := obj.TypeFlag; got != 2 {
				t.Fatalf("fixture type = %d, want scroll type 2", got)
			}
			if got := obj.Values[1]; got != -1 {
				t.Fatalf("fixture spell slot = %d, want -1", got)
			}
		}
	}
	if !foundObject {
		t.Fatal("fixture object 8038 was not parsed")
	}
}

func TestApplyRoomFixturesReplacesExitsAndSetsFlags(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	worldDir := filepath.Join(t.TempDir(), "world")
	if err := os.CopyFS(worldDir, os.DirFS(filepath.Join(repoRoot, "lib", "world"))); err != nil {
		t.Fatalf("copy world: %v", err)
	}
	if err := applyRoomFixtures(worldDir,
		[]oraclediff.RoomExitFixture{{RoomVNum: 8162, Direction: "all", ToRoom: 8161, DoorState: 1, Keyword: "gate"}},
		[]oraclediff.RoomFlagFixture{{RoomVNum: 8161, Bit: 1, Enabled: true}},
		[]oraclediff.RoomSectorFixture{{RoomVNum: 8161, Sector: 7}},
	); err != nil {
		t.Fatalf("applyRoomFixtures: %v", err)
	}
	parsed, err := parser.ParseWorld(worldDir)
	if err != nil {
		t.Fatalf("parse disposable world: %v", err)
	}
	rooms := make(map[int]parser.Room, len(parsed.Rooms))
	for _, room := range parsed.Rooms {
		rooms[room.VNum] = room
	}
	if got := rooms[8162].Exits; len(got) != 6 || got["west"].ToRoom != 8161 || got["up"].ToRoom != 8161 || got["north"].Keywords != "gate" || got["north"].DoorState != 1 {
		t.Fatalf("room 8162 exits = %#v, want all six directions to 8161", got)
	}
	deathRoom := rooms[8161]
	if !deathRoom.HasFlag(1) {
		t.Fatal("room 8161 missing ROOM_DEATH bit")
	}
	if deathRoom.Sector != 7 {
		t.Fatalf("room 8161 sector = %d, want 7", deathRoom.Sector)
	}
}

func TestObservationMobFixturesPrepareDisposableWorld(t *testing.T) {
	worldDir := t.TempDir()
	zoneDir := filepath.Join(worldDir, "zon")
	mobDir := filepath.Join(worldDir, "mob")
	if err := os.MkdirAll(zoneDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(mobDir, 0o750); err != nil {
		t.Fatal(err)
	}
	zonePath := filepath.Join(zoneDir, "80.zon")
	zone := "#80\nFixture Zone~\n8199 30 2\nM 0 1 1 8000\nG 1 2 1\nE 1 3 1 5\nO 0 4 1 8000\nS\n$\n"
	if err := os.WriteFile(zonePath, []byte(zone), 0o600); err != nil {
		t.Fatal(err)
	}
	mobPath := filepath.Join(mobDir, "183.mob")
	mobs := "#18306\ncuchi~\nScript: cuchi.lua\nE\n#18307\nother~\nScript: other.lua\nE\n$\n"
	if err := os.WriteFile(mobPath, []byte(mobs), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := applyQuietZoneFixtures(worldDir, []int{80}); err != nil {
		t.Fatalf("quiet zone: %v", err)
	}
	fixture := oraclediff.MobFixture{MobVNum: 18306, MaxExisting: 1, RoomVNum: 8162, ZoneNumber: 80}
	if err := applyMobFixtures(worldDir, []oraclediff.MobFixture{fixture}); err != nil {
		t.Fatalf("spawn mob: %v", err)
	}
	if err := applyScriptlessMobFixtures(worldDir, []int{18306}); err != nil {
		t.Fatalf("strip mob script: %v", err)
	}

	gotZoneBytes, err := os.ReadFile(zonePath)
	if err != nil {
		t.Fatal(err)
	}
	gotZone := string(gotZoneBytes)
	for _, reset := range []string{"M 0 1 1 8000", "G 1 2 1", "E 1 3 1 5"} {
		if !strings.Contains(gotZone, "* oracle fixture: "+reset) {
			t.Fatalf("zone did not suppress reset %q:\n%s", reset, gotZone)
		}
	}
	if !strings.Contains(gotZone, "O 0 4 1 8000") || !strings.Contains(gotZone, "M 0 18306 1 8162\nS\n$") {
		t.Fatalf("zone did not preserve objects and insert deterministic mob:\n%s", gotZone)
	}

	gotMobBytes, err := os.ReadFile(mobPath)
	if err != nil {
		t.Fatal(err)
	}
	gotMobs := string(gotMobBytes)
	if !strings.Contains(gotMobs, "* oracle fixture: Script: cuchi.lua") || !strings.Contains(gotMobs, "Script: other.lua") {
		t.Fatalf("script fixture changed the wrong mob record:\n%s", gotMobs)
	}
}

func TestPrepareOracleDataEmptiesOnlyDisposablePlayerFile(t *testing.T) {
	source := filepath.Join(t.TempDir(), "lib")
	if err := os.MkdirAll(filepath.Join(source, "etc"), 0o750); err != nil {
		t.Fatal(err)
	}
	sourcePlayers := filepath.Join(source, "etc", "players")
	if err := os.WriteFile(sourcePlayers, []byte("existing players"), 0o600); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "lib")
	if err := prepareOracleData(source, destination, true); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "etc", "players"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("disposable player file = %q, want empty", got)
	}
	original, err := os.ReadFile(sourcePlayers)
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != "existing players" {
		t.Fatalf("source player file changed to %q", original)
	}
}

func TestPrepareOracleDataPreservesPlayersByDefault(t *testing.T) {
	source := filepath.Join(t.TempDir(), "lib")
	if err := os.MkdirAll(filepath.Join(source, "etc"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "etc", "players"), []byte("existing players"), 0o600); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "lib")
	if err := prepareOracleData(source, destination, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "etc", "players"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing players" {
		t.Fatalf("disposable player file = %q, want preserved contents", got)
	}
}

func TestWithFreshMUDEnvFollowsEmptyPlayersFixture(t *testing.T) {
	base := []string{"KEEP=1", "DP_FRESH_MUD=stale"}
	enabled := withFreshMUDEnv(base, true)
	if got := strings.Join(enabled, ","); got != "KEEP=1,DP_FRESH_MUD=1" {
		t.Fatalf("enabled env = %q", got)
	}
	disabled := withFreshMUDEnv(base, false)
	if got := strings.Join(disabled, ","); got != "KEEP=1" {
		t.Fatalf("disabled env = %q", got)
	}
}

func TestProbeClientsSelectsNamedPeerAndKeepsOtherAudiences(t *testing.T) {
	var primary oraclediff.Conn = &scriptedTestConn{name: "primary"}
	var mortal oraclediff.Conn = &scriptedTestConn{name: "mortal"}
	var observer oraclediff.Conn = &scriptedTestConn{name: "observer"}
	actor, audience := probeClients(primary, map[string]oraclediff.Conn{
		"mortal":   mortal,
		"observer": observer,
	}, "mortal")

	if actor != mortal {
		t.Fatal("named peer was not selected as probe actor")
	}
	if audience["primary"] != primary || audience["observer"] != observer {
		t.Fatalf("audience = %#v, want primary and observer", audience)
	}
	if _, ok := audience["mortal"]; ok {
		t.Fatal("probe actor was also retained as an audience peer")
	}
}

type scriptedTestConn struct {
	name string
}

func (*scriptedTestConn) Send(string) error                                { return nil }
func (*scriptedTestConn) ReadUntilQuiescent(time.Duration) (string, error) { return "", nil }
func (*scriptedTestConn) Close() error                                     { return nil }
