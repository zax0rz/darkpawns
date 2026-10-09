package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func tattooDepthWorld(t *testing.T) (*World, *Player, *Player, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Tattoo room", Zone: 1}, {VNum: 1002, Name: "Outside", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 9, Keywords: "skull", ShortDesc: "a flaming skull", Level: 10, Position: combat.PosStanding, DefaultPos: combat.PosStanding, HP: parser.DiceRoll{Num: 10, Sides: 5, Plus: 110}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	a, b := NewPlayer(1, "Actor", 1001), NewPlayer(2, "Observer", 1001)
	for _, p := range []*Player{a, b, NewPlayer(3, "Outside", 1002)} {
		p.Stats = CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10}
		p.CopyBaseAttributes()
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	return w, a, b, out
}

// C: act.other.c:906-924; tattoo.c:36-90. State assertions distinguish
// no tattoo (accepted, timer 24) from unsupported tattoos (no timer).
func TestUseTattooCooldownAndUnsupported(t *testing.T) {
	for _, n := range []int{1, 2, 24, -1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			w, a, _, out := tattooDepthWorld(t)
			a.Tattoo = TattooSkull
			a.TatTimer = n
			w.DoUse(a, "TaTtOo ignored")
			suffix := ""
			if n > 1 {
				suffix = "s"
			}
			want := fmt.Sprintf("You can't use your tattoo's magick for %d more hour%s.\r\n", n, suffix)
			if out[a.Name] != want || a.TatTimer != n || len(w.GetMobsInRoom(1001)) != 0 {
				t.Fatalf("cooldown: output=%q timer=%d", out[a.Name], a.TatTimer)
			}
		})
	}
	for _, tattoo := range []int{TattooNone, TattooDragon, TattooTribal, TattooTiger, TattooWorm, TattooSwords, TattooEagle, TattooHeart, TattooStar, TattooSpider, TattooJyhadi, TattooMom, TattooFox, TattooOwl, 99} {
		t.Run(fmt.Sprintf("tattoo%d", tattoo), func(t *testing.T) {
			w, a, _, out := tattooDepthWorld(t)
			a.Tattoo = tattoo
			w.DoUse(a, "tattoo")
			want := "Your tattoo can't be 'use'd.\r\n"
			timer := 0
			if tattoo == TattooNone {
				want = "You don't have a tattoo.\r\n"
				timer = 24
			}
			if out[a.Name] != want || a.TatTimer != timer || len(a.ActiveAffects) != 0 {
				t.Fatalf("unsupported: output=%q timer=%d", out[a.Name], a.TatTimer)
			}
		})
	}
	w, a, _, out := tattooDepthWorld(t)
	a.Tattoo = TattooEye
	w.DoUse(a, "tat")
	if a.TatTimer != 0 || len(a.ActiveAffects) != 0 || out[a.Name] != "You don't seem to have a tat.\r\n" {
		t.Fatalf("abbreviation activates tattoo: %q", out[a.Name])
	}
}

