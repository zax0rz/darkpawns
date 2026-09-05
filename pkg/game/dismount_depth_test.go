package game

import (
	"reflect"
	"testing"
)

func TestDismountDepthMatchesCOrderAndPairCleanup(t *testing.T) {
	w, rider := newMovementTestWorld(t)
	mount := addMovementMount(t, w, rider, 1001)
	observer := addMovementPlayer(t, w, "Observer", 1001)
	rider.SetFollowing("A different leader")

	var events []string
	w.MessageSink = func(name string, message []byte) {
		events = append(events, name+":"+string(message))
	}

	if !w.doDismount(rider, nil, "dismount", "ignored argument") {
		t.Fatal("doDismount returned false")
	}

	wantEvents := []string{
		rider.Name + ":You hop off your mount.\r\n",
		observer.Name + ":TestPlayer dismounts from the back of a pony.\r\n",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("dismount events = %#v, want %#v", events, wantEvents)
	}
	if rider.IsMounted() || rider.GetMountName() != "" {
		t.Fatalf("rider remained mounted: affect=%t mount=%q", rider.IsMounted(), rider.GetMountName())
	}
	if mount.IsAffected(affMounted) || mount.GetMountRider() != "" {
		t.Fatalf("mount pair state = affect:%t rider:%q", mount.IsAffected(affMounted), mount.GetMountRider())
	}
	if got := rider.GetFollowing(); got != "A different leader" {
		t.Fatalf("ordinary following relation = %q, want preserved", got)
	}
}
