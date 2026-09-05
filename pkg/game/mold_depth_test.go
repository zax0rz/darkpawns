package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMoldParsingMirrorsCOneWord(t *testing.T) {
	objName, rest := OneArgument(`the clay "statue idol" A fine statue`)
	if objName != "clay" {
		t.Fatalf("object name = %q, want clay", objName)
	}
	newName, newDesc := OneWord(rest)
	if newName != "statue idol" {
		t.Fatalf("quoted mold name = %q, want statue idol", newName)
	}
	if newDesc != "A fine statue" {
		t.Fatalf("mold description = %q, want A fine statue", newDesc)
	}
}

func TestDoMoldRewritesLiveObjectFields(t *testing.T) {
	ch := NewPlayer(1, "Moldactor", 8162)
	obj := NewObjectInstance(&parser.Obj{
		VNum:      1003,
		Keywords:  "clay",
		ShortDesc: "a pile of clay",
		LongDesc:  "A pile of clay lies here.",
	}, -1)
	if err := ch.Inventory.AddItem(obj); err != nil {
		t.Fatalf("add clay: %v", err)
	}

	result := DoMold(ch, "clay", "statue", "A fine statue")
	if !result.Success {
		t.Fatalf("DoMold failed: %+v", result)
	}
	if got, want := obj.Runtime.Name, "statue _Moldactor_ mold_item"; got != want {
		t.Errorf("runtime object name = %q, want %q", got, want)
	}
	if got, want := obj.GetKeywords(), "statue _Moldactor_ mold_item"; got != want {
		t.Errorf("object keywords = %q, want %q", got, want)
	}
	if got, want := obj.GetShortDesc(), "A fine statue"; got != want {
		t.Errorf("object short description = %q, want %q", got, want)
	}
	if got, want := obj.GetLongDesc(), "A fine statue has been left here."; got != want {
		t.Errorf("object long description = %q, want %q", got, want)
	}
}
