package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMobFile_SingleBasic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nkeyword~\nA small test mob~\nA small test mob stands here.\nThis is a detailed description.\n~\n0 0 -100 7 E\n1 20 0 5 10 20 1 4 2\n100 500\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("expected 1 mob, got %d", len(mobs))
	}
	m := mobs[0]
	if m.VNum != 100 {
		t.Errorf("expected vnum 100, got %d", m.VNum)
	}
	if m.Keywords != "keyword" {
		t.Errorf("expected keywords 'keyword', got %q", m.Keywords)
	}
	if m.ShortDesc != "a small test mob" {
		t.Errorf("expected short desc 'a small test mob', got %q", m.ShortDesc)
	}
	if m.LongDesc != "A small test mob stands here." {
		t.Errorf("expected long desc 'A small test mob stands here.', got %q", m.LongDesc)
	}
	if m.DetailedDesc != "This is a detailed description.\n" {
		t.Errorf("expected detailed desc 'This is a detailed description.\\n', got %q", m.DetailedDesc)
	}
	if m.Alignment != -100 {
		t.Errorf("expected alignment -100, got %d", m.Alignment)
	}
	if m.Race != 7 {
		t.Errorf("expected race 7, got %d", m.Race)
	}
	if m.Level != 1 {
		t.Errorf("expected level 1, got %d", m.Level)
	}
	if m.THAC0 != 20 {
		t.Errorf("expected thac0 20, got %d", m.THAC0)
	}
	if m.AC != 0 {
		t.Errorf("expected AC 0, got %d", m.AC)
	}
	if m.HP.Num != 5 || m.HP.Sides != 10 || m.HP.Plus != 20 {
		t.Errorf("expected HP 5d10+20, got %s", m.HP.String())
	}
	if m.Damage.Num != 1 || m.Damage.Sides != 4 || m.Damage.Plus != 2 {
		t.Errorf("expected damage 1d4+2, got %s", m.Damage.String())
	}
	if m.Gold != 100 {
		t.Errorf("expected gold 100, got %d", m.Gold)
	}
	if m.Exp != 500 {
		t.Errorf("expected exp 500, got %d", m.Exp)
	}
	if m.Position != 8 {
		t.Errorf("expected position 8, got %d", m.Position)
	}
	if m.DefaultPos != 3 {
		t.Errorf("expected default pos 3, got %d", m.DefaultPos)
	}
	if m.Sex != 0 {
		t.Errorf("expected sex 0, got %d", m.Sex)
	}
}

func TestParseMobFile_ShortDescArticleLowercasing(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	cases := []struct {
		input    string
		expected string
	}{
		{"A dragon~", "a dragon"},
		{"An ogre~", "an ogre"},
		{"The king~", "the king"},
		{"a goblin~", "a goblin"},
		{"some creature~", "some creature"},
	}

	for _, tc := range cases {
		content := "#100\nkeyword~\n" + tc.input + "\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}
		mobs, err := ParseMobFile(testFile)
		if err != nil {
			t.Fatalf("parse error for %q: %v", tc.input, err)
		}
		if len(mobs) != 1 {
			t.Fatalf("expected 1 mob for %q, got %d", tc.input, len(mobs))
		}
		if mobs[0].ShortDesc != tc.expected {
			t.Errorf("short desc %q: expected %q, got %q", tc.input, tc.expected, mobs[0].ShortDesc)
		}
	}
}

func TestParseMobFile_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	// Minimal mob with no race specified (only 3 fields in flags line)
	content := "#100\nkeyword~\nA mob~\nA mob stands here.\n~\n0 0 0 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	m := mobs[0]
	if m.Race != 7 {
		t.Errorf("expected default race 7, got %d", m.Race)
	}
	if m.Weight != 200 {
		t.Errorf("expected default weight 200, got %d", m.Weight)
	}
	if m.Height != 198 {
		t.Errorf("expected default height 198, got %d", m.Height)
	}
	if m.Str != 11 {
		t.Errorf("expected default str 11, got %d", m.Str)
	}
	if m.Int != 11 {
		t.Errorf("expected default int 11, got %d", m.Int)
	}
	if m.Wis != 11 {
		t.Errorf("expected default wis 11, got %d", m.Wis)
	}
	if m.Dex != 11 {
		t.Errorf("expected default dex 11, got %d", m.Dex)
	}
	if m.Con != 11 {
		t.Errorf("expected default con 11, got %d", m.Con)
	}
	if m.Cha != 11 {
		t.Errorf("expected default cha 11, got %d", m.Cha)
	}
}

