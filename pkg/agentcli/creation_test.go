package agentcli

import (
	"encoding/json"
	"testing"
)

func TestCreatorStartsIdle(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{})
	if cc.state != CreationIdle {
		t.Errorf("expected CreationIdle, got %d", cc.state)
	}
	if cc.IsDone() {
		t.Error("expected IsDone() false at start")
	}
}

func TestCreatorIdleAdvancesToColor(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{Color: "normal"})
	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Color:","options":["ansi","normal"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if (*resp)["choice"] != "normal" {
		t.Errorf("expected normal, got %v", (*resp)["choice"])
	}
	if cc.state != CreationColor {
		t.Errorf("expected CreationColor, got %d", cc.state)
	}
}

func TestFullCreationFlow(t *testing.T) {
	cfg := CreationConfig{
		Color:    "ansi",
		Sex:      "female",
		Race:     "elf",
		Class:    "mage",
		Hometown: "midgaard",
	}
	cc := NewCharacterCreator(cfg)

	steps := []struct {
		prompt string
		want   string
	}{
		{`{"prompt":"Color:","options":["ansi","normal"]}`, "ansi"},
		{`{"prompt":"Sex:","options":["male","female"]}`, "female"},
		{`{"prompt":"Race:","options":["human","elf","dwarf","kender","minotaur","rakshasa","ssaur"]}`, "elf"},
		{`{"prompt":"Class:","options":["mage","cleric","thief","warrior"]}`, "mage"},
		{`{"prompt":"Hometown:","options":["midgaard","greyhawk"]}`, "midgaard"},
		{`{"prompt":"Stats: STR=14 INT=16 WIS=12 DEX=10 CON=13 CHA=11","stats":{"STR":14,"INT":16,"WIS":12,"DEX":10,"CON":13,"CHA":11}}`, "accept"},
	}

	for i, step := range steps {
		resp, err := cc.HandleCharCreate(json.RawMessage(step.prompt))
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if resp == nil {
			t.Fatalf("step %d: expected response", i)
		}
		if (*resp)["choice"] != step.want {
			t.Errorf("step %d: expected %s, got %v", i, step.want, (*resp)["choice"])
		}
	}

	if !cc.IsDone() {
		t.Error("expected IsDone() true after full flow")
	}
}

func TestCreatorDefaultHometown(t *testing.T) {
	cfg := CreationConfig{
		Color: "ansi",
		Sex:   "male",
		Race:  "human",
		Class: "warrior",
		// No hometown set — should pick first option
	}
	cc := NewCharacterCreator(cfg)

	// Advance through to hometown
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Color:","options":["ansi","normal"]}`))
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Sex:","options":["male","female"]}`))
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Race:","options":["human","elf"]}`))
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Class:","options":["mage","warrior"]}`))

	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Hometown:","options":["midgaard","greyhawk"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if (*resp)["choice"] != "midgaard" {
		t.Errorf("expected midgaard (first option), got %v", (*resp)["choice"])
	}
}

func TestAcceptFirstRollWhenNoRerollMin(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{RerollMin: 0})

	// Fast-forward to stats state
	for i := 0; i < 5; i++ {
		cc.HandleCharCreate(json.RawMessage(`{"prompt":"step","options":[]}`))
	}

	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":10,"INT":10,"WIS":10,"DEX":10,"CON":10,"CHA":10}}`))
	if err != nil {
		t.Fatal(err)
	}
	if (*resp)["choice"] != "accept" {
		t.Errorf("expected accept, got %v", (*resp)["choice"])
	}
	if !cc.IsDone() {
		t.Error("expected done")
	}
}

func TestRerollWhenBelowMin(t *testing.T) {
	cfg := CreationConfig{
		RerollMin:  80,
		MaxRerolls: 3,
	}
	cc := NewCharacterCreator(cfg)

	// Fast-forward to stats state
	for i := 0; i < 5; i++ {
		cc.HandleCharCreate(json.RawMessage(`{"prompt":"step","options":[]}`))
	}

	// Stats total = 60, below reroll_min of 80
	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":10,"INT":10,"WIS":10,"DEX":10,"CON":10,"CHA":10}}`))
	if err != nil {
		t.Fatal(err)
	}
	if (*resp)["choice"] != "reroll" {
		t.Errorf("expected reroll, got %v", (*resp)["choice"])
	}
	if cc.state != CreationStats {
		t.Errorf("expected still in CreationStats, got %d", cc.state)
	}
}

