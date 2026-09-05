package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// act_informative.go — ported from act.informative.c
// Player-level commands of an informative nature: look, exa, who, score, etc.
// ---------------------------------------------------------------------------

// dirList is the canonical direction order.
var dirList = []string{"north", "east", "south", "west", "up", "down"}

// ---------------------------------------------------------------------------
// doLook — ACMD(do_look) — room, target, direction, or "read"
// ---------------------------------------------------------------------------

func splitArg(arg string) (string, string) {
	arg = strings.TrimSpace(arg)
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

func chCanSee(ch *Player, target interface{}) bool {
	return !ch.IsAffected(affBlind)
}

func chCanSeeObj(ch *Player, obj *ObjectInstance) bool {
	if obj == nil {
		return false
	}
	if obj.HasExtraFlag(0, extraFlagInvisible) {
		// ITEM_INVISIBLE - immortals can see invisible items
		if ch.Level >= lvlImmort {
			return true
		}
		return false
	}
	return chCanSee(ch, nil)
}

func chCanSeeInDark(ch *Player) bool {
	// CAN_SEE_IN_DARK(ch) is exactly AFF_INFRAVISION or PRF_HOLYLIGHT
	// (src/utils.h:451-452). Immortal level alone does not grant dark vision;
	// the first-player God reaches LVL_IMPL through init_char without the
	// advance_level() holy-light assignment (src/db.c:3014-3074).
	return ch.GetHolyLight() || ch.IsAffected(affInfravision)
}

func (w *World) isRoomDark(vnum int) bool {
	// Delegate to the full IS_DARK implementation which checks
	// room light counter, ROOM_DARK flag, and nighttime.
	return w.IsRoomDark(vnum)
}

func (w *World) roomIsDeath(room *parser.Room) bool {
	for _, f := range room.Flags {
		if f == "death" {
			return true
		}
	}
	return false
}

// findCharInRoom finds a character by name in the same room.
// Returns the player and mob — exactly one will be non-nil.
func (w *World) findCharInRoom(ch *Player, roomVNum int, name string) (*Player, *MobInstance) {
	argLower := strings.ToLower(name)
	// Check players first
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		if strings.Contains(strings.ToLower(p.GetName()), argLower) {
			return p, nil
		}
	}
	// Check mobs
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if strings.Contains(strings.ToLower(m.Prototype.ShortDesc), argLower) {
			return nil, m
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Under-ported helpers (from act.informative.c / act.wizard.c / spec_procs2.c)
// ---------------------------------------------------------------------------

// FindTargetRoom resolves a target room string to a VNum (from act.wizard.c:184).
func (w *World) FindTargetRoom(ch *Player, raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return -1
	}
	vnum := 0
	if _, err := fmt.Sscanf(raw, "%d", &vnum); err == nil && vnum > 0 {
		if _, ok := w.rooms[vnum]; ok {
			return vnum
		}
		return -1
	}
	lower := strings.ToLower(raw)
	for vnum, room := range w.rooms {
		if room == nil {
			continue
		}
		if strings.Contains(strings.ToLower(room.Name), lower) {
			return vnum
		}
	}
	return -1
}

// PrintObjectLocation formats where an object is (in room, carried, worn, inside another).
func (w *World) PrintObjectLocation(num int, obj *ObjectInstance, ch *Player, recur bool) string {
	var b strings.Builder
	if num > 0 {
		fmt.Fprintf(&b, "O%3d. %-25s - ", num, obj.Prototype.ShortDesc)
	} else {
		fmt.Fprintf(&b, "%33s", " - ")
	}
	switch {
	case obj.RoomVNum > 0:
		if room, ok := w.rooms[obj.RoomVNum]; ok && room != nil {
			fmt.Fprintf(&b, "[%5d] %s\r\n", obj.RoomVNum, room.Name)
		} else {
			fmt.Fprintf(&b, "[%5d] (unknown room)\r\n", obj.RoomVNum)
		}
	case obj.Location.Kind == ObjInInventory || obj.Location.Kind == ObjEquipped:
		name := "someone"
		switch obj.Location.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[obj.Location.PlayerName]; ok {
				name = p.GetName()
			}
		case OwnerMob:
			if m, ok := w.activeMobs[obj.Location.MobID]; ok {
				name = m.GetName()
			}
		}
		if obj.Location.Kind == ObjEquipped {
			fmt.Fprintf(&b, "worn by %s\r\n", name)
		} else {
			fmt.Fprintf(&b, "carried by %s\r\n", name)
		}
	case obj.Location.Kind == ObjInContainer:
		if container, ok := w.objectInstances[obj.Location.ContainerObjID]; ok {
			fmt.Fprintf(&b, "inside %s\r\n", container.Prototype.ShortDesc)
		} else {
			b.WriteString("in an unknown container\r\n")
		}
	default:
		b.WriteString("in an unknown location\r\n")
	}
	return b.String()
}

