package olc

import "sync"

// ZoneSaveLock serializes the whole in-memory commit -> dirty-mark pair with
// a zone-file snapshot/save. Both telnet and web frontends use this lock so a
// concurrent disk save cannot clear a marker for a commit it did not write.
func ZoneSaveLock(zone int) *sync.Mutex {
	zoneSaveMu.Lock()
	defer zoneSaveMu.Unlock()
	mu, ok := zoneSaveLocks[zone]
	if !ok {
		mu = &sync.Mutex{}
		zoneSaveLocks[zone] = mu
	}
	return mu
}

var (
	zoneSaveMu    sync.Mutex
	zoneSaveLocks = make(map[int]*sync.Mutex)
)