// C: tattoo.c:69-80; magic.c:912-926,1278-1287,1339-1350.
func TestUseTattooSpellStateAndPosition(t *testing.T) {
	for _, tc := range []struct {
		tattoo, spell, duration, count int
		msg                            string
	}{
		{TattooEye, spells.SpellGreatPercept, 10, 2, "Your eyes glow briefly.\r\n"},
		{TattooShip, spells.SpellChangeDensity, 20, 1, "Your molecular density shifts.\r\n"},
		{TattooAngel, spells.SpellBless, 6, 2, "You feel righteous.\r\n"},
	} {
		for _, pos := range []int{combat.PosStanding, combat.PosResting, combat.PosFighting, combat.PosSitting} {
			t.Run(fmt.Sprintf("%d/%d", tc.tattoo, pos), func(t *testing.T) {
				w, a, _, out := tattooDepthWorld(t)
				a.Tattoo = tc.tattoo
				a.Level = 100
				a.Position = pos
				mana, move := a.Mana, a.Move
				w.DoUse(a, "tattoo")
				if a.TatTimer != 24 || a.Mana != mana || a.Move != move {
					t.Fatalf("timer/resources %d/%d/%d", a.TatTimer, a.Mana, a.Move)
				}
				if pos == combat.PosSitting {
					if out[a.Name] != "You cannot do this sitting!\r\n" || len(a.ActiveAffects) != 0 {
						t.Fatalf("sitting: %q %+v", out[a.Name], a.ActiveAffects)
					}
					return
				}
				if out[a.Name] != tc.msg || len(a.ActiveAffects) != tc.count {
					t.Fatalf("spell: %q affects=%+v", out[a.Name], a.ActiveAffects)
				}
				for _, af := range a.ActiveAffects {
					if af.SpellID != tc.spell || af.Duration != tc.duration {
						t.Fatalf("affect %+v", af)
					}
				}
				if tc.tattoo == TattooEye && (!a.IsAffected(affDetectInvisible) || !a.IsAffected(affSenseLife)) {
					t.Fatal("missing percept flags")
				}
				if tc.tattoo == TattooShip && !a.IsAffected(affWaterWalk) {
					t.Fatal("missing waterwalk")
				}
				if tc.tattoo == TattooAngel && (a.GetHitroll() != 2 || a.GetSavingThrow(4) != -2) {
					t.Fatalf("bless stats hitroll=%d saves=%v", a.Hitroll, a.SavingThrows)
				}
			})
		}
	}
}

// C call_magic gates NOMAGIC before sitting, exempts immortals, and uses
// distinct psionic/mystic bytes (spell_parser.c:419-439).
func TestUseTattooNoMagic(t *testing.T) {
	for _, class := range []int{ClassMageUser, ClassPsionic, ClassMystic} {
		for _, level := range []int{1, combat.LVL_IMMORT} {
			t.Run(fmt.Sprintf("%d/%d", class, level), func(t *testing.T) {
				w, a, b, out := tattooDepthWorld(t)
				w.SetRoomFlagBit(1001, 7)
				a.Tattoo = TattooEye
				a.Class = class
				a.Level = level
				w.DoUse(a, "tattoo")
				if a.TatTimer != 24 {
					t.Fatal("blocked spell must still set cooldown")
				}
				if level == combat.LVL_IMMORT {
					if len(a.ActiveAffects) != 2 {
						t.Fatal("immortal blocked")
					}
					return
				}
				actor, room := "Your magic fizzles out and dies.\r\n", "Actor's magic fizzles out and dies.\r\n"
				if class == ClassPsionic || class == ClassMystic {
					actor = "Your will fades, disturbed by an unseen force.\r\n"
					room = "Actor's will fades, disturbed by an unseen force.\r\n"
				}
				if out[a.Name] != actor || out[b.Name] != room || len(a.ActiveAffects) != 0 {
					t.Fatalf("no magic: %v", out)
				}
			})
		}
	}
}