// KenderSteal ports kender_steal() and its NPC do_steal(..., subcmd=1)
// follow-up (spec_procs2.c:594-650). It is called after look_at_char has
// rendered, matching the C call order. Player victims return before this
// procedure is entered in C; this boundary therefore accepts only mobs.
func (w *World) KenderSteal(ch *Player, mob *MobInstance) {
	if ch == nil || mob == nil || ch.GetRace() != RaceKender ||
		ch.GetLevel() >= LVL_IMMORT || mob.GetLevel() >= LVL_IMMORT {
		return
	}

	// C walks a linked list while do_steal may unlink the selected object. A
	// pointer snapshot preserves the same next-object traversal without letting
	// slice removal skip the following object.
	items := append([]*ObjectInstance(nil), mob.Inventory...)
	for _, obj := range items {
		if obj == nil || obj.Prototype == nil || !chCanSeeObj(ch, obj) {
			continue
		}
		if !mobCarriesObject(mob, obj) {
			continue
		}
		if dprng.Number(0, 600) >= ch.GetLevel() {
			continue
		}
		// kender_steal computes this roll even though the NPC branch passes
		// through to do_steal, whose own percent roll is authoritative.
		_ = dprng.Number(1, 101) - dexAppSkill(ch.GetDex()).PPocket
		if w.roomHasFlag(ch.GetRoom(), "peaceful") || isShopKeeper(mob) ||
			ch.GetLevel() <= 10 || mob.GetLevel() <= 10 {
			return
		}
		if ch.GetLevel() <= 5 {
			continue
		}
		w.kenderStealItem(ch, mob, obj)
	}
}

func mobCarriesObject(mob *MobInstance, wanted *ObjectInstance) bool {
	for _, item := range mob.Inventory {
		if item == wanted {
			return true
		}
	}
	return false
}

// kenderStealItem is the NPC carrying-object arm of do_steal. The caller has
// already consumed kender_steal's outer selection and pocket rolls.
func (w *World) kenderStealItem(ch *Player, mob *MobInstance, obj *ObjectInstance) {
	percent := dprng.Number(1, 101) - dexAppSkill(ch.GetDex()).PPocket
	if mob.GetLevel() >= LVL_IMMORT || w.roomHasFlag(ch.GetRoom(), "peaceful") || isShopKeeper(mob) {
		percent = 101
	}
	if ch.GetLevel() > LVL_IMMORT && mob.GetLevel() < ch.GetLevel() {
		percent = -1
	}
	percent += obj.GetTotalWeight()
	if mob.GetLevel() > ch.GetLevel() {
		percent += mob.GetLevel() - ch.GetLevel()
	}

	if percent > ch.GetSkill(SkillSteal) {
		Act(w, false, ch, mob, nil, nil, "$N catches you trying to steal something...", "", ToChar)
		Act(w, false, ch, mob, nil, nil, "$n tried to steal something from you!", "", ToVict)
		Act(w, true, ch, mob, nil, nil, "$n tries to steal something from $N.", "", ToNotVict)
		if mob.GetPosition() > combat.PosSleeping && w.combatEngine != nil {
			if err := w.combatEngine.StartCombat(mob, ch); err != nil {
				slog.Error("kender steal retaliation failed", "mob", mob.GetName(), "player", ch.GetName(), "error", err)
			}
		}
		ch.SetWaitState(1)
		return
	}

	if ok, message := canCarryStolenItem(ch, obj); !ok {
		if message != "" {
			ch.SendMessage(message + "\r\n")
		}
		ch.SetWaitState(1)
		return
	}
	if !mob.RemoveFromInventory(obj) {
		ch.SetWaitState(1)
		return
	}
	if err := ch.Inventory.AddItem(obj); err != nil {
		mob.AddToInventory(obj)
		if !mobCarriesObject(mob, obj) {
			slog.Error("kender steal rollback failed", "mob", mob.GetName(), "player", ch.GetName(), "item", obj.GetShortDesc(), "error", err)
		}
		ch.SetWaitState(1)
		return
	}
	obj.Location = LocInventoryPlayer(ch.Name)
	Act(w, true, ch, nil, obj, nil, "Somehow $p makes it's way into your pack.", "", ToChar)
	ImproveSkill(ch, SkillSteal)
	ch.SetWaitState(1)
}

// FindClassBitvector moved to class_tables.go (with ranger/mystic support).
