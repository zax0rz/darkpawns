package game

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func GainCondition(p *Player, condition int, value int) {
	p.mu.RLock()
	cond := p.Conditions[condition]
	p.mu.RUnlock()

	if cond == -1 { // Immortal / no change
		return
	}

	intoxicated := false
	p.mu.RLock()
	if p.Conditions[CondDrunk] > 0 {
		intoxicated = true
	}
	p.mu.RUnlock()

	newVal := cond + value
	if newVal < 0 {
		newVal = 0
	}
	if newVal > 48 {
		newVal = 48
	}
	p.SetCondition(condition, newVal)
	newCond := p.GetCondition(condition)

	// Messages only at threshold 0 or 1
	if newCond > 1 {
		return
	}

	// Also skip messages if player is writing (PLR_WRITING flag)
	// PLR_WRITING = bit 4 — check p.Flags
	p.mu.RLock()
	writing := p.Flags&(1<<4) != 0
	p.mu.RUnlock()
	if writing {
		return
	}

	var msg string
	if newCond > 0 {
		switch condition {
		case CondFull:
			msg = "Your stomach growls with hunger.\r\n"
		case CondThirst:
			msg = "You feel a bit parched.\r\n"
		case CondDrunk:
			if intoxicated {
				msg = "Your head starts to clear.\r\n"
			}
		}
	} else {
		switch condition {
		case CondFull:
			msg = "You are hungry.\r\n"
		case CondThirst:
			msg = "You are thirsty.\r\n"
		case CondDrunk:
			if intoxicated {
				msg = "You are now sober.\r\n"
			}
		}
	}

	if msg != "" {
		p.SendMessage(msg)
	}
}

