package combat

import (
	"math/rand/v2"
	"sync"
)

// Roller defines an interface that mirrors C random primitives and Go slice indexing helpers.
type Roller interface {
	Number(from, to int) int // C number(from,to): inclusive both ends
	Dice(num, size int) int  // C dice(num,size): sum of num d{size}
	IntN(n int) int          // escape hatch for the few raw [0,n) sites (e.g. slice picks)
}

// productionRoller is the production implementation wrapping math/rand/v2.
type productionRoller struct{}

func (p *productionRoller) Number(from, to int) int {
	if to < from {
		from, to = to, from
	}
	return rand.IntN(to-from+1) + from
}

func (p *productionRoller) Dice(num, size int) int {
	if num <= 0 || size <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < num; i++ {
		total += rand.IntN(size) + 1
	}
	return total
}

func (p *productionRoller) IntN(n int) int {
	return rand.IntN(n)
}

var (
	rollerMu      sync.RWMutex
	defaultRoller Roller = &productionRoller{}
)

// GetRoller returns the current default Roller.
func GetRoller() Roller {
	rollerMu.RLock()
	defer rollerMu.RUnlock()
	return defaultRoller
}

// SetRoller overrides the global default Roller.
func SetRoller(r Roller) {
	rollerMu.Lock()
	defer rollerMu.Unlock()
	defaultRoller = r
}

// WithRoller overrides the default Roller for the scope of the given function.
func WithRoller(r Roller, fn func()) {
	rollerMu.Lock()
	old := defaultRoller
	defaultRoller = r
	rollerMu.Unlock()

	defer func() {
		rollerMu.Lock()
		defaultRoller = old
		rollerMu.Unlock()
	}()

	fn()
}

// SeededRoller is a seedable deterministic Roller for tests.
type SeededRoller struct {
	rng *rand.Rand
}

// NewSeededRoller creates a new SeededRoller.
func NewSeededRoller(seed1, seed2 uint64) *SeededRoller {
	return &SeededRoller{
		rng: rand.New(rand.NewPCG(seed1, seed2)),
	}
}

func (s *SeededRoller) Number(from, to int) int {
	if to < from {
		from, to = to, from
	}
	return s.rng.IntN(to-from+1) + from
}

func (s *SeededRoller) Dice(num, size int) int {
	if num <= 0 || size <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < num; i++ {
		total += s.rng.IntN(size) + 1
	}
	return total
}

func (s *SeededRoller) IntN(n int) int {
	return s.rng.IntN(n)
}

// ScriptedRoller is a mock Roller that yields a scripted list of values.
type ScriptedRoller struct {
	mu     sync.Mutex
	Values []int
	Index  int
}

// NewScriptedRoller creates a new ScriptedRoller.
func NewScriptedRoller(values []int) *ScriptedRoller {
	return &ScriptedRoller{Values: values}
}

func (s *ScriptedRoller) next() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Values) == 0 {
		return 0
	}
	val := s.Values[s.Index]
	s.Index = (s.Index + 1) % len(s.Values)
	return val
}

func (s *ScriptedRoller) Number(from, to int) int {
	return s.next()
}

func (s *ScriptedRoller) Dice(num, size int) int {
	total := 0
	for i := 0; i < num; i++ {
		total += s.next()
	}
	return total
}

func (s *ScriptedRoller) IntN(n int) int {
	return s.next()
}
