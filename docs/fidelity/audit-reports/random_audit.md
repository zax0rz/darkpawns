# Port Fidelity Audit: Module 44 (`random.c`)

This audit examines the port fidelity between the legacy C source file `src/random.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/random.c` (74 lines)
- **Functions & Features**:
  - **Complementary-Multiply-With-Carry (CMWC) Generator**: Implements the advanced CMWC pseudo-random number generator designed by George Marsaglia.
  - **Massive Period ($2^{131104}$)**: Provides high-quality pseudo-randomness specifically tailored to MUD combat rolls, level gains, and AI triggers.
  - **Xorshift Seeding**: Seeds the CMWC generator array using an initial xorshift sequence.

### Go Port Files
- **Go Implementation**:
  - **OMITTED (CMWC Bypass)**: There is **no Go port of the CMWC algorithm**.
  - **Standard Library Replacement**: The MUD port replaces the Marsaglia CMWC custom generator entirely with Go's standard library `"math/rand"` and `"math/rand/v2"` packages.
  - **Ad-Hoc Inline Formulas**: MUD math formulas (like `number(min, max)`) are scattered inline across all packages using standard `rand.Intn` expressions (e.g. `rand.Intn(5) + 4` for `number(4,8)`).

---

## 2. High-Fidelity Validation & Gaps

This audit has identified a **critical concurrency bottleneck** and **code maintenance risks** within the Go pseudo-random implementation:

### 1. Global Mutex Contention Bottleneck (math/rand)
- **The Gap**: In Go, the legacy `"math/rand"` standard library utilizes a single, global, package-level pseudo-random generator. This package-level generator is protected by a single global mutex (`sync.Mutex`) inside the Go runtime to ensure thread safety.
- In Dark Pawns, dozens of concurrent goroutines (session write/read pumps, combat violence loops, weather updates, mob wandering tickers, Lua script triggers, and AI agent threads) constantly call `rand.Intn()` concurrently.
- **Impact**: All threads block each other on the single shared global rand mutex, causing massive **lock contention, CPU waste, and thread stalling** under high player or AI load.

### 2. High Translation & Off-by-One Range Risks
- **The Gap**: Legacy C has a unified `number(min, max)` helper macro. It guarantees inclusive random bounds.
- In Go, standard `rand.Intn(N)` takes an exclusive bound `[0, N)`. Instead of utilizing a central unified wrapper helper, the port translates all Diku formulas to ad-hoc inline statements:
  - `number(4,8)` -> `rand.Intn(5) + 4`
  - `number(GET_LEVEL(ch), 3 * GET_LEVEL(ch))` -> `rand.Intn(3*p.Level - p.Level + 1) + p.Level`
- **Impact**: Distributing complex, manual arithmetic inline across ~73K lines of Go code is highly prone to off-by-one bugs, range violations, and typos during updates, significantly hurting long-term code quality.

---

## 3. Go's Architectural Improvements Over C

- **Zero Memory/State Bloat**: In C, the Marsaglia CMWC algorithm requires maintaining an internal static state array `Q[1024]` in memory. Go standard library generators have extremely low footprints and are optimized at the compiler level.
- **Modern Algorithms**: Go 1.22+ (`math/rand/v2`) incorporates state-of-the-art PCG (Permuted Congruential Generator) algorithms (`chacha8` and `pcgdxsm`), delivering excellent statistical distribution, higher quality than legacy C, and fast performance.

---

## 4. Concurrency & Thread Safety

- **Old math/rand Thread Contention**: As documented, using package-level `math/rand` is thread-safe but suffers from terrible mutex contention.
- **math/rand/v2 Lock-Free Mode**: Go's newer `math/rand/v2` package uses lock-free, thread-local generators for global functions (like `rand.IntN`), completely avoiding mutex contention. Currently, only a few packages (like `pkg/engine/affect.go`) use `math/rand/v2`, while the majority of packages still use the bottleneck-prone `"math/rand"`.

---

## 5. Summary of Recommended Next Steps

1. **Migrate fully to `math/rand/v2`**:
   Refactor all package imports from `"math/rand"` to `"math/rand/v2"` to utilize lock-free global random number functions and prevent mutex bottlenecks.
2. **Implement a Unified `Number(min, max)` Utility**:
   Define a central helper function `Number(min, max int) int` inside a general utility package (e.g. `pkg/common/random.go`) to isolate the exclusive-bound arithmetic and prevent off-by-one translation bugs.
