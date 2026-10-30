---
tags: [audit, fidelity, color, ansi, combat, open]
last_updated: 2026-10-29
author: Cline (deepseek) with The Architect
---
# ANSI Color Audit — C vs Go

Triggered by a player report ("color is wrong") with no reproduction. Color had
never been audited as a class: the ordinary oracle normalizer **strips ANSI**
(`internal/oraclediff/normalize.go:99-102`, "color capability is transport
presentation, not game text"), so every green census since the harness existed
has been silent about color. This document is the class audit (R5b/R5c).

Companion data: [`ansi-color-c-inventory.tsv`](ansi-color-c-inventory.tsv) — all
355 C color-emitting sites, machine-extracted with file:line, macro, level and
enclosing function.

---

## 1. The color law

### 1.1 Three emission mechanisms in C

| # | Mechanism | Where | Gate | Port |
|---|---|---|---|---|
| 1 | Level-gated macros `CCxxx(ch, lvl)` / `Qxxx` | `src/screen.h:50-57,61-68` | `clr(ch,lvl)` → `_clrlevel(ch) >= lvl` | Go call sites; no central macro layer |
| 2 | Raw literals `KNRM..KWHT` | `src/screen.h:32-39` | none (caller decides) | inlined `"\x1b[…m"` |
| 3 | `&`-code expansion | `src/comm.c:1289-1345`, applied in `vwrite_to_output` (`src/comm.c:1376-1378`) | `COLOR_ON(desc)` — any nonzero level | `expandTerminalColors` (`pkg/session/terminal_colors.go:33`) |

`_clrlevel(ch)` (`src/screen.h:47-48`) is `PRF_COLOR_1 ? 1 : 0` plus
`PRF_COLOR_2 ? 2 : 0`; NPCs report 0 (`db.c:1281` dummy_mob). Levels are
`C_OFF 0`, `C_SPR 1`, `C_NRM 2`, `C_CMP 3` (`src/screen.h:43-46`). The
in-game `color` command maps off/sparse/normal/complete onto those bits
(`src/act.informative.c:2489-2492`), and answering `Y` to "Do you want ANSI
color" at creation sets **both** bits, i.e. C_CMP (`src/interpreter.c:1992-1998`).

`vt100.h` is a fourth, near-unused path: only `VT_BOLDTEX`/`VT_NORMALT` appear
in code, once (`src/act.informative.c:2796`).

### 1.2 The `&` table is 18 codes, and the port matches

`src/comm.c:1289-1345` walks `"&ndbgcrmywDBGCRMYW"` (18 chars) against a
19-entry `ANSI[]` table. The 19th entry (`"!"`) is **unreachable**: the scan
loop terminates on `OUR[18] == '\0'` and the post-loop `if (OUR[j])` test then
fails, so a trailing `&` stays literal. `expandTerminalColors` has exactly 18
values, so `&!` is unhandled in both engines. Verified char-by-char: the tables
agree, including `&&` → `&` and a lone trailing `&` → `&`.

### 1.3 Where color actually appears

`ansi-color-c-inventory.tsv` groups the 355 sites into 29 functions. The
distribution is not where intuition puts it:

| Level | Sites (top recipients) |
|---|---|
| `ch, C_NRM` | 161 — `do_infobar` alone is 80 |
| `ch, C_CMP` | 96 — `do_score` 22, `make_prompt` 42 |
| `d->character, C_CMP` | 30 |
| `ch, C_SPR` | 25 — `do_who` 21, `do_users` 2, `do_color` 1, `do_gsay` 1 |
| `tch, C_NRM` | 18 — `do_report` |
| `victim/vict, C_SPR` | 2 — **`damage()`'s wounded-band line** |

**Combat is the smallest color surface in the game.** `src/spells.c`,
`src/magic.c`, `src/act.offensive.c`, `src/spec_procs2.c` and
`src/spec_procs3.c` contain **zero** color macros and zero `&` codes. All
combat color is 17 macro sites in `src/fight.c`:

- `damage_message()` — `src/fight.c:1001-1014` (C_CMP)
- `skill_message()` — `src/fight.c:1049-1087` (C_CMP)
- `damage()`'s position switch — `src/fight.c:1596` (C_SPR)

That is why "combat color" reduces to three questions, and two of them were
wrong. See §3.

---

## 2. Method

Two independent passes, because either alone is blind:

1. **Static inventory** (this document + the TSV): every C site, its level and
   audience, matched to the Go function that replaces it. Catches a missing
   macro, a wrong level gate, and a wrong recipient.
2. **Differential proof**: `keep-ansi` oracle scenarios. `NormalizeKeepANSI`
   disables normalizer rule 1 so raw escapes are compared. Only 17 of 1801
   scenarios used it before this audit, and **none of them was a combat
   scenario** — the comm channels, the five OLC editors, `gsay`,
   `terminal-amp-colors` and one `gain_exp` level-up were the entire colored
   proof surface. Two scenarios are added here.

Static alone cannot see an RNG-gated branch; differential alone cannot see a
surface nobody wrote a scenario for. The work queue in §7 is the list of
surfaces where only pass 1 has run.

---

## 3. Findings

### F1 — `damage_message()` emitted no color (FIXED, combat)

**C** (`src/fight.c:995-1015`) brackets the two private lines of an ordinary
weapon hit:

```c
   act(buf, FALSE, ch, NULL, victim, TO_NOTVICT);      /* room: no color */
   send_to_char(CCYEL(ch, C_CMP), ch);
   act(buf, FALSE, ch, NULL, victim, TO_CHAR);
   send_to_char(CCNRM(ch, C_CMP), ch);
   send_to_char(CCRED(victim, C_CMP), victim);
   act(buf, FALSE, ch, NULL, victim, TO_VICT | TO_SLEEP);
   send_to_char(CCNRM(victim, C_CMP), victim);
```

**Go** `DamMessage` (`pkg/combat/fight_core.go`) broadcast the room line and then
called `cbSendToChar` twice with no escapes at all. A player at complete color
saw their own damage line uncolored.

This is *every* damaging melee/weapon hit — the commonest combat byte in the
game. The skill-message path (`skill_message()`) already implemented its
sandwich, which is exactly why the gap survived review: bash/backstab/kick all
looked right, and only the plain weapon swing was wrong.

**Proof** — `cmd/dp-oracle-diff/scenarios/damage-message-color-depth.txt`,
red before the fix at seed 1:

```
-^[[33mYou scratch a guard trainee as you hit him.
-^[[0m
+You scratch a guard trainee as you hit him.
+
```

Green after, at seeds 1, 2, 3, 5 and 8. **Fix**: `pkg/combat/fight_core.go` now
calls a shared `sendColorBracketed` helper, which also models the fact that C's
two `send_to_char` color writes sit *outside* `act()`'s `SENDOK` gate
(`src/comm.c:2481-2484`), so a writing or sleeping recipient still receives the
escapes while the message is suppressed.

**R5h**: `TestDamMessageColorFramingAtCMP` (levels 0/2/3) and
`TestDamMessageColorBracketsSurviveSendOKSuppression` — both observed failing
against the reverted code.

### F2 — `damage()`'s wounded-band line emitted no color (FIXED, combat)

**C** (`src/fight.c:1593-1597`):

```c
     sprintf(buf2, "%sYou wish that your wounds would stop BLEEDING "
         "so much!%s\r\n", CCRED(victim, C_SPR), CCNRM(victim, C_SPR));
     send_to_char(buf2, victim);
```

Two details that matter: the gate is **C_SPR (1)**, not C_CMP, so a player with
a single color bit still sees it; and the reset is inside the same `sprintf`, so
it lands *before* the CRLF.

**Go** emitted the bare literal at three production sites —
`pkg/combat/fight_core.go:514`, `pkg/combat/engine.go:1012`,
`pkg/game/spec_fighter.go:160`. (An existing unit test,
`pkg/game/bad_stats_test.go:197`, asserted the uncolored bytes; it passes only
because its fixture player carries no color flags — an R5h "green that cannot
fail" instance, left correct and unchanged.)

**Proof** — `cmd/dp-oracle-diff/scenarios/bleed-line-color-depth.txt`, red
before the fix at seed 1:

```
-^[[0m^[[31mYou wish that your wounds would stop BLEEDING so much!^[[0m
+^[[0mYou wish that your wounds would stop BLEEDING so much!
```

Green after, at seeds 1, 2, 3, 5 and 8. **Fix**: one exported
`combat.BleedLine(victim)` builds the C string; all three sites call it.
**R5h**: `TestBleedLineColorGatedAtSPR` (levels 0/1/2/3, reset-before-CRLF
asserted byte for byte), `TestHandleSurvivingVictimStateBleedingLineCarriesColor`
(which drives the position switch itself so reverting only that call site still
fails) and `TestReplayCrashLoadBadStatsBleedingLineIsRedAtSparseColor` in
`pkg/game` for the third site. All three were observed failing with the three
call sites reverted to the bare literal.

### F3 — R5g citation drift in the combat color constants (FIXED)

`pkg/combat/skill_messages.go` cited the color macros at `screen.h:26/27/28`.
Those lines are inside `screen.h`'s header comment block; the definitions are
`KNRM` 32, `KRED` 33, `KYEL` 35 and the `CC*` wrappers 50-57. The citations were
written from the *macro name's* expected position, not from a grep (exactly the
PF-032 failure R5g exists to prevent). Corrected, and the constants now name the
level they gate on (`colorLevelSPR`/`colorLevelCMP`) instead of repeating `3`.

### F4 — an invented combat message table is live as a fallback (OPEN, R4)

`pkg/combat/skill_messages.go:30-542` holds a hand-written `skillMessageTable`
whose header comment says it "restores that flavor for Dark Pawns". It is not
from the C corpus. `pkg/session/manager.go:660-663` wires it **only** when
`InitEmbeddedFightMessages` fails to parse the embedded corpus — and that corpus
(`lib/misc/messages`) is byte-identical to the oracle's
(sha256 `42e6fbb4…c5c42df`), so it never fails in practice.

The risk is latent, not live: a parse failure would silently replace every
skill's combat text with invented prose. It also implements no color at all, so
it would reintroduce F1's class on the skill path. Recommended disposition:
delete the fallback and fail loudly, or move it behind a test-only name.
**Not changed here** — it is an R4 decision, not a color fix.

### F5 — color proof coverage was 17 scenarios, and combat was zero (OPEN)

`keep-ansi` scenarios before this audit: 10 OLC editor menus
(`medit`/`redit`/`oedit`/`sedit`/`zedit`, color on and off), 3 `do_gen_comm`
channel levels, `gsay-act-depth`, `terminal-amp-colors`,
`gain-exp-level-up-color-depth`. Nothing else.

Every other colored surface — `do_score` (26 sites), `do_infobox` (80),
`do_who` (21), `do_stat_character` (42), `do_report` (18), `print_group` (16),
`look_at_char`, `list_char_to_char`, `do_exits`, `do_help`, `do_skills`,
`do_where`, `do_review`, `list_skills`, `do_users`, `do_color`, `nanny`,
`gain_exp`, `make_prompt` — has **no** differential color proof. §7 is the
queue.

---

## 4. Per-function inventory and Go counterpart

Levels are as extracted in the TSV. "Proof" is the differential state after
this audit.

| C function | sites | levels | Go site | proof |
|---|---|---|---|---|
| `do_infobar` | 80 | C_NRM | `pkg/session/infobar` | none |
| `do_score` | 26 | C_CMP 22, C_NRM 4 | `pkg/session` score | none |
| `do_who` | 21 | C_SPR 21 | `pkg/session/cmd_info.go` (`whoColorEnabled`, ≥1) | none |
| `do_report` | 18 | C_NRM | `pkg/session/act_informative.go` | none |
| `print_group` | 16 | C_CMP | `pkg/session` group | none |
| `damage_message` | 16 | C_CMP | `pkg/combat/fight_core.go` | **keep-ansi (new)** |
| `do_stat_character` | 42 | C_NRM 36, C_CMP 6 | `pkg/session/wiz_stats.go` | none |
| `make_prompt` | 42 | C_CMP | `pkg/session/terminal_colors.go`, prompt | indirect |
| `do_exits`/`do_auto_exits` | 13 | C_NRM 12, C_CMP 1 | `pkg/game/look.go` (`autoExitsText`) | none |
| `do_gsay` | 6 | C_CMP 4, C_SPR 1, C_NRM 1 | `pkg/game/comm_channel.go` | **keep-ansi** |
| `do_help` | 6 | C_CMP | `pkg/session/pager.go` | none |
| `do_skills` | 6 | C_CMP | `pkg/session` | none |
| `perform_tell` | 4 | C_NRM 2, C_CMP 2 | `pkg/game/comm_channel.go` | **keep-ansi** |
| `do_ctell` | 4 | C_CMP | `pkg/game/comm_channel.go` | **keep-ansi** |
| `do_where` | 4 | C_CMP | `pkg/session` | none |
| `get_char_cols` | 4 | C_NRM | OLC editors | **keep-ansi** (10 scenarios) |
| `do_stat_room` | 16 | C_NRM | `pkg/session/wiz_stats.go` | none |
| `do_stat_object` | 8 | C_NRM | `pkg/session/wiz_stats.go` | none |
| `do_users` | 2 | C_SPR | `pkg/session` who/users | none |
| `do_color` | 2 | C_SPR 1, C_OFF 1 | `pkg/session/act_informative.go` | unit |
| `look_at_char` | 2 | C_CMP | `pkg/game/look.go` | none |
| `list_char_to_char` | 2 | C_CMP | `pkg/game/look.go` | none |
| `damage` (wounded band) | 2 | **C_SPR** | `pkg/combat` `BleedLine` | **keep-ansi (new)** |
| `nanny` | 2 | C_SPR | `pkg/session` login | none |
| `gain_exp` | 2 | C_CMP | `pkg/game` level-up | **keep-ansi** |
| `do_review` | 3 | C_CMP | `pkg/session` | none |
| `list_skills` | 2 | C_CMP | `pkg/session` | none |
| `do_wiznet` | 2 | C_NRM | `pkg/session` | none |
| `mudlog` | 2 | C_NRM | `pkg/game/logging.go` | n/a (log file) |

### Level gates that were checked and are correct

The two Go gate helpers are `observationColors` (≥ 2, C_NRM) and
`whoColorEnabled` (≥ 1, C_SPR). Verified against the C sites they replace:

- `do_who` uses `CCYEL(ch, C_SPR)` at all 21 sites (`src/act.informative.c:1780-1865`);
  `whoColorEnabled` accepts either bit. **Correct.**
- `look_at_room`'s room name and the dark/blind banner use `CCCYN(ch, C_NRM)`
  (`src/act.informative.c:736-765`); `observeRoom` uses `observationColors`. **Correct.**
- The room object/character lists use `CCGRN`/`CCYEL` at `C_NRM`
  (`src/act.informative.c:824,834`); `observeRoom` uses `observationColors`. **Correct.**
- `do_color`'s confirmation is `CCRED(ch, C_SPR)` + `CCNRM(ch, C_OFF)`
  (`src/act.informative.c:2494-2495`) — note the *asymmetric* levels; the Go
  path and `pkg/session/color_confirmation_test.go` match. **Correct.**

No gate mismatch was found outside F1/F2. The two remaining level-sensitive
surfaces with no proof are `do_score`'s mixed C_CMP/C_NRM split and
`do_stat_character`'s 36/6 split — both are §7 rows.

---

## 5. Combat deep dive (the reported area)

All 17 `src/fight.c` color sites, and their disposition:

| C site | level | line | disposition |
|---|---|---|---|
| `damage_message` attacker | C_CMP | 1001, 1006 | **F1** — was missing, now fixed |
| `damage_message` victim | C_CMP | 1009, 1014 | **F1** — was missing, now fixed |
| `skill_message` die/hit/miss attacker | C_CMP | 1049/1052, 1064/1067, 1080/1082 | correct before audit |
| `skill_message` die/hit/miss victim | C_CMP | 1054/1057, 1069/1072, 1084/1087 | correct before audit |
| `damage()` wounded band | **C_SPR** | 1596 | **F2** — was missing, now fixed |

Nothing else in the combat path emits color:

- `src/spells.c`, `src/magic.c`, `src/spell_parser.c`: 0 macro sites, 0 `&` codes.
  Spell text is uncolored in C; the port must not invent color there (R4). Spot
  checked: `pkg/spells` has 0 `\x1b[` in production code.
- `src/act.offensive.c` (bash, kick, trip, …): 0 macro sites, 0 `&` codes. These
  skills colour only through `skill_message`, which the port already matches.
- The `god_msg` arm of `skill_message` is deliberately **uncolored** in C
  (the `send_to_char(CCYEL…)` calls sit only in the die/hit/miss arms) — the
  port models this with a `colored` flag. Verified, correct.

### Ordering and audience

C's `skill_message` writes attacker → victim → room; `damage_message` writes
room → attacker → victim. The port emits the room line first in both. This is
**not** a divergence: `TO_CHAR`, `TO_VICT` and `TO_NOTVICT` are mutually
exclusive, so no single audience's transcript can observe the relative order.

---

## 6. What this audit proves

- The `&`-code expander is a faithful port of `color_expansion`, including the
  dead 19th table entry and the `&&`/trailing-`&` edge cases.
- Combat's entire color surface is three sites; two were wrong and are fixed,
  with keep-ansi oracle proof red-before/green-after and R5h unit tests.
- The level gates on the two high-traffic colored commands (`do_who` C_SPR,
  `look` C_NRM) are correct.
- No Go code invents color in combat (spells, offensive skills).

## 7. Work queue (color proof still owed)

Ordered by player exposure. Each row needs a `keep-ansi` scenario; a row is
"done" when it has one that can fail.

1. `do_score` — 26 sites, mixed C_CMP/C_NRM in one function.
2. `do_infobar` — 80 sites, the densest colored output in the game.
3. `do_who` / `do_users` — 23 sites at C_SPR; the only C_SPR commands a mortal
   sees constantly.
4. `do_stat_character` / `do_stat_room` / `do_stat_object` — 66 sites; also the
   wizard surface where a gate error is least likely to be reported.
5. `do_report` / `print_group` — 34 sites.
6. `do_exits` / `do_auto_exits` / `look_at_char` / `list_char_to_char`.
7. `do_help` / `do_skills` / `do_where` / `do_review` / `list_skills` / `do_wiznet`.
8. `nanny`'s login load-result line (C_SPR).
9. F4 disposition (delete the invented fallback or scope it to tests).

## 8. Proposed rulebook amendment (for Zach — not applied)

This audit found the failure mode, not just two bugs: **a normalizer rule that
is right for transport made an entire player-facing class unprovable, and no
rule said so.** Proposed text, for `docs/fidelity/RULEBOOK.md`, as a new
sub-rule of R1:

> **R1b. Normalization may not erase a player-facing byte class.** A
> normalization rule that removes bytes from the compared transcript is a claim
> that those bytes are transport, not game. Before such a rule is relied on,
> the class it erases needs its own proof mode and at least one scenario per
> colored surface that can fail. The ordinary census strips ANSI
> (`internal/oraclediff/normalize.go`), so a green census says nothing about
> color; colored surfaces are proved with `keep-ansi` fixtures.
> **Taught by** the 2026-10-29 ANSI color audit: `damage_message()`'s C_CMP
> yellow/red sandwich and `damage()`'s C_SPR wounded-band red were absent from
> the port, and no census could see them because only 17 of 1801 scenarios kept
> ANSI and none was a combat scenario.

Incident citation for the amendment log: this audit; the two red oracle runs are
retained at `/tmp/ansi-audit-D1-PROVEN.log` and `/tmp/ansi-audit-d2.log` on the
auditing workstation and should be re-run into `$ORACLE_RUNS_ROOT` before the
amendment lands.

## 9. Evidence

| artifact | content |
|---|---|
| `ansi-color-c-inventory.tsv` | 355 C color sites: file:line, macro, level, function, source line |
| `scenarios/damage-message-color-depth.txt` | keep-ansi proof for F1 (RED before, GREEN at seeds 1,2,3,5,8 after) |
| `scenarios/bleed-line-color-depth.txt` | keep-ansi proof for F2 (RED before, GREEN at seeds 1,2,3,5,8 after) |
| `pkg/combat/damage_message_color_test.go` | R5h unit tests for F1/F2, observed failing on revert |
| `pkg/game/bad_stats_test.go` | `TestReplayCrashLoadBadStatsBleedingLineIsRedAtSparseColor`, the game-layer site |
| `docs/fidelity/depth/fight-output.tsv` | manifest rows `damage-message.color-cmp`, `damage.bleed-line-color` (oracle-green-multiseed) |
| `cmd/dp-oracle-diff/scenarios/*.txt` (17 pre-existing) | the entire prior colored proof surface |

### How to re-run the two proofs

```bash
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario damage-message-color-depth --seed 1 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario bleed-line-color-depth --seed 1 --show-oracle
```

Both are targeted single-scenario probes; neither is a release gate. The full
combined census remains Claude's pre-release step (AGENTS.md).