// C creates a fresh skull, adds a quiet follower, then applies charm and
// two act audiences (tattoo.c:49-67; utils.c:463-475).
func TestUseTattooSkullAudienceAndIdentity(t *testing.T) {
	w, a, b, out := tattooDepthWorld(t)
	a.Tattoo = TattooSkull
	w.DoUse(a, "tattoo")
	actor := "Your tattoo glows brightly for a second, and a flaming skull appears!\r\n"
	room := "Actor's tattoo glows brightly for a second, and a flaming skull appears!\r\n"
	if out[a.Name] != actor || out[b.Name] != room || out["Outside"] != "" {
		t.Fatalf("audiences: %v", out)
	}
	mobs := w.GetMobsInRoom(1001)
	if len(mobs) != 1 || mobs[0].GetFollowing() != a.Name || !mobs[0].IsAffected(affCharm) || len(mobs[0].Inventory) != 0 {
		t.Fatalf("skull state: %+v", mobs)
	}
	old := mobs[0]
	af, ok := old.CustomData[fmt.Sprintf("affect_%d", spells.SpellCharm)].(*engine.Affect)
	if !ok || af.Duration != 20 || af.SpellID != spells.SpellCharm || af.Location != 0 || af.Magnitude != 0 {
		t.Fatalf("charm: %+v", af)
	}
	w.DoUse(a, "tattoo")
	if len(w.GetMobsInRoom(1001)) != 1 {
		t.Fatal("cooldown spawned another skull")
	}
	// A second actor's same-named skull must follow its own summoner.
	b.Tattoo = TattooSkull
	w.DoUse(b, "tattoo")
	count := 0
	for _, mob := range w.GetMobsInRoom(1001) {
		if mob.GetFollowing() == b.Name {
			count++
		}
	}
	if count != 1 || old.GetFollowing() != a.Name {
		t.Fatal("same-name skull corrupted original leader")
	}
}

func TestUseTattooHeldPrecedenceAndExpiry(t *testing.T) {
	w, a, _, out := tattooDepthWorld(t)
	a.Tattoo = TattooEye
	obj := NewObjectInstance(&parser.Obj{VNum: 999, Keywords: "token tattoo", ShortDesc: "a token", TypeFlag: ITEM_OTHER, WearFlags: [4]int{(1 << 0) | (1 << 14)}}, -1)
	if err := w.MoveObject(obj, LocEquippedPlayer(a.Name, SlotHold)); err != nil {
		t.Fatal(err)
	}
	w.DoUse(a, "tattoo")
	if a.TatTimer != 0 || len(a.ActiveAffects) != 0 || !strings.HasPrefix(out[a.Name], "You can't seem to figure out how to use it.") {
		t.Fatalf("held precedence: %q", out[a.Name])
	}
	if err := w.MoveObject(obj, LocInventoryPlayer(a.Name)); err != nil {
		t.Fatal(err)
	}
	w.DoUse(a, "tattoo")
	if a.TatTimer != 24 {
		t.Fatal("inventory shadowed tattoo")
	}
	w.PointUpdate()
	if a.TatTimer != 23 {
		t.Fatal("timer 24 did not decrement")
	}
	a.TatTimer = 1
	w.PointUpdate()
	if a.TatTimer != 0 {
		t.Fatal("timer did not expire")
	}
	w.PointUpdate()
	if a.TatTimer != 0 {
		t.Fatal("timer underflow")
	}
	w.DoUse(a, "tattoo")
	if a.TatTimer != 24 {
		t.Fatal("expired tattoo cannot be reused")
	}
	a.TatTimer = 0
	worn := NewObjectInstance(&parser.Obj{VNum: 998, Keywords: "tattoo", ShortDesc: "a tattoo blade", TypeFlag: ITEM_WEAPON, WearFlags: [4]int{(1 << 0) | (1 << 13)}}, -1)
	if err := w.MoveObject(worn, LocEquippedPlayer(a.Name, SlotWield)); err != nil {
		t.Fatal(err)
	}
	w.DoUse(a, "tattoo")
	if a.TatTimer != 24 {
		t.Fatal("other worn slot shadowed innate tattoo")
	}
}

// TestUseTattooDepth is the manifest-owned state matrix.
func TestUseTattooDepth(t *testing.T) {
	t.Run("cooldown", TestUseTattooCooldownAndUnsupported)
	t.Run("spells", TestUseTattooSpellStateAndPosition)
	t.Run("nomagic", TestUseTattooNoMagic)
	t.Run("gate-order", TestUseTattooNoMagicPrecedesSitting)
	t.Run("skull", TestUseTattooSkullAudienceAndIdentity)
	t.Run("precedence", TestUseTattooHeldPrecedenceAndExpiry)
	t.Run("keyword-boundaries", TestUseTattooHeldKeywordBoundaries)
	t.Run("draws", TestUseTattooDrawParity)
	t.Run("visibility", TestUseTattooSkullVisibility)
	t.Run("eye-audience", TestUseTattooEyeAudienceGates)
	t.Run("room-visibility-law", TestUseTattooRoomVisibilityLaw)
	t.Run("no-magic-names", TestUseTattooNoMagicAudienceNames)
}

