# Audit Report: act.comm.c vs Go Communication Engines

**C file:** `src/act.comm.c` (1,567 lines)
**Go file(s):** `pkg/session/comm_cmds.go` (490 lines), `pkg/session/act_comm.go` (160 lines), `pkg/game/act_comm.go` (428 lines), `pkg/game/comm_say.go` (169 lines), `pkg/game/comm_tell.go` (125 lines), `pkg/game/comm_channel.go` (483 lines)
**Mapping type:** 1:N
**Functions audited:** 20 C functions / ~12 Go command entrypoints

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Immersion Syllable-Replacement Engine is Dead Code (Stub Bypassed)
- **Location:** `pkg/game/comm_say.go` (in `doRaceSay`), `pkg/game/act_comm.go` (translation tables), and `pkg/session/act_comm.go` (in `cmdRaceSay`).
- **C behavior:** In `act.comm.c:635` (`do_race_say`), if a character speaks in their racial language (e.g. `RACE_RAKSHASA`, `RACE_ELF`, `RACE_DWARF`), the input string is processed character-by-character using race-specific translation syllable arrays (e.g., `rak_syls`, `elf_syls`, `dwa_syls`). Characters of different races hear the processed fantasy syllables (e.g. "nec arrl fess"), while characters of the same race hear the original plaintext translation with a prefix marker: `Bob says, '(In Elven) Hello there'`.
- **Go behavior:** The Go game-side implementation `doRaceSay` (`pkg/game/comm_say.go`) correctly ports the syllable substitution logic via `applySyllableSubstitution()`. However, this function is **completely dead and un-wired** (marked with `//lint:file-ignore U1000 ... not yet wired to command registry`). Instead, the active player command registry maps the `race_say` command directly to `cmdRaceSay` in `pkg/session/act_comm.go`. This session-side implementation does **no translation or syllable replacement at all**. It simply broadcasts the verbatim plaintext message across the room with a `says in [race]` prefix.
- **Discrepancy:** The MUD's highly immersive fantasy language translation system is completely broken and bypassed. All players hear the plaintext messages of other races immediately without any syllable barrier.
- **Severity:** CRITICAL
- **Type:** STUB / BYPASS

### [FINDING-002]: Critical Quest Quest-Say (`qcomm`) Flag Security Bypass
- **Location:** `pkg/session/act_comm.go:39` in `cmdQcomm()`.
- **C behavior:** In `act.comm.c:1301` (`do_qcomm`), quest-communication commands are strictly gated by the quest participant flag:
  ```c
  if (!PRF_FLAGGED(ch, PRF_QUEST)) {
      send_to_char("You aren't even part of the quest!\r\n", ch);
      return;
  }
  ```
- **Go behavior:** The active session command `cmdQcomm` has absolutely no checks for the `PRF_QUEST` preference flag. It executes global broadcasts to all sessions:
  ```go
  // Broadcast to all online players
  s.manager.mu.RLock()
  for _, sess := range s.manager.sessions {
      if sess.player == nil || sess == s {
          continue
      }
      sess.Send(formatted)
  }
  s.manager.mu.RUnlock()
  ```
- **Discrepancy:** Any player (including level 1 mortals) can abuse the `qcomm` command to perform a global broadcast to every single active connection on the server, bypassing all quest constraints. This represents a severe server-wide chat spam and moderation security boundary vulnerability.
- **Severity:** CRITICAL
- **Type:** SECURITY / BYPASS

### [FINDING-003]: Global Mute Flag (`PLR_NOSHOUT`) Bypassed in Active Commands
- **Location:** `pkg/session/comm_cmds.go` (in `cmdTell`/`cmdReply`), `pkg/session/act_comm.go` (in `cmdWhisper`/`cmdAsk`), and `pkg/game/act_comm.go`.
- **C behavior:** In `act.comm.c`, players flagged with `PLR_NOSHOUT` (representing globally muted or gagged characters) are blocked from performing *all* outbound verbal communications, including tells, replies, shouts, whispers, and asks:
  ```c
  if (PLR_FLAGGED(ch, PLR_NOSHOUT)) {
      stc("You cannot tell anyone anything!\r\n", ch);
      return;
  }
  ```
