# Modernization Phase 4 — mechanical handler dedup handoff

Date: 2026-09-05
Branch: `glm/modernize-phase-4-1` (item 4.1 is first)
Base: `origin/main` after PR #1388 merge (`9588f759567dc3b0e526aa2d1ddf43413071b85a`)

## Queue and process

Phase 4 is being executed as seven serial PRs, in roadmap order:

1. 4.1 clan family through `resolveClanForImmortal` (−250)
2. 4.2 skill-command prologue helpers (−275)
3. 4.3 small pairs (−180)
4. 4.4 parameterized channel wrapper (−58)
5. 4.5 table-driven position commands (−50)
6. 4.6 verbatim duplicates plus the `LVL_IMMORT` import-cycle fix (−85)
7. 4.7 script-trigger shared spine (−37)

No item N+1 branch will be created until item N's PR is merged. Each PR must
carry its changed-file list and named proven scenarios, a complete
`make oracle-regression` tally, standard local gates, and hosted CI evidence.
The RED set remains human-only.

## Item 4.1 coverage boundary

The clan family already has depth-proven scenarios in:

- `clan-depth.txt`
- `clan-member-depth.txt`
- `clan-applicant-depth.txt`
- `clan-plan-depth.txt`
- `clan-plan-mortal-depth.txt`
- `clan-rename-depth.txt`

The implementation must remain a pure refactor of the repeated clan context
selection in the handlers named by the roadmap. Any changed file without a
direct mapping to these clan rows is human-merge-only under the amendment.

## Standing fidelity constraints

Apply R1 (player-facing bytes), R3 (draw/order parity), R4 (no invention), and
R5e (verify the reachable call path). Do not edit `src/` or
`darkpawns-c-oracle/`, change save format, touch the RED set, or broaden the
scope beyond the numbered item.
