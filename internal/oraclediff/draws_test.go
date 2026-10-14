package oraclediff

import (
	"os"
	"path/filepath"
	"testing"
)

func n(a, b, v int) DrawCall { return DrawCall{Method: "Number", A: a, B: b, Value: v, Consumed: 1} }
func d(k, s, v int) DrawCall { return DrawCall{Method: "Dice", A: k, B: s, Value: v, Consumed: k} }

func TestCompareDrawsExactMatch(t *testing.T) {
	c := []DrawCall{n(1, 50, 43), n(0, 11, 3), d(26, 4, 73)}
	g := []DrawCall{n(1, 50, 43), n(0, 11, 3), d(26, 4, 73)}
	if r := CompareDraws(c, g); !r.Match || r.Stream != 28 {
		t.Fatalf("got %+v, want match over 28 stream values", r)
	}
}

func TestCompareDrawsSingleDieEqualsNumber(t *testing.T) {
	// C dice(1,50) and Go Number(1,50) are the same single draw.
	if r := CompareDraws([]DrawCall{d(1, 50, 43)}, []DrawCall{n(1, 50, 43)}); !r.Match {
		t.Fatalf("dice(1,50) vs Number(1,50): %+v", r)
	}
}

func TestCompareDrawsDiceAgainstNumbers(t *testing.T) {
	// dice(3,6)=9 against three Number(1,6) calls summing to 9, both ways.
	c := []DrawCall{d(3, 6, 9), n(0, 3, 1)}
	g := []DrawCall{n(1, 6, 2), n(1, 6, 3), n(1, 6, 4), n(0, 3, 1)}
	if r := CompareDraws(c, g); !r.Match {
		t.Fatalf("C dice vs Go numbers: %+v", r)
	}
	if r := CompareDraws(g, c); !r.Match {
		t.Fatalf("reverse: %+v", r)
	}
	// A wrong sum is a mismatch.
	g[2] = n(1, 6, 5)
	if r := CompareDraws(c, g); r.Match {
		t.Fatal("dice with a different sum must not match")
	}
}

// The PF-067 shape: C makes one extra loud_mobs number(0,20) gate draw.
func TestCompareDrawsReportsFirstExtraDraw(t *testing.T) {
	c := []DrawCall{n(0, 15, 13), n(0, 20, 0), n(0, 3, 1), n(0, 3, 1), n(1, 20, 15)}
	g := []DrawCall{n(0, 15, 13), n(0, 3, 0), n(0, 3, 1), n(1, 20, 10)}
	r := CompareDraws(c, g)
	if r.Match || r.CEntry != 1 || r.GoEntry != 1 || r.Stream != 1 {
		t.Fatalf("got %+v, want first mismatch at entry 1 after 1 agreed value", r)
	}
}

func TestCompareDrawsOneSideEndsEarly(t *testing.T) {
	r := CompareDraws([]DrawCall{n(0, 3, 1), n(0, 3, 2)}, []DrawCall{n(0, 3, 1)})
	if r.Match || r.CEntry != 1 || r.GoEntry != 1 {
		t.Fatalf("got %+v, want mismatch where Go ends", r)
	}
}

func TestStepFor(t *testing.T) {
	b := []DrawBoundary{{"setup", 10}, {"warmup", 12}, {"probe 1: hit gil-glash", 20}}
	cases := map[int]string{0: "setup", 9: "setup", 10: "warmup", 12: "probe 1: hit gil-glash", 19: "probe 1: hit gil-glash", 20: "after the last recorded step"}
	for idx, want := range cases {
		if got := StepFor(b, idx); got != want {
			t.Errorf("StepFor(%d) = %q, want %q", idx, got, want)
		}
	}
}

func TestReadDrawLogParsesBothFormats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "draws")
	body := "DRAW 0 1 Number(1,50) = 43\nDRAW 1 26 Dice(26,4) = 73\nDRAW 27 1 Uniform(0,0) = 512345\nnoise\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	calls, err := ReadDrawLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls[1] != d(26, 4, 73) || calls[2].Method != "Uniform" {
		t.Fatalf("parsed %+v", calls)
	}
	if missing, err := ReadDrawLog(filepath.Join(t.TempDir(), "absent")); err != nil || missing != nil {
		t.Fatalf("missing log = %v, %v; want empty", missing, err)
	}
}
