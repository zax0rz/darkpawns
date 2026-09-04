# Depth-fidelity handoff — `yuball`

Date: 2026-09-04

Feature branch: `glm/depth-yuball`

## Queue position and scope

This slice starts from merged main at `b983311d5` after the `yodel` depth
slice. The scheduled worktree prune was completed after the prior tenth slice.
The special-procedure inventory remains exhausted; the designated Phase 2
red/blocked families, blocked clinic vehicles, and the later off-command-table
phases remain queued. Phase 1 concludes with this final remaining social. The
next genuinely unmanifested reachable `do_action` row in `src/interpreter.c`
is `yuball` at line 842. No `yuball` manifest, scenario, or focused
registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remain
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:842 */
{ "yuball"   , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, parses no target when `char_found` is absent, sends the authored
no-argument bytes to the actor, and routes the authored room audience through
`act`. The authored record at `lib/misc/socials:1257-1260` is:

```text
yuball 0 5
You teach everyone how to do the "Yub! Yub!" song and dance.
$n teaches you how to do the "Yub! Yub!" song and dance.
#
```

This is a self-only record despite its C minimum victim position of 5: it has
no `char_found`, `vict_found`, or not-found slot, so typed targets, a named
self, and an unresolved target all remain on the same no-argument actor/room
path. The command row requires `POS_RESTING`; the C social hide field is 0 and
its minimum victim position is 5 (represented by the legacy Go `HideFlag`
field), while the explicit Go override remains 0. Position, `PLR_NOSHOUT`, and
shared Act visibility are delegated rather than duplicated.

No Go source behavior is expected to change unless the differential run
confirms a divergence. No file under `src/` or `darkpawns-c-oracle/` may be
edited.

