# DP-1212 RNG divergence: paired-trace root-cause report

**Date:** 2026-07-24  
**Status:** Root cause proven; implementation not changed  
**Investigated branch:** `glm/god-advancelevel-draws` at `304b4d3d`  
**Comparison baseline:** pre-#469 commit `7e3c36a4`  
**Oracle:** `darkpawns-c-oracle`, `DP_SEED=1`, `DP_CLOCK=1`

## Executive verdict

PR #469 found and fixes a real R3d violation: Go created the first-player God
through a shared constructor that unconditionally consumed two
`AdvanceLevel()` draws which C skips for an already-`LVL_IMPL` God.

The apparent evidence for a second, still-unlocated pre-opener offset was
misleading. Paired C/Go tracing proves that, with #469 applied, both processes
consume the same ranges and produce the same values from boot through the kick
success roll at draw 4,099. This rules out, for this scenario:

- the four mortal movement commands;
- world population/reset draws;
- God creation after the #469 correction;
- mortal creation and `AdvanceLevel`;
- a dynamic pre-opener discrepancy.

The first divergence is inside the successful kick handler:

```text
draw       C oracle                         Go port
4099       number(1,101) = 44               number(1,101) = 44
4100       skill_message dice(1,2) = 1      improve_skill number(1,200) = 40
4101       improve_skill (1,200) = 194      skill_message dice(1,2) = 2
```

C calls `damage()`—and therefore `skill_message()`—before
`improve_skill()`. Go calls `improveSkill()` inside `DoKick`, then returns a
`SkillResult`; `sendSkillResult()` emits the deferred `SkillMessage` afterward.
The two draws are reversed. This violates R3b (operation ordering), and the
different `dice(1,2)` value selects a different player-facing message variant
(R1).

A second, independent issue explains the empty combat pulses: the level-one
successful kick deals `GET_LEVEL(ch) >> 1 == 0` damage. The successful Go
`SkillResult` lacks `StartCombat: true`, so `sendSkillResult()` neither calls
`DoSpellDamage` nor explicitly enrolls the combatants. C's `damage(..., 0,
SKILL_KICK)` still calls `set_fighting`.

The correct disposition is:

1. Keep #469; it is necessary and proven.
2. Do not search for another pre-opener offset in this scenario.
3. Fix the deferred-message/eager-improvement ordering class.
4. Fix successful-skill combat enrollment.
5. Run successful-path oracle gates for all five combat skills, checking raw
   roll values and draw order as required by R5a—not only normalized bytes.

## Governing fidelity rules

- **R1:** player-facing bytes are law. The selected fight-message variant must
  match C.
- **R3a:** draw counts and ranges must match C.
- **R3b:** operation order is behavior. Matching the number of draws is
  insufficient if their consumers are reversed.
- **R3d:** conditional draws must use the same gates. #469 fixes this for the
  first-player God.
- **R5a:** a byte-green random outcome is not proof that the underlying rolls
  match.
- **R5c:** once one handler has the ordering defect, audit the whole class.
- **R5e:** the finding must be demonstrated on the live call path. The paired
  trace and `combat-kick-opener` do so.

## Original symptom and working hypothesis

DP-1212 began as a family of seeded combat-skill outcome flips:

- bash;
- trip;
- headbutt;
- kick;
- backstab.

PR #469 removed two phantom Go draws from first-player God creation. That made
previously red scenarios greener, but regressed the previously byte-green
`combat-kick-opener` scenario:

```diff
-You miss your kick at a guard trainee's groin, much to his relief...
+Your beautiful full-circle kick misses a guard trainee by a mile.
```

The first interpretation was that another offset elsewhere had been
compensating for the God's `+2`, because a correct alignment fix should not
normally turn a green scenario red.

That interpretation treated the scenario's old normalized-byte green as proof
of equal control flow. R5a says it is not. The trace shows that the old run was
green by coincidence: C and Go used different draw indices, took different
branches, and happened to select the same final message variant.

## Method

Temporary, environment-gated instrumentation was added in disposable working
state:

- Go: trace every process-wide `dprng` draw as
  `(index, from, to, result)`;
- C: copy the read-only oracle to `/private/tmp`, instrument `uniform()` and
  `number()`, and build the copied oracle;
- run `combat-kick-opener` against the instrumented pair;
- compare #469 against the pre-#469 parent;
- remove the Go instrumentation from the repository checkout.

The production oracle source was not edited. The trace implementation was not
retained in this branch.

Primary reproduction command:

```bash
DP_ORACLE_BIN=/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario combat-kick-opener
```

## Empirical results

### 1. Pre-#469 remains byte-green

At `7e3c36a4`, without the God creation correction:

```text
scenario: combat-kick-opener
result: no normalized divergence
```

The relevant draws are not aligned:

```text
C oracle:
4099  number(1,101) = 44   # kick roll
4100  number(1,2)   = 1    # fight-message variant
4101  number(1,200) = 194  # improve_skill gate

