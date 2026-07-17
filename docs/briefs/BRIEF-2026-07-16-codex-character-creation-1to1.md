# BRIEF (codex) — character creation must be 1:1 with C `nanny()` (creation fidelity campaign)

**Owner:** codex. **Gate:** Claude owns `scenarios/character-creation.txt` and runs the oracle red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** This is a real domain (comparable to cast/quit), not a nit — a small stacked pair is welcome (see "Suggested shape").

## North star (governs every choice here)
The Go port must be **byte-for-byte 1:1 with original C Dark Pawns on the entire player-facing surface** — every prompt, message, menu, ordering, and byte a player can see. *The game is the game.* Any place the port is "nicer" than C is a **bug to remove**, not a feature. Modern engineering is allowed only where it's **not player-facing** (transport, DB, concurrency). See `docs/briefs` history and the oracle-proof gate.

## The gap (proven RED: `--scenario character-creation`)
Creation was never differentially tested — it was only ever drained `[setup]`. Now diffed, the port re-skins `nanny()` heavily:
1. **Two creation layers.** The port collects name+password in the **telnet transport auth layer** (`pkg/telnet/listener.go:291-421`, the DP-909 design) and the nanny port (`pkg/session/char_creation.go`) skips those prompts. C has **one** flow (`nanny()`), and the transport layer leaks non-C prompts (see list below).
2. **Fabricated MOTD.** `lib/world/text/motd` ships an invented `Welcome to Dark Pawns MUD / Rules of the Realm: 1…4 / Enjoy your stay!` block. The real C motd is the `darkpawns.com` text. It's also **emitted twice**, and the **start room is displayed twice** on entry.
3. **Invented bracket menus** (`[Y] Yes/[N] No`, `[M] Male/[F] Female`) that C does not have — C's prompts are bare inline.
4. **Wrong prompt wording**, missing lines (`New character.`, `WELC_MESSG`), and mis-ordered lines.

## Read-only source of truth
C: `~/.openclaw/workspace/darkpawns-c-oracle/src/interpreter.c` `nanny()` (**1693-2210**, states `CON_GET_NAME`→`CON_MENU`); text constants in `src/config.c` (`MENU` :209, `WELC_MESSG` :258) and `src/class.c`/`src/constants.c` (race/class/hometown menus); the real motd at `~/.openclaw/workspace/darkpawns-c-oracle/lib/text/motd` (or wherever `motd` resolves under that lib). **Never edit the oracle tree.**
Go: `pkg/telnet/listener.go` (transport auth prompts), `pkg/session/char_creation.go` (nanny port), `pkg/session/session_send.go`, `lib/world/text/motd` (data).

## The exact C new-character flow — reproduce byte-for-byte
Every string below is fidelity, including punctuation/`\r\n`. Order is law.

1. **Connect** → greeting logo + credits + `As of 10-17-2008 there has been a pwipe.  Enjoy your new adventures!` then the name prompt `By what name do you wish to be known? ` *(the port already sends this — keep it).*
2. **Unknown (new) name** → `Please remember to choose an appropriate fantasy-oriented name.\r\n` then `Did I get that right, <name> (Y/N)? ` → CON_NAME_CNFRM.
   - Invalid name → `Invalid name, please try another.\r\n` + `Name: ` (re-prompt, do **not** disconnect).
3. **Confirm `Y`** → `New character.\r\n` then `Give me a password for <name>: ` (echo off) → CON_NEWPASSWD.
   - `N` → `Okay, what IS it, then? ` (back to name). Other → `Please type Yes or No: `.
4. **New password** (valid: 3..MAX len, ≠ name) → `\r\nPlease retype password: ` → CON_CNFPASSWD. Illegal → `\r\nIllegal password.\r\n` + `Password: `.
5. **Confirm password:** match → `Do you want ANSI color (Y/N)? ` → CON_COLOR. **Mismatch → `\r\nPasswords don't match... start over.\r\n` + `Password: ` and restart password (C does NOT disconnect).**
6. **Color** `Y`/`N` (invalid → `Please answer Y or N.\r\nDo you want ANSI color (Y/N)? `) → `What is your sex (M/F)? ` → CON_QSEX.
7. **Sex** `M`/`F` (invalid → `That is not a sex..\r\nWhat IS your sex? `) → `<race_menu>` + `\r\nRace: ` → CON_QRACE.
8. **Race** letter (`?`/`?X` = help; invalid → `That is not a race..\r\nWhat IS your race? `) → `<human_class_menu>` if Human else `<class_menu>` + `\r\nClass: ` → CON_QCLASS.
9. **Class** (invalid → `\r\nThat's not a class.\r\nClass: `) → `<hometown_menu>` + `\r\nSelect: ` → CON_HOMETOWN.
10. **Hometown** `K`/`O`/`A` (invalid → `Invalid choice!\r\nSelect: `). **NOTE: C's `CON_HOMETOWN` has no `break` — it falls through into the roll immediately, same input cycle. Do not require an extra keystroke.**
11. **Roll:** `\r\nYour ability scores:\r\n` + the stat block (`  Str: %-13s Dex: %-13s Int: %-13s\r\n  Wis: %-13s Con: %-13s Cha: %-13s\r\n`) + `\r\nPress 'Y' to keep these stats, and 'N' to reroll:` → CON_ROLLABL2.
12. **Keep `Y`** → `<motd>` + `\r\n\n*** PRESS RETURN: ` → CON_RMOTD. `N` → reroll (reprint scores block). Invalid → `Invalid choice! Select 'Y' or 'N':`.
13. **RETURN** → `<MENU>` → CON_MENU. `MENU` (config.c:209) is exactly:
    ```
    \n\rWelcome to Dark Pawns!\n\r0) Exit from Dark Pawns.\n\r1) Enter the game.\r\n2) Enter description.\r\n3) Read the background story.\r\n4) Change password.\r\n5) Delete this character.\r\n\r\n   Make your choice: 
    ```
