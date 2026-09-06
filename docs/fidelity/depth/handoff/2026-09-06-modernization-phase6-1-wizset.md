# Modernization Phase 6.1 — `wiz_set` binary flag table

Date: 2026-09-06  
Scope: mechanical `do_set` binary-field subset of roadmap item 6.1

## Change

Moved the direct flag assignments for twelve binary `wiz_set` fields from the
large `applySetField` switch into `setBinaryFieldTable`:

```text
brief, invstart, nosummon, outlaw, roomflag, siteok,
deleted, nowizlist, quest, color, nodelete, chosen
```

The table records whether each field targets C's player flags or preference
flags, including `color`'s two preference bits. `nohassle` and `frozen` remain
explicit because their authority and self-target gates are behaviorally
distinct. `loadroom` remains explicit because it combines a binary flag with
room validation and a numeric state update. No side effects were added to the
existing `deleted` path (R4).

The ordered field/authority/type table remains unchanged. Prefix matching,
binary parsing, acknowledgements, flag synchronization, and special gates
remain on the existing call path (R1/R2/R5e).

## Fidelity basis

- C field order and type declarations: `src/act.wizard.c:2537-2603`.
- C binary dispatch and flag assignments: `src/act.wizard.c:2701-2712` and
  `src/act.wizard.c:2721-2733`, `src/act.wizard.c:2850-2863`,
  `src/act.wizard.c:2885-2921`, `src/act.wizard.c:2941-2967`, and
  `src/act.wizard.c:2991-2993`.
- Existing proofs: `pkg/session/set_depth_test.go` and the `set` oracle depth
  scenarios. This refactor changes no player-facing bytes or state semantics;
  it only replaces repeated direct assignments with a data table (R1/R3/R4).

## Verification

Focused unit test:

```text
/usr/local/go/bin/go test ./pkg/session       PASS
```

Focused oracle matrix:

```text
set-depth             PASS
set-extended-depth   PASS
set-gate-depth       PASS
```

Oracle summary: 3 scenarios, 3 passed, 0 failed, 0 infrastructure failures,
0 unpinnable, 0 timed out.

Full repository gates remain required before merge:

```text
/usr/local/go/bin/go build ./...
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
PATH=/usr/local/go/bin:$PATH golangci-lint run ./...
make fidelity-depth
python3 scripts/gen_expected_divergences.py --check-pins
git diff --check
```
