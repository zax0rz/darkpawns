package game

import (
	"reflect"
	"testing"
)

func TestDismountDepthMatchesCOrderAndAffectUnmount(t *testing.T) {
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
	if mount.IsAffected(affMounted) || mount.GetMountRider() != rider.Name {
		t.Fatalf("mount state after C unmount = affect:%t rider:%q", mount.IsAffected(affMounted), mount.GetMountRider())
	}
	if got := rider.GetFollowing(); got != "A different leader" {
		t.Fatalf("rider's ordinary following relation = %q, want preserved", got)
	}
	if got := mount.GetFollowing(); got != rider.Name {
		t.Fatalf("mount's master link = %q, want retained", got)
	}
}

func TestCombatDismountCallbackUsesCDoDismount(t *testing.T) {
	w, rider := newMovementTestWorld(t)
	addMovementMount(t, w, rider, 1001)
	observer := addMovementPlayer(t, w, "Observer", 1001)

	var events []string
	w.MessageSink = func(name string, message []byte) {
		events = append(events, name+":"+string(message))
	}
	callback := w.WireCombatCallbacks().Dismount
	if callback == nil {
		t.Fatal("combat callbacks have no dismount route")
	}

	callback(rider)

	want := []string{
		rider.Name + ":You hop off your mount.\r\n",
		observer.Name + ":TestPlayer dismounts from the back of a pony.\r\n",
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("combat dismount events = %#v, want %#v", events, want)
	}
	if rider.IsMounted() {
		t.Fatal("combat dismount left rider mounted")
	}
}

func TestCombatActCharPreservesCEmbeddedLineBreak(t *testing.T) {
	w, rider := newMovementTestWorld(t)
	var got string
	w.MessageSink = func(name string, message []byte) {
		if name == rider.Name {
			got += string(message)
		}
	}
	callback := w.WireCombatCallbacks().ActChar
	if callback == nil {
		t.Fatal("combat callbacks have no private act route")
	}

	callback(rider, "The hit knocks you off of your mount!\r\n")

	want := "The hit knocks you off of your mount!\r\n\r\n"
	if got != want {
		t.Fatalf("private act bytes = %q, want %q", got, want)
	}
}
