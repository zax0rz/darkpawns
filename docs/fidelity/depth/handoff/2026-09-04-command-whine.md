# Depth-fidelity handoff — `whine`

Date: 2026-09-04

Feature branch: `glm/depth-whine`

## Queue position and scope

This slice starts from merged main at `c8a57e9ac` after the `whimper` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `whine` at line 822. No `whine` manifest, scenario,
or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remain
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:822 */
{ "whine"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, parses no target when `char_found` is absent, sends the authored
no-argument bytes to the actor, and routes the authored room audience through
`act`. The authored record at `lib/misc/socials:989-992` is:

```text
whine 0 0
You whine pitifully.
$n whines pitifully about the whole situation.
#
```

This is a self-only record: typed targets, a named self, and an unresolved
target all remain on the same no-argument actor/room path because there is no
`char_found`, `vict_found`, or not-found slot. The command row requires
`POS_RESTING`; the C social hide field and minimum victim position are both 0
(represented by the legacy Go `MinLevel` and `HideFlag` fields). Position,
`PLR_NOSHOUT`, and shared Act visibility are delegated rather than duplicated.

No Go source behavior is expected to change unless the differential run
confirms a divergence. No file under `src/` or `darkpawns-c-oracle/` may be
edited.

