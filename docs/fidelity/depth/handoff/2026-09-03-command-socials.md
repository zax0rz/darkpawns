# Depth-fidelity handoff — `socials`

Date: 2026-09-03

Branch: `glm/depth-socials`

Feature PR: #1271 (merged green)

Feature commit: `41070e0c2`

Main merge: `e2935304f`

Handoff branch: `handoff/2026-09-03-command-socials`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, confirmed the
frontier with `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`
and the newest dated handoff, and took the next unclaimed interpreter row
after `snuggle`.

The special-procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed
cast-sleep outlaw/reagent vehicle and was not repicked. The next source-order
row after this slice is `split` at `src/interpreter.c:726`; no dedicated
`split` depth manifest exists on this fresh `main` audit.

Pre-slice frontier: 3,810 total, 3,707 proven/delegated, 48 blocked, and 55
excluded. The `socials-command` manifest adds 8 proven/delegated cases.
Post-slice frontier: 3,818 total, 3,715 proven/delegated, 48 blocked, and 55
excluded; actionable completion is 3,715/3,763 = 98.7%.

## C call path and confirmed divergence

The registered C row is:

```c
/* src/interpreter.c:725 */
{ "socials"  , POS_DEAD    , do_commands , 0, SCMD_SOCIALS },
```

`src/act.informative.c:2633-2675` uses `one_argument` to optionally resolve a
visible world character, rejects NPCs and targets above the actor's level,
sets `socials = 1`, and walks C's sorted command table. `sort_commands` marks
all `do_action` rows social and explicitly marks `insult`; rows at or above
LVL_IMMORT are excluded because this is not `wizhelp`. It formats seven
`%-11s` entries per line, appends a final CRLF, and calls `page_string`.
`src/handler.c:1277-1331` supplies the visible room-first/world lookup.

The clean-main-equivalent oracle vehicle was RED before the fix. Go listed
the four social records that have no C command row (`hiss`, `kneel`, `mutter`,
and `roll`), included the level-gated `snowball` for a level-one viewer,
omitted C's special `insult`, used 16-column formatting, bypassed paging, and
ignored self/peer/NPC/higher-level target branches.

The confirmed fix added the C-derived `social_command_order.tsv`, target
resolution and level/NPC gates, the exact C 11-column layout and final CRLF,
and `PageString`. No behavior was invented for the four Go-only records.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/socials-command-depth.txt`;
- `docs/fidelity/depth/socials-command.tsv`;
- `pkg/session/social_command_order.tsv`; and
- `pkg/session/socials_command_depth_test.go`.

The corrected oracle vehicle reported no normalized divergence with
`--show-oracle` at seed 1 and with seeds 2, 3, 5, and 8. It covers no
argument, self target, same-level peer, first-token/trailing input,
higher-level target, missing target, NPC target, the full two-page listing,
and pager continuation.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check`

PR #1271's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was needed. The PR was self-merged only after the applicable
checks were green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (deterministic listing and pager transcript), R4 (no invented behavior),
R5/R5e (verify the actual C path and let C win), and R5b/R5c (shared target
and pager ownership boundaries).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`split` at `src/interpreter.c:726`. Follow `do_split` through its actual C
call path and compare it against the current Go implementation before
creating `glm/depth-split`. Do not repick `socials`, the generic `social`
family, or the blocked sleep-entry row.
