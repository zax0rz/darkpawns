# Depth-fidelity handoff — `report`

Date: 2026-09-01

## Queue position

This round began from refreshed `main` after `git pull --ff-only`, a
successful `make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md`
plus the latest `qsay` handoff. The special-procedure inventory remains
exhausted, the one blocked row `objmagic.sleep-entry-gates` remains queued after
its single cast-sleep vehicle, and the interpreter sweep advanced from `qsay`
to `report`.

The next unmanifested interpreter family is `reload` at
`src/interpreter.c:638`. `raise` and `reel` are shared socials, `read` is
covered by the boards manifest, and `recite` is covered by the object-magic and
recite manifests. The claimed `qecho` family and its open feature and handoff
PRs must not be repicked. The open purge and plot handoff PRs likewise remain
unmerged because their checks did not fire after their permitted retries.

Frontier before this slice: 2,937 total; 2,864 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,943 total; 2,870 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:634 */
{ "report"   , POS_RESTING , do_report   , 0, 0 },
```

`src/act.other.c:799-825` first emits `act("$n reports:", FALSE, ch, 0, 0,
TO_ROOM)`, then sends the actor `You report:\r\n`. It walks every character in
the room and formats the actor's hit, mana, movement, kills, PKs, and deaths;
the C color wrappers are evaluated against the recipient (`tch`). The handler
does not parse its argument, so trailing words are ignored. The shared command
dispatcher rejects a sleeping caller with `In your dreams, or what?` before the
handler runs. NPC/no-descriptor behavior is outside the player descriptor
command surface.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/report-depth.txt`

Manifest: `docs/fidelity/depth/report.tsv` (6 rows)

Focused test: `pkg/session/report_depth_test.go`, pinning the C
`POS_RESTING`/level-zero command gate.

The clean-main RED exposed one confirmed divergence: Go emitted the actor's
report and stat lines but omitted C's room-wide `$n reports:` header. The fix
routes that header through the canonical Go `Act` room adapter, preserving the
C recipient exclusion and player-facing formatting. The corrected vehicle was
GREEN at seeds 1, 2, 3, 5, and 8; seed 1 was run with `--show-oracle` before
the final multiseed proof. No `src/` or C-oracle file was edited.

## Verification and integration

All local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-report`

Feature commit: `b8b9d63e0` (`fix: match report room audience`)

Feature PR: #1104 — merged as `aa7bc6d27`. Hosted lint, security, and test
checks were green in run `33572941682`; build/deploy were skipped by workflow
conditions. The PR was merged only after every reported hosted check was green.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), R5b
and R5c (shared behavior ownership), and R5e (verify the actual C call path).
The source-order claim is maintained for the next `reload` slice.