func TestAcceptWhenAboveMin(t *testing.T) {
	cfg := CreationConfig{
		RerollMin:  50,
		MaxRerolls: 3,
	}
	cc := NewCharacterCreator(cfg)

	// Fast-forward to stats state
	for i := 0; i < 5; i++ {
		cc.HandleCharCreate(json.RawMessage(`{"prompt":"step","options":[]}`))
	}

	// Stats total = 60, above reroll_min of 50
	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":12,"INT":10,"WIS":10,"DEX":8,"CON":10,"CHA":10}}`))
	if err != nil {
		t.Fatal(err)
	}
	if (*resp)["choice"] != "accept" {
		t.Errorf("expected accept, got %v", (*resp)["choice"])
	}
	if !cc.IsDone() {
		t.Error("expected done")
	}
}

func TestStopsAtMaxRerolls(t *testing.T) {
	cfg := CreationConfig{
		RerollMin:  100,
		MaxRerolls: 2,
	}
	cc := NewCharacterCreator(cfg)

	// Fast-forward to stats state
	for i := 0; i < 5; i++ {
		cc.HandleCharCreate(json.RawMessage(`{"prompt":"step","options":[]}`))
	}

	// First roll: 60, below 100 → reroll (rerollCount becomes 1)
	resp, _ := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":10,"INT":10,"WIS":10,"DEX":10,"CON":10,"CHA":10}}`))
	if (*resp)["choice"] != "reroll" {
		t.Fatalf("expected reroll, got %v", (*resp)["choice"])
	}

	// Second roll: 50, below 100, rerollCount hits max (2) → forced accept
	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":8,"INT":8,"WIS":10,"DEX":8,"CON":8,"CHA":8}}`))
	if err != nil {
		t.Fatal(err)
	}
	if (*resp)["choice"] != "accept" {
		t.Errorf("expected accept at max rerolls, got %v", (*resp)["choice"])
	}
	if !cc.IsDone() {
		t.Error("expected done")
	}
}

func TestIsDoneFalseUntilComplete(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{
		Color: "ansi",
		Sex:   "male",
		Race:  "human",
		Class: "warrior",
	})

	// After color — not done
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Color:","options":["ansi","normal"]}`))
	if cc.IsDone() {
		t.Error("not done yet")
	}

	// After sex — not done
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Sex:","options":["male","female"]}`))
	if cc.IsDone() {
		t.Error("not done yet")
	}

	// After stats — done
	for i := 0; i < 3; i++ {
		cc.HandleCharCreate(json.RawMessage(`{"prompt":"step","options":[]}`))
	}
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":14}}`))
	if !cc.IsDone() {
		t.Error("should be done now")
	}
}

func TestHandleCharCreateBadJSON(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{})
	_, err := cc.HandleCharCreate(json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestHandleCharCreateAfterComplete(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{})
	cc.state = CreationComplete

	resp, err := cc.HandleCharCreate(json.RawMessage(`{"prompt":"Color:","options":["ansi","normal"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil {
		t.Error("expected nil response when already complete")
	}
}

func TestRerollTracksBestTotal(t *testing.T) {
	cfg := CreationConfig{
		RerollMin:  90,
		MaxRerolls: 3,
	}
	cc := NewCharacterCreator(cfg)

	// Fast-forward to stats state
	for i := 0; i < 5; i++ {
		cc.HandleCharCreate(json.RawMessage(`{"prompt":"step","options":[]}`))
	}

	// First roll: 60
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":10,"INT":10,"WIS":10,"DEX":10,"CON":10,"CHA":10}}`))
	if cc.bestTotal != 60 {
		t.Errorf("expected bestTotal 60, got %d", cc.bestTotal)
	}

	// Second roll: 50 — bestTotal stays 60, rerollCount=2
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":8,"INT":8,"WIS":10,"DEX":8,"CON":8,"CHA":8}}`))
	if cc.bestTotal != 60 {
		t.Errorf("expected bestTotal 60, got %d", cc.bestTotal)
	}

	// Third roll: 55 — bestTotal stays 60, rerollCount=3 = max → accept
	cc.HandleCharCreate(json.RawMessage(`{"prompt":"Stats:","stats":{"STR":9,"INT":9,"WIS":10,"DEX":9,"CON":9,"CHA":9}}`))
	if cc.bestTotal != 60 {
		t.Errorf("expected bestTotal 60, got %d", cc.bestTotal)
	}
}

func TestNewCreatorSetsDefaultMaxRerolls(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{RerollMin: 50})
	if cc.cfg.MaxRerolls != 3 {
		t.Errorf("expected default MaxRerolls 3, got %d", cc.cfg.MaxRerolls)
	}
}

func TestNewCreatorPreservesCustomMaxRerolls(t *testing.T) {
	cc := NewCharacterCreator(CreationConfig{RerollMin: 50, MaxRerolls: 10})
	if cc.cfg.MaxRerolls != 10 {
		t.Errorf("expected MaxRerolls 10, got %d", cc.cfg.MaxRerolls)
	}
}
