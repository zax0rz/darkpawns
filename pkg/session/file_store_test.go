package session

import (
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStatFileSQLiteUsesDurableValues(t *testing.T) {
	t.Chdir(t.TempDir())
	e := newRecoveryEnv(t)
	e.s.player.Level = game.LVL_IMMORT
	e.s.player.Practices, e.s.player.OrigCon = 9, 0
	e.s.player.Stats.Con = 14
	e.s.player.CopyBaseAttributes()
	e.s.player.SetPlrFlag(game.PrfBrief, true)
	e.s.player.SetCondition(game.CondFull, 11)
	e.s.player.SetCondition(game.CondThirst, 12)
	e.s.player.SetCondition(game.CondDrunk, 13)
	e.s.olcZone = 99
	if e.s.saveCharacter("fixture", -1) != game.SaveSucceeded {
		t.Fatal("fixture save")
	}
	savedID := e.s.player.ID
	e.s.player.ID = 999
	e.s.player.Practices = 99
	e.s.olcZone = 22
	wizard := makeCommandTestSession(t, e.m, "Wizard", game.LVL_IMPL, 1001)
	if err := cmdStat(wizard, []string{"file", "Recoverer"}); err != nil {
		t.Fatal(err)
	}
	out := drainFrames(wizard)
	for _, want := range []string{"STL[9]", ", OLC[99]", "PRF: BRIEF", "Hunger: 11, Thirst: 12, Drunk: 13", "Con: [14/0]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "999") || savedID == 999 {
		t.Fatal("live ID substituted for durable ID")
	}
	if err := cmdStat(wizard, []string{"file", "Missing"}); err != nil {
		t.Fatal(err)
	}
	if got := drainFrames(wizard); got != "There is no such player.\r\n" {
		t.Fatalf("missing player=%q", got)
	}
}

func TestSQLiteLevelAdvanceSave(t *testing.T) {
	e := newRecoveryEnv(t)
	e.s.olcZone = 99
	e.s.player.Level = 2
	e.s.player.AdvanceLevel()
	r, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	if r.Level != 2 || r.MaxHealth != e.s.player.MaxHealth || r.OlcZone != 99 {
		t.Fatal("advance_level did not save SQLite at class.c:712")
	}
}

func TestSaveSQLiteLeavesLegacyJSONUntouched(t *testing.T) {
	t.Chdir(t.TempDir())
	e := newRecoveryEnv(t)
	if err := os.MkdirAll("data/players", 0o750); err != nil {
		t.Fatal(err)
	}
	path := "data/players/Recoverer.json"
	if err := os.WriteFile(path, []byte("legacy sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	e.s.player.Gold = 777
	e.world.ExecSave(e.s.player)
	r, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	p, err := db.RecordToPlayer(r, e.world)
	if err != nil || p.Gold != 777 {
		t.Fatal("save command missed SQLite")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "legacy sentinel" {
		t.Fatal("save command changed legacy file")
	}
	files, err := os.ReadDir("data/players")
	if err != nil || len(files) != 1 {
		t.Fatal("save command created sidecar")
	}
}

func TestCreationResumeAcceptedRecord(t *testing.T) {
	s, c := creationProof(t)
	fresh := entrySession(t, c)
	fresh.player = s.player
	fresh.authenticated = true
	if err := fresh.reloadCharacterFromStore(); err != nil {
		t.Fatal(err)
	}
	if err := fresh.completeCharCreation(); err != nil {
		t.Fatal(err)
	}
	if fresh.player.Health != fresh.player.MaxHealth || fresh.player.Practices != 4 || fresh.player.OrigCon != 14 || !fresh.player.AutoExit || fresh.player.Drunk != 0 {
		t.Fatal("accepted-record resume changed initial live defaults")
	}
	assertCreationPhase(t, creationRead(t, c, fresh.manager.world), 1, 1, fresh.player.MaxHealth, 2)
}

func TestSQLiteFileAdmissionGates(t *testing.T) {
	e := newRecoveryEnv(t)
	e.s.player.Level = game.LVL_IMPL
	if e.s.saveCharacter("fixture", -1) != game.SaveSucceeded {
		t.Fatal("fixture save")
	}
	wizard := makeCommandTestSession(t, e.m, "Wizard", game.LVL_IMMORT, 1001)
	if err := cmdStat(wizard, []string{"file", "Recoverer"}); err != nil {
		t.Fatal(err)
	}
	if got := drainFrames(wizard); got != "Sorry, you can't do that.\r\n" {
		t.Fatalf("stat file level gate=%q", got)
	}
	wizard.player.Level = game.LVL_IMPL
	if err := cmdSet(wizard, []string{"file", "Recoverer", "gold", "777"}); err != nil {
		t.Fatal(err)
	}
	if got := drainFrames(wizard); got != "Sorry, you can't do that.\r\n" {
		t.Fatalf("set file level gate=%q", got)
	}
	r, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	p, err := db.RecordToPlayer(r, e.world)
	if err != nil || p.Gold == 777 {
		t.Fatal("refused file edit reached record")
	}
}