// C read_mobile rolls the skull's ten HP dice; benign self spells and
// cooldown/refusal paths draw nothing (db.c read_mobile; tattoo.c:36-90).
func TestUseTattooDrawParity(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3, 5, 8} {
		for _, tattoo := range []int{TattooNone, TattooDragon, TattooEye, TattooShip, TattooAngel, TattooSkull} {
			t.Run(fmt.Sprintf("%d/%d", seed, tattoo), func(t *testing.T) {
				w, a, _, _ := tattooDepthWorld(t)
				a.Tattoo = tattoo
				dprng.ResetStream(seed)
				control := dprng.New(seed)
				hp := 110
				if tattoo == TattooSkull {
					for i := 0; i < 10; i++ {
						hp += control.Number(1, 5)
					}
				}
				w.DoUse(a, "tattoo")
				if tattoo == TattooSkull && w.GetMobsInRoom(1001)[0].GetHP() != hp {
					t.Fatalf("skull HP differs from ten C dice")
				}
				if got, want := dprng.Next(), control.Next(); got != want {
					t.Fatalf("draw parity: got %d want %d", got, want)
				}
			})
		}
	}
}

func TestUseTattooSkullVisibility(t *testing.T) {
	for _, sleep := range []bool{false, true} {
		t.Run(fmt.Sprint(sleep), func(t *testing.T) {
			w, a, b, out := tattooDepthWorld(t)
			a.Tattoo = TattooSkull
			if sleep {
				b.Position = combat.PosSleeping
			} else {
				a.SetAffect(affInvisible, true)
			}
			w.DoUse(a, "tattoo")
			if out[b.Name] != "" {
				t.Fatalf("hidden/asleep observer saw %q", out[b.Name])
			}
			if !strings.HasPrefix(out[a.Name], "Your tattoo glows") {
				t.Fatal("actor missed activation")
			}
		})
	}
}

func TestUseTattooNoMagicPrecedesSitting(t *testing.T) {
	w, a, _, out := tattooDepthWorld(t)
	w.SetRoomFlagBit(1001, 7)
	a.Tattoo = TattooEye
	a.Position = combat.PosSitting
	w.DoUse(a, "tattoo")
	if out[a.Name] != "Your magic fizzles out and dies.\r\n" || a.TatTimer != 24 {
		t.Fatalf("room-before-position gate: %v", out)
	}
}

func TestUseTattooHeldKeywordBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		shadow bool
	}{
		{"tattoo", true},
		{"TaTtOo mark", true},
		{"mark tattoo", true},
		{"tattoo-mark", true},
		{"mark-tattoo", true},
		{"tattoo1", true},
		{"tattoox", false},
		{"tat", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, a, _, _ := tattooDepthWorld(t)
			a.Tattoo = TattooEye
			obj := NewObjectInstance(&parser.Obj{VNum: 997, Keywords: tc.name, ShortDesc: "a token", TypeFlag: ITEM_OTHER, WearFlags: [4]int{(1 << 0) | (1 << 14)}}, -1)
			if err := w.MoveObject(obj, LocEquippedPlayer(a.Name, SlotHold)); err != nil {
				t.Fatal(err)
			}
			w.DoUse(a, "tattoo")
			if got := a.TatTimer == 0; got != tc.shadow {
				t.Fatalf("held shadow=%v want %v", got, tc.shadow)
			}
		})
	}
}

