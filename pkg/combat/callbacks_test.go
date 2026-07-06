package combat

import (
	"strings"
	"testing"
)

// mockCallbacks returns a GameCallbacks that records every call made to it.
func mockCallbacks() (*GameCallbacks, *callbackLog) {
	log := &callbackLog{}
	cb := &GameCallbacks{
		Broadcast: func(roomVNum int, msg string, exclude string) {
			log.broadcasts = append(log.broadcasts, callbackCall{room: roomVNum, msg: msg, exclude: exclude})
		},
		SendToChar: func(name string, msg string) {
			log.sendToChar = append(log.sendToChar, callbackCall{target: name, msg: msg})
		},
		SkillMessage: func(dam int, ch, vict string, attackType int, roomVNum int) bool {
			log.skillMessages = append(log.skillMessages, skillCall{dam: dam, ch: ch, vict: vict, attackType: attackType, room: roomVNum})
			return false
		},
		BroadChat: func(chName string, msg string) {
			log.broadChats = append(log.broadChats, callbackCall{target: chName, msg: msg})
		},
		Log: func(msg string, level string, minLevel int, toLog bool) {
			log.logs = append(log.logs, logCall{msg: msg, level: level, minLevel: minLevel, toLog: toLog})
		},
	}
	return cb, log
}

type callbackCall struct {
	room    int
	target  string
	msg     string
	exclude string
}

type skillCall struct {
	dam        int
	ch         string
	vict       string
	attackType int
	room       int
}

type logCall struct {
	msg      string
	level    string
	minLevel int
	toLog    bool
}

type callbackLog struct {
	broadcasts    []callbackCall
	sendToChar    []callbackCall
	skillMessages []skillCall
	broadChats    []callbackCall
	logs          []logCall
}

func TestGameCallbacks_DamMessage(t *testing.T) {
	cb, log := mockCallbacks()

	// Wire callbacks without touching the legacy package variables.
	origCallbacks := callbacks
	SetCallbacks(cb)
	defer SetCallbacks(origCallbacks)

	attacker := NewNamedCombatant("Alice", 100)
	defender := NewNamedCombatant("Bob", 100)

	DamMessage(25, attacker, defender, 0) // attackType 0 = "hit"

	if len(log.broadcasts) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(log.broadcasts))
	}
	if !strings.Contains(log.broadcasts[0].msg, "Alice") || !strings.Contains(log.broadcasts[0].msg, "Bob") {
		t.Errorf("broadcast message missing names: %q", log.broadcasts[0].msg)
	}
	if log.broadcasts[0].exclude != "Alice Bob" {
		t.Errorf("broadcast exclude = %q, want %q", log.broadcasts[0].exclude, "Alice Bob")
	}

	if len(log.sendToChar) != 2 {
		t.Fatalf("expected 2 sendToChar calls, got %d", len(log.sendToChar))
	}
	if log.sendToChar[0].target != "Alice" {
		t.Errorf("first sendToChar target = %q, want Alice", log.sendToChar[0].target)
	}
	if log.sendToChar[1].target != "Bob" {
		t.Errorf("second sendToChar target = %q, want Bob", log.sendToChar[1].target)
	}
}

func TestGameCallbacks_InitSkillMessages(t *testing.T) {
	cb, log := mockCallbacks()
	InitSkillMessages(cb)

	if cb.SkillMessage == nil {
		t.Fatal("InitSkillMessages did not set cb.SkillMessage")
	}

	// Use a known skill from the table (BACKSTAB = 131).
	sent := cb.SkillMessage(50, "Alice", "Bob", 131, 200)
	if !sent {
		t.Fatal("expected SkillMessage to return true for a known skill")
	}

	if len(log.broadcasts) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(log.broadcasts))
	}
	if log.broadcasts[0].room != 200 {
		t.Errorf("broadcast room = %d, want 200", log.broadcasts[0].room)
	}
	if len(log.sendToChar) != 2 {
		t.Fatalf("expected 2 sendToChar calls, got %d", len(log.sendToChar))
	}
	if log.sendToChar[0].target != "Alice" || log.sendToChar[1].target != "Bob" {
		t.Errorf("sendToChar targets = %v, want [Alice Bob]", []string{log.sendToChar[0].target, log.sendToChar[1].target})
	}
}

func TestGameCallbacks_LogAndBroadChatHelpers(t *testing.T) {
	cb, log := mockCallbacks()

	origCallbacks := callbacks
	SetCallbacks(cb)
	defer SetCallbacks(origCallbacks)

	cbLog("test log", "BRF", LVL_IMMORT, true)
	cbBroadChat("Alice", "brag message")

	if len(log.logs) != 1 {
		t.Fatalf("expected 1 log call, got %d", len(log.logs))
	}
	if log.logs[0].msg != "test log" {
		t.Errorf("log msg = %q, want %q", log.logs[0].msg, "test log")
	}

	if len(log.broadChats) != 1 {
		t.Fatalf("expected 1 broadChat call, got %d", len(log.broadChats))
	}
	if log.broadChats[0].target != "Alice" || log.broadChats[0].msg != "brag message" {
		t.Errorf("broadChat = %+v, want {target:Alice msg:brag message}", log.broadChats[0])
	}
}
