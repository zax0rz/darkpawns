# Depth-fidelity handoff — `whimper`

Date: 2026-09-04

Feature branch: `glm/depth-whimper`

## Queue position and scope

This slice starts from merged main at `4451e22e8` after the `whap` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `whimper` at line 816. No `whimper` manifest,
scenario, or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remain
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:816 */
{ "whimper"  , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, parses no target when `char_found` is absent, sends the authored
no-argument bytes to the actor, and routes the authored room audience through
`act`. The authored record at `lib/misc/socials:984-987` is:

```text
whimper 0 0
You whimper cowardly.
$n whimpers in the corner, what a coward.
#
```

This is a self-only record: typed targets, a named self, and an unresolved
target all remain on the same no-argument actor/room path because there is no
`char_found`, `vict_found`, or not-found slot. The command row requires
`POS_RESTING`; the C social hide field and minimum victim position are both 0
(represented by the legacy Go `MinLevel` and `HideFlag` fields). Position,
`PLR_NOSHOUT`, and shared Act visibility are delegated rather than duplicated.

## Planned proof

Add a focused unit test for the C command gate, social metadata, and all three
parsed message slots. Add a deterministic oracle scenario with an actor and a
named peer, probing no argument, typed peer, missing target, and named self;
the scenario will carry D1-D3 annotations and run at seeds 1, 2, 3, 5, and 8,
with `--show-oracle` at seed 1. The manifest will record the self-only branch
as oracle-green-multiseed and the shared position, `PLR_NOSHOUT`, and
visibility vehicles as delegated.

No Go source behavior is expected to change unless the pre-fix differential
run confirms a divergence. No file under `src/` or `darkpawns-c-oracle/` may
be edited.