func TestParseMobFile_TwoMobsLineBufferUnread(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob1~\nMob One~\nMob one stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n#200\nmob2~\nMob Two~\nMob two stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 2 {
		t.Fatalf("expected 2 mobs, got %d", len(mobs))
	}
	if mobs[0].VNum != 100 {
		t.Errorf("expected first vnum 100, got %d", mobs[0].VNum)
	}
	if mobs[1].VNum != 200 {
		t.Errorf("expected second vnum 200, got %d", mobs[1].VNum)
	}
	if mobs[0].Keywords != "mob1" {
		t.Errorf("expected first keywords 'mob1', got %q", mobs[0].Keywords)
	}
	if mobs[1].Keywords != "mob2" {
		t.Errorf("expected second keywords 'mob2', got %q", mobs[1].Keywords)
	}
}

func TestParseMobFile_ThreeMobs(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob1~\nMob One~\nOne.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n#200\nmob2~\nMob Two~\nTwo.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n#300\nmob3~\nMob Three~\nThree.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 3 {
		t.Fatalf("expected 3 mobs, got %d", len(mobs))
	}
	if mobs[0].VNum != 100 || mobs[1].VNum != 200 || mobs[2].VNum != 300 {
		t.Errorf("expected vnums [100,200,300], got [%d,%d,%d]", mobs[0].VNum, mobs[1].VNum, mobs[2].VNum)
	}
}

func TestParseMobFile_VNumZero(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#0\nmob0~\nMob Zero~\nZero stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("expected 1 mob, got %d", len(mobs))
	}
	if mobs[0].VNum != 0 {
		t.Errorf("expected vnum 0, got %d", mobs[0].VNum)
	}
}

func TestParseMobFile_VNum99999Sentinel(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob1~\nMob One~\nOne.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n#99999\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("expected 1 mob before sentinel, got %d", len(mobs))
	}
	if mobs[0].VNum != 100 {
		t.Errorf("expected vnum 100, got %d", mobs[0].VNum)
	}
}

func TestParseMobFile_HighVNum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#32767\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("expected 1 mob, got %d", len(mobs))
	}
	if mobs[0].VNum != 32767 {
		t.Errorf("expected vnum 32767, got %d", mobs[0].VNum)
	}
}

func TestParseMobFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 0 {
		t.Errorf("expected 0 mobs from empty file, got %d", len(mobs))
	}
}

func TestParseMobFile_CommentsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "* This is a comment\n* Another comment\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 0 {
		t.Errorf("expected 0 mobs from comment-only file, got %d", len(mobs))
	}
}

func TestParseMobFile_TruncatedMissingKeywords(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	_, err := ParseMobFile(testFile)
	if err == nil {
		t.Fatal("expected error for truncated file, got nil")
	}
}

func TestParseMobFile_TruncatedMissingShortDesc(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nkeyword~\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	_, err := ParseMobFile(testFile)
	if err == nil {
		t.Fatal("expected error for truncated file missing short desc, got nil")
	}
}

func TestParseMobFile_EspecBareHandAttack(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nBareHandAttack: 5\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("expected 1 mob, got %d", len(mobs))
	}
	if mobs[0].BareHandAttack != 5 {
		t.Errorf("expected bare hand attack 5, got %d", mobs[0].BareHandAttack)
	}
}

func TestParseMobFile_EspecAllStats(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nStr: 18\nInt: 16\nWis: 14\nDex: 12\nCon: 17\nCha: 10\nStrAdd: 50\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	m := mobs[0]
	if m.Str != 18 {
		t.Errorf("expected str 18, got %d", m.Str)
	}
	if m.Int != 16 {
		t.Errorf("expected int 16, got %d", m.Int)
	}
	if m.Wis != 14 {
		t.Errorf("expected wis 14, got %d", m.Wis)
	}
	if m.Dex != 12 {
		t.Errorf("expected dex 12, got %d", m.Dex)
	}
	if m.Con != 17 {
		t.Errorf("expected con 17, got %d", m.Con)
	}
	if m.Cha != 10 {
		t.Errorf("expected cha 10, got %d", m.Cha)
	}
	if m.StrAdd != 50 {
		t.Errorf("expected stradd 50, got %d", m.StrAdd)
	}
}

func TestParseMobFile_EspecRace(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nRace: 3\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if mobs[0].Race != 3 {
		t.Errorf("expected race 3, got %d", mobs[0].Race)
	}
	if mobs[0].RaceStr != "3" {
		t.Errorf("expected raceStr '3', got %q", mobs[0].RaceStr)
	}
}