- **Go behavior:** Go's active session commands (`cmdTell`, `cmdReply`, `cmdWhisper`, `cmdAsk`) have **zero checks** for the `plrNoShout` (or `PlrNoshout`) flag.
- **Discrepancy:** Globally muted or gagged players can continue to tell, reply, whisper, and ask freely, completely rendering admin-applied moderation gags ineffective for target channels.
- **Severity:** HIGH
- **Type:** DRIFT

### [FINDING-004]: Complete Omission of Three Main Communication Channels & Clan-Tell
- **Location:** `pkg/session/commands.go` (command registry) and `pkg/game/comm_channel.go`.
- **C behavior:** In `act.comm.c:1146` (`do_gen_comm`), the channels `SCMD_SHOUT`, `SCMD_GOSSIP`, `SCMD_AUCTION`, `SCMD_GRATZ`, and `SCMD_NEWBIE` are all fully registered and accessible to characters. Additionally, `do_ctell` (`act.comm.c:1451`) provides clan communication.
- **Go behavior:** Go's game layer implements these channels inside `pkg/game/comm_channel.go:308` (`doGenComm` and `doCTell`), but they are **completely missing** from the active player session command registry in `pkg/session/commands.go`. Only `shout` and `gossip` are registered.
- **Discrepancy:** The MUD is missing three out of its six core public communication channels (`auction`, `gratz`, and `newbie`) along with the `clantell` / `ctell` channel.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-005]: Soundproof Room Flags Completely Ignored by Active Commands
- **Location:** `pkg/session/comm_cmds.go` (in `cmdTell`/`cmdReply`/`cmdShout`).
- **C behavior:** In C, rooms flagged with `ROOM_SOUNDPROOF` absorb all sound. Sounds cannot cross the room boundary:
  ```c
  else if (ROOM_FLAGGED(ch->in_room, ROOM_SOUNDPROOF))
      send_to_char("The walls seem to absorb your words.\r\n", ch);
  ```
  This applies to shouts (preventing sending or hearing them), tells, and replies entering or leaving the room.
- **Go behavior:** Active session-side commands (`cmdTell`, `cmdReply`, `cmdShout`, `cmdGossip`) completely ignore the `soundproof` flag. They only exist in the dead/un-wired game-side counterparts (`doTell`, `doReply`, `doShout`).
- **Discrepancy:** Players inside soundproof rooms can shout, tell, reply, and hear shouts/whispers with zero restrictions, neutralizing the tactical and mechanical utility of soundproof zones.
- **Severity:** MEDIUM
- **Type:** DRIFT

---

## Type & Boundary Vulnerabilities

### [FINDING-006]: Invalid Gender Pronoun Usage in AFK Warning
- **Location:** `pkg/game/comm_tell.go:13` inside `performTell()`.
- **C behavior:** In `act.comm.c:928` and `957` (perform tell AFK notice), C utilizes gendered third-person pronouns through formatting tags like `$E` (translates to `he`, `she`, or `it` in `act()`):
  `act("$N is AFK right now, $E may not hear you.", ...)`
- **Go behavior:** Go replaces this with a direct call to the `hisHer` helper function:
  ```go
  ch.SendMessage(fmt.Sprintf("%s is AFK right now, %s may not hear you.\r\n", vict.Name, hisHer(vict.Sex)))
  ```
- **Risk:** Grammatical output boundary error. `hisHer()` returns possessive pronouns `"his"`, `"her"`, or `"its"`. The resulting string sent to the player's telnet buffer is broken:
  `"Bob is AFK right now, his may not hear you."` (should be `"he may not hear you"`).
- **Severity:** MEDIUM
- **Type:** DRIFT / BUG

### [FINDING-007]: Self-Whispering and Lack of Proximity/Dead checks
- **Location:** `pkg/session/act_comm.go:68` inside `cmdWhisper()`.
- **C behavior:** C blocks characters from whispering to themselves with a classic error:
  `"You can't get your mouth close enough to your ear...\r\n"`
  C also blocks characters from sending tells to NPCs that lack descriptors (`IS_NPC(vict) && !vict->desc`) or characters who are asleep/incapacitated.
- **Go behavior:** Go's session-side `cmdWhisper` does not block a player from whispering to themselves. The system will find the player's own session, deliver the whisper to themselves, and broadcast to the room that they whispered to themselves. Proximity loops also lack status checking.
- **Severity:** LOW
- **Type:** DRIFT

