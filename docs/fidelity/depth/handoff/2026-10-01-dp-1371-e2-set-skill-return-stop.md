# DP-1371 E2: set-skill return-value stop

Fresh `origin/main` at `8d8702483` (merged #1737). Scope:
`lua.bind-set-skill`. The row remains blocked; this PR is diagnosis only.

## Confirmed reachable divergence (R1, R5e, R5g)

C `lua_set_skill` validates table/number/number, pushes `struct`, obtains
the character userdata, sets the numbered skill, calls affect_total, then
returns one value without removing that userdata
(`src/scripts.c:1365-1383`, especially :1372-1375 and :1382).
Thus a valid call returns the character userdata.
Go `bridgeSetSkill` calls the game adapter then returns one value without
pushing the userdata; the script receives its final skill-level argument.
Its comment incorrectly says that C likewise returns the last argument.

A disposable oncmd healer script calls `set_skill(ch, 134, 37)` and says
the result's type. Full paired output:

```text
C:  An old ssauran healer says, 'set_skill result userdata'
Go: An old ssauran healer says, 'set_skill result number'
```

The real Lua-engine unit uses the same binding over a real world/player.
Kick becomes 37 and copied effective DEX 25 becomes 18, confirming #1737's
prerequisite works; the in-script result-type assertion fails. No practice
report, missing globals or shared-global persistence is used as readback.
Skill 134 is SKILL_KICK (`src/spells.h:181`), with SET_SKILL storage at
`src/utils.h:344`. The source is the unmodified fresh-main implementation,
so this is a reproduced pre-existing binding defect.

```text
oracle-regression: scenarios=1 passed=0 expected=0 unpinnable=0 stale=0 failed=1 infra=0 timed_out=0 unstable=0 elapsed=22.880s started=2026-10-01T08:15:03-0400 finished=2026-10-01T08:15:26-0400 verdict=NOT_CLEAN
```

Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-01/`:
`dp-1371-p4-set-skill-return-main/` holds the wrapper manifest and C dump;
`dp-1371-p4-set-skill-return-reproducer/` holds the exact diagnostic test,
scenario, assertion failure, complete --show-oracle report and commands.
Neither failing fixture remains in the shipped test/scenario corpus.
No pins, expected rows, seeds or governing documents changed.

## Class check (R5c)

All `return 1` sites in `pkg/scripting/bindings_bridge.go` were inventoried
against `src/scripts.c`. The valid-call struct-return siblings already
push handles explicitly: objfrom, objto, extra and follow; spell preserves
its selected top value. Other valid return sites explicitly push a boolean,
number, nil or table. The no-push invalid lua_log branch matches C's
`src/scripts.c:780-793`. This check identifies no second confirmed instance
of set-skill's missing valid-call push. It does not certify every binding.

## Proposed narrow repair

The goal says to stop when a decision is not covered, including a new
divergence, and E2's proposal says to start proof-only and trace reds before
a separate fix. This finding is that stop; no speculative repair is included.

Approve a separate set-skill repair: after a valid binding call, push the
first table's actual struct userdata before returning one. Preserve invalid
argument behavior (no SET_SKILL/total, existing stack return) and numeric
string coercion. Keep the game adapter and shared bounds unchanged.
Then prove numbered skill readback, PC/NPC total boundaries, valid-return
identity/type and invalid-argument controls through the real engine, with
assertion-based revert triples. Retain a paired oracle return probe and
run full and claims censuses for the binding change. Only then mark
`lua.bind-set-skill` proven and replace its stale blocked note.

## Diagnosis branch validation

All required gates exited 0 individually: make fmt, build, vet, all Go
tests, clean lint cache/lint, diff check, fidelity depth, fidelity units
(1151 passing claims), string census. There remain 71 blocked cases.
These gates validate this docs-only proposal, not the failing diagnostic.
No full or claims census is required for this documentation-only change.
