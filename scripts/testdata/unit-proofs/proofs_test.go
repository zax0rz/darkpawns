package fixture

import "testing"

const commentProof = "TestOnlyString"

func TestPass(t *testing.T) {
	if 2+2 != 4 {
		t.Fatal("arithmetic")
	}
}
func TestPrefixSuffix(t *testing.T) { t.Fatal("must never be selected by TestPrefix") }
func TestSkip(t *testing.T)         { t.Skip("not proven") }
func TestFail(t *testing.T)         { t.Fatal("expected fixture failure") }
func TestEmpty(t *testing.T)        {}
func TestSub(t *testing.T) {
	t.Run("empty", func(t *testing.T) {})
	t.Run("real name", func(t *testing.T) {
		if 2+2 != 4 {
			t.Fatal("arithmetic")
		}
	})
	t.Run("skipped", func(t *testing.T) { t.Skip("not proven") })
	if false {
		t.Run("unreachable", func(t *testing.T) { t.Fatal("not executed") })
	}
}
