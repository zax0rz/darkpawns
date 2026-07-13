# BRIEF 2026-07-12 — Build & boot the original C Dark Pawns (oracle server), native macOS

**Executor:** daeron or kimi (this is C-build debugging — needs a capable coder who can
read clang errors; whoever's free). **NOT** a fidelity/logic task.
**Deliverable:** a running C server you can telnet into + a reproducible build-notes doc
committed to the Go repo. **One PR** (notes only — see §Deliverables).
**Companion doc:** read `docs/research/drafts/2026-07-12-c-oracle-differential-testing.md`
first — it explains why we're doing this (differential/oracle testing) and the two-tier plan.

---

## ⛔ THE ONE RULE THAT MATTERS

**The C source is our ground-truth oracle. Do NOT change game behavior — not a formula,
not a message string, not a table value, not a branch.** You are allowed to make the
tree *compile and boot on macOS*, nothing more. Permitted changes are limited to:
- build flags / `configure` invocation,
- adding missing `#include`s or function **prototypes** (declarations, not definitions),
- fixing hardcoded **paths** or **permissions** so it can find `lib/`,
- creating **empty runtime files/dirs** the server expects (player index, etc.).

**Every change you make to the C tree must be logged** in the build-notes doc with the
exact diff and a one-line "why this is behavior-preserving." If a compile or boot error
can only be resolved by altering game logic — **STOP and report it**; do not guess. A
wrong "fix" here silently corrupts the oracle and every test built on it.

**Do NOT apply the `DP_SEED` determinism patch** (`comm.c:263`) — that's a separate,
deliberately-designed step Claude owns. Leave the RNG seeding untouched.

---

## Where things live

- **Clone the C repo to a sibling dir of the Go repo, NOT inside it:**
  `git clone https://github.com/rparet/darkpawns.git /Users/zach/.openclaw/workspace/darkpawns-c-oracle`
- The Go repo (`/Users/zach/.openclaw/workspace/darkpawns_repo`) gets **only the notes
  doc** — never the C source or build artifacts.

## Build (native macOS / clang — primary path)

```
cd /Users/zach/.openclaw/workspace/darkpawns-c-oracle
# If ./configure balks on timestamps/aclocal:  brew install autoconf automake && autoreconf -fi
./configure CFLAGS="-g -O2 -Wno-implicit-function-declaration -Wno-implicit-int -Wno-int-conversion -Wno-return-type"
make
```

Known/expected friction (all pre-diagnosed — see companion doc §2):
- **clang implicit-declaration errors** are the likely wall — the `-Wno-*` CFLAGS above
  demote them to warnings. If new hard errors appear, add the minimal matching
  `-Wno-...` flag first; only touch source if a flag can't fix it (then log it).
- **libcrypt**: macOS has no `crypt.h`/`-lcrypt`; configure will *warn* and continue —
  that's expected and fine (test accounts, plaintext pw is acceptable for the oracle).
- **libm/zlib**: present on macOS; configure's checks pass.
- If `make` dies on a specific `.c`, capture the full error verbatim in the notes before
  attempting the smallest possible prototype/include fix.

## Boot to a live prompt

- CircleMUD-lineage: run from the dir containing `lib/`. Try `./start_dp.sh` first; if it
  fights you, run the binary directly (`bin/` after build) with a port arg. Default port
  is `DFLT_PORT` in `comm.c` — **use a non-conflicting local port (e.g. 4000)** and note it.
- Expect boot friction (README warns "significant effort might be required"): missing
  `lib/` runtime files, empty player index, stale paths. Fix by **creating empty
  files/dirs** the loader expects — log each. Keep `lib/world/*` pristine (that's game data).
- Success = the server stays up and accepts a telnet connection.

## Success criteria (all four)

1. `make` completes; a server binary exists.
2. Server boots and stays running on the chosen port without crash-looping.
3. `telnet localhost <port>` reaches the **login / "By what name..." prompt** (the
   `nanny()` state machine, `interpreter.c:1693`).
4. You can create a test character and reach the **in-game playable prompt**, then
   `quit` cleanly.

**Capture a raw transcript** of that telnet session (connect → create char → look →
quit) and paste it into the notes doc as proof-of-life.

## Deliverables (the PR)

Write `docs/research/drafts/2026-07-12-c-oracle-build-notes.md` in the **Go repo** on
branch `docs/c-oracle-build-notes`, containing:
- Exact working `./configure` + `make` command(s) and any `brew`/autoreconf steps.
- **Every change made to the C tree**: file, diff, and the behavior-preserving rationale
  (or "none needed" if it built clean).
- Every empty runtime file/dir you had to create to boot.
- The chosen port and the exact start/stop commands.
- The captured proof-of-life telnet transcript.
- Any error you hit but could NOT resolve without touching logic — flagged loudly for Claude.

Then: `go build ./... && go vet ./... && go test ./...` (should be untouched/green, since
this PR is docs-only), commit, push, `gh pr create`, and **STOP** — Zach/Claude review & merge.

## Out of scope
- The `DP_SEED` RNG patch (Claude owns it).
- Any test harness / diffing (that's the next brief, after this boots).
- Committing C source or build artifacts into the Go repo.
- Any gameplay-behavior change whatsoever (see THE ONE RULE).
