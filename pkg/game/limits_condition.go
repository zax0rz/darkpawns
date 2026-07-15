package game

import (
	"fmt"
	"log/slog"
	"os"

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
	// Snapshot players under read lock, operate without lock
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, m := range w.activeMobs {
		mobs = append(mobs, m)
	}
	w.mu.RUnlock()

	// --- Players ---
	for _, p := range players {
		p.mu.RLock()
		pos := p.Position
		p.mu.RUnlock()

		// Condition decay — skip if inactive (PRF_INACTIVE)
		if p.Flags&(1<<PrfInactive) == 0 {
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

		// Jail timer — decrement each tick, auto-release when it hits 0
		p.mu.Lock()
		if p.JailTimer > 0 {
			p.JailTimer--
			if p.JailTimer == 0 {
				p.SetRoom(MortalStartRoom)
				p.SendMessage("\r\nYour jail sentence is served. You are free!\r\n")
			}
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
			poisoned := p.Affects&(1<<AffPoison) != 0
			cutthroat := p.Affects&(1<<AffCutthroat) != 0
			p.mu.RUnlock()

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

			// Poison damage — limits.c:503-504
			if poisoned {
				p.TakeDamage(10)
			}

			// Cutthroat damage — limits.c:505-506
			if cutthroat {
				p.TakeDamage(13)
			}

			// Poison/cutthroat can drive HP into the wounded band (or past the
			// POS_DEAD threshold, HP <= -11). Re-derive position from the new HP
			// and only die at POS_DEAD — C: damage(ch, ch, dam, ...) runs
			// update_pos then die() only once GET_HIT <= -11 (DP-1021).
			updatePosFromHP(p, p.GetHP())
			if p.GetPosition() == PosDead {
				w.HandleNonCombatDeath(p)
				continue
			}
		} else if pos == PosIncap {
			// Incapacitated: 1 damage per tick — limits.c:511
			p.TakeDamage(1)
			updatePosFromHP(p, p.GetHP())
			if p.GetPosition() == PosDead {
				w.HandleNonCombatDeath(p)
				continue
			}
		} else if pos == PosMortally {
			// Mortally wounded: 2 damage per tick — limits.c:513
			p.TakeDamage(2)
			updatePosFromHP(p, p.GetHP())
			if p.GetPosition() == PosDead {
				w.HandleNonCombatDeath(p)
				continue
			}
		}

		// Memory clearing for NPCs — limits.c:516-518
		// (handled in NPC section below)

		// Idle check for players — limits.c:521-524
		w.CheckIdling(p)
	}

	// --- NPCs ---
	// Object decay is driven per-room, not per-mob. C's point_update() iterates
	// the global object_list exactly once per tick (src/limits.c:525-686); the
	// decay pass is NOT nested inside the character loop. Without dedup, a room
	// with N mobs would decay its objects N× per tick (DP-1036).
	decayedRooms := make(map[int]bool)
	for _, m := range mobs {
		pos := m.GetPosition()
		roomVNum := m.GetRoomVNum()

		if pos >= PosStunned {
			// HP regen — limits.c:498-501
			if m.GetHP() < m.GetMaxHP() {
				gain := MobHitGain(m)
				m.SetHealth(m.GetHP() + gain)
				if m.GetHP() > m.GetMaxHP() {
					m.SetHealth(m.GetMaxHP())
				}
			}
			// Poison damage — limits.c:503-504 (applies to ALL chars including NPCs)
			if m.HasAffect(AffPoison) {
				m.TakeDamage(10)
			}
			// Cutthroat damage — limits.c:505-506
			if m.HasAffect(AffCutthroat) {
				m.TakeDamage(13)
			}
			// Re-derive position from the new HP; only die at POS_DEAD
			// (HP <= -11) so poison/cutthroat progress through the wounded
			// band instead of killing instantly at 0 (DP-1021).
			updateMobPosFromHP(m, m.GetHP())
			if m.GetPosition() == PosDead {
				w.handleMobDeath(m, nil, -1)
				continue
			}
		} else if pos == PosIncap {
			m.TakeDamage(1)
			updateMobPosFromHP(m, m.GetHP())
			if m.GetPosition() == PosDead {
				w.handleMobDeath(m, nil, -1)
				continue
			}
		} else if pos == PosMortally {
			m.TakeDamage(2)
			updateMobPosFromHP(m, m.GetHP())
			if m.GetPosition() == PosDead {
				w.handleMobDeath(m, nil, -1)
				continue
			}
		}

		// Memory clearing — limits.c:516-518
		// 1 in 99 chance of clearing mob memory
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if len(m.GetMemory()) > 0 && dprng.Number(0, 98) == 0 {
			clearMemory(m)
		}

		// Object decay for things in this mob's room — once per room per tick.
		// C iterates object_list globally; Go iterates per-mob, so dedup by room
		// to avoid N× decay when N mobs share a room (DP-1036).
		_ = roomVNum
		if !decayedRooms[roomVNum] {
			decayedRooms[roomVNum] = true
			w.decayObjectsInRoom(roomVNum)
		}
	}

	// Second pass: decay objects in rooms that have no mobs. C's point_update()
	// iterates the global object_list, so corpses in cleared rooms still decay.
	// The dedup map keeps this from double-decaying rooms already handled above.
	rooms := w.Rooms()
	for i := range rooms {
		room := &rooms[i]
		if !decayedRooms[room.VNum] {
			decayedRooms[room.VNum] = true
			w.decayObjectsInRoom(room.VNum)
		}
	}
}

// clearMemory clears a mob's memory — from handler.c
func clearMemory(m *MobInstance) {
	m.ClearMemory()
}

// ShowMOTD reads and returns the MOTD file content.
// Source: comm.c nanny() CON_MOTD reads lib/text/motd
func ShowMOTD(worldPath string) string {
	paths := []string{
		worldPath + "/text/motd",
		worldPath + "/motd",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

// ShowBackground reads the setting's background story, falling back to a
// short built-in introduction when an older world install has no text file.
func ShowBackground(worldPath string) string {
	paths := []string{
		worldPath + "/text/background",
		worldPath + "/background",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
	}
	return "Darkness has settled over the old kingdoms. From the ruins, rival powers " +
		"move their pawns across a world of forgotten magick, ancient grudges, and dangerous ambition.\r\n"
}

// decayObjectsInRoom decays objects in the given room.
// Ported from limits.c point_update() object section (lines 527-686).
func (w *World) decayObjectsInRoom(roomVNum int) {
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		// Allow corpses (IsCorpse=true) through even with nil Prototype so the
		// corpse decay block below can fire. All other nil-prototype objects skip.
		if obj.Prototype == nil && !obj.IsCorpse {
			continue
		}
		objVNum := obj.GetVNum()

		// Corpse decay — ITEM_CONTAINER with val[3] set (corpse flag)
		if obj.IsContainer() && obj.GetValue(3) != 0 {
			if obj.GetTimer() > 0 {
				obj.SetTimer(obj.GetTimer() - 1)
			}
			if obj.GetTimer() == 0 {
				// Scatter contents to room
				for _, contained := range obj.GetContents() {
					obj.RemoveFromContainer(contained)
					contained.SetRoomVNum(roomVNum)
					if err := w.MoveObjectToRoom(contained, roomVNum); err != nil {
						slog.Warn("MoveObjectToRoom failed in decay", "obj_vnum", contained.GetVNum(), "room", roomVNum, "error", err)
					}
				}
				// Random decay message
				msgs := []string{
					"A quivering horde of maggots consumes %s.\r\n",
					"Dissolving into the ground, %s disappears.\r\n",
					"Dissolving into the ground, %s disappears.\r\n",
					"A horde of flesh-eating ants consumes %s.\r\n",
					"A horde of flesh-eating ants consumes %s.\r\n",
					"The earth opens up and swallows %s.\r\n",
					"The earth opens up and swallows %s.\r\n",
				}
				// #nosec G404 — game RNG, not cryptographic
				// #nosec G404
				msg := fmt.Sprintf(msgs[dprng.Number(0, len(msgs)-1)], obj.GetShortDesc())
				w.SendToRoom(roomVNum, msg)
				w.ExtractObject(obj, roomVNum)
				continue
			}
		}

		// Puddle/puke decay
		if objVNum == 20 || objVNum == 21 {
			if obj.GetTimer() > 0 {
				obj.SetTimer(obj.GetTimer() - 1)
			}
			if obj.GetTimer() == 0 {
				w.ExtractObject(obj, roomVNum)
				continue
			}
		}

		// Dust decay
		if objVNum == 18 {
			if obj.GetTimer() > 0 {
				obj.SetTimer(obj.GetTimer() - 1)
			}
			if obj.GetTimer() == 0 {
				w.SendToRoom(roomVNum, "The pile of dust is blown away by a draft of wind.\r\n")
				w.ExtractObject(obj, roomVNum)
				continue
			}
		}

		// Circle of summoning (COC_VNUM = 64)
		if objVNum == 64 {
			if obj.GetTimer() > 0 {
				obj.SetTimer(obj.GetTimer() - 1)
			}
			if obj.GetTimer() <= 0 {
				w.SendToRoom(roomVNum, "The circle on the ground slowly fades away.\r\n")
				w.ExtractObject(obj, roomVNum)
				continue
			}
		}

		// Field objects — check against fieldObjs table
		for _, fo := range fieldObjs {
			if objVNum == fo.ObjVNum {
				if obj.GetTimer() > 0 {
					obj.SetTimer(obj.GetTimer() - 1)
				}
				if obj.GetTimer() == 0 {
					if fo.WornOffObjNum > 0 {
						if proto, ok := w.GetObjPrototype(fo.WornOffObjNum); ok {
							spawned := NewObjectInstance(proto, roomVNum)
							spawned.SetTimer(2)
							if err := w.MoveObjectToRoom(spawned, roomVNum); err != nil {
								slog.Warn("MoveObjectToRoom failed in worn-off spawn", "obj_vnum", spawned.GetVNum(), "room", roomVNum, "error", err)
							}
						}
					}
					w.SendToRoom(roomVNum, fo.WearOffMsg+"\r\n")
					w.ExtractObject(obj, roomVNum)
				}
				break
			}
		}
	}
}

// updatePosFromHP updates a player's position based on their HP.
// Ported from fight.c update_pos()
