package admin

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// recoverPendingCharacterRenames restores the old filesystem state for every
// journal entry. The database rename and journal removal share one SQLite
// transaction, so a surviving entry always means the relational commit did not
// happen.
func recoverPendingCharacterRenames(world *game.World, database *db.DB) error {
	if database == nil {
		return nil
	}
	pending, err := database.ListPendingCharacterRenames()
	if err != nil {
		return fmt.Errorf("list interrupted character renames: %w", err)
	}
	for _, rename := range pending {
		if len(rename.BoardSnapshot) > 0 {
			if world == nil || world.Boards == nil {
				return fmt.Errorf("cannot restore board files for interrupted rename %q to %q: board system is unavailable", rename.OldName, rename.NewName)
			}
			if err := world.Boards.RestoreSnapshot(rename.BoardSnapshot); err != nil {
				return fmt.Errorf("restore board files for interrupted rename %q to %q: %w", rename.OldName, rename.NewName, err)
			}
		}
		if err := game.RestoreAliasRename(rename.OldName, rename.NewName); err != nil {
			return fmt.Errorf("restore aliases for interrupted rename %q to %q: %w", rename.OldName, rename.NewName, err)
		}
		if err := database.ClearPendingCharacterRename(rename.OldName, rename.NewName); err != nil {
			return fmt.Errorf("clear recovered rename journal for %q: %w", rename.OldName, err)
		}
	}
	return nil
}
