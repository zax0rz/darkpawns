package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func writeObjFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return p
}

// basic single object with all standard fields populated
func TestParseObjFile_SingleObject(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", `#500
sword blade sharp~
A sharp sword~
A sharp sword lies here.
~
1 0 0 0 0 1 0 0 0
0 0 0 0
10 100 50.0
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objs))
	}
	o := objs[0]
	if o.VNum != 500 {
		t.Errorf("vnum: expected 500, got %d", o.VNum)
	}
	if o.Keywords != "sword blade sharp" {
		t.Errorf("keywords: expected 'sword blade sharp', got %q", o.Keywords)
	}
	if o.ShortDesc != "A sharp sword" {
		t.Errorf("shortdesc: expected 'A sharp sword', got %q", o.ShortDesc)
	}
	if o.LongDesc != "A sharp sword lies here." {
		t.Errorf("longdesc: expected 'A sharp sword lies here.', got %q", o.LongDesc)
	}
	if o.ActionDesc != "" {
		t.Errorf("actiondesc: expected empty, got %q", o.ActionDesc)
	}
	if o.TypeFlag != 1 {
		t.Errorf("typeflag: expected 1, got %d", o.TypeFlag)
	}
	if o.Weight != 10 {
		t.Errorf("weight: expected 10, got %d", o.Weight)
	}
	if o.Cost != 100 {
		t.Errorf("cost: expected 100, got %d", o.Cost)
	}
	if o.LoadPercent != 50.0 {
		t.Errorf("loadpercent: expected 50.0, got %f", o.LoadPercent)
	}
}

// multiple objects in one file
func TestParseObjFile_MultiObject(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", `#100
obj one~
short one~
long one here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 1 100.0
#200
obj two~
short two~
long two here.
~
2 0 0 0 0 0 0 0 0
0 0 0 0
2 2 100.0
#300
obj three~
short three~
long three here.
~
3 0 0 0 0 0 0 0 0
0 0 0 0
3 3 100.0
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(objs))
	}
	if objs[0].VNum != 100 {
		t.Errorf("obj0 vnum: expected 100, got %d", objs[0].VNum)
	}
	if objs[1].VNum != 200 {
		t.Errorf("obj1 vnum: expected 200, got %d", objs[1].VNum)
	}
	if objs[2].VNum != 300 {
		t.Errorf("obj2 vnum: expected 300, got %d", objs[2].VNum)
	}
}

// parseFlag with plain integer
func TestParseFlag_Integer(t *testing.T) {
	if v := parseFlag("8193"); v != 8193 {
		t.Errorf("parseFlag(\"8193\") = %d, want 8193", v)
	}
	if v := parseFlag("0"); v != 0 {
		t.Errorf("parseFlag(\"0\") = %d, want 0", v)
	}
	if v := parseFlag("32768"); v != 32768 {
		t.Errorf("parseFlag(\"32768\") = %d, want 32768", v)
	}
}

// parseFlag with letter-encoded bitmask
func TestParseFlag_Letters(t *testing.T) {
	// a = 1 << 0 = 1
	if v := parseFlag("a"); v != 1 {
		t.Errorf("parseFlag(\"a\") = %d, want 1", v)
	}
	// b = 1 << 1 = 2
	if v := parseFlag("b"); v != 2 {
		t.Errorf("parseFlag(\"b\") = %d, want 2", v)
	}
	// A = 1 << 26
	if v := parseFlag("A"); v != 1<<26 {
		t.Errorf("parseFlag(\"A\") = %d, want %d", v, 1<<26)
	}
}

// parseFlag with mixed letters
func TestParseFlag_MixedLetters(t *testing.T) {
	// aB = a (1) + B (1 << 27)
	expected := 1 | (1 << 27)
	if v := parseFlag("aB"); v != expected {
		t.Errorf("parseFlag(\"aB\") = %d, want %d", v, expected)
	}
}

// long description first letter auto-capitalized
func TestParseObjFile_LongDescCapitalization(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", `#501
axe~
An axe~
a rusty axe lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
5 10 100.0
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if objs[0].LongDesc != "A rusty axe lies here." {
		t.Errorf("longdesc: expected 'A rusty axe lies here.', got %q", objs[0].LongDesc)
	}
}

// container weight validation for ITEM_DRINKCON (17)
func TestParseObjFile_DrinkconWeightAdjustment(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", `#502
waterskin~
A waterskin~
A waterskin lies here.
~
17 0 0 0 0 0 0 0 0
0 20 0 0
1 5 100.0
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	// weight 1 < Values[1]=20, so weight should be adjusted to 25
	if objs[0].Weight != 25 {
		t.Errorf("drinkcon weight: expected 25, got %d", objs[0].Weight)
	}
}

