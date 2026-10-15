package game

import (
	"strings"
	"testing"
)

func TestReviewGossipTraversesHistoryOldestFirst(t *testing.T) {
	w, viewer, _, _, _ := newChannelWorld(t)
	w.updateGossipHistory("First", "oldest", 0)
	w.updateGossipHistory("Second", "newest", 0)

	want := "Last Gossips:\r\n-------------\r\nFirst: oldest\n\rSecond: newest\n\r"
	if got := w.ReviewGossip(viewer); got != want {
		t.Fatalf("review = %q, want %q", got, want)
	}
}

func TestReviewGossipHidesHigherInvisibility(t *testing.T) {
	w, viewer, _, _, _ := newChannelWorld(t)
	viewer.Level = 10
	w.updateGossipHistory("Hidden", "secret", viewer.Level+1)
	w.updateGossipHistory("Visible", "public", viewer.Level)

	want := "Last Gossips:\r\n-------------\r\nSomeone invisible: secret\n\rVisible: public\n\r"
	if got := w.ReviewGossip(viewer); got != want {
		t.Fatalf("review = %q, want %q", got, want)
	}
}

func TestReviewGossipRetainsTheCappedHistoryWindow(t *testing.T) {
	w, viewer, _, _, _ := newChannelWorld(t)
	for index := 1; index <= maxGossipHistory+1; index++ {
		w.updateGossipHistory("Speaker", "message "+string(rune('a'+index-1)), 0)
	}

	if got := len(w.gossipHistory); got != maxGossipHistory {
		t.Fatalf("history length = %d, want %d", got, maxGossipHistory)
	}
	review := w.ReviewGossip(viewer)
	if strings.Contains(review, "message a\n\r") {
		t.Fatal("review retained the oldest entry beyond the C window")
	}
	if !strings.Contains(review, "message b\n\r") || !strings.Contains(review, "message z\n\r") {
		t.Fatalf("review window = %q, want messages b through z", review)
	}
}

// TestReviewGossipColorsAtComplete pins do_review's C_CMP color sandwich
// (src/new_cmds.c:1378-1387): CCCYN before the name, CCYEL before the text,
// CCNRM after it, and the "\n\r" entry terminator. Below complete color every
// macro is KNUL, so the bytes are plain.
func TestReviewGossipColorsAtComplete(t *testing.T) {
	w, viewer, _, _, _ := newChannelWorld(t)
	w.updateGossipHistory("Stink", "peee-YEW", 0)

	viewer.SetPlrFlag(PrfColor1, true)
	viewer.SetPlrFlag(PrfColor2, true)
	want := "Last Gossips:\r\n-------------\r\n\x1b[36mStink: \x1b[33mpeee-YEW\x1b[0m\n\r"
	if got := w.ReviewGossip(viewer); got != want {
		t.Fatalf("complete-color review = %q, want %q", got, want)
	}

	// C_NRM (color 2) is below C_CMP: no color bytes at all.
	viewer.SetPlrFlag(PrfColor1, false)
	want = "Last Gossips:\r\n-------------\r\nStink: peee-YEW\n\r"
	if got := w.ReviewGossip(viewer); got != want {
		t.Fatalf("normal-color review = %q, want %q", got, want)
	}
}
