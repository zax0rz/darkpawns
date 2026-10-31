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
| `do_report` | 18 | C_NRM | `pkg/game/other_character.go` (`doReport`) | **keep-ansi (new)** |
| `print_group` | 16 | C_CMP | `pkg/session/cmd_group.go` (`printGroup`) | **keep-ansi (new)** |
| `damage_message` | 16 | C_CMP | `pkg/combat/fight_core.go` | **keep-ansi (new)** |
| `do_stat_character` | 42 | C_NRM 36, C_CMP 6 | `pkg/session/wiz_stats.go` | **keep-ansi (new)** |
| `make_prompt` | 42 | C_CMP | `pkg/session/terminal_colors.go`, prompt | indirect |
| `do_exits`/`do_auto_exits` | 13 | C_NRM 12, C_CMP 1 | `pkg/game/look.go` (`autoExitsText`) | **keep-ansi (new)** |
| `do_gsay` | 6 | C_CMP 4, C_SPR 1, C_NRM 1 | `pkg/game/comm_channel.go` | **keep-ansi** |
| `do_help` | 6 | C_CMP | `pkg/session/pager.go` | none |
| `do_skills` | 6 | C_CMP | `pkg/session` | none |
| `perform_tell` | 4 | C_NRM 2, C_CMP 2 | `pkg/game/comm_channel.go` | **keep-ansi** |
| `do_ctell` | 4 | C_CMP | `pkg/game/comm_channel.go` | **keep-ansi** |
| `do_where` | 4 | C_CMP | `pkg/session` | none |
| `get_char_cols` | 4 | C_NRM | OLC editors | **keep-ansi** (10 scenarios) |
| `do_stat_room` | 16 | C_NRM | `pkg/session/wiz_stats.go` | **keep-ansi (new)** |
| `do_stat_object` | 8 | C_NRM | `pkg/session/wiz_stats.go` | **keep-ansi (new)** |
| `do_users` | 2 | C_SPR | `pkg/session` who/users | none |
| `do_color` | 2 | C_SPR 1, C_OFF 1 | `pkg/session/act_informative.go` | unit |
| `look_at_char` | 2 | C_CMP | `pkg/game/look.go` | **keep-ansi (new)** |
| `list_char_to_char` | 2 | C_CMP | dead in C (§15) | n/a — unreachable |
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

Companion class audit for the same surface:
[`skills-spells-attacktype-audit.md`](skills-spells-attacktype-audit.md) — the
attack type every skill passes to `damage()` was wrong for six skills and
changed their corpse wording.

Ordered by player exposure. Each row needs a `keep-ansi` scenario; a row is
"done" when it has one that can fail.

1. ~~`do_score` — 26 sites, mixed C_CMP/C_NRM in one function.~~ **Done —
   `score-color-levels-depth`.** The real surface is 11 C_CMP field pairs
   (`src/act.informative.c:1184-1275`), and the port emitted no colour at all.
2. `do_infobar` — 80 sites, the densest colored output in the game. **Not
   coverable by `keep-ansi` alone:** the sites live in the `IB_*` renderers,
   which draw with `VT_CURSPOS` cursor addressing from the prompt, so their
   escapes are interleaved with cursor moves that a transcript comparison cannot
   isolate.
3. ~~`do_who` / `do_users` — 23 sites at C_SPR; the only C_SPR commands a mortal
   sees constantly.~~ **`do_who` done — `who-color-levels-depth` is green at all
   four levels.** The port replaced C's literal `&r`/`&y`/`&n` codes with a
   conditional ANSI escape, which differs at colour off. `do_users` is still owed.
4. ~~`do_stat_character` / `do_stat_room` / `do_stat_object` — 66 sites; also the
   wizard surface where a gate error is least likely to be reported.~~
   **Done — `wizard-stat-color-depth`, §11.** It also found two text-level
   inventions the normalizer had been hiding (**F7**, **F8**).
5. ~~`do_report` / `print_group` — 34 sites.~~ **Done — `report-color-levels-depth`
   and `group-color-levels-depth`, §12.** `do_report` was already correct but had
   no proof that could fail on colour; `print_group` emitted no colour at all.
