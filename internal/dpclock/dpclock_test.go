package dpclock

import (
	"os"
	"testing"
)

func TestFrozenTracksEnvironmentPresence(t *testing.T) {
	if err := os.Unsetenv("DP_CLOCK"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("DP_CLOCK") })
	if Frozen() {
		t.Fatal("Frozen() = true with DP_CLOCK unset")
	}

	t.Setenv("DP_CLOCK", "")
	if !Frozen() {
		t.Fatal("Frozen() = false with DP_CLOCK present")
	}
}
