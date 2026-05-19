package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestConcurrentCharCreation is a regression test for the World RWMutex deadlock
// (LRN-20260519-001). Manager.Register() and World.GiveStartingItems() both need
// locks; this test verifies they can run concurrently without deadlocking.
//
// If the deadlock returns, this test will hang and timeout after 30 seconds.
func TestConcurrentCharCreation(t *testing.T) {
	m := makeTestManager(t)
	const numClients = 5

	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			name := fmt.Sprintf("TestCharUser%d", id)
			s := makeTestSession(t, m, name, 1001, false)

			// Login — nil DB path requires a password and enters char creation.
			loginMsg, _ := json.Marshal(map[string]interface{}{
				"type": MsgLogin,
				"data": map[string]interface{}{
					"player_name": name,
					"password":    "testpass123",
					"mode":        "player",
				},
			})
			if err := s.handleMessage(loginMsg); err != nil {
				t.Errorf("client %d: login error: %v", id, err)
				return
			}

			// Login triggers startCharCreation which sends the color prompt.
			select {
			case <-s.send:
			case <-time.After(5 * time.Second):
				t.Errorf("client %d: no response after login", id)
				return
			}

			// Walk through char creation stages: color→sex→race→class→hometown→stats_roll.
			// Each choice triggers a prompt sent to s.send (except the final accept).
			stages := []struct {
				choice   string
				expectMsg bool // whether this stage sends a response we must consume
			}{
				{"Y", true},  // color: yes ANSI → sex prompt
				{"M", true},  // sex: male → race prompt
				{"0", true},  // race: human → class prompt
				{"3", true},  // class: warrior → hometown prompt
				{"O", true},  // hometown: Old City → stats_roll prompt
				{"Y", false}, // stats_roll: accept → completeCharCreation (no msg if start room missing)
			}

			for _, stage := range stages {
				charMsg, _ := json.Marshal(map[string]interface{}{
					"type": MsgCharInput,
					"data": map[string]interface{}{"choice": stage.choice},
				})
				if err := s.handleMessage(charMsg); err != nil {
					t.Errorf("client %d: char_input %q error: %v", id, stage.choice, err)
					return
				}
				if stage.expectMsg {
					select {
					case <-s.send:
					case <-time.After(5 * time.Second):
						t.Errorf("client %d: timeout waiting for response after choice %q", id, stage.choice)
						return
					}
				}
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("All %d clients completed char creation without deadlock", numClients)
	case <-time.After(30 * time.Second):
		t.Fatal("DEADLOCK DETECTED: concurrent char creation timed out after 30s")
	}
}
