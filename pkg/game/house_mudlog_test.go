package game

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1371 batch 7 — house.c:570 (hcontrol pay) and house.c:699 (house transfer).
//
// C (src/house.c):
//
//	:569-570  sprintf(buf, "Payment for house %s collected by %s.", arg, GET_NAME(ch));
//	          mudlog(buf, NRM, MAX(LVL_IMMORT, GET_INVIS_LEV(ch)), TRUE);
//	:698-699  sprintf(buf, "%s transfered %s house to %s.", GET_NAME(ch), HSHR(ch), CAP(arg2));
//	          mudlog(buf, NRM, MAX(LVL_IMMORT, GET_INVIS_LEV(ch)), TRUE);
//
// Go producers: pkg/game/house_control.go:274 (World.HcontrolPayHouse) and
// pkg/game/house_player.go:165 (doHouseTransfer, reached through World.DoHouse).
//
// The house rows are NRM at MAX(LVL_IMMORT, invis) with file TRUE, so the
// discriminators are the same pair as the quit/LOSTEQ batch: a LOG2-only observer
// receives (type <= NRM) while a LOG1-only observer does not (type > BRF), and
// the level is the LVL_IMMORT floor raised by invisibility.

// mudlogRig is the shared observer set for this batch. Every assertion names one
// observer and one producer's exact bracketed bytes.
type mudlogRig struct {
	log1       *Player // 40, LOG1 only  (level 1): BRF
	log2       *Player // 40, LOG2 only  (level 2): NRM
	noFlag     *Player // 40, no bits    (level 0)
	floor      *Player // LVL_IMMORT, both bits
	underFloor *Player // LVL_IMMORT-1, both bits
	god        *Player // LVL_GOD, both bits (level 3): CMP
	underGod   *Player // LVL_GOD-1, both bits
	under36    *Player // 35, both bits
	at36       *Player // 36, both bits
}

func newMudlogRig(t *testing.T) (*milestoneProvider, *bytes.Buffer, mudlogRig) {
	t.Helper()
	mk := func(id int, name string, level int, log1, log2 bool) *Player {
		p := NewPlayer(id, name, 1001)
		p.SetLevel(level)
		p.SetPlrFlag(PrfLog1, log1)
		p.SetPlrFlag(PrfLog2, log2)
		return p
	}
	rig := mudlogRig{
		log1:       mk(401, "Riglog1", 40, true, false),
		log2:       mk(402, "Riglog2", 40, false, true),
		noFlag:     mk(403, "Rignone", 40, false, false),
		floor:      mk(404, "Rigfloor", LVL_IMMORT, true, true),
		underFloor: mk(405, "Rigunderfloor", LVL_IMMORT-1, true, true),
		god:        mk(406, "Riggod", LVL_GOD, true, true),
		underGod:   mk(407, "Rigundergod", LVL_GOD-1, true, true),
		under36:    mk(408, "Rigunder36", 35, true, true),
		at36:       mk(409, "Rigat36", 36, true, true),
	}
	provider := &milestoneProvider{
		players: []*Player{rig.log1, rig.log2, rig.noFlag, rig.floor, rig.underFloor, rig.god, rig.underGod, rig.under36, rig.at36},
		lines:   map[string]string{},
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(old) })
	file := &bytes.Buffer{}
	oldWriter := getLogWriter()
	SetLogWriter(file)
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	return provider, file, rig
}

// bracket is the exact consumer bytes of one mudlog payload.
func bracket(payload string) string { return "[ " + payload + " ]\r\n" }

// rigSaw reports whether an observer's mudlog stream holds this producer's exact
// bracketed line — never "received something".
func rigSaw(provider *milestoneProvider, p *Player, payload string) bool {
	return strings.Contains(provider.lines[p.Name], bracket(payload))
}

