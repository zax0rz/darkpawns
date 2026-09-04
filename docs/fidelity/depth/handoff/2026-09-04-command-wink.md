# Depth-fidelity handoff — `wink`

Date: 2026-09-04

Feature branch: `glm/depth-wink`

## Queue position and scope

This slice starts from merged main at `3608e164f` after the `wiggle` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `wink` at line 828. No `wink` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:828 */
{ "wink"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, visible-target, and sleeping-target branches, and routes the
authored audiences through `act`. The authored record at
`lib/misc/socials:1004-1012` is:

```text
wink 0 5
Have you got something in your eye?
$n winks suggestively.
You wink suggestively at $N.
$n winks at $N.
$n winks suggestively at you.
No one with that name is present.
You wink at yourself?? -- what are you up to?
$n winks at $mself -- something strange is going on...
```

The command row requires `POS_RESTING`. The C social hide field is 0 and its
minimum victim position is 5 (represented by the legacy Go `HideFlag` field),
while the explicit Go override remains 0. This target-capable record reaches
no-argument, visible player/NPC, named self, missing target, first-token/
trailing-argument, and sleeping-target branches. Shared command position,
`PLR_NOSHOUT`, target lookup, visibility, and Act audience behavior are not
duplicated beyond this slice.

No Go source behavior is expected to change unless the differential run
confirms a divergence. No file under `src/` or `darkpawns-c-oracle/` may be
edited.

