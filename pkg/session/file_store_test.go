package session

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSetFileSQLitePreservesSessionFields(t *testing.T) {
	t.Chdir(t.TempDir())
	e := newRecoveryEnv(t)
	e.s.olcZone = 99
	e.s.player.SetLoadRoom(1001)
	e.s.player.SetPlrFlag(game.PlrLoadroom, true)
	if e.s.saveCharacter("fixture", 1001) != game.SaveSucceeded {
		t.Fatal("fixture save")
	}
	e.m.Unregister("Recoverer")
	e.world.RemovePlayer("Recoverer")
	// Disconnect saves first; seed the offline record after that save.
	before, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	before.Inventory, before.Equipment = []byte(" [ ] "), []byte(" { } ")
	if err := e.store.SavePlayer(before); err != nil {
		t.Fatal(err)
	}
	wizard := makeCommandTestSession(t, e.m, "Wizard", game.LVL_IMPL, 1001)
	// A conflicting legacy file must neither be read nor written.
	if err := os.MkdirAll("data/players", 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("data", "players", "recoverer.json")
	sentinel := []byte("legacy sentinel; not a record")
	if err := os.WriteFile(path, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdSet(wizard, []string{"file", "Recoverer", "gold", "777"}); err != nil {
		t.Fatal(err)
	}
	if got := drainFrames(wizard); got != "Recoverer's gold set to 777.\r\nSaved in file.\r\n" {
		t.Fatalf("set output=%q", got)
	}
	after, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	p, err := db.RecordToPlayer(after, e.world)
	if err != nil {
		t.Fatal(err)
	}
	if p.Gold != 777 || after.OlcZone != 99 || after.RoomVNum != before.RoomVNum || after.Password != before.Password || after.ID != before.ID {
		t.Fatalf("SQLite update lost fields: %+v", after)
	}
	if !bytes.Equal(after.Inventory, before.Inventory) || !bytes.Equal(after.Equipment, before.Equipment) {
		t.Fatalf("offline playerfile edit rewrote crash-object payloads: before=%q/%q after=%q/%q", before.Inventory, before.Equipment, after.Inventory, after.Equipment)
	}
	// Each command reads the current record; no stale snapshot is retained.
	after.OlcZone = 42
	if err := e.store.SavePlayer(after); err != nil {
		t.Fatal(err)
	}
	if err := cmdSet(wizard, []string{"file", "Recoverer", "bank", "88"}); err != nil {
		t.Fatal(err)
	}
	after, err = e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	p, err = db.RecordToPlayer(after, e.world)
	if err != nil {
		t.Fatal(err)
	}
	if p.Gold != 777 || p.BankGold != 88 || after.OlcZone != 42 {
		t.Fatal("stale record overwrote newer values")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, sentinel) {
		t.Fatal("legacy JSON changed")
	}
	if err := cmdSet(wizard, []string{"file", "Recoverer", "olc", "7"}); err != nil {
		t.Fatal(err)
	}
	after, err = e.store.GetPlayer("Recoverer")
	if err != nil || after.OlcZone != 7 {
		t.Fatal("set file olc did not reach SQLite")
	}
}

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

func TestSQLiteOfflineClanReaderAndEdit(t *testing.T) {
	e := newRecoveryEnv(t)
	e.s.player.ClanID, e.s.player.ClanRank = 3, 2
	e.s.olcZone = 99
	if e.s.saveCharacter("fixture", -1) != game.SaveSucceeded {
		t.Fatal("fixture save")
	}
	e.m.Unregister("Recoverer")
	e.world.RemovePlayer("Recoverer")
	players, err := e.world.StoredPlayers()
	if err != nil || len(players) != 1 || players[0].ClanID != 3 {
		t.Fatalf("stored players=%v err=%v", players, err)
	}
	if err := e.world.EditStoredPlayer("Recoverer", func(p *game.Player) { p.ClanID, p.ClanRank = 0, 0 }); err != nil {
		t.Fatal(err)
	}
	r, err := e.store.GetPlayer("Recoverer")
	if err != nil {
		t.Fatal(err)
	}
	p, err := db.RecordToPlayer(r, e.world)
	if err != nil {
		t.Fatal(err)
	}
	if p.ClanID != 0 || p.ClanRank != 0 || r.OlcZone != 99 {
		t.Fatal("offline clan edit not durable or lost session field")
	}
	clans := game.NewClanManager()
	clans.Clans = []*game.Clan{{ID: 3}}
	clans.RecountMembers([]*game.Player{{ClanID: 3, ClanRank: 0, Level: 10}, {ClanID: 3, ClanRank: 1, Level: 7}})
	if clans.Clans[0].Members != 1 || clans.Clans[0].Power != 7 {
		t.Fatal("clan census counted rank-zero applicants (clan.c:870-873)")
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

func TestSetFileSQLiteNameAndPassword(t *testing.T) {
	e := newRecoveryEnv(t)
	e.m.Unregister("Recoverer")
	e.world.RemovePlayer("Recoverer")
	wizard := makeCommandTestSession(t, e.m, "Wizard", game.LVL_IMPL, 1001)
	if err := cmdSet(wizard, []string{"file", "Recoverer", "name", "Renamed"}); err != nil {
		t.Fatal(err)
	}
	old, err := e.store.GetPlayer("Recoverer")
	if err != nil || old != nil {
		t.Fatal("old identity remains after rename")
	}
	r, err := e.store.GetPlayer("Renamed")
	if err != nil || r == nil {
		t.Fatal("new SQLite identity missing")
	}
	if err := cmdSet(wizard, []string{"file", "Renamed", "passwd", "newsecret"}); err != nil {
		t.Fatal(err)
	}
	r, err = e.store.GetPlayer("Renamed")
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(r.Password), []byte("newsecret")); err != nil {
		t.Fatal("offline password not durable")
	}
	_ = drainFrames(wizard)
	live := makeTestSession(t, e.m, "Renamed", 1001, true)
	live.player.ID = r.ID
	if err := e.m.Register("Renamed", live); err != nil {
		t.Fatal(err)
	}
	if err := e.world.AddPlayer(live.player); err != nil {
		t.Fatal(err)
	}
	if err := cmdSet(wizard, []string{"Renamed", "passwd", "wrong"}); err != nil {
		t.Fatal(err)
	}
	if got := drainFrames(wizard); got != "You must use set file with this command.\r\nThe player *must* not be logged in when this command is run.\r\n" {
		t.Fatalf("non-file refusal = %q", got)
	}
	r, err = e.store.GetPlayer("Renamed")
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(r.Password), []byte("newsecret")); err != nil {
		t.Fatal("non-file password refusal changed password")
	}
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

func TestSetFileSQLiteCombatBase(t *testing.T) {
	e := newRecoveryEnv(t)
	e.m.Unregister("Recoverer")
	e.world.RemovePlayer("Recoverer")
	wizard := makeCommandTestSession(t, e.m, "Wizard", game.LVL_IMPL, 1001)
	for _, field := range []string{"ac", "hitroll", "damroll"} {
		if err := cmdSet(wizard, []string{"file", "Recoverer", field, "7"}); err != nil {
			t.Fatal(err)
		}
		r, err := e.store.GetPlayer("Recoverer")
		if err != nil {
			t.Fatal(err)
		}
		p, err := db.RecordToPlayer(r, e.world)
		if err != nil {
			t.Fatal(err)
		}
		if p.AC != 100 || p.Hitroll != 0 || p.Damroll != 0 {
			t.Fatalf("%s leaked into char_to_store bases: %d/%d/%d", field, p.AC, p.Hitroll, p.Damroll)
		}
	}
}
