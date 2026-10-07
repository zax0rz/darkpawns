package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// VULN-027: the unauthenticated login body is a name and a password — an
// oversized body must be rejected before decode allocates, with 413, while a
// normal-sized body still reaches the credential path.
func TestAdminLoginBodyLimit(t *testing.T) {
	tracker := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{Threshold: 1000, Lockout: time.Minute})
	handler := handleLogin(nil, tracker)

	// Oversized: 100 KB of JSON padding in the password field.
	big := []byte(`{"player_name":"a","password":"` + strings.Repeat("x", 100*1024) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader(big))
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized login body: status = %d, want 413", rec.Code)
	}

	// Normal-sized wrong credentials: must still reach the auth path (401),
	// proving the limit did not break legitimate requests.
	req = httptest.NewRequest(http.MethodPost, "/admin/login",
		bytes.NewReader([]byte(`{"player_name":"someone","password":"wrong"}`)))
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusTooManyRequests && rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("normal login body: status = %d, want 401/429/503 (auth path reached)", rec.Code)
	}
}

// VULN-028: the player-detail response is built from live Inventory.Items /
// Equipment.Slots while gameplay goroutines mutate them — a fatal concurrent
// iteration. The builder must use the snapshot accessors. Run under -race.
func TestPlayerDetailSnapshotVsMutation(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "R", Zone: 1}},
		Objs:  []parser.Obj{{VNum: 7001, Keywords: "rock", ShortDesc: "a rock", LongDesc: "A rock."}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	p := game.NewPlayer(1, "Racer", 1001)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		obj := game.NewObjectInstance(&parser.Obj{VNum: 7001, Keywords: "rock", ShortDesc: "a rock", LongDesc: "A rock."}, -1)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = p.Inventory.AddItem(obj)
			p.Inventory.RemoveItem(obj)
		}
	}()
	for i := 0; i < 2000; i++ {
		_ = playerDetailToResponse(p)
	}
	close(stop)
	wg.Wait()
}

// VULN-031: store mutators and getters previously returned live pointers that
// handlers encoded after the lock released — racing concurrent updates. Every
// returned record must be a copy. Run under -race.
func TestAgentStoreReturnsCopies(t *testing.T) {
	store, err := NewAgentStore(t.TempDir() + "/agents.json")
	if err != nil {
		t.Fatalf("NewAgentStore: %v", err)
	}
	store.UpdateAgentStatus("daeron", "running")

	agents := store.GetAgents()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			_, _, _ = store.UpdateAgentStatus("daeron", "idle")
			_, _, _ = store.UpdateAgentStatus("daeron", "running")
		}
		close(stop)
	}()
	for {
		select {
		case <-stop:
			wg.Wait()
			goto done
		default:
		}
		for i := range agents {
			_ = agents[i].Status // would race live pointers; copies are stable
		}
	}
done:
	return
}

// VULN-030: prototype list getters previously returned live pointers read
// after the world lock released, racing admin PUTs to the same prototypes.
// Copies are stable. Run under -race.
func TestPrototypeGettersReturnCopies(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "R", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 5001, ShortDesc: "a mob"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	mobs := w.GetAllMobPrototypes()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			w.CommitEditedMob(parser.Mob{VNum: 5001, ShortDesc: "mutated"})
			w.CommitEditedMob(parser.Mob{VNum: 5001, ShortDesc: "a mob"})
		}
		close(stop)
	}()
	for {
		select {
		case <-stop:
			wg.Wait()
			goto done
		default:
		}
		for i := range mobs {
			_ = mobs[i].ShortDesc
		}
	}
done:
	return
}
