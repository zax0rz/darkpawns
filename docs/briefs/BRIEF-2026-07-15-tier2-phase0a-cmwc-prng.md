# BRIEF — Tier-2 Phase 0a: CMWC PRNG port + one seeded RNG stream

**For:** codex (frontier). **Owner of gate:** Claude (parity proof + review vs C).
**Branch:** `refactor/tier2-cmwc-prng` off `main`.
**This is the KEYSTONE of the skills/spells endeavor** — nothing RNG-driven (combat, skills,
spells) is oracle-testable until Go and the C oracle roll the *same numbers in the same order*
for a shared seed. **Method rules:** read `src/random.c` + `src/utils.c` `number`/`dice` directly.
Gated by a **generator-parity proof** (Go stream == checked-in C golden), not just a green build.

---

## 0. Why this exists / scope boundary
Combat/skill/spell effects all draw from C's one global PRNG (`random.c`). We already seed the C
oracle deterministically via `DP_SEED` (comm.c seam, proven). The Go port currently uses
`math/rand/v2` (PCG) in **two** competing abstractions plus ~45 direct call sites — none seed-
compatible with C. **Phase 0a delivers the generator + the single stream + seed wiring.** It does
NOT touch spell/skill *logic* (that's Phase 2) or the skill-system *commands* (Phase 1). Keep this
PR to the RNG substrate.

## 1. The C generator — port byte-exact (`src/random.c`)
Complementary-multiply-with-carry (Marsaglia). Port into a new `pkg/dprng` (name TBD) as a struct
holding all state — **no globals, no `math/rand`**.

```c
static uint32_t Q[1024];
static void cmwc_seed(uint32_t j) {           /* fills Q via xorshift */
  for (int i = 0; i < 1024; i++) {
    j ^= j << 13;  j ^= j >> 17;  j ^= j << 5;   /* all uint32, wrapping */
    Q[i] = j;
  }
}
static uint32_t cmwc_next() {
  uint64_t t, a = 123471786ULL;
  static uint32_t c = 362436, i = 1023;         /* PERSIST across calls; seed does NOT reset them */
  uint32_t x, r = 0xfffffffe;
  i = (i + 1) & 1023;
  t = a * Q[i] + c;
  c = (t >> 32);
  x = t + c;                                     /* uint32 truncation */
  if (x < c) { x++; c++; }
  return (Q[i] = r - x);                          /* uint32 wrap */
}
```
**Fidelity landmines (each has burned ports before):**
1. **`c` and `i` are function-static, initialized ONCE to `362436` / `1023`.** `cmwc_seed` fills
   `Q` but leaves `c`/`i` alone. So the FIRST `next()` after any seed uses `i=(1023+1)&1023=0`,
   `c=362436`. In Go: struct fields `c=362436, i=1023` set at construction, seed only rewrites `Q`.
2. **All arithmetic is unsigned 32/64-bit with wraparound.** Use `uint32`/`uint64`; the xorshift and
   `r - x` MUST wrap, not panic/clamp.
3. **`number()` uses float32, NOT float64** (§2) — this is the single most common parity break.

## 2. `number()` / `dice()` — port from `src/utils.c` (float32-exact)
```c
/* uniform() == prng_uniform(): float f = 2.328306437e-10f;  return prng_next() * f;  (FLOAT32) */
int number(int from, int to){
  if (from > to) { int t=from; from=to; to=t; }
  return (int)(uniform() * (to - from + 1)) + from;   /* float32 mul, C-style trunc-toward-zero */
}
int dice(int num, int size){
  if (size <= 0 || num <= 0) return 0;
  int sum = 0; while (num--) sum += number(1, size); return sum;
}
```
In Go: `u := float32(g.next()) * float32(2.328306437e-10)` then `int(u*float32(to-from+1)) + from`.
Use `float32` throughout the multiply and Go's `int()` conversion (truncates toward zero, matching
C's `(int)` on a non-negative product). **Draw-count parity is law:** `number()` consumes exactly
one `next()`; `dice(n,s)` consumes exactly `n`. Never batch, cache, or reorder draws.

## 3. Golden parity target (captured from the real C generator, `DP_SEED=42`)
Check these into `testdata/` and assert them in a unit test. **Claude generated these from the C
oracle's `random.c`** (probe compiled against the real object) — they are the source of truth:

Raw `prng_next()`, seed 42, first 16:
```
3718346876 3507964525 1302043376 3853678134 3224156370 1202735980 146185864 771127629
755436127 1898814665 203008726 4229820860 3246251873 2521895320 2882264546 2716584201
```
`number()`/`dice()` from a FRESH seed-42 generator (each line consumes from where the previous left
off — i.e. one continuous stream, 10+10+10 `number()` draws then 8×`dice(2,6)`=16 draws):
```
number(1,100): 87 82 31 90 76 29 4 18 18 45
number(0,99):  4 98 75 58 67 63 50 93 61 45
number(1,6):   1 5 5 6 6 6 4 3 4 5
dice(2,6):     10 10 6 12 10 5 9 10
```
Sanity anchor: `next[0]=3718346876`; `3718346876 / (2³²−1) · 100 = 86.57 → (int)=86, +1 = 87`. If
your `number(1,100)[0]` isn't 87, the float path is wrong (likely float64, or wrong constant).
**Ask Claude to regenerate the golden for any additional seed/length** (the probe is Claude's; do
not reconstruct it from a re-implementation — that would be marking your own homework).

## 4. One stream: consolidation (the anti-"second-iteration" work)
A perfect generator is useless if spells bypass it. Today ~45 non-test files call `math/rand`
directly (incl. `pkg/spells/{saving_throws,damage_spells,affect_spells}.go`,
`pkg/game/skill_*.go`), plus a second helper `pkg/common.Number`, plus `pkg/combat.Roller`
(PCG). **Everything must route through the ONE seeded generator.**
1. Make `pkg/combat`'s production `Roller` (and `pkg/common.Number`) delegate to the new CMWC
   generator. Keep the `Roller` interface (`Number`/`Dice`/`IntN`) as the seam.
2. **Audit every `IntN` and direct `rand.*` site → express it as C `number(from,to)`/`dice`.** The
   `IntN` "escape hatch" is a fidelity hole: `rand.IntN(100)+1` must become `number(1,100)`;
   `rand.IntN(n)` used as a pick is C `number(0, n-1)`. Each site's C equivalent must match the
   real C call at that logic point — where a C source exists, cite it; where it's Go-invented
   (e.g. Go-only spells), route it through the generator anyway so the stream stays coherent, and
   list those sites for Claude to review (they'll matter when Phase 2 defines their oracle status).
3. **Add an import-guard ratchet** (mirror `internal/lintguard/u1000_ratchet_test.go`): fail CI if
   any non-test file outside the generator package imports `math/rand`/`math/rand/v2`. This is what
   stops the sprawl from regrowing.

## 5. Seed wiring
- Go reads `DP_SEED` at boot (env), seeds the global generator; **unset → seed from time** (matches
  C's `init_game` fallback — byte-identical non-determinism when unused). Mirror the C seam's
  contract exactly.
- The oracle-diff harness already passes `DP_SEED` to the C oracle (`cmd/dp-oracle-diff/main.go:184`);
  extend it to pass the SAME `DP_SEED` to the Go port process so both share a seed in scenarios.
- Concurrency: the generator is process-global mutable state. C is single-threaded; Go isn't. Guard
  with a mutex (like `rollerMu`) — but note this only makes each draw atomic, NOT ordered across
  goroutines. For oracle determinism, RNG-driven scenarios must drive a single actor so draws are
  serialized (document this; ordering across concurrent players is inherently non-deterministic and
  out of scope — Phase 2 scenarios are single-threaded).

## 6. Acceptance gate
1. **Generator parity (the proof):** unit test asserts Go `next()` stream == the §3 golden (seed 42,
   ≥16 values) AND `number()`/`dice()` == the §3 golden. Add ≥1 more seed (ask Claude for the golden).
2. **One stream:** import-guard ratchet green; `grep` shows no direct `math/rand` in non-test game/
   spell/combat code; `pkg/common.Number` + `combat.Roller` delegate to CMWC.
3. **Seed wiring:** `DP_SEED=K` reproducible across Go runs; unset → varies. Harness passes it to both.
4. **End-to-end smoke:** one oracle scenario exercising a *single* deterministic RNG draw path
   (Claude will design the scenario — likely a bare-hand combat swing or a fixed `dice` readout) shows
   Go and C agree with `DP_SEED` set. This validates the whole chain before Phase 2 scales it.
5. `make check-fmt vet` + `go test ./...` green; no WS schema break; Roller/Scripted test doubles
   still work (keep `ScriptedRoller` for unit tests that assert on specific rolls).

## 7. Roadmap (context — NOT this PR)
- **Phase 0b:** finish any consolidation stragglers + the end-to-end combat-round parity scenario.
- **Phase 1 (deterministic, C-faithful model — DECIDED):** `cast` grammar/targeting/gates (O39),
  guild-owned `practice` + use-based `improve_skill` (O36), reconcile `skills`/`spells` listing
  (O37/O38), **retire the Go-invented learn/forget/catalog/skillinfo**. Oracle-gated like prior domains.
- **Phase 2:** per-spell/skill effect parity — ~163 `spello`/`skillo` entries — each with a
  red→green oracle scenario under a shared seed, enforcing draw-order parity. The multi-day bulk;
  built on a per-spell scenario template Claude will define once Phase 0/1 land.

## 8. Gotchas
- **Never touch the oracle** to make parity pass ([[darkpawns-oracle-proof-gate]]). The generator is
  ported TO Go; the C side is the reference.
- **float32.** Say it three times. `number()[0]` for seed 42 is 87 or the port is wrong.
- **Draw order is the whole game.** Same numbers, same order, same count. A Go site that draws in a
  different sequence than its C counterpart desyncs everything downstream in that scenario.
