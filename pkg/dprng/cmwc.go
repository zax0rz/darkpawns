// Package dprng implements Dark Pawns' process-wide C-compatible random
// stream. The generator and range conversion intentionally mirror
// src/random.c and src/utils.c rather than Go's math/rand algorithms.
package dprng

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

const uniformScale = float32(2.328306437e-10)

// Generator is the complementary-multiply-with-carry generator from
// src/random.c. Generator methods are not safe for concurrent use; callers
// that share one must serialize access.
type Generator struct {
	q     [1024]uint32
	carry uint32
	index uint32
}

// New returns a freshly constructed CMWC generator. As in C, construction
// initializes the function-static carry and index before seeding Q.
func New(seed uint32) *Generator {
	g := &Generator{carry: 362436, index: 1023}
	g.Seed(seed)
	return g
}

// Seed fills Q using the C xorshift loop. It deliberately does not reset the
// carry or index: cmwc_seed() leaves cmwc_next()'s function-static state alone.
func (g *Generator) Seed(seed uint32) {
	value := seed
	for i := range g.q {
		value ^= value << 13
		value ^= value >> 17
		value ^= value << 5
		g.q[i] = value
	}
}

// Next returns the next raw uint32 from the CMWC stream.
func (g *Generator) Next() uint32 {
	const (
		multiplier = uint64(123471786)
		r          = uint32(0xfffffffe)
	)

	g.index = (g.index + 1) & 1023
	t := multiplier*uint64(g.q[g.index]) + uint64(g.carry)
	g.carry = uint32(t >> 32)
	x := uint32(t + uint64(g.carry)) // #nosec G115 -- intentional modular truncation required to match the C CMWC algorithm
	if x < g.carry {
		x++
		g.carry++
	}
	g.q[g.index] = r - x
	return g.q[g.index]
}

// Uniform returns the C prng_uniform() value in [0, 1). Both multiplications
// in Number remain float32 to preserve C rounding and draw parity.
func (g *Generator) Uniform() float32 {
	return float32(g.Next()) * uniformScale
}

// Number returns one random integer in the inclusive range [from, to]. It
// consumes exactly one generator draw, including when from equals to.
func (g *Generator) Number(from, to int) int {
	if from > to {
		from, to = to, from
	}
	return int(g.Uniform()*float32(to-from+1)) + from
}

// Dice returns the sum of num rolls in [1, size], consuming exactly num draws.
func (g *Generator) Dice(num, size int) int {
	if num <= 0 || size <= 0 {
		return 0
	}
	sum := 0
	for range num {
		sum += g.Number(1, size)
	}
	return sum
}

var (
	streamMu sync.Mutex
	stream   = New(uint32(time.Now().Unix()))
)

// Seed replaces Q in the process-wide stream while preserving C's persistent
// carry/index contract. Boot code must call this before production draws.
func Seed(seed uint32) {
	streamMu.Lock()
	defer streamMu.Unlock()
	stream.Seed(seed)
}

// ResetStream reinstalls a fresh process-wide stream seeded with `seed`, discarding
// any accumulated carry/index. Unlike Seed (which preserves C's persistent carry
// contract across reseeds), this yields a byte-identical stream on every call. It
// exists so tests of code that draws from the global stream can establish a
// reproducible starting state; production boot uses Seed/ConfigureFromEnvironment.
func ResetStream(seed uint32) {
	streamMu.Lock()
	defer streamMu.Unlock()
	stream = New(seed)
}

// ConfigureFromEnvironment seeds the process-wide stream from DP_SEED, or
// from the current Unix time when DP_SEED is unset, matching C init_game().
func ConfigureFromEnvironment() (uint32, error) {
	seed, err := seedFromEnvironment(os.LookupEnv, time.Now)
	if err != nil {
		return 0, err
	}
	Seed(seed)
	return seed, nil
}

func seedFromEnvironment(lookup func(string) (string, bool), now func() time.Time) (uint32, error) {
	seed := uint32(now().Unix()) // #nosec G115 -- intentional truncation of Unix time to a 32-bit PRNG seed
	if value, ok := lookup("DP_SEED"); ok {
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse DP_SEED %q: %w", value, err)
		}
		seed = uint32(parsed)
	}
	return seed, nil
}

// Next returns one raw value from the process-wide stream.
func Next() uint32 {
	streamMu.Lock()
	defer streamMu.Unlock()
	return stream.Next()
}

// Uniform returns one C-compatible float32 from the process-wide stream.
func Uniform() float32 {
	streamMu.Lock()
	defer streamMu.Unlock()
	return stream.Uniform()
}

// Number returns one value in [from, to] from the process-wide stream.
func Number(from, to int) int {
	streamMu.Lock()
	defer streamMu.Unlock()
	return stream.Number(from, to)
}

// Dice returns one C-compatible dice sum from the process-wide stream.
func Dice(num, size int) int {
	streamMu.Lock()
	defer streamMu.Unlock()
	return stream.Dice(num, size)
}
