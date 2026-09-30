// Legacy file helpers exist only in tests to retain unchanged save-format
// round-trip and version-migration fixtures. Production never uses sidecars.
package game

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const saveDir = "./data/players"

// SavePlayer serializes a player's state to disk as JSON.
// Save path: ./data/players/{name}.json
func SavePlayer(player *Player) error {
	if player == nil {
		return fmt.Errorf("cannot save nil player")
	}

	if err := os.MkdirAll(saveDir, 0o750); err != nil {
		return fmt.Errorf("create save dir: %w", err)
	}

	data := playerToSaveData(player)

	path := filepath.Join(saveDir, sanitizeName(player.Name)+".json")
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("create save file: %w", err)
	}

	if err := encodeSave(f, f.Close, data); err != nil {
		return err
	}

	slog.Debug("Player saved", "name", player.Name, "path", path)
	return nil
}

// LoadPlayer loads a player's state from disk.
// Returns a Player with runtime fields initialized.
func LoadPlayer(name string) (*Player, error) {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open save file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var data savePlayerData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode save data: %w", err)
	}

	// Version check: 0 means old format (pre-versioning), silently upgrade.
	// Non-zero mismatch means a future or corrupted save — warn but still load.
	// Version 1 is migrated on load (migrateFlags); older than that is
	// version 0, which shares version 1's layout.
	if data.SaveVersion > CurrentSaveVersion {
		slog.Warn("player save version mismatch",
			"player", name,
			"file_version", data.SaveVersion,
			"expected_version", CurrentSaveVersion,
			"action", "loading with possible data loss")
	}

	return saveDataToPlayer(data), nil
}

// DeletePlayer removes a player's save file from disk.
func DeletePlayer(name string) error {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove save file: %w", err)
	}
	return nil
}

// PlayerSaveExists checks if a player save file exists.
func PlayerSaveExists(name string) bool {
	path := filepath.Join(saveDir, sanitizeName(name)+".json")
	_, err := os.Stat(path)
	return err == nil
}
