# Depth handoff — 2026-08-31 — `gag`

## Frontier and queue position

- Started from clean `main` at `01db3d893` after the merged queue-order
  correction, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-queue-correction-get-complete.md`.
- The frontier before this slice was 1,836 total, with 1,779
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `gag` manifest
  adds ten proven/delegated cases: seven direct cases and three shared
  delegations. The post-slice frontier is 1,846 total, with 1,789
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,789/1,805 (99.1%).
- The source-order command gap was `gag`, registered at
  `src/interpreter.c:460`. The next command-table gap is `gasp` at line 461;
  the next session must return to clean `main`, pull, rerun the frontier
  check, reread this handoff, and begin `gasp`.

## C call path and branch inventory

`src/interpreter.c:460` registers `gag` with `POS_RESTING`, no minimum level,
and `do_action`. Its authored social record is `lib/misc/socials:1188-1196`:
`gag 0 5`, meaning no hide-invisible flag and a `POS_RESTING` minimum victim
position, followed by distinct no-argument, target-found, not-found,
self-target, and audience messages. The actual handler path is
`src/act.social.c:102-151`: shared PLR_NOSHOUT, conditional C `one_argument`,
visible room lookup, self branch, victim-position gate, and actor/
non-victim/victim `act` audiences.

The standing vehicle uses an awake peer and observer to prove the no-target
pair, target trio, exact missing-target text, self pair, and leading fill-word
with trailing-token parsing. The companion forces the target asleep; the
record's POS_RESTING minimum rejects it before any social audience emission.
Shared command position, PLR_NOSHOUT, visibility, and Act filtering are
delegated to existing owners. The Go social data intentionally uses legacy
field names: C hide=0 is `MinLevel=0`, and C victim position=5 is
`HideFlag=5` with no explicit `MinVictimPosition` override.

## Coverage proof

The clean-main vehicles were GREEN. `gag-depth --show-oracle` and
`gag-sleeping-depth --show-oracle` reported no normalized divergence for
seeds `1,2,3,5,8`; one standing seed-8 boot attempt hit an external transient
port collision and its single honest retry was GREEN. `TestGagRegistrationUsesCEntryGate`
pins both the C command gate and the social record metadata. No Go behavior
change was inferred, and no `src/` or `darkpawns-c-oracle/` file was edited.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the registered command
surface remain authoritative, the actual `do_action` call path was verified,
the C-to-Go legacy field mapping was checked against its consumer, and shared
social behavior is delegated rather than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,846 total / 1,789 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

Implementation PR #887 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. This handoff must itself be merged with green
checks before the next session begins `gasp`.
