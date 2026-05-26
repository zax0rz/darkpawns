# Claude Code Batch — Run 7: Clan System Fixes

## Overview
5 fidelity issues in the clan system. The critical prerequisite is fixing the save format — `ClanID`/`ClanRank` are on the `Player` struct but not in `savePlayerData`, so they're never written to disk. This blocks all offline player fixes.

## Issues
- DP-430: doClanBank nil pointer panic (URGENT)
- DP-431: Clan destroy only clears online players (HIGH)
- DP-434: Clan plan set wipes description without editor (HIGH)
- DP-433: InitClans trusts cached JSON — member counts drift (MEDIUM)
- DP-432: Clan enroll/members only see online players (MEDIUM)

---

## Task 0: Add ClanID/ClanRank to save format (PREREQUISITE)

**File:** `pkg/game/save.go`

**Problem:** `Player.ClanID` and `Player.ClanRank` (defined at `pkg/game/player.go:37-38`) are not included in `savePlayerData`. This means they're never written to disk. All offline player operations (destroy, enroll, members, InitClans) depend on being able to read clan data from saved players.

**Fix — 3 changes in `save.go`:**

### 1. Add fields to `savePlayerData` struct (after `BankGold` at line ~52):
```go
ClanID    int `json:"clan_id"`
ClanRank  int `json:"clan_rank"`
```

### 2. Add to `playerToSaveData` (after `BankGold: data.BankGold,` around line ~186):
```go
ClanID:    p.ClanID,
ClanRank:  p.ClanRank,
```

### 3. Add to `saveDataToPlayer` (after `BankGold: data.BankGold,` around line ~286):
```go
ClanID:    data.ClanID,
ClanRank:  data.ClanRank,
```

**Verification:** `go build ./...` must pass. This is a safe additive change — old saves without these fields will decode as 0 (no clan), which is correct default behavior.

---

## Task 1: Fix doClanBank nil panic (DP-430)

**File:** `pkg/game/clan_bank.go:7-19`

`doClanBank` declares `var c *Clan` (nil) and never resolves it. Line 14: `if action == CBWithdraw && !c.CanWithdraw(ch)` dereferences nil → server panic.

**Fix:** Add clan resolution after the empty arg check. Look at how `doClanRanks` in `clan_settings.go` resolves the clan — use the same pattern:
```go
clanNum, c := w.Clans.FindClanByID(ch.ClanID)
if c == nil {
    ch.SendMessage("You don't belong to any clan!\r\n")
    return
}
```

Remove the unused `var clanNum int` and `var c *Clan` declarations at the top.

**C source:** `src/clan.c:881` — `do_clan_bank()` resolves `clan_num = find_clan_by_id(GET_CLAN(ch))` before any bank operations.

---

## Task 2: Clan destroy — clear offline players (DP-431)

**File:** `pkg/game/clan_admin.go:132-138`

**C source:** `src/clan.c:240` — `do_clan_destroy()` iterates `player_table` (all players), loads each with `load_char()`, clears clan ID/rank if it matches, saves with `save_char_file_u()`.

**Go behavior:** Only loops `w.players` (online players). Offline members keep stale `ClanID`.

**Fix:** After clearing online players, iterate over saved player files on disk:
1. Read the player index at `lib/etc/players` (each line: `name level last_login`)
2. For each player NOT currently online, call `game.LoadPlayer(name)` 
3. If `loaded.ClanID == c.ID`, set `loaded.ClanID = 0` and `loaded.ClanRank = 0`
4. Call `game.SavePlayer(loaded)` to persist

Use the `saveDir` constant (`./data/players`) and `PlayerSaveExists` / `LoadPlayer` / `SavePlayer` from `pkg/game/save.go`.

**Important:** This task depends on Task 0 (ClanID/ClanRank in save format). Without Task 0, LoadPlayer won't return clan data.

---

## Task 3: Clan plan — activate editor (DP-434)

**File:** `pkg/game/clan_settings.go:64-110` (`doClanPlan`)

