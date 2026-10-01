# DP-1371 E2: lua.bind-set-skill implementation

Fresh origin/main `d053581d7` (merged #1738). Scope is the set-skill
binding row; Zach approved its return repair and clarified that unambiguous
in-batch C-fidelity repairs proceed within the batch. R1, R3, R5e, R5g, R5h.

## Behavior and C authority

`src/scripts.c:1365-1383` checks table/number/number. A valid call retrieves
its character struct userdata (:1372-1375), stores the numbered skill
(:1378; SET_SKILL `src/utils.h:344`), calls affect_total (:1379), and returns
one value (:1382). That top value is the struct userdata, not the percentage.
On an invalid argument, it performs none of those operations and returns the
original stack top. A trailing fourth argument is irrelevant for a valid
call and is the return value for an invalid call. Numeric strings are accepted
by C's lua_isnumber, with fractional values truncated to int.

The repaired bridge pushes the first table's existing struct userdata after
the adapter call. This preserves identity rather than constructing another
handle. Invalid calls keep the original stack. The game adapter and shared
attribute computation from #1737 are unchanged. The existing reader/writer
class audit is in that batch's implementation note. #1738's return-site audit
found no second confirmed missing valid-call handle push.

## Proof boundaries

`lua.bind-set-skill` is now unit-green, tied to three real-engine tests:

- TestLuaSetSkillDepthValid: numbered kick 134/bash 132 (`src/spells.h:179,181`),
  zero/one/many/100 percentages, numeric strings, fractional truncation and
  a trailing argument; exact skill readback, unrelated-skill preservation,
  immortal PC total, exceptional strength, uncapped CHA and permanent bases.
- TestLuaSetSkillDepthInvalid: invalid/nil character, skill and percentage,
  table percentage and trailing stack top; exact returned value and absence
  of skill write or affect_total.
- TestLuaSetSkillDepthModifiers: current tattoo, worn equipment and active
  negative affect participate in the PC rebuild; valid NPC call returns its
  own handle and uses the NPC ceiling 25. This proves the NPC total boundary,
  not a separate NPC player-skill persistence contract.

Effective arithmetic is owned by the shared #1737 repair
(`src/handler.c:350-373`); these tests prove the binding reaches it.
The state proof does not depend on unrelated practice output.
The additional `lua.bind-set-skill-return` oracle-green row explicitly
records the script-observable return contract in the same binding.

The paired healer scenario has eight exercised C blocks: valid player
struct, numeric player struct, valid mobile struct, invalid character top 37,
invalid skill top top, invalid level top not a number, invalid nil top nil,
and invalid table top unchanged. The --show-oracle report confirms each.
Before repair the valid real-engine tests fail on the return assertion;
#1738 retained the paired userdata-versus-number divergence on fresh main.
After repair the targeted census is CLEAN (one passing scenario).

Eight isolated revert triples each fail on an assertion, never compilation
or timeout: valid return, userdata identity, invalid stack top, invalid
no-write/no-total, numeric coercion, skill-number mapping, PC total, NPC total.
The compact table is under `docs/fidelity/depth/evidence/2026-10-01-lua-set-skill/`.
Full logs, mutation script, exact source and development report are retained
at `~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-set-skill-proofs/`.
Targeted wrapper evidence is sibling `dp-1371-p4-set-skill-targeted/`.

No save format, oracle, governing document, divergence ledger, pin or seed
change is included. C-undefined malformed struct tables or out-of-bounds
skill indices are not used as fixtures. No RNG call is added by this binding.

## Final validation

All local gates pass individually: formatting, build, vet, all tests,
clean lint cache/lint, diff check, fidelity depth (5189 cases, 70 blocked),
fidelity units (1154 claims), string census. Full and claims census verdicts
will be recorded after the final source is committed and validated.
