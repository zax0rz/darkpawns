# Depth-fidelity handoff — `qui`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest available qecho handoff. The special-procedure inventory remains
exhausted, the one blocked row `objmagic.sleep-entry-gates` remains queued
after its single cast-sleep vehicle, and the interpreter sweep advanced from
the claimed `qecho` row to `qui`.

The next unmanifested interpreter family is `quit` at
`src/interpreter.c:630`. `qecho` remains claimed by its handoff and its
feature PR is open; it must not be repicked. `quaff` and `quest` are already
represented by existing depth manifests.

Frontier before this slice: 2,904 total; 2,833 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,911 total; 2,839 proven/delegated; 22 blocked; 50
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:629 */
{ "qui"      , POS_DEAD    , do_quit     , 0, 0 },
```

`src/act.other.c:72-181` begins with an NPC/no-descriptor early return, then
computes the safe-room state and reaches the abbreviated-command guard at
lines 114-116. For mortal players, `qui` is rejected with exactly
`You have to type quit--no less, to quit!\r\n`, before room safety, combat,
health, equipment, or extraction logic. Arguments are ignored by this
handler. For immortals, the guard is skipped and the same `do_quit` call
continues through the real logout path: the actor receives the goodbye line,
the room peer receives the leave act, and the player is extracted. The full
shared `do_quit` teardown remains owned by the separate `quit`/`reallyquit`
surface.

## Evidence and confirmed divergences

Scenarios:

- `cmd/dp-oracle-diff/scenarios/qui-depth.txt`
- `cmd/dp-oracle-diff/scenarios/qui-immortal-depth.txt`

Manifest: `docs/fidelity/depth/qui.tsv` (7 rows)

Focused tests: `pkg/session/qui_depth_test.go`

Both vehicles were GREEN on main-equivalent code at seeds 1, 2, 3, 5, and 8,
with `--show-oracle` confirming the intended mortal refusal and immortal
fall-through paths. No production divergence was confirmed, so this slice adds
only the two C-first scenarios, the manifest, and focused command/refusal
proofs. The unreachable NPC/no-descriptor branch is explicitly excluded from
this player-descriptor command surface. No `src/` or C-oracle file was edited.

## Verification and integration

All local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-qui`

Feature commit: `361d4f3c8` (`test: prove qui depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1098 — merged as `1e7edf3cd`. Hosted lint, security, and test
checks were green; build-and-push and deploy were skipped by repository
workflow conditions. The PR was merged only after required hosted checks were
green.

The qecho feature PR #1096 and handoff PR #1097 remain open because their
checks did not fire after their one permitted exact workflow retries. Purge
handoff PR #1095 and plot handoff PR #1064 likewise remain open for the same
reason; none was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The shared quit boundary and source-order
claim are maintained under R5b/R5c.
