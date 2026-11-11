package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Depth tests for C's full_moon() and lunar_hunter() (src/new_cmds.c:1310-1328
// and :2662-2683, called from the sunset arm at src/weather.c:71-75 on days
// 22-25). The port previously broadcast two invented lines (R4); these tests
// pin the faithful behavior: force-transforms and hunter spawns with draws.

func fullMoonTestWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 8004, Name: "Moon Test Room", Zone: 8},
			{VNum: 8067, Name: "Hunter Lair", Zone: 80},
			{VNum: huntingStartRoom, Name: "Temple", Zone: 182},
		},
		Mobs: []parser.Mob{
			// Mirrors the real vnum-7 prototype (lib/world/mob/0.mob): gold 9000
			// and hit dice 12d5+130, so read_mobile's HP/gold draws match the
			// oracle's hunter spawns draw for draw.
			{VNum: 7, Keywords: "lunatic", ShortDesc: "a lunatic", LongDesc: "A lunatic stands here.\r\n", Level: 1, Gold: 9000, AC: 100, HP: parser.DiceRoll{Num: 12, Sides: 5, Plus: 130}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func setSunsetClimate(t *testing.T, day, moon int) {
	t.Helper()
	weatherMu.Lock()
	originalTime, originalWeather := timeInfo, weatherInfo
	timeInfo = TimeInfoData{Hours: 21, Day: day, Month: 0, Year: 0, Moon: moon}
	weatherInfo = WeatherData{Pressure: 1013, Change: 0, Sky: SkyCloudless, Sunlight: SunSet}
	weatherMu.Unlock()
	t.Cleanup(func() {
		weatherMu.Lock()
		timeInfo, weatherInfo = originalTime, originalWeather
		weatherMu.Unlock()
	})
}

// TestFullMoonTransformsWerewolf pins src/new_cmds.c:1310-1328: a PLR_WEREWOLF
// player who is not already transformed receives the two lunar lines and is
// force-transformed by do_transform. Before the fix this test failed because
// fullMoon broadcast an invented line and transformed nobody.
func TestFullMoonTransformsWerewolf(t *testing.T) {
	w := fullMoonTestWorld(t)
	setSunsetClimate(t, 21, MoonFull)

	messages := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

	wolf := NewPlayer(1, "Moonwolf", 8004)
	if err := w.AddPlayer(wolf); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	wolf.SetPlrFlag(PlrWerewolf, true)
	wolf.SetMaxHP(100)
	wolf.SetHP(100)

	plain := NewPlayer(2, "Plainone", 8004)
	if err := w.AddPlayer(plain); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	SetWeatherWorld(w)
	t.Cleanup(func() { SetWeatherWorld(nil) })

	fullMoon()

	if !wolf.IsAffected(affWerewolf) {
		t.Fatal("werewolf not transformed by full_moon")
	}
	// C adds GET_MAX_HIT/2 at MoonFull (src/act.other.c:1440-1442), capped at 666.
	if got := wolf.GetHP(); got != 150 {
		t.Errorf("werewolf HP after transform = %d, want 150 (100 + 100/2)", got)
	}
	got := messages["Moonwolf"]
	if !strings.Contains(got, "The lunar light infuses your body, forcing you to transform!\r\n") {
		t.Errorf("werewolf messages missing lunar line 1, got %q", got)
	}
	if !strings.Contains(got, "Racked with the pain of the transformation, your head is thrown\r\nback and an unearthly moan escapes your lips.\r\n") {
		t.Errorf("werewolf messages missing lunar line 2, got %q", got)
	}
	if strings.Contains(got, "FULL MOON RISES") {
		t.Errorf("werewolf received invented broadcast, got %q", got)
	}
	if plain.IsAffected(affWerewolf) || strings.Contains(messages["Plainone"], "lunar light") {
		t.Error("non-werewolf player was transformed or messaged")
	}
	if strings.Contains(messages["Plainone"], "FULL MOON RISES") || strings.Contains(messages["Plainone"], "LUNAR HUNTER") {
		t.Errorf("plain player received invented broadcast, got %q", messages["Plainone"])
	}
}

// TestLunarHunterSpawnsOnZeroRoll pins src/new_cmds.c:2662-2683: for each
// playing player at level 13..immort, number(0,5) is rolled; on 0 a vnum-7
// hunter at level (level/2)+2 is created hunting that player, moved to room
// 8067, and tells them "By the light of the full moon.. DIE!". Before the fix
// this test failed because lunarHunter broadcast an invented line and spawned
// nobody.
func TestLunarHunterSpawnsOnZeroRoll(t *testing.T) {
	w := fullMoonTestWorld(t)

	var gotTell string
	w.MessageSink = func(name string, msg []byte) {
		if name == "Moonhigh" && strings.Contains(string(msg), "By the light of the full moon") {
			gotTell += string(msg)
		}
	}

	high := NewPlayer(1, "Moonhigh", 8004)
	if err := w.AddPlayer(high); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	high.SetLevel(20)

	SetWeatherWorld(w)
	t.Cleanup(func() { SetWeatherWorld(nil) })

	// Find a seed whose first number(0,5) is 0 so the spawn is forced.
	seed := uint32(0)
	for s := uint32(1); s < 1000; s++ {
		if dprng.New(s).Number(0, 5) == 0 {
			seed = s
			break
		}
	}
	if seed == 0 {
		t.Fatal("no seed in 1..1000 yields number(0,5)==0")
	}
	dprng.ResetStream(seed)
	t.Cleanup(func() { dprng.ResetStream(1) })

	lunarHunter()

	hunters := 0
	for _, mob := range w.GetAllMobs() {
		if mob.GetVNum() != 7 {
			continue
		}
		hunters++
		if got := mob.GetLevel(); got != 12 {
			t.Errorf("hunter level = %d, want 12 (20/2+2)", got)
		}
		if got := mob.GetRoom(); got != 8067 {
			t.Errorf("hunter room = %d, want 8067", got)
		}
		if got := mob.GetHunting(); got != "Moonhigh" {
			t.Errorf("hunter target = %q, want Moonhigh", got)
		}
	}
	if hunters != 1 {
		t.Fatalf("hunters spawned = %d, want 1", hunters)
	}
	if !strings.Contains(gotTell, "By the light of the full moon.. DIE!") {
		t.Errorf("Moonhigh tell = %q, want hunter DIE line", gotTell)
	}
	// Draw sequence for one spawn: the number(0,5) roll, then read_mobile's
	// HP dice(12,5) and gold number(0,1)+number(1,20) (src/db.c:1749-1774).
	// The reference replays the same calls on an independent stream so any
	// moved, added, or re-gated draw fails here.
	replaySpawn := func(r *dprng.Generator) {
		r.Dice(12, 5)
		r.Number(0, 1)
		r.Number(1, 20)
	}
	ref := dprng.New(seed)
	ref.Number(0, 5)
	replaySpawn(ref)
	want := ref.Number(0, 999)
	if got := dprng.Number(0, 999); got != want {
		t.Errorf("next draw = %d, want %d (roll + spawn draws consumed)", got, want)
	}
}

// TestLunarHunterRollsInDescriptorOrder pins the draw order: C iterates
// descriptor_list newest-first (src/comm.c:1591-1603 prepends), so the newest
// eligible player consumes the first number(0,5). Go must match or the hunter
// targets diverge.
func TestLunarHunterRollsInDescriptorOrder(t *testing.T) {
	w := fullMoonTestWorld(t)

	// "First" is created first (oldest); "Second" is created last (newest).
	first := NewPlayer(1, "Firstel", 8004)
	if err := w.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	first.SetLevel(20)
	second := NewPlayer(2, "Secondel", 8004)
	if err := w.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	second.SetLevel(14)

	SetWeatherWorld(w)
	t.Cleanup(func() { SetWeatherWorld(nil) })

	// Find a seed where the first roll is 0 and, after the spawn's
	// read_mobile draws, the second roll is not, so the single hunter's
	// target reveals which player consumed the first draw. The spawn draws
	// sit between the two rolls (C create_mobile runs inside the loop), so
	// the search must interleave them.
	seed := uint32(0)
	for s := uint32(1); s < 50000; s++ {
		r := dprng.New(s)
		if r.Number(0, 5) != 0 {
			continue
		}
		r.Dice(12, 5)
		r.Number(0, 1)
		r.Number(1, 20)
		if r.Number(0, 5) != 0 {
			seed = s
			break
		}
	}
	if seed == 0 {
		t.Fatal("no seed in 1..50000 yields first=0, second!=0 with spawn draws")
	}
	dprng.ResetStream(seed)
	t.Cleanup(func() { dprng.ResetStream(1) })

	lunarHunter()

	hunters := 0
	for _, mob := range w.GetAllMobs() {
		if mob.GetVNum() != 7 {
			continue
		}
		hunters++
		// Newest descriptor first: Secondel (level 14 → hunter level 9).
		if got := mob.GetHunting(); got != "Secondel" {
			t.Errorf("hunter target = %q, want Secondel (descriptor order)", got)
		}
		if got := mob.GetLevel(); got != 9 {
			t.Errorf("hunter level = %d, want 9 (14/2+2)", got)
		}
	}
	if hunters != 1 {
		t.Fatalf("hunters spawned = %d, want 1", hunters)
	}
	// Both eligible players must have consumed a draw, with the first
	// spawn's read_mobile draws interleaved between the rolls.
	ref := dprng.New(seed)
	ref.Number(0, 5)
	ref.Dice(12, 5)
	ref.Number(0, 1)
	ref.Number(1, 20)
	ref.Number(0, 5)
	if got, want := dprng.Number(0, 999), ref.Number(0, 999); got != want {
		t.Errorf("next draw = %d, want %d (two rolls + spawn draws consumed)", got, want)
	}
}

// TestLunarHunterSkipsIneligibleWithoutDraw pins the C short-circuit: the
// number(0,5) roll happens only when the level gate passes, so levels <=12
// and >= immort consume no draw at all.
func TestLunarHunterSkipsIneligibleWithoutDraw(t *testing.T) {
	w := fullMoonTestWorld(t)

	low := NewPlayer(1, "Lowel", 8004)
	if err := w.AddPlayer(low); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	low.SetLevel(12)
	high := NewPlayer(2, "Godhigh", 8004)
	if err := w.AddPlayer(high); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	high.SetLevel(lvlImmort)

	SetWeatherWorld(w)
	t.Cleanup(func() { SetWeatherWorld(nil) })

	dprng.ResetStream(7)
	t.Cleanup(func() { dprng.ResetStream(1) })
	lunarHunter()

	for _, mob := range w.GetAllMobs() {
		if mob.GetVNum() == 7 {
			t.Fatalf("hunter spawned for ineligible player (level %d)", mob.GetLevel())
		}
	}
	if got, want := dprng.Number(0, 999), dprng.New(7).Number(0, 999); got != want {
		t.Errorf("next draw = %d, want %d (no rolls consumed)", got, want)
	}
}
