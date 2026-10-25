package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// Retain exact comparator inputs and raw draws for every completed attempt.
// The worker supplies a unique attempt directory; no retry overwrites a red.
func retainEvidence(root, directory, cDraw, goDraw string, diffs []oraclediff.BlockDiff, cLog, goLog string) error {
	if root == "" && directory == "" {
		return nil
	}
	if !filepath.IsAbs(root) {
		return fmt.Errorf("retain oracle evidence: root must be absolute")
	}
	root = filepath.Clean(root)
	relative, err := evidencePath(root, directory)
	if err != nil {
		return err
	}
	scope, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("retain oracle evidence root: %w", err)
	}
	defer func() { _ = scope.Close() }()
	// Root operations also reject symlink escapes after lexical validation.
	if err := scope.MkdirAll(relative, 0o750); err != nil {
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
		path, err := evidencePath(root, filepath.Join(relative, name))
		if err != nil {
			return err
		}
		if err := scope.WriteFile(path, data, 0o600); err != nil {
			return fmt.Errorf("retain oracle evidence: %w", err)
		}
	}
	return nil
}

// Names derived from scenario IDs or run names must stay below the chosen root.
// Check components before Clean, which would otherwise erase a hostile "..".
func evidencePath(root, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || strings.Contains(name, `\`) {
		return "", fmt.Errorf("retain oracle evidence: invalid relative name %q", name)
	}
	for _, component := range strings.Split(filepath.ToSlash(name), "/") {
		if component == ".." {
			return "", fmt.Errorf("retain oracle evidence: parent component in %q", name)
		}
	}
	target := filepath.Clean(filepath.Join(root, name))
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || !filepath.IsLocal(relative) {
		return "", fmt.Errorf("retain oracle evidence: path outside root: %q", name)
	}
	return relative, nil
}
