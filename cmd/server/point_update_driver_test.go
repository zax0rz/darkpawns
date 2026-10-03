package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The engine count proof needs the actual server wiring too: a fake test
// callback cannot detect a frozen-only production callback or a second ticker.
func TestPointUpdateDriverWiring(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	drivers := 0
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "OnPointUpdate" {
			return true
		}
		sel, ok := kv.Value.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "PointUpdate" {
			t.Error("hourly callback must bind World.PointUpdate unconditionally in both modes")
		} else {
			drivers++
		}
		return true
	})
	if drivers != 1 {
		t.Errorf("server hourly drivers=%d, want exactly 1", drivers)
	}
	files, err := os.ReadDir("../../pkg/game")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range files {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("../../pkg/game", entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if ok && (sel.Sel.Name == "StartPointUpdateTicker" || sel.Sel.Name == "PointUpdate") {
				t.Errorf("extra world point-update driver call in %s: %s", entry.Name(), sel.Sel.Name)
			}
			return true
		})
	}
}
