# Depth handoff — 2026-08-31 — `fwap`

## Frontier and queue position

- Started from clean `main` at `a5326f389` after the merged `future` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-future.md`.
- The frontier before this slice was 1,826 total, with 1,769
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `fwap`
  manifest adds ten proven/delegated cases: eight direct cases and two shared
  delegations. The post-slice frontier is 1,836 total, with 1,779
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,779/1,795 (99.1%).
- The source-order command gap was `fwap`, registered at
  `src/interpreter.c:457`. The next command-table gap is `gag` at line 460;
  the next session must return to clean `main`, pull, rerun the frontier
  check, reread this handoff, and begin `gag`.

## C call path and branch inventory

`src/interpreter.c:457` registers `fwap` with `POS_RESTING`, no minimum level,
and `do_action`. Its authored social record is `lib/misc/socials:285-293`:
hide flag zero, victim minimum position zero, distinct no-argument,
target-found, not-found, self-target, and audience messages. The actual
handler path is `src/act.social.c:102-151`: the shared PLR_NOSHOUT gate,
conditional C `one_argument`, visible room lookup, self branch, victim
position gate, and the actor/non-victim/victim `act` audiences.

The standing vehicle uses an awake peer and observer to prove the no-target
pair, target trio, exact missing-target text, self pair, and leading fill-word
with trailing-token parsing. The companion vehicle forces the target asleep;
because the record's minimum victim position is zero, the target still passes
the social gate, while the plain `TO_VICT` act suppresses its private line and
the awake actor/observer retain their C bytes. Shared command position,
PLR_NOSHOUT, visibility, and Act filtering are delegated to existing owners.

## Coverage proof

The clean-main vehicles were GREEN. `fwap-depth --show-oracle` and
`fwap-sleeping-depth --show-oracle` reported no normalized divergence for
seeds `1,2,3,5,8`. `TestFwapRegistrationUsesCEntryGate` pins both the C command
gate and the social record metadata. No Go behavior change was inferred, and
no `src/` or `darkpawns-c-oracle/` file was edited.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the registered command
surface remain authoritative, the actual `do_action` call path was verified,
and shared social behavior is delegated rather than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,836 total / 1,779 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

Implementation PR #883 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. This handoff must itself be merged with green
checks before the next session begins `gag`.