// houseMudlogFixture stages a house record, the player lookup, the observer rig
// and a redirected house-control file so HcontrolPayHouse's save is harmless.
func houseMudlogFixture(t *testing.T) (*World, *milestoneProvider, *bytes.Buffer, mudlogRig, *Player) {
	t.Helper()
	oldFilename := houseControlFilename
	houseControlFilename = filepath.Join(t.TempDir(), "house", "house_control.json")
	t.Cleanup(func() { houseControlFilename = oldFilename })

	oldNameByID, oldIDByName := getPlayerNameByID, getPlayerIDByName
	RegisterHousePlayerLookup(
		func(id int64) string {
			if id == 1 {
				return "Houselord"
			}
			return ""
		},
		func(name string) int64 {
			if strings.EqualFold(name, "Guest") {
				return 2
			}
			return -1
		},
	)
	t.Cleanup(func() { RegisterHousePlayerLookup(oldNameByID, oldIDByName) })

	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "House", Zone: 1, Flags: []string{"house"}}}})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.HouseControl = []HouseControl{{VNum: 1001, Owner: 1}}

	actor := NewPlayer(1, "Houselord", 1001)
	actor.SetSex(0) // HSHR(ch) -> "his"
	actor.SetLevel(LVL_IMMORT)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	w.MessageSink = func(string, []byte) {}

	provider, file, rig := newMudlogRig(t)
	return w, provider, file, rig, actor
}

func payHousePayload() string { return "Payment for house 1001 collected by Houselord." }
func transferPayload() string { return "Houselord transfered his house to Guest." }

// TestHouseMudlogProducers pins payload, file and the two negative observers for
// each house producer.
func TestHouseMudlogProducers(t *testing.T) {
	cases := []struct {
		name    string
		payload func() string
		drive   func(w *World, ch *Player)
	}{
		{"pay_house", payHousePayload, func(w *World, ch *Player) { w.HcontrolPayHouse(ch, "1001") }},
		{"transfer", transferPayload, func(w *World, ch *Player) { w.DoHouse(ch, "transfer Guest") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, provider, file, rig, actor := houseMudlogFixture(t)
			tc.drive(w, actor)
			payload := tc.payload()
			if got := strings.Count(file.String(), payload); got != 1 {
				t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
			}
			if !rigSaw(provider, rig.log2, payload) {
				t.Fatalf("LOG2-only observer did not receive %q", bracket(payload))
			}
			if rigSaw(provider, rig.log1, payload) {
				t.Fatalf("LOG1-only observer received %q: type <= BRF", bracket(payload))
			}
			if rigSaw(provider, rig.noFlag, payload) {
				t.Fatalf("no-flag observer received %q: type would be OFF", bracket(payload))
			}
			if rigSaw(provider, rig.underFloor, payload) {
				t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracket(payload))
			}
		})
	}
}

// TestHouseMudlogLevelDiscriminators: the LVL_IMMORT floor and the invisibility
// override (MAX(LVL_IMMORT, GET_INVIS_LEV(ch))), per producer.
func TestHouseMudlogLevelDiscriminators(t *testing.T) {
	cases := []struct {
		name    string
		payload func() string
		drive   func(w *World, ch *Player)
	}{
		{"pay_house", payHousePayload, func(w *World, ch *Player) { w.HcontrolPayHouse(ch, "1001") }},
		{"transfer", transferPayload, func(w *World, ch *Player) { w.DoHouse(ch, "transfer Guest") }},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/lv_immort_floor", func(t *testing.T) {
			w, provider, _, rig, actor := houseMudlogFixture(t)
			tc.drive(w, actor)
			payload := tc.payload()
			if rigSaw(provider, rig.underFloor, payload) {
				t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracket(payload))
			}
			if !rigSaw(provider, rig.floor, payload) {
				t.Fatalf("level-%d observer did not receive %q", LVL_IMMORT, bracket(payload))
			}
		})
		t.Run(tc.name+"/invis_raises_threshold", func(t *testing.T) {
			w, provider, _, rig, actor := houseMudlogFixture(t)
			actor.SetInvisLevel(36)
			tc.drive(w, actor)
			payload := tc.payload()
			if rigSaw(provider, rig.under36, payload) {
				t.Fatalf("level-35 observer received %q", bracket(payload))
			}
			if !rigSaw(provider, rig.at36, payload) {
				t.Fatalf("level-36 observer did not receive %q at invisibility 36", bracket(payload))
			}
		})
	}
}
