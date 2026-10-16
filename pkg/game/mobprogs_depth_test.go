package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newMobProgDispatchWorld(t *testing.T, targetVNum int, targetName string) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Source", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
		{VNum: 1002, Name: "Target", Zone: 1},
		{VNum: 8117, Name: "Cell Exit", Zone: 1},
		{VNum: 13154, Name: "Tunnel Exit", Zone: 1},
		{VNum: 13193, Name: "Gremlin Exit", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	player := NewPlayer(1, "Briber", 1001)
	player.SetLevel(10)
	player.SetGold(1000)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer player: %v", err)
	}
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	registerMobProgProto(t, w, targetVNum, targetName)
	mob, err := w.SpawnMob(targetVNum, 1001)
	if err != nil {
		t.Fatalf("SpawnMob %d: %v", targetVNum, err)
	}
	mob.SetPosition(combat.PosStanding)
	mob.SetGold(500)

	output := map[string]string{}
	w.MessageSink = func(name string, message []byte) { output[name] += string(message) }
	return w, player, observer, mob, output
}

func registerMobProgProto(t *testing.T, w *World, vnum int, keyword string) {
	t.Helper()
	w.mu.Lock()
	w.mobs[vnum] = &parser.Mob{
		VNum:      vnum,
		Keywords:  keyword,
		ShortDesc: "the " + keyword,
		LongDesc:  "The " + keyword + " is here.",
		Level:     10,
		HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		Race:      1,
		Sex:       1,
	}
	w.mu.Unlock()
}

