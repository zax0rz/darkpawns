package game

import (
	"fmt"
	"math/rand"
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

	p.mu.Lock()
	p.Conditions[condition] += value
	if p.Conditions[condition] < 0 {
		p.Conditions[condition] = 0
	}
	if p.Conditions[condition] > 48 {
		p.Conditions[condition] = 48
	}
	newCond := p.Conditions[condition]
	p.mu.Unlock()

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
			curPos := p.Position
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

			// Update position if HP has dropped low — limits.c:507-508
			if curPos <= PosStunned {
				p.mu.RLock()
				hp := p.Health
				p.mu.RUnlock()
				updatePosFromHP(p, hp)
			}
		} else if pos == PosIncap {
			// Incapacitated: 1 damage per tick — limits.c:511
			p.TakeDamage(1)
		} else if pos == PosMortally {
			// Mortally wounded: 2 damage per tick — limits.c:513
			p.TakeDamage(2)
		}

		// Memory clearing for NPCs — limits.c:516-518
		// (handled in NPC section below)

		// Idle check for players — limits.c:521-524
		w.CheckIdling(p)
	}

	// --- NPCs ---
	for _, m := range mobs {
		pos := m.GetPosition()
		roomVNum := m.GetRoomVNum()

		if pos >= PosStunned {
			if m.CurrentHP < m.MaxHP {
				gain := MobHitGain(m)
				m.CurrentHP += gain
				if m.CurrentHP > m.MaxHP {
					m.CurrentHP = m.MaxHP
				}
			}
		} else if pos == PosIncap {
			m.TakeDamage(1)
		} else if pos == PosMortally {
			m.TakeDamage(2)
		}

		// Memory clearing — limits.c:516-518
		// 1 in 99 chance of clearing mob memory
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if m.Memory != nil && rand.Intn(99) == 0 {
			clearMemory(m)
		}

		// Object decay for things in this mob's room
		_ = roomVNum
		w.decayObjectsInRoom(roomVNum)
	}
}

// clearMemory clears a mob's memory — from handler.c
func clearMemory(m *MobInstance) {
	m.Memory = nil
}

// decayObjectsInRoom decays objects in the given room.
// Ported from limits.c point_update() object section (lines 527-686).
func (w *World) decayObjectsInRoom(roomVNum int) {
	items := w.GetItemsInRoom(roomVNum)
	for _, obj := range items {
		if obj.Prototype == nil {
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
// #nosec G104
					w.MoveObjectToRoom(contained, roomVNum)
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
				msg := fmt.Sprintf(msgs[rand.Intn(len(msgs))], obj.GetShortDesc())
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
// #nosec G104
							w.MoveObjectToRoom(spawned, roomVNum)
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