6. ~~`do_exits` / `do_auto_exits` — 13 sites; `look_at_char` /
   `list_char_to_char` were **done with F6** (#2244).~~ **Done —
   `exits-auto-exits-color-depth`, §13.** `do_exits` itself is uncoloured; all 13
   sites are in `do_auto_exits`. §13 also records **F10** (the harness cannot
   build a closed door), **F11** (`exits.closed-filter` claims a filter it does
   not build) and **F12** (the character list gains a leading space).
   `look_at_char`'s tattoo sites are **done — `look-char-color-depth`, §15**,
   and `list_char_to_char`'s are **dead in C** (§15).
7. `do_help` / `do_skills` / `do_where` / `do_review` / `list_skills` / `do_wiznet`.
8. `nanny`'s login load-result line (C_SPR).
9. F4 disposition (delete the invented fallback or scope it to tests).

**Inventory defect found while building these proofs.** `ansi-color-c-inventory.tsv`
attributes each site to the last function-like line before it, and that pattern
only matched lowercase signatures, so it filed the capitalised `IB_*` infobar
renderers under `do_infobar`, and gave `do_score` 26 sites when four of them
(`src/act.informative.c:1406,1441`, `CCCYN` at `C_NRM`) belong to a later
function. `do_score`'s real surface is the 11 C_CMP field pairs in
`src/act.informative.c:1184-1275`; `do_stat_character`'s is
`src/act.wizard.c:746-968`. Re-derive the table with a corrected pattern before
trusting its per-function counts.

**`stat`'s fix was subtler than `score`'s.** C's `sprintbit` leaves a trailing
space after each flag list and the port's `wizardPlayerFlags` /
`wizardPreferenceFlags` / `wizardAffectFlags` trimmed it, so wrapping those lines
in colour put the space on the wrong side of the reset. That is **F7 in §11**,
fixed with the whole stat surface in #2247.

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
## 10. Fireball raw-byte follow-up (2026-10-30)

Egon's report pointed at fireball. Its text in `lib/misc/messages` (set 26) carries
no colour codes, so every escape around it comes from `skill_message`'s delivery.
Eight `keep-ansi` scenarios now cover it — `fireball-color-{off,sparse,normal,complete}-hit`,
plus `{off,complete}-{miss,kill}` — with the caster setting its own level by
`color` and the peers taking theirs from the creation ANSI answer, so the
transcript also proves the two gates are per-recipient (at sparse/normal the
caster's line is plain while the victim's is still red).

**Result: the fireball colour bytes match C exactly.** Telnet: targeted census
`deepseek-fireball-color-telnet`, 8 scenarios, `passed=8 failed=0
draw_divergent=0 verdict=CLEAN`. WebSocket: all eight run through
`--go-transport ws` with the same verdicts. So the escapes are ordered as C
orders them, the reset lands where C lands it, and Go does not split a C write
across frames in a way the browser client shows differently.

**F6 — the room-contents list colours each entry instead of the list (OPEN).**
The one divergence the fireball work did surface is not in the fireball path.
C's `look_at_room` brackets the two room-content lists once each
(`src/act.informative.c:823-833`):

```c
  send_to_char(CCGRN(ch, C_NRM), ch);      /* once, before the object list  */
  list_obj_to_char(world[ch->in_room].contents, ch, 20, FALSE);
  send_to_char(CCYEL(ch, C_NRM), ch);      /* once, before the character list */
  list_char_to_char(world[ch->in_room].people, ch);
  send_to_char(CCNRM(ch, C_NRM), ch);      /* once, after it                 */
```

`pkg/game/look.go`'s `observeRoom` wraps **each line** (`green+line+normal`,
`yellow+line+normal`), so it opens and closes the colour per entry and never
emits C's trailing bare reset. A kill-then-look transcript, identical on telnet
and WebSocket:

```
 C: ^[[32mThe charred corpse of FireVictim is lying here, still smoking.
    ^[[33mFireWatch the Warrior is standing here.
    ^[[0m
Go: ^[[32mThe charred corpse of FireVictim is lying here, still smoking.^[[0m
    ^[[33mFireWatch the Warrior is standing here.^[[0m
```