func TestMpBribeDispatchRendersCBranches(t *testing.T) {
	tests := []struct {
		name           string
		vnum           int
		amount         int
		keyword        string
		mobFollows     bool
		wantPlayer     []string
		wantObserver   []string
		wantPlayerGold int
		wantMobGold    int
		wantRoom       int
		wantSleeping   bool
		seedCityguard  bool
		spec           string
	}{
		{
			name: "dog eats coins", vnum: 8063, amount: 7, keyword: "warg",
			wantPlayer:     []string{"Okay.", "sniffs the coins and proceeds to eat them"},
			wantObserver:   []string{"gives a tiny pile of gold coins to the warg.", "sniffs the coins and proceeds to eat them"},
			wantPlayerGold: 993, wantMobGold: 493, wantRoom: 1001,
		},
		{
			name: "janitor accepts", vnum: 8061, amount: 7, keyword: "janitor",
			wantPlayer:     []string{"Okay.", "tips his hat and smiles a thank you"},
			wantObserver:   []string{"gives a tiny pile of gold coins to the janitor.", "tips his hat and smiles a thank you"},
			wantPlayerGold: 993, wantMobGold: 500, wantRoom: 1001,
		},
		{
			name: "mercenary hires", vnum: 3063, amount: 100, keyword: "mercenary",
			wantPlayer:     []string{"The mercenary counts the coins then secrets them away.", "The mercenary swears his allegiance to you.", "starts following you"},
			wantObserver:   []string{"gives a small pile of gold coins to the mercenary.", "hires the mercenary", "starts to follow Briber"},
			wantPlayerGold: 900, wantMobGold: 400, wantRoom: 1001,
		},
		{
			name: "mercenary declines", vnum: 3063, amount: 50, keyword: "mercenary", mobFollows: true,
			wantPlayer:     []string{"Okay.", "laughs somewhat rudely"},
			wantObserver:   []string{"gives a little pile of gold coins to the mercenary.", "laughs somewhat rudely"},
			wantPlayerGold: 950, wantMobGold: 500, wantRoom: 1001,
		},
		{
			name: "cell bribe too small", vnum: 8088, amount: 50, keyword: "jailer",
			wantPlayer:     []string{"not quite enough cash", "grins evilly"},
			wantObserver:   []string{"against the law you\r\nknow...", "not quite enough cash", "grins evilly"},
			wantPlayerGold: 1000, wantMobGold: 450, wantRoom: 1001,
		},
		{
			name: "cell bribe ejects", vnum: 8088, amount: 100, keyword: "jailer",
			wantPlayer:     []string{"Okay.", "Thank you very much, monsieur.", "Now get outta here!", "The jailer throws you out of the cell!"},
			wantObserver:   []string{"gives a small pile of gold coins to the jailer.", "Thank you very much, monsieur.", "Now get outta here!", "The jailer throws Briber out of the cell!"},
			wantPlayerGold: 900, wantMobGold: 400, wantRoom: 8117,
		},
		{
			name: "cityguard refuses", vnum: 2747, amount: 50, keyword: "guard",
			wantPlayer:     []string{"Okay.", "trying to bribe me?"},
			wantObserver:   []string{"gives a little pile of gold coins to the guard.", "trying to bribe me?"},
			wantPlayerGold: 950, wantMobGold: 500, wantRoom: 1001,
		},
		{
			name: "cityguard sleeps", vnum: 2747, amount: 500, keyword: "guard", seedCityguard: true,
			wantPlayer:     []string{"Okay.", "I am off duty now...", "lays down and falls asleep on the job!"},
			wantObserver:   []string{"gives a pile of gold coins to the guard.", "I am off duty now...", "lays down and falls asleep on the job!"},
			wantPlayerGold: 500, wantMobGold: 0, wantRoom: 1001, wantSleeping: true,
		},
		{
			name: "citizen accepts", vnum: 2749, amount: 7, keyword: "citizen",
			wantPlayer:     []string{"Okay.", "bows before you"},
			wantObserver:   []string{"gives a tiny pile of gold coins to the citizen.", "Thanks for the investment", "bows before Briber"},
			wantPlayerGold: 993, wantMobGold: 500, wantRoom: 1001,
		},
		{
			name: "prostitute entertains", vnum: 8023, amount: 5, keyword: "prostitute", spec: "prostitute",
			wantPlayer:     []string{"Okay.", "pulls you into the shadows", "A few coins lighter"},
			wantObserver:   []string{"to the prostitute.", "pulls Briber into the shadows"},
			wantPlayerGold: 995, wantMobGold: 500, wantRoom: 1001,
		},
		{
			name: "prostitute declines", vnum: 8023, amount: 4, keyword: "prostitute", spec: "prostitute",
			wantPlayer:     []string{"Okay.", "ain't THAT cheap"},
			wantObserver:   []string{"to the prostitute.", "ain't THAT cheap"},
			wantPlayerGold: 996, wantMobGold: 500, wantRoom: 1001,
		},
		{
			name: "gremlin leads through tunnel", vnum: 13108, amount: 1000, keyword: "gremlin",
			wantPlayer:     []string{"Okay.", "You follow the gremlin through a series of tunnels"},
			wantObserver:   []string{"gives a pile of gold coins to the gremlin.", "leads Briber through a hidden door"},
			wantPlayerGold: 0, wantMobGold: 500, wantRoom: 13154,
		},
		{
			name: "gremlin throws out miser", vnum: 13108, amount: 999, keyword: "gremlin",
			wantPlayer:     []string{"Okay.", "kicks you out on your ass!\r\n\r\n"},
			wantObserver:   []string{"gives a pile of gold coins to the gremlin.", "Cheap bastard", "gets kicked out on his ass"},
			wantPlayerGold: 1, wantMobGold: 500, wantRoom: 13193,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, player, observer, mob, output := newMobProgDispatchWorld(t, tt.vnum, tt.keyword)
			if tt.mobFollows {
				mob.SetFollowing("someone")
			}
			if tt.spec != "" {
				old, had := MobSpecAssign[tt.vnum]
				MobSpecAssign[tt.vnum] = tt.spec
				t.Cleanup(func() {
					if had {
						MobSpecAssign[tt.vnum] = old
					} else {
						delete(MobSpecAssign, tt.vnum)
					}
				})
			}
			if tt.seedCityguard {
				var seed uint32
				for seed = 1; seed < 8192; seed++ {
					dprng.ResetStream(seed)
					if dprng.Number(0, 2) != 0 {
						break
					}
				}
				if seed == 8192 {
					t.Fatal("no seed found that accepts the cityguard bribe")
				}
				dprng.ResetStream(seed)
				t.Cleanup(func() { dprng.ResetStream(1) })
			}

			w.DoGive(player, fmt.Sprintf("%d coins %s", tt.amount, tt.keyword))

			for _, want := range tt.wantPlayer {
				if got := output[player.Name]; !strings.Contains(got, want) {
					t.Errorf("player output missing %q: %q", want, got)
				}
			}
			for _, want := range tt.wantObserver {
				if got := output[observer.Name]; !strings.Contains(got, want) {
					t.Errorf("observer output missing %q: %q", want, got)
				}
			}
			if got := player.GetGold(); got != tt.wantPlayerGold {
				t.Errorf("player gold = %d, want %d", got, tt.wantPlayerGold)
			}
			if got := mob.GetGold(); got != tt.wantMobGold {
				t.Errorf("mob gold = %d, want %d", got, tt.wantMobGold)
			}
			if got := player.GetRoom(); got != tt.wantRoom {
				t.Errorf("player room = %d, want %d", got, tt.wantRoom)
			}
			if got := mob.GetPosition() <= PosSleeping; got != tt.wantSleeping {
				t.Errorf("mob sleeping = %v, want %v", got, tt.wantSleeping)
			}
		})
	}
}

