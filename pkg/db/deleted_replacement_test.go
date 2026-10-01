package db

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestDeletedReplacementAtomic(t *testing.T) {
	for _, mode := range []string{"success", "insert-failure", "changed-record", "not-deleted"} {
		t.Run(mode, func(t *testing.T) {
			database := openGameStore(t, "sqlite://"+filepath.Join(t.TempDir(), "deleted.db"))
			defer func() { _ = database.Close() }()
			p := game.NewCharacter(0, "Reclaim", game.ClassThief, game.RaceKender)
			p.SetPlrFlag(game.PlrDeleted, mode != "not-deleted")
			old, err := PlayerToRecord(p, nil)
			if err != nil {
				t.Fatal(err)
			}
			old.Password = "oldhash"
			if err := database.CreatePlayer(old); err != nil {
				t.Fatal(err)
			}
			fresh := &PlayerRecord{Name: "Reclaim", Password: "freshhash", Inventory: []byte("[]"), Equipment: []byte("{}"), CharacterData: []byte("{}")}
			if mode == "insert-failure" {
				if _, err := database.Exec("CREATE TRIGGER reject_replacement BEFORE INSERT ON players BEGIN SELECT RAISE(ABORT,'replacement refused'); END"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "changed-record" {
				if _, err := database.Exec("UPDATE players SET character_data='{}' WHERE id=?", old.ID); err != nil {
					t.Fatal(err)
				}
			}
			err = ReplaceDeletedPlayer(database, fresh, old)
			got, getErr := database.GetPlayer("reclaim")
			if getErr != nil || got == nil {
				t.Fatalf("atomic replacement lost record: %v", getErr)
			}
			if mode == "success" {
				if err != nil || got.ID <= old.ID || got.Password != "freshhash" || fresh.ID != got.ID {
					t.Fatalf("replacement did not allocate a new identity: err=%v id=%d old=%d", err, got.ID, old.ID)
				}
			} else {
				if err == nil || got.ID != old.ID || got.Password != "oldhash" || fresh.ID != 0 {
					t.Fatalf("failed replacement altered old identity: err=%v id=%d old=%d", err, got.ID, old.ID)
				}
				if mode != "insert-failure" && !errors.Is(err, ErrPlayerRecordChanged) {
					t.Fatalf("record guard: %v", err)
				}
			}
		})
	}
}
