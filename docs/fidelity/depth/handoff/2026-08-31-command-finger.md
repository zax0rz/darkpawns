# Depth handoff — 2026-08-31 — `finger`

## Frontier and queue position

- Started the slice from clean `main` at `34f070dfd` after the merged `fill`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-fill.md`.
- The frontier before this slice was 1,722 total, with 1,665
  proven/delegated, 16 blocked, and 41 excluded. The `finger` manifest adds
  nine cases, all proven/delegated: eight oracle cases and one entry-gate unit
  case. The post-slice frontier is 1,731 total, 1,674 proven/delegated, 16
  blocked, and 41 excluded; actionable completion is 1,674/1,690 (99.1%).
- The source-order command gap was `finger`, registered at
  `src/interpreter.c:444`; its handler is also registered as `whois` at
  `src/interpreter.c:819`. The next unmanifested interpreter-table family is
  `flex` at line 447: `flee` at 445 and `flesh` at 446 are already covered.

## C call path and branch inventory

`src/interpreter.c:444` registers `finger` with `POS_DEAD`, no minimum level,
and `do_whois` in `src/new_cmds.c:1402-1421`. The same handler is registered
again as `whois` at line 819. `do_whois` calls C `one_argument` at line 1407,
returns the exact no-argument prompt when no token remains, calls `load_char`
for the selected first token, returns the exact not-found line on failure, and
formats level, class abbreviation, name, and title on success. The shared C
parser at `src/interpreter.c:1265-1283` skips fill words, lowercases the first
non-fill token, and discards trailing input; `load_char` is implemented at
`src/db.c:2342-2391`.

The clean-main baseline was RED for the confirmed parser divergence: Go joined
all command arguments, so `finger the Fingerpeer` and
`finger Fingerpeer trailing words are ignored` looked up a multi-word name and
returned `There is no such player.` while C selected `Fingerpeer` and printed
the saved record. No-argument, exact-name, missing-name, and the `whois` exact
name path were already byte-identical.

The Go fix uses the existing exported `game.OneArgument` helper before the
online/database lookup, sweeping both `finger` and `whois` registrations.
The focused linkless vehicle drops a saved peer before probing; C still runs
`load_char`, while Go retains the linkless player in its live world, and both
produce the same player-facing record bytes. No speculative persistence
schema change was made.

## Coverage proof

`finger-depth` covers no arguments, found player formatting, leading fill-word
parsing, trailing-token discard, not-found, and both the `finger` command and
the `whois` registration. `finger-offline-depth` covers the saved linkless
target path. Both scenarios were run with `--show-oracle` at seed 1; all
listed vehicles were GREEN for seeds `1,2,3,5,8`.

`TestFingerRegistrationUsesCEntryGate` verifies the C `POS_DEAD`/level-zero
registration. The manifest records all nine cases. No `src/` or
`darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4, R5e, and
R5c: C bytes and actual dispatch/parser paths remain authoritative, the
confirmed shared `do_whois` parsing gap was fixed in Go, and both registrations
were swept together.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,731 total / 1,674 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #863 (`fix: align finger argument parsing`) was merged only after hosted
`lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, reread the newest
handoff, and begin `flex`.