The corpse **text** is right — C's `The charred corpse of %s is lying here, still
smoking.` (`src/fight.c:286-290`) is what both engines print. What differs is the
colour framing around the list, and it affects every room look that has objects
or characters in it, not just fireball kills.

Fixing it needs a no-line-ending write on the observation path: `Raw` messages
currently go through `Player.SendMessage`, which adds the line ending C's
`send_to_char(CCGRN(ch, C_NRM), ch)` does not have. That is new plumbing through
`ObservationMessage` → `RenderObservationMessages` → the session frame, on the
game's most-used output path, so it wants its own change and vehicle rather than
a rushed edit. The kill-then-look vehicle above is red on `main` and is the
starting point.

This also closes a row from §7's work queue: `look_at_char` / `list_char_to_char`
were listed as having no colour proof, and now have one — which is how the
divergence was found.

DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario damage-message-color-depth --seed 1 --show-oracle
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario bleed-line-color-depth --seed 1 --show-oracle
```

Both are targeted single-scenario probes; neither is a release gate. The full
combined census remains Claude's pre-release step (AGENTS.md).




## 11. The stat surface (2026-10-30)

Queue item 4: `do_stat_character` (42 sites), `do_stat_room` (16) and
`do_stat_object` (8) — 66 sites in `pkg/session/wiz_stats.go`, the largest
coloured surface after `do_infobar`, and the wizard command where a wrong gate is
least likely to be reported. **All 66 now match C**, proved by
`wizard-stat-color-depth`, which walks the viewer through every level
(`color sparse` → `normal` → `complete` → `off`) and re-renders all three report
bodies at each, raw escapes intact.

The gates are C's own. Everything is C_NRM (`clr(ch, C_NRM)` is `_clrlevel >= 2`,
so PRF_COLOR_2 alone is enough) except the kills/pkills/deaths row, the one C_CMP
site (`src/act.wizard.c:831-835`), which needs both bits. `color normal` is
therefore the discriminating level: the rest of the report is coloured and that
row is plain. Before this work `wiz_stats.go` emitted **no** colour at all in
`do_stat_character` and `do_stat_room`, and only the room/object extra-description
and contents lists in `do_stat_object`.

Two things came out of it that are not about colour at all.

**F7 — the stat flag lists dropped `sprintbitarray`'s trailing space (FIXED, R4).**
C's `sprintbitarray` appends `"NAME "` per set bit and `"NOBITS "` when none are
set, and never trims (`src/utils.c:825-843`); `sprintbit` behaves the same way
(`src/utils.c:243-267`). C prints the result straight into the report
(`src/act.wizard.c:879, 885, 888, 894, 968`), so the trailing space is part of the
line. The port's `wiz_stats.go` wrapped nine call sites in `strings.TrimSpace`,
which C does not do:

```
 C: NPC flags: ^[[36mSCAVENGER ISNPC AGGR STAY-ZONE ^[[0m
Go: NPC flags: ^[[36mSCAVENGER ISNPC AGGR STAY-ZONE^[[0m
```

**Why no census had ever seen it.** Uncoloured, the space sits directly before
the CRLF, and the oracle normalizer strips trailing whitespace — both engines
normalized to the same line, so `stat.player-success` was green on a divergence
that had been there since the port was written. Colour moves `CCNRM` after the
space, which puts it **mid-line**, where the trailing-whitespace rule cannot reach
it. This is the same failure shape as R1b: a normalizer rule that is right for its
own purpose silently made a byte class unprovable. Fix: delete the nine
`TrimSpace` wrappers.

**F8 — the PLR line fabricated a `CSH` marker (FIXED, R4).** `wizardPlayerFlags`
masked the PLR half of the merged flag word and then, when that half was empty but
any other bit was set, **invented** `PLR_CRASH` to "preserve the visible CSH
marker". C prints `PLR_FLAGS(k)` verbatim (`src/act.wizard.c:884-886`), so a player
whose only flags are the two colour preferences reads `NOBITS`, not `CSH`:

```
 C: PLR: ^[[36mNOBITS ^[[0m
Go: PLR: ^[[36mCSH ^[[0m
```

The fallback is not needed for what it claimed: the runtime already models
PLR_CRASH as bit 6 of that half and sets it through `MarkCrashNeeded`
(`pkg/game/persistence_seam.go:104-109`) exactly where C sets it on inventory
movement. Deleting it is safe, and the proof is that all eleven stat scenarios —
including the eight that predate this work — stay green without it, so it was not
covering a missing flag-set anywhere.

### Evidence

- `wizard-stat-color-depth` (new, `keep-ansi`): one viewer, four levels, three
  report bodies per level. Targeted census `stat-color3` over every
  `wizard-stat*` and `wizard-valid-reports*` scenario at `8624f88b7`:
  `scenarios=11 passed=11 failed=0 verdict=CLEAN`.