Go before #469:
4101  number(1,101) = 98   # kick roll, shifted by the phantom God +2
4102  number(1,2)   = 1    # fight-message variant
```

C's roll enters kick's success branch. Go's shifted roll enters the failure
branch, which does not call `improveSkill`. Both message-variant draws happen
to return `1`, so the player-facing opener is identical despite different
stream positions and control flow.

This is an exact example of the failure mode described by R5a.

### 2. #469 aligns the entire pre-opener stream

At `304b4d3d`, the paired traces match through draw 4,099:

```text
             C range/result        Go range/result
4087         120..180 = 127        120..180 = 127
4088         160..200 = 184        160..200 = 184
4089         11..14 = 14           11..14 = 14
4090         1..4 = 3              1..4 = 3
4091         0..18 = 17            0..18 = 17
4092         0..15 = 12            0..15 = 12
4093         0..3 = 0              0..3 = 0
4094         0..3 = 0              0..3 = 0
4095         0..3 = 1              0..3 = 1
4096         0..3 = 1              0..3 = 1
4097         0..3 = 2              0..3 = 2
4098         0..3 = 0              0..3 = 0
4099         1..101 = 44           1..101 = 44
```

Because this is a whole-process trace, it includes:

- boot and world loading;
- zone resets and mob/object creation;
- first-player God creation;
- mortal creation and level-one advancement;
- the mortal's `north`, `east`, `south`, `east` movement;
- setup/warmup activity;
- the kick roll itself.

The static C/Go movement audit agrees with the runtime evidence: ordinary
directional movement on this path consumes no random values.

### 3. The first post-#469 divergence is draw 4,100

```text
C oracle:
4099  number(1,101) = 44
4100  number(1,2)   = 1
4101  number(1,200) = 194

Go with #469:
4099  number(1,101) = 44
4100  number(1,200) = 40
4101  number(1,2)   = 2
```

The first mismatched range—not merely the first mismatched result—is at draw
4,100. This proves the defect is consumer ordering:

- C consumes draw 4,100 for `skill_message`;
- Go consumes draw 4,100 for `improveSkill`;
- C consumes draw 4,101 for `improve_skill`;
- Go consumes draw 4,101 for `SkillMessage`.

## Exact live call paths

### C kick success

`src/act.offensive.c:622-633`:

```c
percent = ((7 - (GET_AC(vict) / 10)) << 1) + number(1, 101);

if (percent > prob)
  damage(ch, vict, 0, SKILL_KICK);
else
{
  damage(ch, vict, GET_LEVEL(ch) >> 1, SKILL_KICK);
  improve_skill(ch, SKILL_KICK);
}
```

`damage()` calls `skill_message()` for the skill attack type.
`skill_message()` calls:

```c
nr = dice(1, fight_messages[i].number_of_attacks);
```

Therefore C success order is:

```text
skill roll -> damage/skill_message draw -> improve_skill draw
```

At level one, `GET_LEVEL(ch) >> 1` is zero. C still took the success branch,
but `skill_message(dam == 0, ...)` correctly selects a miss-message variant.
This is why the observed text says "miss" even though the skill check passed.

### Go kick success

`pkg/game/skill_combat.go:249-283`:

```go
percent := ((7 - (target.GetAC() / 10)) * 2) + dprng.Number(1, 101)
// ...
dam := ch.GetLevel() >> 1
improveSkill(ch, SkillKick)

