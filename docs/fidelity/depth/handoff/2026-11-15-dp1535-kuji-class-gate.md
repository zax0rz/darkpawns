# DP-1535 Kuji Kiri mobile class gate

## Change

PR fixes two shipped switched-body skill lanes whose Go mobile handlers checked
only level and missed C's body-class allowance:

- `do_kuji_kiri` allows Ninja or immortal bodies (`src/new_cmds.c:1559-1563`).
  NPC prototypes default to class zero (`src/db.c:1096`), but `set <mob> class
  ninja` is supported (`src/act.wizard.c:2582,2935-2940`). Go now checks the
  active mobile's class with its level in `CharacterKujiKiri`.
- `do_peek` allows Thief, Assassin, or immortal bodies (`src/act.other.c:1673-
  1679`). Go's mobile path had hardcoded the mage prototype class; it now reads
  the active mobile class.

The C-first witnesses set the clone's class before switching. The Kuji witness
issues `rin`: C passes the class gate, draws `number(1,101)`, then emits the
zero-skill refusal. The Peek witness issues bare `peek` and reaches the
no-argument prompt. Ordinary player handlers remain unchanged.

## R5c audit

Searched shipped switched-body skill handlers for the same class/immortal gate
shape and followed each C path:

- Kuji Kiri and Peek were the two missed live-body class gates; both are fixed.
- `do_retreat` reads class to choose Escape/Retreat labels and skill, but has no
  class/immortal eligibility gate (`src/act.offensive.c:1008,1040,1048`). With
  the current zero mobile skills its mobile branch returns `Huh?` first, as
  Go does.
- `do_track` uses warrior/paladin/ranger macros that exclude NPCs
  (`src/graph.c:192`; `src/utils.h:408-416`); the mobile refusal is already
  faithful.
- `do_cast` returns for NPCs before reaching its class-indexed level checks
  (`src/spell_parser.c:923-924,963`).
- No additional shipped switched-body skill lane with the Kuji/Peek
  class-or-immortal gate was found. No rulebook change is proposed.

## Evidence

- Candidate census at implementation commit `e2fb5252b`:
  `/home/zach/Archives/darkpawns/oracle-runs/2026-11-15/luna-dp1535-classgate-candidate`
  — 52/52 clean, `--draw-parity`, zero divergent scenarios. Both new witnesses
  are non-hollow and match output and draws. Two unrelated older combat-wait
  probes account for the census's 2 hollow steps.
- Retained revert commit `8caedb452` in
  `/home/zach/wt/luna-dp1535-kuji-gate-revert`, with the two new C-first
  witnesses preserved. Run:
  `/home/zach/Archives/darkpawns/oracle-runs/2026-11-15/luna-dp1535-classgate-revert`
  — both class-qualified switched witnesses fail; ordinary `kuji-depth` and
  `peek-depth` pass. In the Kuji revert, C emits “You have not mastered this
  art yet!” after drawing `Number(1,101)=94`; Go emits “You know nothing of
  kuji-kiri!” and skips the draw. The Peek revert says “You're not a thief!”
  instead of C's no-argument prompt. This demonstrates the new witnesses fail
  on the reverted behavior.
- Gate results before the docs-only evidence update: `make fmt`,
  `go build ./...`, `go vet ./...`, `GOEXPERIMENT=nojsonv2 go test -p 4 ./...`,
  `golangci-lint run --concurrency 4 ./...`, and `make deadcode-ratchet` all
  passed. These are rerun on the final commit before the PR is opened.

## Decisions for Zach

- Review this as its own stop-tier PR from current main. Do not merge here.
- The `do_peek` correction is included because the R5c class-gate audit found
  the same switched-body defect while tracing the Kuji omission.
- DP-1535 queue work does not advance in this PR.
