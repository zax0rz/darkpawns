package dprng

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"
)

type goldenStream struct {
	Seed       uint32   `json:"seed"`
	Next       []uint32 `json:"next"`
	Number1100 []int    `json:"number_1_100"`
	Number099  []int    `json:"number_0_99"`
	Number16   []int    `json:"number_1_6"`
	Dice26     []int    `json:"dice_2_6"`
}

func readGolden(t *testing.T, path string) goldenStream {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var golden goldenStream
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	return golden
}

func TestCMWCRawStreamMatchesCGolden(t *testing.T) {
	for _, path := range []string{"testdata/seed-1.json", "testdata/seed-42.json"} {
		golden := readGolden(t, path)
		generator := New(golden.Seed)
		got := make([]uint32, len(golden.Next))
		for i := range got {
			got[i] = generator.Next()
		}
		if !slices.Equal(got, golden.Next) {
			t.Fatalf("seed %d raw stream = %v, want %v", golden.Seed, got, golden.Next)
		}
	}
}

func TestNumberAndDiceMatchCGoldenContinuousStream(t *testing.T) {
	golden := readGolden(t, "testdata/seed-42.json")
	generator := New(golden.Seed)

	assertNumbers := func(name string, from, to int, want []int) {
		t.Helper()
		got := make([]int, len(want))
		for i := range got {
			got[i] = generator.Number(from, to)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
	assertNumbers("number(1,100)", 1, 100, golden.Number1100)
	assertNumbers("number(0,99)", 0, 99, golden.Number099)
	assertNumbers("number(1,6)", 1, 6, golden.Number16)

	gotDice := make([]int, len(golden.Dice26))
	for i := range gotDice {
		gotDice[i] = generator.Dice(2, 6)
	}
	if !slices.Equal(gotDice, golden.Dice26) {
		t.Fatalf("dice(2,6) = %v, want %v", gotDice, golden.Dice26)
	}
}

func TestSeedPreservesCarryAndIndexLikeC(t *testing.T) {
	reseeded := New(42)
	_ = reseeded.Next()
	reseeded.Seed(42)

	fresh := New(42)
	if got, first := reseeded.Next(), fresh.Next(); got == first {
		t.Fatalf("reseed reset persistent state: reseeded next %d equals fresh next %d", got, first)
	}
}

func TestNumberRangeSwapAndDiceDrawCount(t *testing.T) {
	forward := New(42)
	reversed := New(42)
	if got, want := reversed.Number(100, 1), forward.Number(1, 100); got != want {
		t.Fatalf("reversed range = %d, want %d", got, want)
	}

	dice := New(42)
	manual := New(42)
	got := dice.Dice(3, 6)
	want := manual.Number(1, 6) + manual.Number(1, 6) + manual.Number(1, 6)
	if got != want || dice.Next() != manual.Next() {
		t.Fatal("dice did not consume exactly one draw per die")
	}
	if got := dice.Dice(0, 6); got != 0 {
		t.Fatalf("Dice(0,6) = %d, want 0", got)
	}
}

func TestSeedFromEnvironment(t *testing.T) {
	fixedTime := func() time.Time { return time.Unix(123456789, 0) }

	seed, err := seedFromEnvironment(func(key string) (string, bool) { return "42", true }, fixedTime)
	if err != nil || seed != 42 {
		t.Fatalf("DP_SEED=42 returned seed %d, err %v", seed, err)
	}

	seed, err = seedFromEnvironment(func(key string) (string, bool) { return "", false }, fixedTime)
	if err != nil || seed != 123456789 {
		t.Fatalf("unset DP_SEED returned seed %d, err %v", seed, err)
	}
	laterSeed, err := seedFromEnvironment(
		func(key string) (string, bool) { return "", false },
		func() time.Time { return fixedTime().Add(time.Second) },
	)
	if err != nil || laterSeed == seed {
		t.Fatalf("time fallback did not vary: first %d, later %d, err %v", seed, laterSeed, err)
	}

	if _, err := seedFromEnvironment(func(key string) (string, bool) { return "not-a-seed", true }, fixedTime); err == nil {
		t.Fatal("invalid DP_SEED did not return an error")
	}
}
