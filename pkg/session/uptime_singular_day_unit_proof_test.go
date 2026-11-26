package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/admin"
)

// TestSendUptimeSingularDayUnitProof pins Round 3 row 27 under the 2026-11-11
// unit-proof exception (Zach's 2026-11-25 ruling 6): C do_date/uptime formats
// exactly 1 elapsed day with the singular "day" (src/act.wizard.c:1825-1826).
func TestSendUptimeSingularDayUnitProof(t *testing.T) {
	s := &Session{send: make(chan []byte, 4)}
	now := admin.ProcessStartTime().Add(25 * time.Hour)
	sendUptime(s, now)

	select {
	case raw := <-s.send:
		var env struct {
			Data struct {
				Text string `json:"text"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("unmarshal send frame: %v", err)
		}
		if !strings.Contains(env.Data.Text, ": 1 day, 1:00") {
			t.Fatalf("sendUptime(+25h) = %q, want substring %q", env.Data.Text, ": 1 day, 1:00")
		}
	default:
		t.Fatal("sendUptime emitted no frame")
	}
}
