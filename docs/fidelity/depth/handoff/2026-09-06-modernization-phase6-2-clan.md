# Modernization Phase 6.2 — clan info builder

Date: 2026-09-06  
Scope: proven clan renderer output-buffer subset of roadmap item 6.2

## Change

Replaced `doClanInfo`'s repeated string concatenation with a
`strings.Builder`. The renderer still writes the same initial carriage return,
source-order clan rows, sparse no-visible-clan fallback, null-plan marker,
war-status line, and detail fields. `Builder.Reset` preserves the existing
replacement behavior when a mortal has no visible clans.

This is output buffering only. Clan lookup, visibility, ordering, state
updates, and the intentionally C-shaped detail bytes are unchanged (R1/R3/R5e).
No other clan command or unproven string loop is included (R4).

## Fidelity basis

- C detail/list paths: `src/clan.c:723-785`.
- Existing depth matrix: `docs/fidelity/depth/clan.tsv`.
- Go path: `pkg/game/clan_info.go`, `doClanInfo`.

## Verification

Focused unit tests:

```text
/usr/local/go/bin/go test ./pkg/game ./pkg/session       PASS
```

Focused oracle matrix:

```text
clan-depth             PASS
clan-member-depth      PASS
clan-applicant-depth   PASS
clan-plan-depth        PASS
clan-plan-mortal-depth PASS
clan-rename-depth      PASS
```

Oracle summary: 6 scenarios, 6 passed, 0 failed, 0 infrastructure failures,
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