// ---------------------------------------------------------------------------
// PointUpdate — from limits.c point_update() (lines 460-686)
// ---------------------------------------------------------------------------
// Main tick function called periodically. Iterates all players and NPCs,
// applies condition decay, regenerates HMV, processes poison/cutthroat
// damage, memory clearing, idle checks, and object decay.
func (w *World) PointUpdate() {
	// Regeneration prints nothing, so structured clients learn of the new
	// vitals only from this notice, sent once the whole tick has applied.
	defer func() {
		if observer := w.outOfBand(); observer != nil {
			observer.PointUpdated()
		}
	}()

	// point_update() walks every body in C's global character_list order.
	for _, character := range w.cOrderedCharacters() {
		if p := character.player; p != nil {
			p.mu.RLock()
			pos := p.Position
			inactive := p.Flags&(1<<PrfInactive) != 0
			p.mu.RUnlock()

			// Condition decay — skip if inactive (PRF_INACTIVE)
			if !inactive {
				GainCondition(p, CondFull, -1)
				GainCondition(p, CondDrunk, -1)
				GainCondition(p, CondThirst, -1)
			}

			// Tattoo timer
			p.mu.Lock()
			if p.TatTimer > 0 {
				p.TatTimer--
			}
			p.mu.Unlock()

			// src/limits.c:486-487 only decrements; the jail room pulse owns release.
			p.mu.Lock()
			if p.JailTimer > 0 {
				p.JailTimer--
			}
			p.mu.Unlock()

			// Dream processing for sleeping characters
			// Source: limits.c:476
			if pos == PosSleeping {
				adapter := &PlayerDreamAdapter{p: p, w: w}
				Dream(adapter)
			}

			if pos >= PosStunned {
				p.mu.RLock()
				hp := p.Health
				maxHP := p.MaxHealth
				mana := p.Mana
				maxMana := p.MaxMana
				move := p.Move
				maxMove := p.MaxMove
				poisoned := p.isAffectedLocked(AffPoison)
				cutthroat := p.isAffectedLocked(AffCutthroat)
				p.mu.RUnlock()

				// C: limits.c:495-501 gates all three player resource gains on
				// PRF_INACTIVE; poison/cutthroat damage below remains outside it.
				if !inactive {
					// HP regen
					if hp < maxHP {
						gain := w.HitGain(p)
						hp += gain
						if hp > maxHP {
							hp = maxHP
						}
						p.mu.Lock()
						p.Health = hp
						p.mu.Unlock()
					}

					// Mana regen
					if mana < maxMana {
						gain := w.ManaGain(p)
						mana += gain
						if mana > maxMana {
							mana = maxMana
						}
						p.mu.Lock()
						p.Mana = mana
						p.mu.Unlock()
					}

					// Move regen — always (even at max, original limits.c:501)
					mvGain := w.MoveGain(p)
					move += mvGain
					if move > maxMove {
						move = maxMove
					}
					p.mu.Lock()
					p.Move = move
					p.mu.Unlock()
				}

				// Poison damage — limits.c:503-504
				if poisoned {
					w.selfDamage(p, 10, 33)
					if p.CombatRetired() {
						continue
					}
				}

				// Cutthroat damage — limits.c:505-506
				if cutthroat {
					w.selfDamage(p, 13, 143)
				}

				// Poison/cutthroat can drive HP into the wounded band (or past the
				// POS_DEAD threshold, HP <= -11). Re-derive position from the new HP
				// and only die at POS_DEAD — C: damage(ch, ch, dam, ...) runs
				// update_pos then die() only once GET_HIT <= -11 (DP-1021).
				if p.CombatRetired() {
					continue
				}
				updatePosFromHP(p, p.GetHP())
			} else if pos == PosIncap {
				// Incapacitated: 1 damage per tick — limits.c:511
				w.selfDamage(p, 1, combat.TYPE_SUFFERING)
				if p.CombatRetired() {
					continue
				}
				updatePosFromHP(p, p.GetHP())
			} else if pos == PosMortally {
				// Mortally wounded: 2 damage per tick — limits.c:513
				w.selfDamage(p, 2, combat.TYPE_SUFFERING)
				if p.CombatRetired() {
					continue
				}
				updatePosFromHP(p, p.GetHP())
			}

			// Memory clearing for NPCs — limits.c:516-518
			// (handled in NPC section below)

			// Idle check for players — limits.c:521-524
			w.CheckIdling(p)
			continue
		}
		m := character.mob
		if m == nil {
			continue
		}

		// --- NPCs ---
		pos := m.GetPosition()

		if pos >= PosStunned {
			// HP regen — limits.c:498-501
			if m.GetHP() < m.GetMaxHP() {
				gain := MobHitGain(m)
				m.SetHealth(m.GetHP() + gain)
				if m.GetHP() > m.GetMaxHP() {
					m.SetHealth(m.GetMaxHP())
				}
			}
			// C point_update restores NPC movement by level every tick, capped
			// at the fixed max_move=50 initialized in read_mobile.
			m.GainMove(MoveGainNPC(m))
			// Poison damage — limits.c:503-504 (applies to ALL chars including NPCs)
			if m.HasAffect(AffPoison) {
				w.selfDamage(m, 10, 33)
				if m.CombatRetired() {
					continue
				}
			}
			// Cutthroat damage — limits.c:505-506
			if m.HasAffect(AffCutthroat) {
				w.selfDamage(m, 13, 143)
			}
			// Re-derive position from the new HP; only die at POS_DEAD
			// (HP <= -11) so poison/cutthroat progress through the wounded
			// band instead of killing instantly at 0 (DP-1021).
			if m.CombatRetired() {
				continue
			}
			updateMobPosFromHP(m, m.GetHP())
		} else if pos == PosIncap {
			w.selfDamage(m, 1, combat.TYPE_SUFFERING)
			if m.CombatRetired() {
				continue
			}
			updateMobPosFromHP(m, m.GetHP())
		} else if pos == PosMortally {
			w.selfDamage(m, 2, combat.TYPE_SUFFERING)
			if m.CombatRetired() {
				continue
			}
			updateMobPosFromHP(m, m.GetHP())
		}

		// Memory clearing — limits.c:516-518
		// 1 in 99 chance of clearing mob memory
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if len(m.GetMemory()) > 0 && dprng.Number(0, 98) == 0 {
			clearMemory(m)
		}

	}

	// C finishes the character list before its single global object_list walk.
	// Snapshot IDs in descending creation order: read_object prepends each new
	// object to C's list, and objects created while processing this snapshot are
	// not visited until the next point_update (src/db.c:1892-1893,1950-1951;
	// src/limits.c:525-686).
	for _, obj := range w.pointUpdateObjectList() {
		if w.objectIsRegistered(obj) {
			w.decayObject(obj)
		}
	}
}

