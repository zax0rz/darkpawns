# Modernization Phase 6.2 — search builder

Date: 2026-09-06  
Scope: proven `DoDetect` output-buffer subset of roadmap item 6.2

## Change

Replaced `DoDetect`'s repeated string concatenation with a
`strings.Builder`. The initial room-check line, failed-roll suffix, exit scan
order, direction-specific wall/ceiling/floor wording, and final message are
unchanged.

This is output buffering only. The C-inclusive RNG draw, case-sensitive
keyword check, secret-room state mutation, and wait-state behavior remain on
the existing call path (R1/R3/R5e). No unproven search behavior was changed
(R4).

## Fidelity basis

- C source path: `src/new_cmds2.c:500-541`.
- Existing depth matrix: `docs/fidelity/depth/search.tsv`.
- Go path: `pkg/game/skills2.go`, `DoDetect`.

## Verification

Focused unit tests:

```text
/usr/local/go/bin/go test ./pkg/game ./pkg/session       PASS
```

Focused oracle matrix:

```text
search-case-depth      PASS
search-depth           PASS
search-failure-depth   PASS
search-gates-depth     PASS
search-secret-depth    PASS
```

Oracle summary: 5 scenarios, 5 passed, 0 failed, 0 infrastructure failures,
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
