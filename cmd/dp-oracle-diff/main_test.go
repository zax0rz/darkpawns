package main

import (
	"os"
	"path/filepath"
	"testing"

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
