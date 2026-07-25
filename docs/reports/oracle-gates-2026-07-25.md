# Oracle gates — DP-1215 / DP-1210 / DP-1200 / DP-1205

Run date: 2026-07-25

Oracle:
`/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle`

Post-fix port commit: `c780921c`

All comparisons used `DP_SEED=1`, the harness's fixed clock/time seams, and the
same scenario line sequence on both servers. A GREEN result means the harness
reported `result: no normalized divergence`.

## Target gates

| Issue | Scenario | Pre-fix port commit | Pre-fix result | Post-fix result |
|---|---|---:|---|---|
| DP-1215 | `combat-bash-opener` | `dc6fcd73` (`07d1e4e8^`) | RED | GREEN |
| DP-1215 / DP-1210 | `combat-trip-opener` | `dc6fcd73` (`07d1e4e8^`) | RED | GREEN |
| DP-1215 / DP-1210 | `combat-headbutt-opener` | `dc6fcd73` (`07d1e4e8^`) | RED | GREEN |
| DP-1200 | `object-give-nobody` | `07d1e4e8` (`5b03c3ff^`) | RED | GREEN |
| DP-1205 | `god-harness-smoke` | `d5f2eeb5` (`f89d8296^`) | RED | GREEN |

No target remains RED, so there is no still-divergent full transcript to
include.

## RED evidence

### DP-1215 — bash

On `dc6fcd73`, the first post-bash combat pulse diverged. C stood the trainee
and let it attack in the same round; Go skipped those lines and consumed a
different melee message:

```diff
-A guard trainee scrambles to his feet!
-A guard trainee tries to hit you but you easily avoid the blow.
-You wildly punch at the air, missing a guard trainee.
+You try to hit a guard trainee who easily avoids the blow.
```

The same scenario is GREEN on `c780921c`.

### DP-1215 / DP-1210 — trip

On `dc6fcd73`, the first post-trip pulse showed the same stand-up-round class
plus downstream draw divergence:

```diff
-A guard trainee scrambles to his feet!
-A guard trainee tries to hit you but you easily avoid the blow.
-You wildly punch at the air, missing a guard trainee.
+a guard trainee scrambles to his feet!
+You try to hit a guard trainee who easily avoids the blow.
```

The second pulse also selected different hit/miss messages. The scenario is
GREEN on `c780921c` with no further Go change after `07d1e4e8`.

### DP-1215 / DP-1210 — headbutt

On `dc6fcd73`, the first post-headbutt pulse exposed the capitalization member
of the same class:

```diff
-A guard trainee scrambles to his feet!
+a guard trainee scrambles to his feet!
```

The scenario is GREEN on `c780921c` with no further Go change after
`07d1e4e8`.

### DP-1200 — give to a missing person

On `07d1e4e8`, the probe diverged byte-for-byte:

```diff
-No-one by that name here.
+There doesn't seem to be anyone here by that name.
```

The scenario is GREEN on `c780921c`.

R5e corrections to the brief summary:

- `do_give` reaches C's shared `NOPERSON` constant at `src/config.c:93`, whose
  exact text is `No-one by that name here.\r\n`, not “around.”
- `do_trip` and `do_headbutt` are implemented in `src/new_cmds.c:735-815` and
  `src/new_cmds.c:368-460`; their command registrations are in
  `src/interpreter.c`.

The scenarios document the verified live paths and preconditions.

### DP-1205 — immortal start room

On `d5f2eeb5`, the mortal peer's `look` incorrectly showed the first-player
God in mortal room 8099:

```diff
+Harnessgod the Warrior is standing here.
```

The scenario is GREEN on `c780921c`: the God starts in immortal room 1204 and
is not co-located with the mortal.

## Regression anchors

The following scenarios are GREEN on `c780921c`:

- Combat: `combat-swing`, `combat-backstab-opener`,
  `combat-unlearned-bash-opener`, `combat-death`
- Creation/start rooms: `character-creation`, `newbie-start-room`,
  `look-start-room`

No new normalized divergence was observed (R5c).

## Linear verdicts

- DP-1215 — oracle-confirmed: bash, trip, and headbutt opener scenarios are RED
  on `07d1e4e8^` and GREEN post-fix; all four combat anchors remain GREEN.
- DP-1210 — oracle-confirmed: trip and headbutt cover the stand-up-round class
  and are GREEN post-`07d1e4e8` without another Go change.
- DP-1200 — oracle-confirmed: `object-give-nobody` is RED on `5b03c3ff^` and
  GREEN post-fix with canonical C `NOPERSON` bytes.
- DP-1205 — oracle-confirmed: `god-harness-smoke` is RED on `f89d8296^` and
  GREEN post-fix; all three creation/start-room anchors remain GREEN.