// clearMemory clears a mob's memory — from handler.c
func clearMemory(m *MobInstance) {
	m.ClearMemory()
}

// pointUpdateObjectList snapshots the live object registry in the order C's
// prepended object_list is traversed.
func (w *World) pointUpdateObjectList() []*ObjectInstance {
	w.mu.RLock()
	objects := make([]*ObjectInstance, 0, len(w.objectInstances))
	for _, obj := range w.objectInstances {
		objects = append(objects, obj)
	}
	w.mu.RUnlock()
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID > objects[j].ID })
	return objects
}

func (w *World) objectIsRegistered(obj *ObjectInstance) bool {
	if obj == nil {
		return false
	}
	w.mu.RLock()
	registered := w.objectInstances[obj.ID] == obj
	w.mu.RUnlock()
	return registered
}

// decayObjectsInRoom remains a narrow helper for existing room-local unit
// regressions. Production point_update uses the global object_list snapshot.
func (w *World) decayObjectsInRoom(roomVNum int) {
	items := append([]*ObjectInstance(nil), w.GetItemsInRoom(roomVNum)...)
	for _, obj := range items {
		w.decayObject(obj)
	}
}

// decayObject ports one object_list iteration from limits.c:527-686.
func (w *World) decayObject(obj *ObjectInstance) {
	if obj == nil || (obj.Prototype == nil && !obj.IsCorpse) {
		return
	}
	vnum := obj.GetVNum()
	location := w.objectLocation(obj)
	roomVNum := 0
	if location.Kind == ObjInRoom {
		roomVNum = location.RoomVNum
	}

	if specName, ok := GetObjSpecName(vnum); ok && specName == "moon_gate" {
		timer := obj.GetValue(2)
		if timer > 0 {
			timer--
			obj.SetValue(2, timer)
		}
		if timer == 0 {
			if location.Kind == ObjInRoom && len(w.actChar(roomVNum)) != 0 {
				w.sendToRoomExact(roomVNum, cap(fmt.Sprintf("%s fades out of existence.\r\n", obj.GetShortDesc())))
			}
			w.ExtractObject(obj, roomVNum)
		}
		return
	}

	if vnum == 20 || vnum == 21 {
		timer := obj.GetTimer()
		if timer > 0 {
			timer--
			obj.SetTimer(timer)
		}
		if timer == 0 {
			w.ExtractObject(obj, roomVNum)
		}
		return
	}

	if vnum == 18 {
		timer := obj.GetTimer()
		if timer > 0 {
			timer--
			obj.SetTimer(timer)
		}
		if timer == 0 {
			if location.Kind == ObjInRoom && len(w.actChar(roomVNum)) != 0 {
				w.sendToRoomExact(roomVNum, "The pile of dust is blown away by a draft of wind.\r\n")
			}
			w.ExtractObject(obj, roomVNum)
		}
		return
	}

	if vnum == 64 {
		timer := obj.GetTimer()
		if timer > 0 {
			timer--
			obj.SetTimer(timer)
		}
		if timer <= 0 {
			if location.Kind == ObjInRoom && len(w.actChar(roomVNum)) != 0 {
				w.sendToRoomExact(roomVNum, "The circle on the ground slowly fades away.\r\n")
			}
			w.ExtractObject(obj, roomVNum)
		}
		return
	}

	if obj.IsContainer() && obj.GetValue(3) != 0 {
		timer := obj.GetTimer()
		if timer > 0 {
			timer--
			obj.SetTimer(timer)
		}
		if timer == 0 {
			w.decayCorpse(obj, location, roomVNum)
		}
		return
	}

	// C reaches the field table only for floor objects and tests the C room
	// index, where world[0] is intentionally excluded (limits.c:658-684).
	if location.Kind != ObjInRoom || w.roomIndex(location.RoomVNum) <= 0 {
		return
	}
	for _, field := range fieldObjs {
		if vnum != field.ObjVNum {
			continue
		}
		timer := obj.GetTimer()
		if timer > 0 {
			timer--
			obj.SetTimer(timer)
		}
		if timer == 0 {
			if field.WornOffObjNum > 0 {
				if proto, ok := w.GetObjPrototype(field.WornOffObjNum); ok {
					spawned := w.NewObjectFromProto(proto, roomVNum)
					spawned.SetTimer(2)
					if err := w.MoveObjectToRoomFront(spawned, roomVNum); err != nil {
						slog.Warn("MoveObjectToRoomFront failed in worn-off spawn", "obj_vnum", spawned.GetVNum(), "room", roomVNum, "error", err)
						w.ExtractObject(spawned, roomVNum)
					}
				}
			}
			w.sendToRoomExact(roomVNum, field.WearOffMsg+"\r\n")
			w.ExtractObject(obj, roomVNum)
		}
		return
	}
}