return SkillResult{
    Success:      true,
    Damage:       dam,
    SkillMsgType: SkillKickNum,
}
```

`pkg/command/skill_commands.go:1511-1536` later processes the result:

```go
eng.SkillMessage(...)
// ...
s.GetWorld().DoSpellDamage(...)
```

Therefore Go success order is:

```text
skill roll -> improveSkill draw -> deferred SkillMessage draw
```

The handler and result-sender split makes the C order impossible while
`improveSkill` remains eager inside `DoKick`.

## Why combat pulses disappear

`sendSkillResult()` applies positive damage through `DoSpellDamage`, which
starts combat. For zero-damage results it starts combat only when
`result.StartCombat` is true.

Kick's success result has:

```go
Damage: ch.GetLevel() >> 1 // zero at level one
```

but omits:

```go
StartCombat: true
```

Consequently:

- `Damage > 0` is false, so `DoSpellDamage` is skipped;
- `StartCombat` is false, so explicit enrollment is skipped;
- the following controlled combat pulses are empty in Go.

C calls `damage(ch, vict, 0, SKILL_KICK)` and enrolls both combatants
regardless of the zero damage.

## R5c class audit: five opener skills

The defect is broader than kick. C's `damage()`/`hit()` performs the
fight-message draw before returning to the command handler, where
`improve_skill()` then runs. Go computes and improves inside `Do*`, but defers
the fight-message draw to `sendSkillResult()`.

| Skill | C successful-path order | Current Go successful-path order | Verdict |
|---|---|---|---|
| backstab | `hit`/`damage`/message, then one `improve_skill` | hit/damage calculation, eager `improveSkill`, deferred message | R3b ordering bug when the initial skill roll succeeds |
| bash | `damage`/message, then `improve_skill` | eager `improveSkill`; currently hardcoded result messages | Ordering class plus the separate bash message-reroute gap |
| kick | `damage`/message, then `improve_skill` | eager `improveSkill`, deferred message | Runtime-proven first divergence |
| trip | `damage`/message, then `improve_skill` | eager `improveSkill`, deferred message | Same ordering class |
| headbutt | `damage`/message, then two `improve_skill` calls | two eager `improveSkill` calls, deferred message | Same ordering class, twice |

This finding refines `docs/reports/improve-skill-callsite-audit.md`. That audit
compared the local placement and gating of `improve_skill` calls, but its kick
verdict did not account for the random `skill_message` draw hidden inside C's
preceding `damage()` call and deferred across the Go `Do*`/`sendSkillResult`
boundary. The paired live trace is the stronger evidence.

### Backstab's current green does not cover this class

The `combat-backstab-opener` scenario grants 75% skill. In the traced run its
initial `number(1,101)` returned `84`, so it took the early skill-failure path:

- C and Go made no improvement draw;
- both immediately selected the miss message;
- their traces remained identical.

That green does not exercise backstab's successful skill-roll path. A
successful-path fixture is required.

## Successful-result combat enrollment audit

The successful `SkillResult` literals for all five opener skills omit
`StartCombat: true`:

- backstab;
- bash;
- kick;
- trip;
- headbutt.

Positive damage currently masks the omission because `DoSpellDamage` starts
combat. Kick at level one exposes it because successful damage truncates to
zero. Setting `StartCombat: true` on successful results is faithful and safe
with the existing sender: explicit enrollment is only attempted when
`Damage <= 0`, avoiding a second `StartCombat` call on positive-damage paths.

## Recommended implementation shape

Do not fix this by burning an extra draw or manually swapping random calls.
The side effects and their ordering must match C.

The result pipeline needs to represent deferred skill improvement:

1. `DoBackstab`, `DoBash`, `DoKick`, `DoTrip`, and `DoHeadbutt` should not call
   `improveSkill` eagerly on their successful paths.
2. Their `SkillResult` should carry the requested post-damage improvement
   operation(s).
3. `sendSkillResult` should:
   1. execute the C-equivalent skill message/damage operation;
   2. execute the deferred improvement operation(s);
   3. continue with position and wait-state effects in C order.
4. The representation must support two improvement calls for headbutt.
5. Improvement must use the real `improveSkill` logic, including optional
   player-facing "improves" output; consuming dummy draws would violate R1/R4.
6. Successful results should set `StartCombat: true`.
7. Bash must first/also use its C `skill_message` set rather than hardcoded Go
   messages; otherwise its message draw cannot be ordered faithfully.

The precise API is an implementation choice. A field such as a list of
deferred skill names on `SkillResult` is one plausible design, but the required
contract is the C operation order, not that particular representation.

## Tests that must change or be added

### Correct misleading unit coverage

Existing tests often call `Do*` and `SkillMessage` separately. That proves draw
counts but can hide their actual production ordering through
`sendSkillResult`.

In particular, the current headbutt hit draw-order test explicitly documents
and asserts:

```text
improve draws -> SkillMessage dice
```

That is the current Go order, not the C order. It must be reversed.

### Add pipeline-level draw-order tests

For each successful path, test through the real result-sending pipeline:

```text
skill roll
optional to-hit/damage dice
skill_message dice
improve_skill number(1,200)
optional improve_skill number(1,3)
```

Headbutt must then perform the second improvement gate/draw.

Tests should assert the next shared-stream value after the full operation, not
only messages or hit/miss.

### Add successful oracle fixtures

The 75% fixtures do not guarantee the class is exercised. Add or temporarily
use fixtures/seeds that prove:

- initial skill check succeeds;
- backstab's subsequent to-hit miss and hit paths are both covered;
- kick's level-one zero-damage success enters combat;
- trip success;
- headbutt success and both improvements;
- bash success after its `skill_message` reroute.

## Acceptance gate

Do not merge #469 alone. Merge it with the ordering/enrollment fix only after:

1. `combat-kick-opener` is normalized-byte green on the #469 stream.
2. Kick's raw roll is `44` on both processes.
3. Kick's message draw occurs before its improvement gate on both processes.
4. Kick's controlled pulses contain combat on both processes.
5. Successful-path backstab, bash, trip, and headbutt scenarios are green.
6. The five-skill trace has matching `(range, result)` sequences through the
   opener and its immediate combat effects.
7. The standard gates pass:

```bash
make fmt
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

## Final conclusion

PR #469 did not regress a truly aligned kick scenario. It removed a real
first-player-God `+2` violation and thereby exposed a second fidelity bug of a
different kind:

```text
C:  message draw -> improvement draw
Go: improvement draw -> message draw
```

Before #469, the God offset moved Go onto a different skill branch and a later
message draw happened to select the same text as C. After #469, the shared
stream reaches the same skill roll and branch, making the local ordering bug
visible.

The investigation therefore closes the remaining "unknown offset" question for
`combat-kick-opener`: no additional pre-opener offset exists on the traced
path. The remaining work is a proven result-pipeline ordering fix, the
successful-result `StartCombat` fix, and R5c coverage of all five successful
skill paths.