**C source:** `src/clan.c:1392` — `do_clan_plan()` calls `string_write(ch->desc, &clan[clan_num].plan, ...)` to activate the line editor for multi-line input.

**Go behavior:** Sets `c.Plan = ""` (clears the plan) and saves immediately. Data loss.

**Fix — simplified approach (no editor needed):**

Instead of clearing the plan, accept text as an argument:
```go
func (w *World) doClanPlan(ch *Player, arg string) {
    _, c := w.Clans.FindClanByID(ch.ClanID)
    if c == nil {
        ch.SendMessage("You don't belong to any clan!\r\n")
        return
    }
    
    if arg == "" {
        // Show current plan
        if c.Plan == "" {
            ch.SendMessage("Your clan has no plan set.\r\n")
        } else {
            ch.SendMessage("Clan plan:\r\n%s\r\n", c.Plan)
        }
        return
    }
    
    // Set plan to argument text
    c.Plan = arg
    w.SaveClans()
    ch.SendMessage("Clan plan updated.\r\n")
}
```

Remove the old code that clears `c.Plan` unconditionally. The full `string_write` editor can be added later — for now, single-line plan setting is better than data loss.

---

## Task 4: InitClans — recalculate from player DB (DP-433)

**File:** `pkg/game/clans.go:162-195` (`InitClans`)

**C source:** `src/clan.c:836` — `init_clans()` loads every player record and recalculates each clan's `members` count and `power` total.

**Go behavior:** Trusts cached JSON values. No recalculation.

**Fix:** After loading clans from JSON, scan all saved player files to recalculate:
1. Read player index from `lib/etc/players`
2. For each player, call `LoadPlayer(name)`
3. If `player.ClanID` matches a clan, increment that clan's `Members` and add `player.Level` to `Power`
4. Overwrite the cached values with recalculated ones

```go
// After loading from JSON cache:
files, _ := os.ReadDir(saveDir)
for _, f := range files {
    if strings.HasSuffix(f.Name(), ".json") {
        name := strings.TrimSuffix(f.Name(), ".json")
        p, err := LoadPlayer(name)
        if err != nil {
            continue
        }
        if p.ClanID != 0 {
            if c, ok := cm.clans[p.ClanID]; ok {
                c.Members++
                c.Power += p.Level
            }
        }
    }
}
```

**Depends on:** Task 0 (ClanID in save format).

---

## Task 5: Clan enroll/members — list offline players (DP-432)

**File:** `pkg/game/clan_membership.go:27-32, 283-295`

**C source:** `src/clan.c:287` (`do_clan_enroll`) and `src/clan.c:579` (`do_clan_members`) — both iterate the full player table, loading offline players with `load_char()`.

**Go behavior:** Both loop only `w.players` (online).

**Fix for `clanEnroll` (around line 27):** After listing online applicants, scan saved player files:
```go
// After online player loop:
files, _ := os.ReadDir(saveDir)
for _, f := range files {
    if strings.HasSuffix(f.Name(), ".json") {
        playerName := strings.TrimSuffix(f.Name(), ".json")
        // Skip if already listed (online)
        if _, ok := w.players[playerName]; ok {
            continue
        }
        p, err := LoadPlayer(playerName)
        if err != nil {
            continue
        }
        if p.ClanID == c.ID && p.ClanRank == 0 {
            ch.SendMessage("%s (offline)\r\n", p.Name)
        }
    }
}
```

**Fix for `clanMembers` (around line 283):** Same pattern — after listing online members, scan disk for offline members with matching `ClanID` and `ClanRank != 0`.

**Depends on:** Task 0 (ClanID in save format).

---

## Execution Order
1. **Task 0** (save format) — MUST go first. All offline tasks depend on it.
2. **Tasks 1 + 3** (bank panic + plan fix) — independent, can run in parallel
3. **Tasks 2 + 4 + 5** (offline player operations) — depend on Task 0, but independent of each other

## Verification
1. `go build ./...` — must pass after each task
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