func (w *World) decayCorpse(obj *ObjectInstance, location ObjectLocation, roomVNum int) {
	switch location.Kind {
	case ObjInInventory, ObjEquipped:
		carrier, carrierRoom := w.objectCarrier(location)
		if carrier != nil {
			Act(w, false, carrier, nil, obj, nil, "$p decays in your hands, and the contents fall to the ground.", "", ToChar)
			w.spillCorpseContents(obj, LocRoom(carrierRoom), carrierRoom)
		}
	case ObjInContainer:
		w.spillCorpseContents(obj, location, roomVNum)
	case ObjInRoom:
		actors := w.actChar(roomVNum)
		if len(actors) != 0 {
			messages := [...]string{
				"A quivering horde of maggots consumes $p.",
				"Dissolving into the ground, $p disappears.",
				"Dissolving into the ground, $p disappears.",
				"A horde of flesh-eating ants consumes $p.",
				"A horde of flesh-eating ants consumes $p.",
				"The earth opens up and swallows $p.",
				"The earth opens up and swallows $p.",
			}
			// #nosec G404 — game RNG, not cryptographic
			// #nosec G404
			message := messages[dprng.Number(0, len(messages)-1)]
			Act(w, true, actors[0], nil, obj, nil, message, "", ToRoom)
			Act(w, true, actors[0], nil, obj, nil, message, "", ToChar)
		}
		w.spillCorpseContents(obj, location, roomVNum)
	}
	w.ExtractObject(obj, roomVNum)
}

func (w *World) spillCorpseContents(obj *ObjectInstance, destination ObjectLocation, roomVNum int) {
	contents := append([]*ObjectInstance(nil), obj.GetContents()...)
	for _, contained := range contents {
		var err error
		switch destination.Kind {
		case ObjInRoom:
			err = w.MoveObjectToRoomFront(contained, roomVNum)
		case ObjInContainer:
			w.mu.RLock()
			parent := w.objectInstances[destination.ContainerObjID]
			w.mu.RUnlock()
			if parent == nil {
				err = fmt.Errorf("parent container %d no longer exists", destination.ContainerObjID)
			} else {
				err = w.MoveObjectToContainer(contained, parent)
			}
		}
		if err != nil {
			slog.Warn("move corpse contents failed", "obj_vnum", contained.GetVNum(), "room", roomVNum, "error", err)
		}
	}
}

func (w *World) objectLocation(obj *ObjectInstance) ObjectLocation {
	w.mu.RLock()
	location := obj.Location
	w.mu.RUnlock()
	return location
}

func (w *World) sendToRoomExact(roomVNum int, message string) {
	for _, actor := range w.actChar(roomVNum) {
		if actor.GetPosition() > PosSleeping {
			actor.SendMessage(message)
		}
	}
}

func (w *World) objectCarrier(location ObjectLocation) (Actor, int) {
	switch location.OwnerKind {
	case OwnerPlayer:
		w.mu.RLock()
		carrier := w.players[location.PlayerName]
		w.mu.RUnlock()
		if carrier != nil {
			return carrier, carrier.GetRoom()
		}
	case OwnerMob:
		w.mu.RLock()
		carrier := w.mobileObjectOwnerLocked(location.MobID)
		w.mu.RUnlock()
		if carrier != nil {
			return carrier, carrier.GetRoom()
		}
	}
	return nil, 0
}

func (w *World) roomIndex(roomVNum int) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for index, vnum := range w.roomOrder {
		if vnum == roomVNum {
			return index
		}
	}
	return -1
}

// updatePosFromHP updates a player's position based on their HP.
// Ported from fight.c update_pos()