// container weight validation for ITEM_FOUNTAIN (23)
func TestParseObjFile_FountainWeightAdjustment(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", `#503
fountain~
A fountain~
A fountain stands here.
~
23 0 0 0 0 0 0 0 0
0 50 0 0
10 100 100.0
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	// weight 10 < Values[1]=50, so weight should be adjusted to 55
	if objs[0].Weight != 55 {
		t.Errorf("fountain weight: expected 55, got %d", objs[0].Weight)
	}
}

// drinkcon with sufficient weight should not be adjusted
func TestParseObjFile_DrinkconWeightNoAdjustment(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", `#504
waterskin~
A waterskin~
A waterskin lies here.
~
17 0 0 0 0 0 0 0 0
0 20 0 0
30 5 100.0
$
`)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	// weight 30 >= Values[1]=20, no adjustment
	if objs[0].Weight != 30 {
		t.Errorf("drinkcon weight: expected 30, got %d", objs[0].Weight)
	}
}

// MAX_OBJ_AFFECT = 6 — 7th affect silently discarded
func TestParseObjFile_MaxObjAffect(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#505
ring~
A ring~
A ring lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 100 100.0
A
1 1
A
2 2
A
3 3
A
4 4
A
5 5
A
6 6
A
7 7
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs[0].Affects) != MAX_OBJ_AFFECT {
		t.Errorf("affects: expected %d, got %d", MAX_OBJ_AFFECT, len(objs[0].Affects))
	}
	// verify the 7th was discarded (location 7 should not be present)
	for _, aff := range objs[0].Affects {
		if aff.Location == 7 {
			t.Error("7th affect (location 7) should have been discarded")
		}
	}
}

// extra description parsing
func TestParseObjFile_ExtraDesc(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#506
book~
A book~
A book lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 50 100.0
E
cover title~
The title reads "Ancient Secrets".
~
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs[0].ExtraDescs) != 1 {
		t.Fatalf("expected 1 extra desc, got %d", len(objs[0].ExtraDescs))
	}
	ed := objs[0].ExtraDescs[0]
	if ed.Keywords != "cover title" {
		t.Errorf("extra desc keywords: expected 'cover title', got %q", ed.Keywords)
	}
	if ed.Description != "The title reads \"Ancient Secrets\".\n" {
		t.Errorf("extra desc description: expected 'The title reads \"Ancient Secrets\".\n', got %q", ed.Description)
	}
}

// multiple extra descriptions
func TestParseObjFile_MultipleExtraDescs(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#507
book~
A book~
A book lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 50 100.0
E
first~
First extra description.
~
E
second~
Second extra description.
~
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs[0].ExtraDescs) != 2 {
		t.Fatalf("expected 2 extra descs, got %d", len(objs[0].ExtraDescs))
	}
	if objs[0].ExtraDescs[0].Keywords != "first" {
		t.Errorf("first extra desc keywords wrong: %q", objs[0].ExtraDescs[0].Keywords)
	}
	if objs[0].ExtraDescs[1].Keywords != "second" {
		t.Errorf("second extra desc keywords wrong: %q", objs[0].ExtraDescs[1].Keywords)
	}
}

// script block parsing
func TestParseObjFile_ScriptBlock(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#508
obj~
An object~
An object lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 10 100.0
S
myscript 3
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if objs[0].ScriptName != "myscript" {
		t.Errorf("scriptname: expected 'myscript', got %q", objs[0].ScriptName)
	}
	if objs[0].LuaFunctions != 3 {
		t.Errorf("luafunctions: expected 3, got %d", objs[0].LuaFunctions)
	}
}

// affect block parsing
func TestParseObjFile_AffectBlock(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#509
obj~
An object~
An object lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 10 100.0
A
13 2
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs[0].Affects) != 1 {
		t.Fatalf("expected 1 affect, got %d", len(objs[0].Affects))
	}
	aff := objs[0].Affects[0]
	if aff.Location != 13 {
		t.Errorf("affect location: expected 13, got %d", aff.Location)
	}
	if aff.Modifier != 2 {
		t.Errorf("affect modifier: expected 2, got %d", aff.Modifier)
	}
}

// #99999 sentinel stops parsing
func TestParseObjFile_Sentinel99999(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#600
obj~
An object~
An object lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 10 100.0
#99999
#700
obj2~
Object two~
Object two lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 10 100.0
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("expected 1 object (stopped at #99999), got %d", len(objs))
	}
	if objs[0].VNum != 600 {
		t.Errorf("vnum: expected 600, got %d", objs[0].VNum)
	}
}

// $ end marker stops parsing
func TestParseObjFile_DollarEnd(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#600
obj~
An object~
An object lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 10 100.0
$
#700
obj2~
Object two~
Object two lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 10 100.0
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 2 {
		t.Fatalf("expected 2 objects ($ is block delimiter, not EOF), got %d", len(objs))
	}
	if objs[0].VNum != 600 {
		t.Errorf("first vnum: expected 600, got %d", objs[0].VNum)
	}
	if objs[1].VNum != 700 {
		t.Errorf("second vnum: expected 700, got %d", objs[1].VNum)
	}
}

// vnum 0 edge case
func TestParseObjFile_VnumZero(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#0
null obj~
A null object~
A null object lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 1 100.0
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objs))
	}
	if objs[0].VNum != 0 {
		t.Errorf("vnum: expected 0, got %d", objs[0].VNum)
	}
}

// empty file returns no objects
func TestParseObjFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", "")

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 0 {
		t.Errorf("expected 0 objects, got %d", len(objs))
	}
}

// file with only comments and whitespace
func TestParseObjFile_CommentsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	f := writeObjFile(t, tmpDir, "test.obj", "* this is a comment\n\n  \n")

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 0 {
		t.Errorf("expected 0 objects, got %d", len(objs))
	}
}

// truncated file (missing fields) should return error
func TestParseObjFile_Truncated(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#700
obj~
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	_, err := ParseObjFile(f)
	if err == nil {
		t.Fatal("expected error for truncated file, got nil")
	}
}

// ParseAllObjFiles from directory
func TestParseAllObjFiles(t *testing.T) {
	tmpDir := t.TempDir()
	c1 := `#100
obj1~
Obj one~
Obj one lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 1 100.0
$
`
	c2 := `#200
obj2~
Obj two~
Obj two lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 1 100.0
$
`
	_ = writeObjFile(t, tmpDir, "a.obj", c1)
	_ = writeObjFile(t, tmpDir, "b.obj", c2)

	objs, err := ParseAllObjFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all obj files: %v", err)
	}
	if len(objs) != 2 {
		t.Errorf("expected 2 objects total, got %d", len(objs))
	}
}

// ParseAllObjFiles ignores non-.obj files
func TestParseAllObjFiles_IgnoresNonObj(t *testing.T) {
	tmpDir := t.TempDir()
	c1 := `#100
obj1~
Obj one~
Obj one lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 1 100.0
$
`
	_ = writeObjFile(t, tmpDir, "a.obj", c1)
	_ = writeObjFile(t, tmpDir, "readme.txt", "not an obj file")
	_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	objs, err := ParseAllObjFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all obj files: %v", err)
	}
	if len(objs) != 1 {
		t.Errorf("expected 1 object, got %d", len(objs))
	}
}

// path traversal rejection
func TestParseObjFile_PathTraversal(t *testing.T) {
	_, err := ParseObjFile("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

// parseFlag with empty string returns 0
func TestParseFlag_Empty(t *testing.T) {
	if v := parseFlag(""); v != 0 {
		t.Errorf("parseFlag(\"\") = %d, want 0", v)
	}
}

// object with all optional blocks (E, A, S)
func TestParseObjFile_FullObject(t *testing.T) {
	tmpDir := t.TempDir()
	content := `#800
shield~
A shield~
A sturdy shield rests here.
~
9 0 0 0 0 0 0 0 0
0 0 0 0
15 500 75.0
E
emblem~
A golden eagle emblem is etched into the surface.
~
A
19 2
S
shieldscript 1
$
`
	f := writeObjFile(t, tmpDir, "test.obj", content)

	objs, err := ParseObjFile(f)
	if err != nil {
		t.Fatalf("parse obj file: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objs))
	}
	o := objs[0]
	if len(o.ExtraDescs) != 1 {
		t.Errorf("extra descs: expected 1, got %d", len(o.ExtraDescs))
	}
	if len(o.Affects) != 1 {
		t.Errorf("affects: expected 1, got %d", len(o.Affects))
	}
	if o.ScriptName != "shieldscript" {
		t.Errorf("scriptname: expected 'shieldscript', got %q", o.ScriptName)
	}
	if o.LuaFunctions != 1 {
		t.Errorf("luafunctions: expected 1, got %d", o.LuaFunctions)
	}
}

// toUpper helper
func TestToUpper(t *testing.T) {
	if toUpper('a') != 'A' {
		t.Errorf("toUpper('a') = %c, want 'A'", toUpper('a'))
	}
	if toUpper('z') != 'Z' {
		t.Errorf("toUpper('z') = %c, want 'Z'", toUpper('z'))
	}
	if toUpper('A') != 'A' {
		t.Errorf("toUpper('A') = %c, want 'A'", toUpper('A'))
	}
	if toUpper('5') != '5' {
		t.Errorf("toUpper('5') = %c, want '5'", toUpper('5'))
	}
}
