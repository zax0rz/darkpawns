package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRequiresRealTestSignature(t *testing.T) {
	root := t.TempDir()
	source := `package fixture
 import alias "testing"
 // TestOnlyComment is not a declaration.
 const marker = "TestOnlyString"
 func TestValid(t *alias.T) { if false {t.Fatal("failure")} }
 func TestWrong(t *alias.B) {}
 func TestReturns(t *alias.T) int {return 1}
 func Testlowercase(t *alias.T) {}
 `
	if err := os.WriteFile(filepath.Join(root, "fixture_test.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	ix, err := scan(root)
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]bool{}
	for _, decl := range ix.Tests {
		valid[decl.Name] = decl.Valid
	}
	if len(valid) != 4 || !valid["TestValid"] || valid["TestWrong"] || valid["TestReturns"] || valid["Testlowercase"] {
		t.Fatalf("declarations = %v", valid)
	}
}

func TestScanSubtestsAndVacuousBodies(t *testing.T) {
	root := t.TempDir()
	source := `package fixture
 import "testing"
 func TestEmpty(t *testing.T) {}
 func TestLogOnly(t *testing.T) {t.Log("placeholder")}
 func TestChecksHelper(t *testing.T) {check(t)}
 func TestSub(t *testing.T) {t.Run("with spaces",func(child *testing.T){ child.Run("nested",func(t *testing.T){t.Fatal("check")}) })}
 `
	if err := os.WriteFile(filepath.Join(root, "fixture_test.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	ix, err := scan(root)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]testDecl{}
	for _, decl := range ix.Tests {
		by[decl.Name] = decl
	}
	if !by["TestEmpty"].Vacuous || !by["TestLogOnly"].Vacuous || by["TestChecksHelper"].Vacuous {
		t.Fatalf("vacuity = %+v", by)
	}
	sub := by["TestSub"].Subtests
	if len(sub) != 2 || sub[0] != "with_spaces" || sub[1] != "with_spaces/nested" {
		t.Fatalf("subtests = %v", sub)
	}
}
