package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// Retain exact comparator inputs and raw draws for every completed attempt.
// The worker supplies a unique attempt directory; no retry overwrites a red.
func retainEvidence(directory, cDraw, goDraw string, diffs []oraclediff.BlockDiff, cLog, goLog string) error {
	if directory == "" {
		return nil
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}
	payload, err := json.Marshal(diffs)
	if err != nil {
		return err
	}
	files := map[string][]byte{"blocks.json": payload, "c-server.log": []byte(cLog), "go-server.log": []byte(goLog)}
	for name, path := range map[string]string{"c-draws.log": cDraw, "go-draws.log": goDraw} {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		files[name] = data
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			return fmt.Errorf("retain oracle evidence: %w", err)
		}
	}
	return nil
}
