# Scoping — Differential (Oracle) Testing Against the Original C Dark Pawns

**Date:** 2026-07-12
**Author:** Claude (recon), for Zach
**Repo under test:** https://github.com/rparet/darkpawns — "Dark Pawns" v2.3, the
real shutdown-state (2010) source, released by the original Implementor ("Frontline").
**Goal:** run the original C server as a *reference oracle*, drive it and the Go port
with identical inputs, and treat any output divergence as a fidelity bug — instead of
(only) reading and hand-diffing the two codebases.

## TL;DR verdict: GREEN LIGHT

It's a **CircleMUD-derived** codebase, already **autotooled** (`configure` +
`automake`), C99, `<stdint.h>`-clean, with only three ordinary dependencies. Because
it's been *modernized* (not raw 1990s C), a **native macOS build alongside the Go repo
is the primary path** (Zach's preference — no Proxmox/container). The one real macOS
friction is modern **clang treating implicit function declarations as errors**, fixed
with a couple of `CFLAGS` (§2), not a VM. Critically, the RNG is a **single
self-contained seam we can make deterministic with a one-line patch.** Very buildable;
the oracle approach is sound.

---

## 1. What it is

- **Lineage:** CircleMUD (file naming `act.*.c`, `interpreter.c`, `nanny()`,
  `number()`/`dice()` in `utils.c`, world files under `lib/world/{mob,obj,wld,zon,shp}`).
  Our Go port is described as DikuMUD/Merc 2.2, but the C ancestor here is Circle-family
  — close cousin, same gameplay DNA. **`src/` still the single source of truth.**
- **Version:** 2.3 (`configure.ac` → `AC_INIT([darkpawns], [2.3])`).
- **Size:** 66 `.c` files in `src/`. World data in `lib/`.
- **License/provenance:** public release "warts and all"; README explicitly warns
  "significant effort might be required to return the game to a useable state." Expect
  boot-time friction (missing/renamed lib files, stale player index), not compile walls.

## 2. Build (native macOS — primary path, alongside the Go repo)

```
# deps: Xcode CLT gives clang; brew for autotools if you need to re-gen configure
brew install autoconf automake            # only needed if you must autoreconf
cd darkpawns
./configure CFLAGS="-g -O2 -Wno-implicit-function-declaration -Wno-implicit-int -Wno-int-conversion"
make
```

- `configure.ac` deps: **libm** (part of macOS libSystem — the `-lm` check passes as a
  no-op), **zlib** (ships with macOS, header present), **libcrypt** (macOS has no
  separate `libcrypt` or `crypt.h` — configure will *warn* and fall back; `crypt()`
  still resolves via libSystem. Passwords may go plaintext, which is FINE for oracle
  testing since we own the test accounts). `arpa/telnet.h` is present on macOS.
- **The one real macOS gotcha:** modern clang (15+, i.e. current Xcode) makes
  *implicit function declarations* and *implicit int* hard **errors**. Old
  CircleMUD-lineage code trips these constantly. The `-Wno-implicit-function-declaration
  -Wno-implicit-int -Wno-int-conversion` CFLAGS above demote them back to warnings and
  let the build through. This is the substitute for "fighting Apple" — a flag string,
  not a VM.
- Requires **C99** (`AC_PROG_CC_C99`); clang is C99 by default. Automake's
  `-Werror foreign` targets *automake's* warnings, not the compiler, so compiler
  warnings won't fail the build.
- `configure` is pre-generated & committed, so `./configure` works directly. If it
  complains about timestamps/aclocal, run `autoreconf -fi` first (hence the brew autotools).
- **Fallback only:** if native clang proves too painful, a Debian container builds it
  trivially (`build-essential zlib1g-dev libcrypt-dev`) — but we're deliberately avoiding
  containers this project, so treat that as last resort.
- Boot: `lib/` holds `world/ etc/ text/ misc/ database/ scripts/`. Launcher is
  `start_dp.sh`. Default port via `DFLT_PORT` in `comm.c`. Game entry / login state
  machine is **`nanny()` at `src/interpreter.c:1693`** — the exact flow we reverse-engineered
  during today's char-creation project, so we already know it cold.

## 3. The determinism seam — THE key unlock

The game ships its **own** portable PRNG (`src/random.c` / `random.h`):
`prng_seed(uint32_t)` and `prng_next()`. Every roll (`number()`, `dice()` in
`src/utils.c` + a couple in `magic.c`) rides `prng_next()`.

**It is seeded in exactly one place:**
```
src/comm.c:263:  prng_seed(time(0));
```

**Patch this one line** to honor an env var:
```c
{ char *s = getenv("DP_SEED"); prng_seed(s ? (uint32_t)strtoul(s,0,10) : time(0)); }
```
Now `DP_SEED=1 ./bin/darkpawns` produces a fully repeatable roll stream. That single
patch converts the server from "noisy" into a real oracle.

## 4. Two tiers of oracle testing (important scoping distinction)

Byte-identical transcripts across C and Go require *either* RNG-free inputs *or* the
same PRNG on both sides. So split the effort:

**Tier 1 — deterministic / structural parity (HIGH ROI, do first, NO PRNG port needed).**
Compare everything that doesn't depend on a roll: room/object/mob descriptions, menus,
help text, char-creation flow, stat/score display, formulas fed fixed inputs (carry
weight, movement cost per sector, XP tables, level thresholds, alignment math). This is
*exactly* the class of bug the audit + char-creation projects have been fixing — and it
needs no RNG matching at all, just `DP_SEED` pinned so nothing wobbles. Estimated to
catch the large majority of remaining divergences.

**Tier 2 — RNG-parity (STRETCH, combat rolls & procs).** To compare `number(1,101)`
to-hit rolls byte-for-byte, the Go port must reproduce `random.c`'s algorithm and seed.
Two options: (a) port `random.c`'s PRNG into Go behind a `--deterministic-seed` flag and
seed both identically; or (b) don't match streams — run N trials on each and compare
*distributions* (means, hit-rates, damage histograms) with tolerance. (a) is more work
but gives exact combat oracles; (b) is cheap and catches gross formula errors. Recommend
starting with (b), graduating hot paths to (a) if a bug hides in the noise.

## 5. Harness design

- **Dual driver:** a telnet/expect-style client (Go — reuse our existing WS/telnet test
  scaffolding) that scripts an identical command sequence to *both* the C server (port A)
  and the Go server (port B), captures both transcripts.
- **Normalizer:** strip/canonicalize the known-acceptable deltas — prompts, timestamps,
  ANSI color, uptime/whod lines, dynamic vnums — then `diff`. Surviving diffs = candidate
  fidelity bugs → file as `[Fidelity]` Linear issues, same pipeline as the audit.
- **Golden transcripts:** check scripted sessions + expected normalized output into the
  repo; CI replays them against the Go server. Captured *human* playtest sessions
  (Zach logging in) become regression transcripts for free — today's ~15-issue
  char-creation project is the proof of concept that *running it* finds what static
  review misses.

## 6. Recommended first moves (in order)

1. **Build + boot the C server natively on macOS** (§2 CFLAGS), alongside the Go repo;
   telnet in, confirm you reach a playable prompt. (This is the only real unknown —
   README warns of boot friction, and clang strictness is the likely compile snag.)
2. **Apply the `DP_SEED` one-line patch** (§3); confirm repeatable rolls.
3. **Tier-1 harness**: script the char-creation + a walk-around-and-look session against
   both servers; normalize + diff. Ship the diffs as issues.
4. Only then consider **Tier-2** RNG parity for combat.

Step 1 is a worker-friendly task (kimi/glm/mimo/deepseek in the container). Steps 2–3
are where the design judgment lives. Zach's rate-limit-preservation plan: workers do the
build/boot/harness plumbing; frontier (gpt / Claude) does the seam design and diff triage.

## 7. Risks / unknowns

- **Boot-state rot** (README's own warning): stale player files, missing lib entries,
  hardcoded paths. Most likely time sink. Mitigate: build in a container, keep `lib/`
  pristine, boot with a fresh player dir.
- **Circle-vs-Merc lineage drift:** the C ancestor is Circle-family; a few subsystems may
  differ in *structure* from what the Go port assumed. That's fine — the oracle tells us
  what the C *actually does*, which is the whole point.
- **Tier-2 PRNG parity** is real work; don't block Tier-1 on it.
