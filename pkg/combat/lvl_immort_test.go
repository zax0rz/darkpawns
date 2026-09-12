package combat

import "testing"

// cLVLImmort is the independent authoritative value from src/structs.h:620.
// Keep this literal separate from LVL_IMMORT so the canonical constant cannot
// drift together with its own fidelity proof.
const cLVLImmort = 31

func TestLVLImmortCanonicalValueMatchesC(t *testing.T) {
	if LVL_IMMORT != cLVLImmort {
		t.Fatalf("LVL_IMMORT = %d, want independent C value %d", LVL_IMMORT, cLVLImmort)
	}
}

func TestLVLImmortCompileTimeContexts(t *testing.T) {
	const mortalRows = LVL_IMMORT - 1
	var _ [LVL_IMMORT]int
	if mortalRows != cLVLImmort-1 {
		t.Fatalf("LVL_IMMORT-1 = %d, want %d mortal rows", mortalRows, cLVLImmort-1)
	}
}