- Both findings were found by the scenario, not by reading: the first run failed
  on F7, the second on F8, and each fix turned exactly those rows green.
- Unit proofs (R5h): `TestStatPlayerColorLevelsMatchC`,
  `TestStatRoomColorLevelsMatchC`, `TestStatObjectColorLevelsMatchC`,
  `TestStatMobColorLevelsMatchC` (each a four-level table that also asserts the
  report is escape-free below C_NRM), `TestStatPlayerSpellLineColorMatchesC`, and
  `TestStatFlagListsKeepSprintbitarrayTrailingSpace` for F7. Expected bytes are
  built from the literal codes in `src/screen.h:34-41`, never from the production
  constants, so a drift in either the constants or the gates fails the test. Both
  fixes were scratch-broken and the tests failed as intended.
- Two pre-existing unit assertions encoded the F7 bug
  (`wiz_stats_test.go`: `"AFF: SANCT NOTHING\r\n"`, `"AFF: BLIND\r\n"`) and were
  corrected to C's bytes.


## 12. `do_report` and `print_group` (2026-10-30)

Queue item 5. The two commands share a shape — a per-field coloured stats line —
but they turned out to be at opposite ends of the problem.

**`print_group` emitted no colour at all (FIXED).** C wraps every stat at
`C_CMP` against the **viewer** and prints two row shapes
(`src/act.other.c:638-681`):

```
head      [%s%3d%sH %s%3d%sM %s%3d%sV] [%2d %s%s%s] $N (Head of group)
          CCGRN on each stat, CCCYN on CLASS_ABBR                (line 659)
follower  [%s%3d%sH %s%3d%sM %s%3d%sV] [%2d %s%s%s] $N
          CCGRN on each stat, CCGRN on CLASS_ABBR                (line 676)
mob       [---H ---M ---V] [-- --] $N   — uncoloured            (line 669)
```

`pkg/session/cmd_group.go` printed the same fields plain. Two details are easy to
get wrong and are now pinned: the **class abbreviation is cyan on the head row
and green on a follower's row** — a genuine asymmetry in C, not a typo — and the
mobile row stays uncoloured at every level.

`C_CMP` is `_clrlevel >= 3` (`src/screen.h:45-49`), so this is the first command
in the audit whose discriminating level is `complete`: at `color normal` every
other coloured surface in this report is live and this listing must still be
plain. `group-color-levels-depth` walks all four levels and would fail a port
that reused the `C_NRM` gate.

**`do_report` was already correct — but nothing could prove it.** `pkg/game`'s
`doReport` (not `pkg/session/act_informative.go`, which is what §4's table
claimed) already formats the line once per recipient through that recipient's own
`C_NRM` wrappers while printing the reporter's numbers, exactly as C does
(`src/act.other.c:806-818`).