14. **Menu `1`** → `WELC_MESSG` = `\r\nWelcome to Dark Pawns! May your visit here be... Interesting.\r\n\r\n`, then the start room (shown **once**). `0` → `Goodbye.\r\n`.

## The Go fix (requirement, not prescription)
The **player-visible transcript must equal C's `nanny()` output byte-for-byte.** How you reconcile the port's transport-auth layer (`listener.go`) with the nanny port (`char_creation.go`) is your call — either make the transport layer emit C's exact prompts in C's exact order, or move the creation dialogue wholly into the nanny and make transport transparent — but the oracle transcript is the judge. Concretely you must:
- **Delete the transport-layer leaks / re-skins** in `listener.go`: `Character does not exist. Do you want to create a new character? (Y/N): `, `No database connection. Create new character? (Y/N): `, `Choose a password: `, `Confirm password: `, `Passwords do not match. Disconnecting.`, the non-C `Invalid name. Use 2-32 characters…`. Replace with C's wording + the `Did I get that right, <name> (Y/N)?` confirm step and C's password prompts, and make a password mismatch **re-prompt** (not disconnect).
- **Remove every bracket menu** (`[Y] Yes/[N] No`, `[M] Male/[F] Female`, and any others) — C's confirm/sex/color prompts are bare inline text.
- **Fix `lib/world/text/motd`** — replace the fabricated "Rules of the Realm" block with the real C motd content (copy from the oracle's `motd`, preserving its exact bytes including the `&c…&n` color codes). This is a **data** fix.
- **De-duplicate**: the motd and the start-room description are each emitted twice on entry — emit once, matching C.
- **Add the missing lines** in the right places: `New character.\r\n`, `WELC_MESSG`, and `Please remember to choose an appropriate fantasy-oriented name.\r\n` in C's position (right after the name prompt, before the confirm).
- **Preserve stat-roll RNG faithfully** — the `<ROLLED_STATS>` normalizer masks the values, but the roll must still consume the same PRNG draws as C `roll_real_abils` (don't change draw counts; a prior campaign, DP-1063..1081, touched creation — verify you don't regress it).

## Transports
The oracle tests the **telnet** path — that must be exact. The **WebSocket** creation path should match too (same player-facing bytes); if you can't unify both in one PR, make telnet exact now and file a follow-up for WS, noting it in the PR.

## Suggested shape (optional stack)
1. **Data + dedupe** (small, low-risk): real `motd`, remove the fabricated blocks, fix the double motd/room emit, add `WELC_MESSG`/`New character.`
2. **Prompt reconciliation**: transport-auth vs nanny wording, bracket-menu removal, password re-prompt-not-disconnect, ordering.

## Oracle gate (Claude owns)
`scenarios/character-creation.txt` is **RED on `main`** now (one `creation` block). Claude re-runs it on your branch and confirms **GREEN** (byte-identical normalized creation transcript). Claude also re-runs the existing green scenarios (their `[setup]` creation is the same flow) to confirm no regression.

## Out of scope
- Non-player-facing internals (how transport/auth is wired) — free to change, but the bytes a player sees are fixed to C.
- Description editor (menu 2), background story (menu 3), delete (menu 5) internals beyond their menu lines — unless trivially in the way.
- The C oracle tree, `website/static/map/world-sphere.json`, `docs/reports/reek/*`.

## Tests you own (deterministic)
- Golden-transcript test of the new-char telnet flow: exact prompt sequence + text through menu `1` (mask the rolled stats).
- Password mismatch re-prompts (does not disconnect); invalid name/sex/race/class re-prompt with C's exact wording.
- `motd` file content equals the C motd (byte compare against a committed fixture).
- No duplicate motd/room emission.

## PR hygiene
- Commits end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- PR body ends with: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`
