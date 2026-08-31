# Depth handoff — 2026-08-31 — `hide` / `kabuki`

## Frontier and queue position

- Started from clean `main` at `fdddbfb42` after the merged `hiccup` handoff,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-hiccup.md`.
- The starting frontier was 2,125 total, with 2,065 proven/delegated, 16
  blocked, and 44 excluded. The shared hide-family slice adds 21
  proven/delegated cases, producing 2,146 total, 2,086 proven/delegated, 16
  blocked, and 44 excluded; actionable completion is 2,086/2,102 (99.2%).
- `hide` is registered at `src/interpreter.c:493`, and `kabuki` at
  `src/interpreter.c:526`; both reach `do_hide` in `src/act.other.c:247-306`
  with different `subcmd` values. An exact interpreter-table sweep leaves
  `highfive` at `src/interpreter.c:494` as the next unmanifested command
  family; the next session must return to clean `main`, pull, rerun the
  frontier check, reread this handoff, and begin `highfive`.

## C call path and branch inventory

The C registration and handler were traced before changing Go:

- Both rows require `POS_RESTING` and command level 1. `hide` selects the
  ordinary `SKILL_HIDE` message/roll; `kabuki` selects `SCMD_KABUKI`, the
  `SKILL_KABUKI` roll, and its alternate attempt message.
- `do_hide` checks `IS_MOUNTED` first and emits `Dismount first!` without
  consulting weather, emitting attempt text, or drawing RNG.
- In daylight (`weather_info.sunlight != SUN_DARK`), the sector switch rejects
  field, desert, grouped water sectors (swim, no-swim, underwater, water), and
  grouped exposed sectors (flying, fire, earth, wind), each with its authored
  refusal. Other sectors continue.
- The ordinary continuation emits its attempt line, clears `AFF_HIDE` if
  present, draws `number(1, 101)`, compares against skill plus the dexterity
  hide bonus, applies `AFF_HIDE` on success, and calls `improve_skill` in C's
  order. Kabuki follows the same state/RNG path with its own skill selector.
- Arguments are not read by the handler. The daytime gate is shared by both
  variants and is independent of `ROOM_INDOORS`.

## Confirmed divergence and fix

The field vehicle was run against pre-fix `main` before the feature branch.
In a fixed daytime FIELD room, C returned `Hide out here during the day? Yeah
right.` for both commands, while Go invented the ordinary hide/kabuki attempt
messages and proceeded toward the roll. This confirmed the missing shared
weather/sector gate.

Only that confirmed divergence was fixed. The live command path now supplies
the authoritative world to a shared `doHide` implementation, which ports the
C sunlight/sector switch and retains a nil-world wrapper for direct game-layer
tests. The same gate is applied to `DoKabukiInWorld`; no source or C-oracle
file was edited.

## Coverage proof

- Added live vehicles for mounted, field, desert, representative water, and
  representative exposed-sector gates, plus an allowed daytime city vehicle
  with ignored arguments for both variants. All matched C; the allowed vehicle
  matched across seeds `1, 2, 3, 5, 8`, and one run used `--show-oracle`.
- Added `TestHideFamilyRegistrationUsesCEntryGates` and
  `TestDoHideDaytimeSectorGates`. The latter checks every grouped water and
  exposed value and the nighttime bypass; the existing
  `TestDoHideDexBonusToggleAndImprove` remains the focused clear/reroll/
  improvement proof.
- Added 21 manifest rows in `docs/fidelity/depth/hide.tsv`; shared position
  and kabuki state branches are delegated only where the owning hide-family
  case is explicit.

This follows R1/R2/R3/R4/R5e and R5c: C refusal and attempt bytes remain law,
both registered command surfaces and deterministic roll paths are covered, no
unreachable branches were invented, the actual shared handler path was
verified, and the full do_hide behavior class was rechecked.

## Changes, gates, and integration

- PR #937 (`glm/depth-hide`, commit `cef13f32b`) passed hosted `test`, `lint`,
  and `security` checks; release-only build/deploy jobs were skipped as
  expected. It was merged only after every reported check was green; merged
  `main` is `b4a04b682`.
- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  `gofumpt -l .`, and `git diff --check`.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `highfive`
at `src/interpreter.c:494`.