---

## Control Flow & Mathematical Fidelity

### [FINDING-008]: Complete Omission of `holler` and Movement Point Exhaustion
- **Location:** `pkg/session/comm_cmds.go` (shout/gossip).
- **C behavior:** In `act.comm.c:1246`, `SCMD_HOLLER` represents a louder form of shouting that spans the entire world instead of being restricted to the zone. Performing a holler charges the character movement points:
  ```c
  if (GET_MOVE(ch) < holler_move_cost) {
      send_to_char("You're too exhausted to holler.\r\n", ch);
      return;
  } else {
      GET_MOVE(ch) -= holler_move_cost;
  }
  ```
- **Go behavior:** Go has completely omitted the `holler` command, and shouts (`cmdShout`) are completely free of charge (never deducting movement points or checking exhaustion).
- **Severity:** MEDIUM
- **Type:** DRIFT

### [FINDING-009]: Clan Tell IMM Rank Filtering & Multi-Target Paging Divergence
- **Location:** `pkg/game/comm_channel.go:250` (in `doPage`), `pkg/session/comm_cmds.go` (in `cmdPage`), and `pkg/game/comm_channel.go:419` (in `doCTell`).
- **C behavior:**
  - **Clan Tells:** Immortals can send a clan tell to any clan via `ctell <clan_num> <msg>`. Furthermore, players can restrict tells to a specific minimum clan rank via `ctell #5 <msg>` (only rank 5+ hears it).
  - **Paging:** Paging in C supports paging everyone if you are godly (`page all <msg>`) and searches via `get_char_vis`.
- **Go behavior:**
  - **Clan Tells:** Go's `doCTell` completely lacks immortal number routing and the `#<rank>` minimum rank targeting filter.
  - **Paging:** Go's active `cmdPage` implements a multi-target argument parser (`page t1 t2 ... msg`) that loops through target names. While useful, this is a distinct control flow extension from the original single-target MUD pager.
- **Severity:** MEDIUM
- **Type:** DRIFT

---

## Concurrency & Mutex Safety

### [FINDING-010]: Data Race Risk on Player Ignoring Maps
- **Location:** `pkg/session/comm_cmds.go` in `cmdTell()` / `cmdIgnore()`.
- **C behavior:** Single-threaded synchronous architecture; inherently thread-safe.
- **Go behavior:** The player ignore system maintains a list/map of ignored player names. Toggled via `cmdIgnore` on a player session, it reads and modifies the player ignoring structures. Meanwhile, `cmdTell` reads `target.player.IsIgnoring(s.player.Name)` concurrently. Because these commands are handled in concurrent goroutines (one for each player session), reading and writing these structures concurrently without holding the player mutex `player.mu` presents a classical data race condition.
- **Severity:** HIGH
- **Type:** CONCURRENCY

---

## Unported Functions

The following legacy C functions from `act.comm.c` have no equivalent behavioral Go implementation in `pkg/session/` or `pkg/game/`:

| C Function | Line | Description | Ported? |
|------------|------|-------------|---------|
| `speak_human` | 553 | Syllable translation table for Humans. | NO |
| `speak_kender` | 220 | Syllable translation table for Kenders. | NO |
| `speak_minotaur` | 387 | Syllable translation table for Minotaurs. | NO |
| `speak_ssaur` | 470 | Syllable translation table for Ssauren (Lizardmen). | NO |

---

## Summary

- **Total findings:** 10
- **Critical:** 2
- **High:** 3
- **Medium:** 4
- **Low:** 1
- **Unported functions:** 4

---

## Verification Plan

### Automated Verification
Run the communication package unit tests and basic compilations to verify baseline safety:
```bash
go build ./pkg/session/...
go build ./pkg/game/...
go test ./pkg/session/...
go test ./pkg/game/...
```

### Manual Verification
1. Log in to two test client sessions.
2. Verify that `/qcomm` allows any player to broadcast server-wide, demonstrating the security vulnerability.
3. Apply `PLR_NOSHOUT` flag manually and verify that the muted player can still send `/tell` and `/whisper`.
4. Test `/race_say` and confirm that other races see the raw plaintext instead of fantasy syllables.