func TestMobEntryProgRunsThroughMobileMovement(t *testing.T) {
	tests := []struct {
		name             string
		companionVNum    int
		companionPos     int
		want             []string
		wantCompanionPos int
	}{
		{
			name: "captain wakes guard", companionVNum: 2747, companionPos: PosSleeping,
			want:             []string{"barks 'On your feet, slacker!'", "wakes up and quickly snaps to attention!", "Report to my office at 0500 tomorrow morning."},
			wantCompanionPos: PosStanding,
		},
		{
			name: "captain salutes standing guard", companionVNum: 2747, companionPos: PosStanding,
			want:             []string{"snaps to attention and salutes!", "At ease, soldier."},
			wantCompanionPos: PosStanding,
		},
		{
			name: "captain greets citizen", companionVNum: 2749, companionPos: PosStanding,
			want:             []string{"frowns at the companion.", "Hail unto the True One, Captain."},
			wantCompanionPos: PosStanding,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, observer, _, output := newMobProgDispatchWorld(t, tt.companionVNum, "companion")
			// Move the observer and companion into the destination before the captain arrives.
			observer.SetRoom(1002)
			companion, err := w.SpawnMob(tt.companionVNum, 1002)
			if err != nil {
				t.Fatalf("SpawnMob companion: %v", err)
			}
			companion.SetPosition(tt.companionPos)
			registerMobProgProto(t, w, 8059, "captain")
			captain, err := w.SpawnMob(8059, 1001)
			if err != nil {
				t.Fatalf("SpawnMob captain: %v", err)
			}
			captain.SetPosition(PosStanding)

			if moved := w.doSimpleMoveMob(captain, 0, false); !moved {
				t.Fatal("captain should enter the destination room")
			}
			for _, want := range tt.want {
				if got := output[observer.Name]; !strings.Contains(got, want) {
					t.Errorf("observer output missing %q: %q", want, got)
				}
			}
			if got := companion.GetPosition(); got != tt.wantCompanionPos {
				t.Errorf("companion position = %d, want %d", got, tt.wantCompanionPos)
			}
		})
	}
}

func TestMobGreetOutputRunsThroughMobileMovement(t *testing.T) {
	tests := []struct {
		name         string
		moverVNum    int
		hostVNum     int
		hostKeywords string
		spec         string
		want         string
	}{
		{name: "shopkeeper warns arriving dog", moverVNum: 8063, hostVNum: 8003, hostKeywords: "shopkeeper", want: "mutters something about filthy animals in his shop"},
		{name: "innkeeper greets arrival", moverVNum: 9402, hostVNum: 19406, hostKeywords: "innkeeper", want: "Have a seat there! Stay a while"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, observer, host, output := newMobProgDispatchWorld(t, tt.hostVNum, tt.hostKeywords)
			observer.SetRoom(1002)
			if tt.spec != "" {
				old, had := MobSpecAssign[tt.hostVNum]
				MobSpecAssign[tt.hostVNum] = tt.spec
				t.Cleanup(func() {
					if had {
						MobSpecAssign[tt.hostVNum] = old
					} else {
						delete(MobSpecAssign, tt.hostVNum)
					}
				})
			}
			registerMobProgProto(t, w, tt.moverVNum, "arriving mobile")
			mover, err := w.SpawnMob(tt.moverVNum, 1001)
			if err != nil {
				t.Fatalf("SpawnMob mover: %v", err)
			}
			mover.SetPosition(PosStanding)
			if tt.moverVNum == 8063 {
				// Production worlds have the deferred combat entry installed;
				// this focused seam accepts the C hit(tch, who) boundary.
				w.combatEngine = &cityguardTestCombatEngine{}
			}
			// Move the host into the destination before the arrival.
			if err := w.MobTransfer(host, 1002); err != nil {
				t.Fatalf("move host: %v", err)
			}

			if moved := w.doSimpleMoveMob(mover, 0, false); !moved {
				t.Fatal("mobile should enter the destination room")
			}
			if got := output[observer.Name]; !strings.Contains(got, tt.want) {
				t.Errorf("observer output missing %q: %q", tt.want, got)
			}
		})
	}
}

// perform_give_gold's checks run before mp_bribe (src/act.item.c:738-754): a
// zero offer is refused before the cityguard sees it, and a god's purse is
// neither checked nor charged.
func TestMobBribeGoesThroughPerformGiveGoldChecks(t *testing.T) {
	t.Run("zero coins never reach the bribe", func(t *testing.T) {
		w, player, observer, mob, output := newMobProgDispatchWorld(t, 2747, "guard")
		w.DoGive(player, "0 coins guard")
		if got, want := output[player.Name], "Heh heh heh ... we are jolly funny today, eh?\r\n"; got != want {
			t.Fatalf("player output = %q, want %q", got, want)
		}
		if got := output[observer.Name]; got != "" {
			t.Fatalf("observer output = %q, want nothing", got)
		}
		if mob.GetFighting() != "" || player.GetGold() != 1000 {
			t.Fatalf("zero bribe changed state: fighting %q, gold %d", mob.GetFighting(), player.GetGold())
		}
	})
	t.Run("a god bribes from an empty purse", func(t *testing.T) {
		w, player, _, _, output := newMobProgDispatchWorld(t, 8061, "janitor")
		player.SetLevel(lvlGod)
		player.SetGold(0)
		w.DoGive(player, "50 coins janitor")
		if got := output[player.Name]; !strings.Contains(got, "Okay.") || !strings.Contains(got, "tips his hat") {
			t.Fatalf("god's bribe output = %q, want Okay. and the janitor's thanks", got)
		}
		if got := player.GetGold(); got != 0 {
			t.Fatalf("god's gold = %d, want 0 (C never charges a god)", got)
		}
	})
}