// magic.c:1418-1421 uses act(... TRUE ... TO_ROOM) for perception.
func TestUseTattooEyeAudienceGates(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		invisible, sleep, detect bool
		want                     string
	}{
		{"visible", false, false, false, "Actor's eyes glow briefly.\r\n"},
		{"invisible", true, false, false, ""},
		{"sleeping", false, true, false, ""},
		{"detect-invisible", true, false, true, "Actor's eyes glow briefly.\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, a, b, out := tattooDepthWorld(t)
			a.Tattoo = TattooEye
			if tc.invisible {
				a.SetAffect(affInvisible, true)
			}
			if tc.sleep {
				b.Position = combat.PosSleeping
			}
			if tc.detect {
				b.SetAffect(affDetectInvisible, true)
			}
			w.DoUse(a, "tattoo")
			if out[b.Name] != tc.want || out[a.Name] != "Your eyes glow briefly.\r\n" {
				t.Fatalf("eye audience: %v", out)
			}
		})
	}
}

// utils.h:515-530 has LIGHT_OK, INVIS_OK and holylight, not AFF_HIDE
// or a blanket immortal bypass. Both room-message branches share this gate.
func TestUseTattooRoomVisibilityLaw(t *testing.T) {
	for _, tattoo := range []int{TattooEye, TattooSkull} {
		for _, tc := range []struct {
			name                                                         string
			hide, invisible, blind, dark, infra, holy, writing, wizinvis bool
			level                                                        int
			visible                                                      bool
		}{
			{name: "hidden", hide: true, visible: true},
			{name: "blind", blind: true},
			{name: "dark", dark: true},
			{name: "infravision", dark: true, infra: true, visible: true},
			{name: "holy-blind", blind: true, holy: true, visible: true},
			{name: "immortal-invisible", invisible: true, level: combat.LVL_IMMORT},
			{name: "wizinvis-holy", wizinvis: true, holy: true},
			{name: "writing", writing: true},
		} {
			t.Run(fmt.Sprintf("%d/%s", tattoo, tc.name), func(t *testing.T) {
				w, a, b, out := tattooDepthWorld(t)
				a.Tattoo = tattoo
				if tc.hide {
					a.SetAffect(affHide, true)
				}
				if tc.invisible {
					a.SetAffect(affInvisible, true)
				}
				if tc.blind {
					b.SetAffect(affBlind, true)
				}
				if tc.dark {
					w.SetRoomFlagBit(1001, 0)
				}
				if tc.infra {
					b.SetAffect(affInfravision, true)
				}
				if tc.holy {
					b.SetHolyLight(true)
				}
				if tc.writing {
					b.SetPlrFlag(PlrWriting, true)
				}
				if tc.wizinvis {
					a.InvisLevel = 100
				}
				if tc.level != 0 {
					b.Level = tc.level
				}
				w.DoUse(a, "tattoo")
				if got := out[b.Name] != ""; got != tc.visible {
					t.Fatalf("room visibility got %q want visible=%v", out[b.Name], tc.visible)
				}
			})
		}
	}
}

func TestUseTattooNoMagicAudienceNames(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		invisible, dark, hide, writing bool
		want                           string
	}{
		{"invisible", true, false, false, false, "Someone's magic fizzles out and dies.\r\n"},
		{"dark", false, true, false, false, "Someone's magic fizzles out and dies.\r\n"},
		{"hidden", false, false, true, false, "Actor's magic fizzles out and dies.\r\n"},
		{"writing", false, false, false, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, a, b, out := tattooDepthWorld(t)
			a.Tattoo = TattooEye
			w.SetRoomFlagBit(1001, 7)
			if tc.invisible {
				a.SetAffect(affInvisible, true)
			}
			if tc.dark {
				w.SetRoomFlagBit(1001, 0)
			}
			if tc.hide {
				a.SetAffect(affHide, true)
			}
			if tc.writing {
				b.SetPlrFlag(PlrWriting, true)
			}
			w.DoUse(a, "tattoo")
			if out[b.Name] != tc.want {
				t.Fatalf("no-magic audience %q want %q", out[b.Name], tc.want)
			}
		})
	}
}
