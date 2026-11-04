# Idiomatic-Go experiment: predictions

Written 2026-11-03, before the run started. Committing this file to main
is the pre-registration: git dates it before any refactor commit exists.
Don't edit the predictions after the run starts. Record results in a new
`RESULTS.md`.

## The experiment

Google's FVCOM post ("Modernizing 40-year-old scientific code with Gemini 4
Argon") ported 174,774 lines of Fortran and C to Rust in 59 minutes. It
checked the output against the original across 100 seeds, reported six
decades-old bugs, and released the workflow as the `migrate-to-rust` skill
(github.com/google/skills, `skills/developers/migrate-to-rust`).

Our port already exists. Much of the Go still reads like translated C: long
functions, package-level state, C-style names. This experiment gives one
overnight GPT-Luna run the job of making the game logic idiomatic Go while
keeping every player-facing byte and every random-number draw identical
(R1, R3). The brief is [`BRIEF.md`](BRIEF.md).

Two questions:

1. **The engineering question:** how much C-shaped code can one unattended
   night remove without changing behaviour?
2. **The research question (Study 1):** when an agent refactors code that is
   meant to change nothing, how much of that code can the oracle census
   actually see? Each commit carries a mutation probe: a deliberate one-byte
   output change in a touched function, run against the same scenarios. A
   probe that stays green marks code the safety net can't see (R5h).

The Google skill borrowed the outline: Stage 0 baseline, audit blueprint,
bounded self-healing (5 tries, then record and move on), before/after scan,
and seed parity. It dropped PGO, the Struct-of-Arrays layout, and the
infographic. A MUD isn't CPU-bound.

## Scope

`pkg/game`, `pkg/combat`, `pkg/command`. Non-test code only.

Out of scope, with reasons:

- **`pkg/spells`:** PF-095 found 52 hollow spellmatrix scenarios, so the net
  is partly blind there until DP-1456 lands.
- **`pkg/session`, `pkg/db`, `pkg/telnet`:** login, persistence and
  transport are stop-tier.
- **The oracle harness:** `internal/oraclediff` and `cmd/dp-oracle-diff`.

## Baseline (origin/main `e08e0a017`)

Measured with `go run ./docs/research/idiom-experiment/metrics . pkg/game pkg/combat pkg/command`
(the script is in [`metrics/`](metrics/)):

| Measure | pkg/game | pkg/combat | pkg/command | Total |
|---|---:|---:|---:|---:|
| Files | 197 | 10 | 6 | 213 |
| Lines | 65,406 | 4,606 | 3,545 | **73,557** |
| Functions | 2,434 | 193 | 96 | 2,723 |
| Median function length (lines) | 10 | 6 | 19 | 10 |
| Functions over 100 lines | 41 | 4 | 3 | **48** |
| Functions over 200 lines | 6 | 2 | 1 | 9 |
| Longest function (lines) | 747 | 237 | 446 | 747 |
| Functions nested 5 or more deep | 19 | 2 | 1 | 22 |
| snake_case declarations | 71 | 55 | 0 | 126 |
| Empty `interface{}` | 83 | 1 | 4 | 88 |
| Package-level vars | 194 | 8 | 3 | 205 |
| `CustomData` references | 41 | 0 | 0 | 41 |

Lint findings, including test files, from
`golangci-lint run --default=none --enable=dupl,funlen,gocognit,gocyclo --max-issues-per-linter=0 --max-same-issues=0 ./pkg/game/... ./pkg/combat/... ./pkg/command/...`:
dupl 26, funlen 190, gocognit 112, gocyclo 4.

Server binary (`go build ./cmd/server`, linux/amd64): 40,280,199 bytes.
Luna measures boot time and memory as the first step of the run, before
changing anything, and repeats the same procedure at the end.

## Predictions

My predictions (Claude), each with an 80% interval: I expect 8 of every 10
true values to land inside it. Zach's column is for your own guesses.
Write them in before you go to bed; there's no wrong answer.

| # | Measure | Baseline | Claude predicts | 80% interval | Zach guesses |
|---|---|---:|---:|---|---|
| 1 | Net lines in scope | 73,557 | **−1,500** (−2%) | +500 to −5,000 |  |
| 2 | Chance the line count goes down at all | | 75% | | |
| 3 | Functions over 100 lines | 48 | 30 | 20 to 42 |  |
| 4 | Longest function | 747 | 350 | 150 to 747 |  |
| 5 | funlen findings | 190 | 150 | 120 to 185 |  |
| 6 | gocognit findings | 112 | 85 | 65 to 108 |  |
| 7 | snake_case declarations | 126 | 40 | 0 to 110 |  |
| 8 | Package-level vars | 205 | 195 | 170 to 205 |  |
| 9 | Refactor commits kept | 0 | 25 | 10 to 45 |  |
| 10 | Commits whose first targeted census went red (the refactor changed behaviour and the net caught it) | | 5 | 1 to 15 |  |
| 11 | **Mutation probes that stay green (code the census can't see)** | | **25%** | 10% to 45% |  |
| 12 | Latent port bugs logged (found by reading, not fixed) | | 8 | 2 to 20 |  |
| 13 | Edits to protected paths (tests, scenarios, expected files, normalizer) | | 0 | 0 to 0 |  |
| 14 | Change in boot time and peak memory | | within ±5% | | |
| 15 | Change in binary size | | within ±1% | | |
| 16 | Final combined census on the branch tip is clean | | 70% | | |
| 17 | Morning review finds at least one kept, green commit that still changes behaviour | | 40% | | |

## Reasons for the main predictions

- **Lines go down only a little.** Idiomatic Go isn't always shorter.
  Splitting a 700-line function into named helpers adds signatures and doc
  comments. Little in scope is duplicated: 26 dupl findings, against 230 in
  the whole repo, where most sit in session, admin and tests. Real removal
  comes from collapsing repeated blocks and dead branches, and the deadcode
  ratchet already took the easy dead code.
- **Global state barely moves.** Package-level state is structural. Moving
  it into structs touches hundreds of call sites. Done safely, under a
  per-commit census, one night covers little of it.
- **No speed change.** Nothing in scope is a hot loop. The FVCOM speedup
  came from memory layout in a numerical solver. This code is text
  formatting and table lookups.
- **The probe number is the one that matters.** PF-095 and PF-096 already
  showed that scenarios can be green without touching the code they claim
  to cover. Breadth scenarios hit common paths. Long functions hide rare
  branches. So about a quarter of refactored code should prove invisible to
  the census. If that's right, "the census is green" overstates how safe a
  refactor is. The fidelity paper can measure that claim directly.
- **Prediction 17:** a refactor that moves a random draw past an early
  return changes the draw count (R3a, R3d). The census only notices when a
  scenario reaches that exact branch with that exact seed. I review the
  diff for draw-order changes in the morning, as an independent check.
