package combat

import (
	"sync"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// Roller defines the injectable combat RNG seam. Production call sites use
// Number and Dice; IntN remains only for compatibility with existing test doubles.
type Roller interface {
	Number(from, to int) int // C number(from,to): inclusive both ends
	Dice(num, size int) int  // C dice(num,size): sum of num d{size}
	IntN(n int) int
}

// productionRoller delegates to Dark Pawns' one process-wide CMWC stream.
type productionRoller struct{}

func (p *productionRoller) Number(from, to int) int {
	return dprng.Number(from, to)
}

func (p *productionRoller) Dice(num, size int) int {
	return dprng.Dice(num, size)
}

func (p *productionRoller) IntN(n int) int {
	if n <= 0 {
		panic("invalid argument to Roller.IntN")
	}
	return dprng.Number(0, n-1)
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
	rng *dprng.Generator
}

// NewSeededRoller creates a CMWC-backed test roller from one C-compatible seed.
func NewSeededRoller(seed uint32) *SeededRoller {
	return &SeededRoller{
		rng: dprng.New(seed),
	}
}

func (s *SeededRoller) Number(from, to int) int {
	return s.rng.Number(from, to)
}

func (s *SeededRoller) Dice(num, size int) int {
	return s.rng.Dice(num, size)
}

func (s *SeededRoller) IntN(n int) int {
	if n <= 0 {
		panic("invalid argument to Roller.IntN")
	}
	return s.rng.Number(0, n-1)
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