func TestParseMobFile_EspecNoise(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nNoise: The mob growls.~\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if mobs[0].Noise != "The mob growls." {
		t.Errorf("expected noise 'The mob growls.', got %q", mobs[0].Noise)
	}
}

func TestParseMobFile_EspecNoiseInline(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nNoise: bark bark\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if mobs[0].Noise != "bark bark" {
		t.Errorf("expected noise 'bark bark', got %q", mobs[0].Noise)
	}
}

func TestParseMobFile_EspecScript(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nScript: myscript.lua 5\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if mobs[0].ScriptName != "myscript.lua" {
		t.Errorf("expected script 'myscript.lua', got %q", mobs[0].ScriptName)
	}
	if mobs[0].LuaFunctions != 5 {
		t.Errorf("expected lua functions 5, got %d", mobs[0].LuaFunctions)
	}
}

func TestParseMobFile_StatClampingLow(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nStr: 1\nInt: 2\nWis: -5\nDex: 0\nCon: -10\nCha: -99\nStrAdd: -5\nBareHandAttack: -1\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	m := mobs[0]
	if m.Str != 3 {
		t.Errorf("str clamp low: expected 3, got %d", m.Str)
	}
	if m.Int != 3 {
		t.Errorf("int clamp low: expected 3, got %d", m.Int)
	}
	if m.Wis != 3 {
		t.Errorf("wis clamp low: expected 3, got %d", m.Wis)
	}
	if m.Dex != 3 {
		t.Errorf("dex clamp low: expected 3, got %d", m.Dex)
	}
	if m.Con != 3 {
		t.Errorf("con clamp low: expected 3, got %d", m.Con)
	}
	if m.Cha != 3 {
		t.Errorf("cha clamp low: expected 3, got %d", m.Cha)
	}
	if m.StrAdd != 0 {
		t.Errorf("stradd clamp low: expected 0, got %d", m.StrAdd)
	}
	if m.BareHandAttack != 0 {
		t.Errorf("barehand clamp low: expected 0, got %d", m.BareHandAttack)
	}
}

func TestParseMobFile_StatClampingHigh(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\nStr: 30\nInt: 99\nWis: 100\nDex: 250\nCon: 500\nCha: 999\nStrAdd: 150\nBareHandAttack: 200\nE\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	m := mobs[0]
	if m.Str != 25 {
		t.Errorf("str clamp high: expected 25, got %d", m.Str)
	}
	if m.Int != 25 {
		t.Errorf("int clamp high: expected 25, got %d", m.Int)
	}
	if m.Wis != 25 {
		t.Errorf("wis clamp high: expected 25, got %d", m.Wis)
	}
	if m.Dex != 25 {
		t.Errorf("dex clamp high: expected 25, got %d", m.Dex)
	}
	if m.Con != 25 {
		t.Errorf("con clamp high: expected 25, got %d", m.Con)
	}
	if m.Cha != 25 {
		t.Errorf("cha clamp high: expected 25, got %d", m.Cha)
	}
	if m.StrAdd != 100 {
		t.Errorf("stradd clamp high: expected 100, got %d", m.StrAdd)
	}
	if m.BareHandAttack != 99 {
		t.Errorf("barehand clamp high: expected 99, got %d", m.BareHandAttack)
	}
}

func TestParseMobFile_SimpleFlag(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 S\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(mobs) != 1 {
		t.Fatalf("expected 1 mob, got %d", len(mobs))
	}
}

func TestParseMobFile_MultiLineDesc(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\nLine two of desc.\nLine three.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	expected := "Line two of desc.\nLine three.\n"
	if mobs[0].DetailedDesc != expected {
		t.Errorf("expected detailed desc %q, got %q", expected, mobs[0].DetailedDesc)
	}
}

func TestParseMobFile_ACMultipliedByTen(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mob")

	content := "#100\nmob~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 -5 1 1 0 1 1 0\n0 0\n8 3 0\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	mobs, err := ParseMobFile(testFile)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if mobs[0].AC != -50 {
		t.Errorf("expected AC -50 (10 * -5), got %d", mobs[0].AC)
	}
}

func TestParseAllMobFiles(t *testing.T) {
	tmpDir := t.TempDir()
	content1 := "#100\nmob1~\nMob one~\nMob one stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	content2 := "#200\nmob2~\nMob two~\nMob two stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "a.mob"), []byte(content1), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "b.mob"), []byte(content2), 0644)

	mobs, err := ParseAllMobFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all mob files: %v", err)
	}
	if len(mobs) != 2 {
		t.Errorf("expected 2 mobs total, got %d", len(mobs))
	}
}