The problem was the proof. `docs/fidelity/depth/report.tsv` carried a
`report.color-audience` row claiming that "color-enabled peer and observer
descriptors cover this audience-dependent branch", proved by `report-depth` — a
scenario that does **not** set `keep-ansi`. Ordinary normalization strips ANSI by
design (`internal/oraclediff/normalize.go:64-67`: "the ordinary mode strips ANSI
and therefore cannot see a missing-color port"), so that row was green no matter
what the port emitted. It is an R5h violation in the manifest: a green that could
not fail on the thing it claimed to cover. The row's note is corrected to say it
covers the audience and the numbers only, and `report-color-levels-raw` is now
the raw-byte proof. This is the second manifest claim this audit has found that
could not fail (the first was `score`'s, which showed no escapes at all).

### Evidence

- `report-color-levels-depth` (new, `keep-ansi`): an actor walks all four levels,
  then a peer is forced to `complete` and back to `off` under a complete actor,
  so both directions of the per-recipient branch appear in one transcript.
- `group-color-levels-depth` (new, `keep-ansi`): leader + player follower + mobile
  follower, four levels. `group all` in warmup builds the group, so the transcript
  carries the head, follower and mobile rows.
- Targeted census `report-group-color` at `8a052ba1e`, `--draw-parity`, seven
  scenarios (the two new ones plus `report-depth`, `group-depth`,
  `group-gates-depth`, `group-mob-depth`, `group-room-depth` to catch regressions
  in the code being changed): `passed=7 failed=0 verdict=CLEAN`,
  `draw_divergent=0`.
- Unit proofs (R5h): `TestGroupListColorLevelsMatchC` (four-level table asserting
  the head row's cyan class, the follower row's green class, the uncoloured mobile
  row, and that the whole listing is escape-free below `C_CMP`) and
  `TestReportColorUsesCNRMNotCMP` (a `PRF_COLOR_2`-only viewer must still get the
  escapes, which a port reusing the `C_CMP` gate would fail). Both were
  scratch-broken — reverting the gate failed the `normal` subtest, dropping the
  head cyan failed the `complete` subtest.
- The stat surface's `statGreen`/`statCyan`/… constants and `wizardTint` are now
  shared by this code, so they are renamed to `ansiGreen`/`ansiCyan`/… and
  `colorWrap`. Mechanical, one file, no behaviour change.


## 13. `do_auto_exits`, and two claims the harness could not make (2026-10-30)

Queue item 6, the exits half. `do_exits` itself is **uncoloured** in C — all 13
sites of the pair are in `do_auto_exits`, the `[ Exits: ... ]` line `look` prints
under `PRF_AUTOEXIT` (`src/act.informative.c:657-680`).

C builds the list one direction at a time and appends a trailing space after
**every** entry, wrapping an imm-only closed door:

```c
sprintf(buf, "%s%s%s%s%s%s ", buf,
  io ? CCRED(ch, C_CMP) : "", io ? "(" : "", dirs[door], io ? ")" : "",
  io ? CCCYN(ch, C_NRM) : "");
...
sprintf(buf2, "%s[ Exits: %s]%s\r\n", CCCYN(ch, C_NRM), *buf ? buf : "None! ", CCNRM(ch, C_NRM));
```

Three things follow that the port had wrong. The port joined entries with a
single space, so the trailing space was gone; it printed the imm-only door as a
bare `(east)` with no red sandwich; and it wrote `[ Exits: %s ]` with its own
space before the bracket. The last two cancel out while colour is off — the space
that C puts inside each entry is the one the port put before `]` — which is
exactly why **no scenario had ever seen the missing C_CMP sandwich**: the only
uncoloured difference is where the space lives, and it lands in the same column.
Note also that C re-emits `CCCYN(ch, C_NRM)` after the `)` whenever the door is
imm-only, *even when the red is empty* (line 673), so the escape is present at
`color normal` too. The unit test caught that detail by disagreeing with my first
expectation.

### F10 — the harness cannot build a closed door (OPEN, tooling)

Getting a closed door into a scenario turned out to be impossible with the
fixture set. `replace-room-exits <room> <dir|all> <to> <DoorState>` writes the
**.wld door *capability* code**, which `pkg/parser/wld.go:83-92` maps to
`ExitIsDoor` (1) and `ExitIsDoor|ExitPickproof` (2) only — and its `DoorState` is
validated to 0..2 (`internal/oraclediff/scenario.go:485`), so `3`
(`EX_ISDOOR|EX_CLOSED`) is rejected outright. The closed state is applied at
runtime by a zone **D reset** (`ApplyDoorReset`), and the fixture list has no
door-reset directive. `replace-room-exits` also **replaces the whole exit list**,
so a second directive for the same room silently discards the first.

The vehicle therefore uses a corpus room that already has a runtime-closed door:
`D 0 11269 1 1` closes room 11269's east exit, and `look` under autoexit renders
it. That is the first time this branch has been exercised at all, which is why the
C_CMP site had gone unnoticed — it needs an immortal, a closed door, and colour at
`complete`, and two of the three were not reachable.

**F11 — `exits.closed-filter` claims a filter it does not build (OPEN, R5h).**
That row in `docs/fidelity/depth/exits.tsv` says "An open north exit is rendered
while an east exit marked EX_CLOSED is omitted", proved by `exits-filtering`. Its
fixture is `replace-room-exits 8105 north 8161 0` followed by
`replace-room-exits 8105 east 8161 1 gate` — which, per F10, (a) discards the
north exit entirely and (b) sets `exit_info = 1`, a door that is **not** closed.
The retained transcript agrees: the command prints
`Obvious exits:` / `East  - Western Vestibule` — east is *shown* and north is
*absent*, the exact opposite of the row's description. The scenario is green
because both engines agree on what it actually builds, so this is the **third
manifest claim in this audit that cannot fail** (after `report.color-audience` and
`score`'s), and the first where the row describes a state the fixture cannot
express.

**F12 — the character list gains a leading space when it follows a room
description (OPEN, R1).** Found while building the vehicle above, and not fixed
here. With a peer in the room, the peer's `look` diverged:

```
 C: \x1b[32m\x1b[33mExitsautoconfig the Warrior is standing here.
Go: \x1b[32m\x1b[33m Exitsautoconfig the Warrior is standing here.
```

C's `linePrefix` space belongs to the room description's trailing position; the
port applies it to the first character-list entry instead. The scenario was
narrowed to an empty room to isolate the exit line (the retained v7 run is the
evidence: `oracle-runs/2026-10-30/exits-auto-v7`), so the exit proof stands on its
own, and F12 needs its own investigation — it is in the F6 room-list area and
belongs with it.

### Evidence

- `exits-auto-exits-color-depth` (new, `keep-ansi`): the God walks
  `complete`/`normal`/`sparse`/`off` in room 11269 (closed east door) and then in
  8162 (open exits). The captured bytes are C's exactly, including the redundant
  cyan: `\x1b[36m[ Exits: \x1b[31m(east)\x1b[36m ]\x1b[0m` at `complete`,
  `\x1b[36m[ Exits: (east)\x1b[36m ]\x1b[0m` at `normal`, and
  `[ Exits: (east) ]` below C_NRM. I checked the captured blocks rather than
  trusting the verdict: an earlier version of this scenario passed **with the fix
  deliberately broken**, because its fixture never produced a closed door. A green
  that cannot fail is the failure mode this audit exists to find, so the
  non-vacuity check is part of the proof here.
- Targeted census `exits-regress` at `a98263eec`, nine scenarios (the new one plus
  `exits-depth`, `exits-filtering`, `exits-no-open`, `exits-immortal`,
  `exits-dark`, `room-desc-exits`, `observation-roomflags`,
  `player-state-toggles`): `passed=9 failed=0 verdict=CLEAN`.
- Unit proofs (R5h): `TestAutoExitsColorLevelsMatchC` (a seven-case table over
  mortal/immortal × the four levels, asserting the imm-only sandwich, the
  redundant cyan, and the plain forms) and
  `TestAutoExitsEmptyListKeepsCTrailingSpace` (C's literal `"None! "`). The first
  was scratch-broken — disabling the `C_CMP` gate failed the `immortal complete`
  case with exactly the old bytes.
## 14. Colour-level consolidation, and a mobile group leader (2026-10-30)

Two follow-ups from the `print_group` work, plus what chasing the second one
found.

**One definition per colour level.** After §11 and §12 the package held seven
copies of C's level arithmetic: `colorCMP` and `wizardHasCompleteColor` (both
`C_CMP`), `wizardHasNormalColor`, `whoColorEnabled` (written as "either bit"),
and inline `flags&(1<<PrfColorN)` expressions in the infobar state, the prompt's
two `C_CMP` sites and `reditCols`. C defines `_clrlevel`, `clr` and the eight
`CC*` macros once (`src/screen.h:34-56`) and every command reaches for them; the
port had re-derived the arithmetic per command, which is how §11's and §12's
gate mistakes survived review.

They now live in `pkg/session/color_levels.go`: `colorLevel` (C's `_clrlevel`),
`hasSparseColor`/`hasNormalColor`/`hasCompleteColor` (`clr` at C_SPR/C_NRM/C_CMP),
`colorWrap` (a `CCxxx … CCNRM` pair) and `colorCMP` (the `C_CMP` code-or-empty
form). Every site above now calls one of them. Behaviour is unchanged — the
predicates are the same arithmetic — which the targeted census confirms by
re-running the scenarios for every rewired surface.

**A grouped mobile can lead a group, and the port rendered it as a follower.**
`print_group`'s head arm has no `IS_NPC` check (`src/act.other.c:651-660`): it
prints `k`'s real stats and `CLASS_ABBR(k)`, which is the literal `"--"` for an
NPC (`src/utils.h:570`). The dashes row (line 669) is inside the **follower**
loop only. The port's `sendGroupMemberLine` showed dashes for any mobile,
including a leader, and `printGroup` bailed before that with an invented
`Your group leader is not online.` — a string C does not have anywhere, already
recorded in the triage baseline as `bug:invented`.

The fix resolves the leader to either body type and gives a mobile leader C's
head row; the unresolvable-name case now falls back to the viewer, which is C's
`k = (ch->master ? ch->master : ch)` null arm, instead of the invented message.
Proof: `TestGroupListMobileLeaderShowsRealStats`, scratch-broken to confirm it
fails (R5h) — restoring the dashes row produced
`     [---H ---M ---V] [-- --] a guard` where C prints
`     [ 40H   5M  12V] [ 9 --] a guard (Head of group)`.

**F9 — the port cannot run `group` as a mobile, and `force` hides it (OPEN, R1/R4).**
Reaching the mobile-leader state needs `do_group` to run as the mobile, because
`perform_group` is the only thing in the tree that sets `AFF_GROUP`
(`src/act.other.c:629`; no mobprog or spec proc sets it) and the viewer has to be
enrolled by its own master. C reaches that through `do_force`, which resolves its
target with `get_char_vis` — mobiles included — and hands the command to
`command_interpreter(vict, to_force)` (`src/act.wizard.c:1869-1880`). The port's
`cmdForceText` resolves mobiles too, but dispatches through
`executeCharacterCommand`, whose surface has no `group`; an unsupported command is
dropped with a `slog.Warn` after the forcer has already been told `Okay.`
(`pkg/session/wiz_communication.go:221-223`).

A `keep-ansi` scenario built exactly as C's transcript requires
(`force Grouppeer follow generic`, `force generic group Grouppeer`) proves the
gap: C answers the follower's `group` with
`Your group consists of:` and a row, while the port answers
`But you are not the member of a group!` because the mobile's `group` never ran.
The scenario is **not** committed — it cannot go green, and a full census would
fail on it — but the run is retained as evidence:
`oracle-runs/2026-10-30/group-mobile-leader`, whose `census-dump/` carries the
scenario. The general shape of F9 is larger than `group`: C's `do_force` runs
*any* command as a mobile, and the port's mobile command surface is a partial
registry that fails silently behind a success message. That needs its own issue
and its own audit, not a fix smuggled into a colour PR.

So this row's coverage is unit-level on purpose: the state is reachable in C but
not currently reachable in the port, which is exactly why the port's wrong branch
had gone unnoticed. It is latent, not dead — the moment `group` joins the mobile
command surface, the head row must already be right.
### The leader lookup is a body, not a name (R3)

`groupLeaderActor` resolved the leader with `GetMobByName`, which matches a
**substring** over a map-ordered slice: with two NPCs sharing a name it could pick
the wrong one, and the pick changed from run to run. C never resolves `ch->master`
that way — it is a live pointer — and `k->followers` is that body's own list.
`printGroup` now uses two exported body-identity accessors: `World.FollowingActor`
(the retained leader, via `combatFollowingBody`, falling back to a uniquely named
player) and `World.FollowersOf`. The follower list had the same defect —
`GetFollowerActors` matches by name, so two duplicates' lists merged — and
`FollowersOf` fixes it for this command; the other name-based callers still need
their own audit.

`TestGroupListResolvesTheFollowedDuplicateNpc` spawns two guards with the same
name and different stats, has an unrelated player follow the first and the viewer
follow the second, and asserts the head row is the *second* guard's and that the
first guard's follower is absent. Run against the pre-fix resolution it fails all
three assertions at once:

```
head row is not the followed duplicate: "…[ 10H   1M   2V] [ 3 --] a guard (Head of group)…"
head row resolved the other same-named guard: "…[ 10H   1M   2V] …"
follower list merged both same-named guards: "…[100H 100M   0V] [ 5 Mu] Bystander…"
```

### One colour definition, now across packages too (R5c)

§14 above folded the seven copies *inside* `pkg/session`, but `pkg/game` kept its
own `colorLevel`, `observationColors` and `completeObservationColor`, plus its own
copy of the K* codes — so the same thresholds were still defined twice, one
package apart, and neither could see the other drift. The gate bugs in §11 and §12
came from exactly that re-derivation.

The definitions now live in a new leaf package, **`pkg/screen`** (named for
`src/screen.h`): the K* codes, C's four levels, `Level`/`LevelFromFlags`, the
`SparseOn`/`NormalOn`/`CompleteOn` predicates and `Wrap`. `pkg/session` and
`pkg/game` each keep a one-line adapter for their own flag word or `*Player`, so
call sites did not move, but there is now **one** place where C's arithmetic is
written down. `pkg/game`'s ANSI names alias `pkg/screen`'s constants rather than
repeating the bytes.

### Evidence

- Full `go test ./...`, `go vet ./...`, `golangci-lint` (0 issues), gofumpt,
  deadcode ratchet and `make fidelity-depth` on the rebased commit.
- Targeted census re-running every surface either package colours: 13 scenarios
  (stat, score, who, report, group, infobar, prompt/pager, redit, display,
  terminal colours), `verdict=CLEAN`, `draw_divergent=0`.
- The duplicate-NPC test above, scratch-broken against the pre-fix resolution.



## 15. `look`'s own colour, and two sites that are dead in C (2026-10-30)

Queue item 6's last piece. The inventory attributes **four** C_CMP sites to
`look`, split across two functions. Two are reachable, two are not.

**`look_at_char`'s tattoo line (reachable, already correct).** C wraps the word
"glows" in `CCWHT(ch, C_CMP)` … `CCNRM(ch, C_CMP)`
(`src/act.informative.c:415-417`). The port already emitted `KWHT` and `KNRM` at
exactly that gate, so the bytes were right; what was wrong was the *spelling* — a
raw `"\x1b[37m"` and a hand-rolled reset, from before §14 gave the port one place
to name these. It now reads `screen.Wrap("glows", screen.KWHT, screen.CompleteOn(...))`.
Byte-identical, and `look-char-color-depth` proves it raw:

```
complete  On her right arm is a tattoo of a green dragon... it \x1b[37mglows\x1b[0m softly.
off/sparse/normal  On her right arm is a tattoo of a green dragon... it glows softly.
```

**`list_char_to_char`'s infravision line is dead in C.** Its two sites wrap "red"
in `CCRED(ch, C_CMP)` … `CCNRM(ch, C_CMP)` (`src/act.informative.c:646-650`), and
the port has no equivalent. It does not need one:

```c
/* look_at_room, src/act.informative.c:731-732 */
if ((IS_DARK(ch->in_room) && !CAN_SEE_IN_DARK(ch)) || (IS_AFFECTED(ch, AFF_BLIND)))
    ... send the dark/blind message and return ...
...
list_char_to_char(world[ch->in_room].people, ch);   /* line 835, the only call */
```

The branch requires `IS_DARK(ch->in_room) && !CAN_SEE_IN_DARK(ch)`
(lines 643-644), which is precisely the condition that returns before line 835,
and line 835 is `list_char_to_char`'s **only** caller in the tree. So those two
C_CMP sites can never reach a player, and R4 forbids inventing output for them.

The **reachable** eyes line is a different one — `look_at_room`'s dark branch at
line 755, `"You see a pair of glowing red eyes.\r\n"`, which is **uncoloured** in
C — and `appendDarkOccupants` already matches it, including the
`"You hear someone or something moving around nearby."` fallback at line 759.
This is the first pair of sites in the audit whose disposition is *unreachable*
rather than "unproven": the distinction matters, because the port would have to
add colour to reproduce them, and adding colour C never emits is exactly the
class R4 exists to stop.

### Evidence

- `look-char-color-depth` (new, `keep-ansi`): the God walks all four levels in
  room 8162, looking at the room and at a tattooed peer (`set <peer> tattoo 1`).
  Room 8162 was chosen because it has **no extra descriptions**: C's
  `do_description` linePrefix space only appears in rooms that have them, and
  keeping it out of this transcript leaves the exit-line proof in
  `exits-auto-exits-color-depth` to stand on its own. Targeted census
  `look-char-color` at `08782cf9b`: `passed=1 failed=0 verdict=CLEAN`.
- I checked the captured blocks rather than trusting the verdict — the white
  sandwich is present at `complete` and absent at the other three levels. The
  exits vehicle had passed once while deliberately broken, so a green alone is
  not evidence here.
- Unit proof (R5h): `TestLookAtCharTattooColorLevelsMatchC`, a four-level table
  that also asserts no `KWHT` appears below C_CMP.
- The dead-branch claim was verified by reading `look_at_room`'s early return and
  grepping `list_char_to_char`'s call sites, not by trusting the inventory (R5e).

