# DeepSeek — round 12: the last draw rows in the lane

Date: 2026-10-21 · Branch: `deepseek/round12` (off `origin/main` `234c8d4ee`, which
has #2126/#2127/#2128) · Rules: R1, R3, R4, R5c, R5e, R5h

Goal: every draw-divergent row from the v0.1.2 release census that is **not**
stop tier is cleared or written up with its owner. Starting point: 36 rows in
`v012-release-41f19a5cd/draw-results.tsv`, 19 of which rounds 10-11 cleared.

**Result: 27 of 36 cleared, 7 stop tier (login fixture, named below), 2 suspected
harness phase (one seed each, 4/5 seeds clean).**

## 1. Fresh first-divergence table (re-run on `234c8d4ee` before any change)

| Scenario | Seed | Fresh verdict | First divergence |
|---|---:|---|---|
| `house-depth` | 1 | MISMATCH | `c=Number(0,3)` vs `go=Number(1,6)` @ 4058, c-step=warmup, go-step=setup |
| `house-depth` | 2 | MISMATCH | same shape @ 4057 |
| `house-depth` | 3 | MISMATCH | same shape @ 4058 |
| `house-depth` | 5 | MISMATCH | same shape @ 4059 |
| `house-depth` | 8 | MISMATCH | same shape @ 4057 |
| `shop-buy-weight-sell-steal` | 1 | **MATCH** | cleared — `entries=36036 stream=67639` |
| `shop-live-inventory-filters` | 1 | **MATCH** | cleared — `entries=30903 stream=62504` |
| `shop-live-inventory-gate` | 1 | **MATCH** | cleared — `entries=30901 stream=62502` |
| `spec-proc-beholder-command` | 1 | MISMATCH | `c=Number(0,2)=0` vs `go=Number(0,18)=1` @ 4121 |
| `spec-proc-key-seller` | 1 | MISMATCH | `c=Number(0,18)=10` vs `go=Number(1,6)=4` @ 4036 |
| `spec-proc-key-seller-non-owner` | 1 | MISMATCH | identical to `spec-proc-key-seller` |
| `spell-charm-shopkeeper-depth` | 1 | MISMATCH | `c=Number(0,900)=662` vs `go=Number(0,3)=2` @ 63584 |
| `mudlog-crash-load-entry` | 3 | **MATCH** | cleared — `entries=47971 stream=81531` |
| `mudlog-crash-load-entry` | 8 | **MATCH** | cleared — `entries=47312 stream=80870` |
| `will-zen-meditation-depth` | 3 | **MATCH** | cleared — `entries=36883 stream=70443` |
| `will-zen-meditation-depth` | 8 | **MATCH** | cleared — `entries=36234 stream=69792` |
| `spec-proc-cityguard` | 5 | MISMATCH | `c=Number(1,20)=13` vs `go=(log ends)` @ 4137 |

Seven rows needed no work: rounds 11's fixes (the mobile race default and the
empty-`dice()` seam) had already cleared the shop trio, both
`mudlog-crash-load-entry` rows and both `will-zen-meditation-depth` rows. Those
are re-verified above, not assumed.

## 2. Fixed this round: `spec-proc-beholder-command` (1 row)

C's `SPECIAL(beholder)` (`src/new_cmds2.c:328-398`) has three arms; the port had
invented a fourth:

| Arm | C | Port before |
|---|---|---|
| `cmd` | cast/recite -> two eyestalk acts, TRUE; else FALSE (:334-343) | matched |
| not fighting | count people, `targets<=1` returns, `number(0,targets)` picks, `number(0,2)` -> sleep/charm/curse, returns **FALSE** (:345-373) | **missing** |
| fighting | scan people, `!number(0,4)` per candidate fighting the beholder, fall back to `FIGHTING(ch)`, `number(0,10)` -> disrupt, else `number(0,5)` -> disintegrate (:377-397) | invented `randN(5)` table of disintegrate/sleep/curse/disrupt with invented room messages, plus an invented `GetHP() < 0` gate |

