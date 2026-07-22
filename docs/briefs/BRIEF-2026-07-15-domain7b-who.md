# BRIEF — Domain 7b: `who` roster (+ `where` residual) — O13 (+ O11)

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/domain-who` off `main`.
**Findings:** DP-1096 / O13 (who filters + visibility + format). **Also folds** the DP-1094 / O11
`where` **residual** (the critical mortal-exposure part is ALREADY FIXED — see §0).
**Part 2 of Character-view.** **Method rules:** read `src/act.informative.c` `do_who`
(1681-1860) + `WHO_FORMAT` + `CLASS_ABBR`, and `do_where` (2244-2307) directly.
Gated by an **oracle red→green run**.

---

## 0. `where` (O11) — critical part already fixed; verify + residual only
The oracle (`--scenario character-view`, `where [actor]`) shows **no divergence**: a mortal
`where` is refused on both servers. The gate table already has `where 31 5` (LVL_IMMORT,
POS_RESTING) and it's enforced — so **the mortal-exposure / vnum-leak-to-mortals bug (the
Critical part of DP-1094) is resolved.** Residual, immortal-only (off the mortal oracle path,
low priority): implement both C forms — no-arg self-context listing + target search
(`do_where`, act.informative.c:2244-2307) — with C's visibility rules and vnum shown only to
immortals. If you don't implement the immortal forms here, leave a note; the Critical finding
can be closed on the strength of the oracle parity. **Do not** re-expose `where` to mortals.

## 1. Oracle-PROVEN RED (`scenarios/character-view.txt`, verified 2026-07-15)
Two level-1 Human Warriors, hometown K, co-located at 8162. `who [actor]` diff (C `-`, Go `+`):
```
 Players
 -------
-[  1  Wa ] Cviewobs the Warrior
-[  1  Wa ] Cviewactor the Warrior
+[  1  Warrior ] Cviewactor      (Human, Warrior, player)
+[  1  Warrior ] Cviewobs        (Human, Warrior, player)

 2 characters displayed.
```
Three divergences: **(a) line format** — C `[ %2d  <ABBR> ] <name> <title>`, Go uses full class +
a `(race, class, type)` parenthetical and drops the title; **(b) ordering** — C is
descriptor-list order (**most-recently-connected first** → Cviewobs before Cviewactor); Go is
creation/actor-first; **(c)** the header `Players\r\n-------\r\n` and trailer `2 characters
displayed.` already MATCH — keep them.

## 2. Root cause
`who` is registered `wrapNoArgs(cmdWhere→cmdWho)` so **all filter args are discarded**
(commands.go:~107), and the handler lists every live session unconditionally with a Go-invented
format (cmd_info.go:~426-463). No viewer-aware visibility, no C grammar, wrong per-line format
and order.

## 3. Faithful C reference (`do_who`, act.informative.c:1681-1860)

### 3a. Mortal (non-short) line format — the RED
`:1826` (the final `else` in the non-short branch):
```c
sprintf(buf, "[ %2d  %s ] %-12.12s ... GET_LEVEL(tch), CLASS_ABBR(tch), GET_NAME(tch));
/* immortal tiers above print [ Wizard ]/[ IMMORT ]/... %s %s (name title) */
```
Non-short mortal line = `"[ %2d  %s ] %s %s"` → level (width 2), `CLASS_ABBR` (e.g. Warrior→`Wa`),
name, `GET_TITLE`. Immortal tiers (LEVEL_IMMORT..IMP) use bracket labels `[ Wizard ]`/`[ IMMORT ]`/
`[ TITAN ]`/`[  GOD  ]`/`[ LEGEND ]`/`[ HIGOD ]`/`[ GRGOD ]`/`[ *IMP* ]` + `name title`. Port
`CLASS_ABBR` (the pc_class abbrev table) faithfully.

### 3b. Ordering
Iterate `descriptor_list` order (CircleMUD prepends new descriptors → newest first). Reproduce
that order, not creation order.

### 3c. Argument grammar + visibility gates (the bulk of O13 — unit-tested)
`do_who` parses (`:1690-1745`): level range `low high` (digits), `-l low high`, `-n name`,
`-c <classletters>` (class mask), `-q` quest-only, `-o` outlaws-only, `-s` short list, `-r`
localwho (same zone), `-z`/who_room (same room). Bad token → `WHO_FORMAT` help. Per-target skips:
- `!CAN_SEE(ch, tch)` → skip (visibility — hidden/invis players hidden from lower viewers)
- level `< low || > high`
- `-q` and not `PRF_QUEST`; `-o` and not `PLR_OUTLAW`
- `ROOM_NO_WHO_ROOM` and viewer `< LEVEL_IMPL` → skip
- localwho zone mismatch; who_room room mismatch; class-mask mismatch
- name filter: `str_cmp(name)` AND not `strstr(title, name)`

Trailer: `"\r\n%d characters displayed.\r\n"` (matches today). Read the source for `WHO_FORMAT`
and the exact option letters; assert them in tests.

## 4. Session adoption
Accept the raw arg string (stop `wrapNoArgs` discarding it); parse per §3c; build a
**viewer-aware roster** from live sessions filtered by `CAN_SEE` + the option predicates; render
C's per-line format (§3a) in descriptor order. Keep header/trailer. If a game-owned roster query
is cleaner (viewer perspective), mirror the F0a DTO pattern — but visibility must be evaluated
against the viewer.

## 5. Acceptance gate
1. **Oracle red→green:** `--scenario character-view` → `who [actor]` block clean (format + order).
2. **Unit tests** (exact C strings — most of O13): each filter (`-l/-n/-c/-q/-o/-s/-r/-z`, bare
   `low high`), `WHO_FORMAT` on bad token, `CAN_SEE` hiding an invis/hidden player, `NO_WHO_ROOM`
   hidden from sub-IMPL, `CLASS_ABBR` per class, immortal bracket tiers, descriptor ordering.
3. `where`: keep the immortal gate; (optional) immortal no-arg + target forms with unit tests.
4. `make check-fmt vet` + `go test ./...` green; no WS schema break.

## 6. Gotchas
- **Two co-located newbies only prove the mortal line format + order** — filters and visibility
  aren't reachable in the scenario; lock them with unit tests (add a hidden/second-zone fixture
  later if you want oracle coverage).
- **`where` stays immortal** — the whole point of O11 was it must not serve mortals.
- **ANSI blind spot** — color codes (CCYEL etc.) are stripped by the normalizer; unit-test color.
