# BRIEF — Domain 7a: `score` self-view — O9 + O14

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/domain-score` off `main`.
**Findings:** DP-1093 / O9 (hometown name), DP-1103 / O14 (residual score fields).
**Part 1 of the Character-view domain** (others: who, consider, time/weather). **Display-only.**
**Method rules:** read `src/act.informative.c` `do_score` (1168-1452) + `src/constants.c` directly.
Gated by an **oracle red→green run** — a green build is NOT sign-off.

---

## 1. Oracle-PROVEN RED (`scenarios/character-view.txt`, verified 2026-07-15)

Fresh level-1 Human Warrior, hometown K, at 8162. Run:
```
DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" \
  go run ./cmd/dp-oracle-diff --scenario character-view
```
`score [actor]` diff (C = `-`, Go = `+`) — the **display** divergences you own:

| # | line | C oracle | Go port | finding |
|---|---|---|---|---|
| 1 | mana label | `... Mana points: 100(100) ...` | `... Mana: 100(100) ...` | O14 — drops ` points` from the `%s points:` label |
| 2 | hometown | `You are a citizen of Kir Drax'in.` | `You are a citizen of Solace.` | O9 — stale stock `Hometowns[]` |
| 3 | rank/title | `This ranks you as Cviewactor the Warrior (level 1).` | `This ranks you as Cviewactor  (level 1).` | O14 — empty title (double space); default title missing |
| 4 | trailing blank | blank line after `You are a Human Warrior.` | (absent) | O14 — missing terminal `\r\n` |

**⚠ OUT OF SCOPE — do NOT touch (tracked as DP-1161):** the same probe also shows `Hit points: 22(22)`→`24(24)`, `Movement 85`→`84`, and `You are naked, have you no shame?`→`You are well armored.` These are **model** divergences (AC/starting-eq/stat-RNG), NOT the score renderer. This PR renders the model faithfully; it must not alter HP/AC/move. Leave them — DP-1161 owns them.

## 2. Root cause
`score` renders `hometown = game.Hometowns[p.Hometown]` (pkg/session/cmd_info.go:~204) but `game.Hometowns` (pkg/game/constants.go:~19) is a **stale stock Dragonlance array** (`{"Kalaman","Solace","Port Storm",…}`), so `Hometowns[1]` = "Solace". A second wrong table exists at `spec_procs3.go:~346` (`{"","Midgaard","Thalos","New Thalos"}`). The correct DP names live only in the creation menu. Other O14 fields (mana label, title, blank line) are formatter drift in the session `score`.

## 3. Faithful C reference (`do_score`, act.informative.c) — exact strings

### 3a. Hometown (O9) — the RED
`src/constants.c:51`:
```c
const char *hometowns[] = { "!Bad Hometown - Tell a God!", "Kir Drax'in", "Kir-Oshi", "Alaozar" };
```
C prints `"You are a citizen of %s.\r\n", hometowns[GET_HOME(ch)]` (:1284). `GET_HOME` = 1/2/3 for K/O/A.
**Fix:** replace the stock `Hometowns[]` with this DP table and index by `p.Hometown`; delete the stray `hometownNames` in spec_procs3.go and point it at the one canonical table. Confirm the value stored at creation aligns (K→1). Index 0 is the "bad hometown" sentinel.

### 3b. Mana label (O14) — the RED
`:1192` — `(!IS_PSIONIC(ch) && !IS_MYSTIC(ch)) ? "Mana" : "Mind/Psi"`, used as `"%s points:"`:
```
Hit points: %d(%d)  <LABEL> points: %d(%d)  Movement points: %d(%d)
```
So a warrior line is `Mana points:` (not `Mana:`). Psionic/mystic → `Mind/Psi points:`. **Fix:** restore ` points` and the psionic/mystic label (the class branch is unit-tested; the warrior case is the oracle RED).

### 3c. Title (O14) — the RED
`:1301` — `"This ranks you as %s %s (level %d).\r\n", GET_NAME(ch), GET_TITLE(ch), GET_LEVEL(ch)`. A fresh character's `GET_TITLE` is the **class default** ("the Warrior" for a warrior). Go stores an empty title → `Name  (level 1)` with a double space. **Fix:** ensure new characters get the C default title per class/level (find where C sets it — `do_start`/`set_title`), and render `GET_TITLE` verbatim. Verify K/O/A + all classes with unit tests.

### 3d. Full layout / other lines (O14)
Match `do_score` line-for-line (already mostly correct in the RED — only 1-4 diverge). Notable exact strings to keep aligned: the alignment prose (`:1204-1228`), AC prose (`:1233-1260`, **rendered from the model — see the DP-1161 caveat**), `Experience:    %d points`, `Coins carried:`/`Coins in bank:`, `Kills: %d  Pks: %d  Deaths: %d`, `You need %d exp to reach your next level.`, `You have been playing for %d days and %d hours.`, veteran line, citizen line (3a), rank line (3c), `You are %s %s <class>.\n\r`, then the pack-weight line and the **trailing blank line** (RED #4). Read the source; don't guess wording.

## 4. Session adoption
Keep `score` in the session render path (it's a pure self-view formatter — no game op needed), OR mirror the F0a pattern with a game-owned `ScoreLines(ch)` if that's cleaner; either way source hometown/title from canonical game state, not session-local tables. Delete the two dead hometown tables' wrong usages.

## 5. Acceptance gate
1. **Oracle red→green:** `--scenario character-view` → `score [actor]` block clean **except** the DP-1161 model lines (HP/move/AC), which stay divergent by design until DP-1161. (If you want a fully-clean block, coordinate — but do NOT fix the model here.)
2. **Unit tests** (exact C strings): hometown K/O/A + sentinel; mana-vs-mind/psi label by class; default title by class; full score golden for a fixed-stat fixture (so HP/AC are pinned and the whole block can assert clean).
3. `make check-fmt vet` + `go test ./...` green; no WS schema break.

## 6. Gotchas
- **Display-only.** The HP/move/AC deltas are DP-1161's; touching the model here muddies both.
- **Two stale hometown tables** — fix both usages, collapse to one canonical DP table.
- **`GET_HOME` index base** — hometowns[0] is the sentinel; K/O/A = 1/2/3. Verify the stored value.
- **ANSI blind spot** — the normalizer strips color; color parity is unit-test-only.
