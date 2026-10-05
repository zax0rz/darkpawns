package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestOfflineRenameDeletesOldRent(t *testing.T) {
	t.Chdir(t.TempDir())
	database := entryDatabase(t)
	entrySeed(t, database, "Returner")
	m := entryWorldManager(t, database)
	wizard := makeCommandTestSession(t, m, "Wizard", game.LVL_IMPL, 1001)
	if err := db.SaveObjectSnapshot(database, "Returner", 2, []byte(`[]`), []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := cmdSet(wizard, []string{"file", "Returner", "name", "Archived"}); err != nil {
		t.Fatal(err)
	}
	renderedOutput(wizard)
	if err := cmdShow(wizard, []string{"rent", "Returner"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(wizard); got != "returner has no rent file.\r\n" {
		t.Fatalf("old name rent: %q", got)
	}
}

func TestOfflineRenamePreventsAliasInheritance(t *testing.T) {
	t.Chdir(t.TempDir())
	database := entryDatabase(t)
	entrySeed(t, database, "Returner")
	m := entryWorldManager(t, database)
	wizard := makeCommandTestSession(t, m, "Wizard", game.LVL_IMPL, 1001)
	if err := game.WriteAliases("Returner", []game.Alias{{Alias: "oldalias", Replacement: " score"}}); err != nil {
		t.Fatal(err)
	}
	if err := cmdSet(wizard, []string{"file", "Returner", "name", "Archived"}); err != nil {
		t.Fatal(err)
	}
	entrySeed(t, database, "Returner")
	later := makeCharSession(t, m)
	if err := later.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if later.player == nil {
		t.Fatal("later character did not log in")
	}
	if len(later.player.Aliases) != 0 {
		t.Fatalf("later character inherited aliases: %+v", later.player.Aliases)
	}
}
