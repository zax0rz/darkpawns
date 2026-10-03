package game

import "log/slog"

// ZoneClockSnapshot exposes runtime state without changing parser.Zone or saves.
type ZoneClockSnapshot struct {
	Ages        map[int]int
	Queue       []int
	MinuteTicks int
}

func (w *World) ZoneClockSnapshot() ZoneClockSnapshot {
	w.zoneResetMu.Lock()
	defer w.zoneResetMu.Unlock()
	snapshot := ZoneClockSnapshot{Ages: make(map[int]int), Queue: append([]int(nil), w.zoneResetQueue...), MinuteTicks: w.zoneMinuteTicks}
	for _, z := range w.GetAllZones() {
		snapshot.Ages[z.Number] = w.zoneAges[z.Number]
	}
	return snapshot
}

func (w *World) setZoneAgeLocked(number, age int) {
	if w.zoneAges == nil {
		w.zoneAges = make(map[int]int)
	}
	w.zoneAges[number] = age
}

// ZoneUpdate ports src/db.c:1967-2045. Called every 100 pulses in both modes.
// Lock order: reset mutex, then brief manager/body snapshots and World reads;
// no World or manager lock is held across reset execution or output.
func (w *World) ZoneUpdate() {
	w.zoneResetMu.Lock()
	defer w.zoneResetMu.Unlock()
	if !w.zoneClockStarted {
		return
	}
	zones := w.GetAllZones()
	w.zoneMinuteTicks++
	if w.zoneMinuteTicks >= 6 {
		w.zoneMinuteTicks = 0
		for _, z := range zones {
			age := w.zoneAges[z.Number]
			if age < z.Lifespan && z.ResetMode != 0 {
				age++
			}
			if age >= z.Lifespan && age < 999 && z.ResetMode != 0 {
				w.zoneResetQueue = append(w.zoneResetQueue, z.Number)
				age = 999
			}
			w.setZoneAgeLocked(z.Number, age)
		}
	}
	occupied := make(map[int]bool)
	if w.OccupiedZoneRooms != nil {
		for _, vnum := range w.OccupiedZoneRooms() {
			if room, ok := w.GetRoom(vnum); ok {
				occupied[room.Zone] = true
			}
		}
	}
	for i, number := range w.zoneResetQueue {
		zone, ok := w.GetZone(number)
		if !ok || (zone.ResetMode != 2 && occupied[number]) {
			continue
		}
		if err := w.spawner.executeZoneResetLocked(zone); err != nil {
			slog.Error("automatic zone reset failed", "zone", number, "error", err)
			return
		}
		w.zoneResetQueue = append(w.zoneResetQueue[:i], w.zoneResetQueue[i+1:]...)
		break
	}
}
