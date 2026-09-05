package game

import (
	"strings"
	"testing"
)

func TestReviewGossipTraversesHistoryOldestFirst(t *testing.T) {
	w, viewer, _, _, _ := newChannelWorld(t)
	w.updateGossipHistory("First", "oldest", 0)
	w.updateGossipHistory("Second", "newest", 0)

	want := "Last Gossips:\r\n-------------\r\nFirst: oldest\r\nSecond: newest\r\n"
	if got := w.ReviewGossip(viewer); got != want {
		t.Fatalf("review = %q, want %q", got, want)
	}
}

func TestReviewGossipHidesHigherInvisibility(t *testing.T) {
	w, viewer, _, _, _ := newChannelWorld(t)
	viewer.Level = 10
	w.updateGossipHistory("Hidden", "secret", viewer.Level+1)
	w.updateGossipHistory("Visible", "public", viewer.Level)

	want := "Last Gossips:\r\n-------------\r\nSomeone invisible: secret\r\nVisible: public\r\n"
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
	if strings.Contains(review, "message a\r\n") {
		t.Fatal("review retained the oldest entry beyond the C window")
	}
	if !strings.Contains(review, "message b\r\n") || !strings.Contains(review, "message z\r\n") {
		t.Fatalf("review window = %q, want messages b through z", review)
	}
}
