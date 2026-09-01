# Depth-fidelity handoff — `policy`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `poke` to `policy`.
The next unclaimed interpreter row is `ponder` at `src/interpreter.c:612`.

Frontier before this slice: 2,823 total; 2,752 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,829 total; 2,758 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:611 */
{ "policy"   , POS_DEAD    , do_gen_ps   , 0, SCMD_POLICIES },
```

`src/act.informative.c:2117-2155` dispatches `SCMD_POLICIES` to
`page_string(ch->desc, policies, 0)`. The `policies` pointer is boot-cached by
`src/db.c:195,324` from `POLICIES_FILE` (`lib/text/policies`,
`src/db.h:59`). The command does not inspect its argument or branch on room,
character, or object state; the only direct behavior surface is the unrestricted
POS_DEAD entry, the complete static six-page text, and the shared pager's
RETURN, quit, and subsequent-command routing.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/policy-depth.txt`

Manifest: `docs/fidelity/depth/policy.tsv` (6 rows)

Test: `pkg/session/policy_depth_test.go`

The seed-1 main vehicle was GREEN with `--show-oracle`. It matched the complete
six-page policy transcript, including the exact authored bytes and pager
prompts, five RETURN navigations to the final page, ignored trailing arguments,
pager quit, and normal `score` dispatch after the pager. The matrix at seeds
`1,2,3,5,8` was GREEN with no normalized divergence.

No confirmed divergence was found, so no production Go behavior was changed.
The round adds only the durable scenario, manifest, and focused gate/registry
test. The existing `cmdPolicy` and shared `PageString` path already matched the
actual C `do_gen_ps` call path and cached source text.

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

Feature branch: `glm/depth-policy`

Feature commit: `f8f70b175` (`test: prove policy depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1072 — merged; main merge commit `3a8971e0f`. Hosted lint,
security, and test checks were green; build-and-push and deploy were skipped by
repository workflow conditions. The PR was merged only after the required
hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