That is an R4 violation, not just a draw bug: the beholder cast spells C never
taught it and could not charm/sleep/curse at all when idle. The missing
`number(0, targets)` is exactly the row: in the fixture the beholder stands with
two players, so `targets == 2` and C's next draw is `number(0,2)`.

After:

```
seed 1  MATCH entries=4096 stream=4149
seed 2  MATCH entries=4095 stream=4148
seed 3  MATCH entries=4098 stream=4130
seed 5  MATCH entries=4099 stream=4152
seed 8  MATCH entries=4098 stream=4151
```

Tests: `pkg/game/spec_beholder_test.go` pins all three arms' draw bounds through
the `beholderNumber` seam (`scriptBeholderDraws`, the same shape as
`scriptHuntDraws`), including the lone-room gate. Revert control: removing the
peaceful arm fails `TestSpecBeholderPeacefulPulseDraws` with
`draws = 0, want 2: number(0, targets) then number(0, 2)`.

## 3. Stop tier: the login fixture (`house-depth` x5, `spec-proc-key-seller` x2)

Both scenario families give C an **existing** player and give Go a **new** one:

```
[setup:oracle]      Testor          [setup:port]     Testor
                    testpassword                     y
                    <ENTER>                           oraclepass
                    1                             ... M H W K Y <ENTER> 1
```

So C runs `nanny()`'s login path -> `store_to_char` and rolls nothing; Go runs the
full creation nanny and rolls a character:

- `pkg/session/char_creation.go:349` (`case "K"`, hometown) ->
  `game.RollRealAbils` (`pkg/game/character.go:97`) -> 24 x `dprng.Number(1,6)`
  (`pkg/game/character.go:196-211`), then `player.go:454-455` (weight/height) and
  `level.go:347,350` (`do_start` -> `advance_level`, `src/class.c:604-700`).
- C's equivalent creation path is `roll_real_abils` (`src/class.c:379-402`) plus
  `do_start` (`src/class.c:501-590`). A pre-seeded pfile never reaches it.

Measured: `TMPROLL2 session="Testor" stage="hometown" idx=4036` and
`TMPROLL2 session="Keypeer" stage="hometown" idx=4066` in `spec-proc-key-seller`
— Go creates twice where C creates once. `house-depth` is the same with Testor at
4030 and Houseguest at 4058.

The harness seeds the oracle's `Testor` because `house-control 8162 8161 north 1
1 19604` needs owner ID 1 and the Go side has no database; the scenario comment
says so ("the Go fixture maps that owner to its first new Testor (ID 1)"). Fixing
it means sending the same creation answers to the oracle in those two scenarios
(or pre-seeding both sides), which is the harness's call — I did not touch
`cmd/dp-oracle-diff` or `scripts/census.sh`.

## 4. Suspected harness phase: 1 of 5 seeds each

Two rows diverge on exactly one seed, with a phase-boundary signature:

| Scenario | Seed | Verdict on the other four seeds |
|---|---:|---|
| `spell-charm-shopkeeper-depth` | 1: `c=Number(0,900)` (a `hit()` bonus-attack roll, `src/fight.c:1917`) vs `go=Number(0,3)` (the pre-command hide roll, `src/interpreter.c:889`); c-step=setup, go-step=warmup | seeds 2, 3, 5, 8 **MATCH** |
| `spec-proc-cityguard` | 5: `c=Number(1,20)` vs `go=(log ends)` — Go's log stops mid-exchange after a `Dice(1,4)` | seeds 1, 2, 3, 8 **MATCH** |

Neither reproduces across seeds with the same fixture, and both signatures are
positional (a step-phase boundary; a truncated tail mid-combat), so I read them
as pump/log-timing in the harness rather than port behaviour. I did not chase
them further because DP_CLOCK accounting and the harness are stop tier; they go
to the same harness session as the login fixture.

## 5. Final state of all 36 rows

`draw` = `draw-results.tsv` row from `v012-release-41f19a5cd`.

| # | Scenario | Seed | State | Owner / round |
|---:|---|---:|---|---|
| 1 | `carve-depth` | 1 | cleared | r10 pending-body |
| 2 | `house-depth` | 1 | stop tier | login fixture (harness) |
| 3 | `shop-buy-weight-sell-steal` | 1 | cleared | r11 race default |
| 4 | `shop-live-inventory-filters` | 1 | cleared | r11 race default |
| 5 | `shop-live-inventory-gate` | 1 | cleared | r11 race default |
| 6 | `spec-proc-beholder-command` | 1 | **cleared** | **r12 beholder** |
| 7 | `spec-proc-key-seller` | 1 | stop tier | login fixture (harness) |
| 8 | `spec-proc-key-seller-non-owner` | 1 | stop tier | login fixture (harness) |
| 9 | `spell-charm-shopkeeper-depth` | 1 | harness phase | 4/5 seeds MATCH; harness session |
| 10 | `carve-depth` | 2 | cleared | r10 pending-body |
| 11 | `circle-aware-depth` | 2 | cleared | r11 empty-`dice()` |
| 12 | `house-depth` | 2 | stop tier | login fixture (harness) |
| 13 | `carve-depth` | 3 | cleared | r10 pending-body |
| 14 | `checkload-depth` | 3 | cleared | r11 race default |
| 15 | `circle-aware-depth` | 3 | cleared | r11 empty-`dice()` |
| 16 | `house-depth` | 3 | stop tier | login fixture (harness) |
| 17 | `idlist-depth` | 3 | cleared | r11 race default |
| 18 | `mudlog-crash-load-entry` | 3 | cleared | r11 race default |
| 19 | `peek-gates-depth` | 3 | cleared | r11 race default |
| 20 | `sedit-entry-depth` | 3 | cleared | r11 race default |
| 21 | `sedit-session-depth` | 3 | cleared | r11 race default |
| 22 | `will-zen-meditation-depth` | 3 | cleared | r11 race default |
| 23 | `carve-depth` | 5 | cleared | r10 pending-body |
| 24 | `circle-aware-depth` | 5 | cleared | r11 empty-`dice()` |
| 25 | `house-depth` | 5 | stop tier | login fixture (harness) |
| 26 | `spec-proc-cityguard` | 5 | harness phase | 4/5 seeds MATCH; harness session |
| 27 | `carve-depth` | 8 | cleared | r10 pending-body |
| 28 | `checkload-depth` | 8 | cleared | r11 race default |
| 29 | `circle-aware-depth` | 8 | cleared | r11 empty-`dice()` |
| 30 | `house-depth` | 8 | stop tier | login fixture (harness) |
| 31 | `idlist-depth` | 8 | cleared | r11 race default |
| 32 | `mudlog-crash-load-entry` | 8 | cleared | r11 race default |
| 33 | `peek-gates-depth` | 8 | cleared | r11 race default |
| 34 | `sedit-entry-depth` | 8 | cleared | r11 race default |
| 35 | `sedit-session-depth` | 8 | cleared | r11 race default |
| 36 | `will-zen-meditation-depth` | 8 | cleared | r11 race default |

Tally: **27 cleared** (5 in r10, 21 in r11, 1 in r12), **7 stop tier**
(`house-depth` x5, `spec-proc-key-seller` + `-non-owner`), **2 harness phase**
(`spell-charm-shopkeeper-depth` seed 1, `spec-proc-cityguard` seed 5).

## 6. Remaining named gap, not a draw row

`combat.pending-body-second-death-path` is still blocked: `kill` -> `hit` on a
non-peaceful room makes C produce a second corpse and repeat the death line,
while the port stops before applying damage. Recorded as a `blocked` row in
`docs/fidelity/depth/handoff/2026-10-20-deepseek-round10-pending-body.md` for
Zach to schedule; nothing in round 12 touches it.

## Decisions for Zach

- **One PR this round** (`fix(specials): port SPECIAL(beholder) as C writes it`).
  The other 35 rows are cleared, stop tier, or harness phase; no second root
  cause in the lane needed a code change.
- **The login fixture is the harness's to fix.** Both house and key-seller need
  the oracle to run a creation, or the port to log in a pre-seeded player; the
  first is a scenario edit, the second needs a database the port does not have
  in oracle runs.
- **Two rows are classified on 4/5 evidence, not proof.** If you would rather
  treat `spell-charm-shopkeeper-depth` seed 1 and `spec-proc-cityguard` seed 5
  as open lane work, say so and I will instrument the pump boundary next round;
  I stopped because the remaining suspects are DP_CLOCK and the harness.

## 7. Add-on: the immortal `version` line

`do_gen_ps`'s SCMD_VERSION arm printed an invented `Built with: <go version>`
line for immortals (`pkg/session/gen_ps_cmds.go`), where C prints
`Compile Time: %s` (`src/act.informative.c:2172-2174`). The string is stamped at
boot by `load_compile_time()` (`src/db.c:313`, `src/version.c:12-62`), which
assigns `compile_time = time(0)` — boot time, not a build stamp — and formats it
in local time as `<Month> <day>, <year>   <HH>:<MM>`:

```c
sprintf(buf, "%s %d, %d   ", mon, now->tm_mday, now->tm_year+1900);
sprintf(buf, "%s%d:", buf, now->tm_hour);      /* "%s0%d:" under 10 */
sprintf(buf, "%s%d",  buf, now->tm_min);       /* "%s0%d"  under 10 */
```

Those calls pass `buf` as both destination and source, which is undefined
behaviour, so the port renders the intended bytes (R1a) with
`"January 2, 2006   15:04"`. C's zone/DST `strftime` into `buf1` feeds nothing —
`buf1` is never sent — so only `compile_time_str` reaches the player.

**Why the census never saw it:** the only version probe in the corpus was
`mortal-batch12`, and the line is immortal-only. `version-immortal-depth.txt`
creates the first player (crowned, so LVL_IMMORT is inherited) and probes
`version` and `version ignored-argument`; seeds 1, 2, 3, 5, 8 are clean on both
transcript and draw parity.

**The stamp is the real boot clock, deliberately not `DP_FIXED_TIME`.**
Measured with the pinned game clock:

```
C : Compile Time: October 21, 2026   09:27   (load_compile_time -> time(0))
Go: Compile Time: February 11, 2026   14:34   (DP_FIXED_TIME=1770838461)
```

so `loadCompileTime()` reads `compileTimeNow = time.Now`, and a unit test pins
that the pinned game clock does not leak into it. Consequence worth knowing: the
stamp is minute-granular on *each* engine's own boot, so a scenario run whose two
boots straddle a minute would differ on that one line. That exposure is closed by
a normalizer mask Zach approved for exactly this case: a line matching C's stamp
shape is masked to `Compile Time: <COMPILE_TIME>`
(`internal/oraclediff/normalize.go`, with `compile_time_test.go` pinning both the
masking and the shapes it must *not* hide — abbreviated month, wrong separator
widths, seconds, invented label). Mask proof: pinning the port's clock back to
`DP_FIXED_TIME` (Go printing `February 11, 2026   14:34` against C's real
`October 21, 2026   09:27`) now reports `no normalized divergence`, where it was
a divergence before the mask.

Revert control: restoring the invented line reddens the scenario on every seed:

```
-Compile Time: October 21, 2026   09:30
+Built with: go1.26.10
```

String ledger: `c-missing.tsv` already named `Compile Time:`
(`src/act.informative.c:2173`) and `go-only-baseline.json` already reviewed the
invented string as `bug:paraphrase` citing the same C line — the ledger knew the
bug before the census could see it. Both rows go in this change; the 36
pre-existing stale baseline entries are left for a separate
`make string-census-update` (regenerating now would drag ~1136 unrelated lines
into the diff).
