# Port Fidelity Audit: Module 12 (`act.social.c`)

This audit examines the port fidelity between the legacy C source file `src/act.social.c` and its Go counterparts in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/act.social.c` (305 lines)
- **Functions**: `find_action`, `do_action` (social dispatcher), `do_insult` (gender-specific random insults), `fread_action`, `boot_social_messages` (boots/sorts socials from `SOCMESS_FILE`), `do_dream` (sleeping dream emote).

### Go Port Files
- **Session Commands**:
  - `pkg/session/cmd_social.go` (Active, custom session-layer implementation of `cmdSocial` representing social emotes)
  - `pkg/session/act_social.go` (Misleadingly named; does not contain social commands. It implements `cmdAlias` for Module 13)
- **Game Logic**:
  - `pkg/game/act_social.go` (Uncalled/inactive `DoAction`, `DoInsult`, and `DoDream` implementations)
  - `pkg/game/socials.go` (Hardcoded map containing 800+ lines of faithful static social records parsed from `lib/misc/socials`)
  - `pkg/game/player_social.go` (Misleadingly named; does not contain social commands. It implements general player getters/setters like AFK, AutoGold, ignore, and conditions)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Buggy Session-Layer Emote Parser Bypasses Robust Game-Layer Parser
- **Source Context**: `pkg/session/commands.go#L473-L476`
- **Fidelity Bug**: The Go codebase contains two parallel implementations of the socials engine:
  1. `DoAction` (in `pkg/game/act_social.go`), which is a robust, clean port that delegates to the core `Act()` engine.
  2. `cmdSocial` (in `pkg/session/cmd_social.go`), which is a custom session-layer dispatcher.
  
  The command registry in `commands.go` completely bypasses `DoAction` and delegates all social emotes to `cmdSocial`. As a result, the robust game-layer engine is entirely dead code, and players are forced to use the highly buggy, custom session-layer implementation.

### 2. Inverted Actor / Victim Gender Pronoun Substitutions
- **Source Context**: `pkg/session/cmd_social.go#L26-L50` (`actSubst`)
- **Fidelity Bug**: In `cmdSocial`, the local text replacer `actSubst` replaces lowercase pronouns (`$m`, `$s`, `$e` — representing the **actor's** gender pronouns in CircleMUD/Diku conventions) based exclusively on the **victim's** gender:
  ```go
  actSubst := func(msg string, charName, targetName string, targetSex int) string {
      ...
      switch targetSex {
      case 0:
          msg = strings.ReplaceAll(msg, "$m", "him")
          msg = strings.ReplaceAll(msg, "$s", "his")
          msg = strings.ReplaceAll(msg, "$e", "he")
      ...
  ```
  If a male player types `smile alice` (female victim), the system replaces `$s` ("Zach shakes $s head") with "her" instead of "his" because the victim is female. This completely scrambles pronoun substitutions whenever the actor and victim are of different genders.

### 3. Missing `$E` Pronoun Token Replacement (Visual Glitch)
- **Source Context**: `pkg/session/cmd_social.go#L26-L50` (`actSubst`)
- **Fidelity Bug**: The uppercase `$E` token (representing the victim's subjective pronoun: he/she/it) is completely neglected in `actSubst`. When a social containing `$E` is triggered (e.g. `"$n smiles at you as $E sits down."`), players in-game see the raw `$E` format code printed directly to their screens.

### 4. Mute Penalty Bypass (`PLR_NOSHOUT`)
- **Source Context**: `pkg/session/cmd_social.go#L8-L161` (`cmdSocial`)
- **Logic Gap**: In legacy C (and in `act_comm.go` for chat channels), muted characters (`PLR_NOSHOUT`) are strictly barred from emoting or using socials:
  ```c
  if (PLR_FLAGGED(ch, PLR_NOSHOUT)) {
      stc("You cannot perform emotes!\r\n", ch);
      return;
  }
  ```
- **Fidelity Bug**: The active `cmdSocial` completely ignores the `PlrNoshout` flag. Muted players can happily spam room-wide and victim-targeted social emotes to harass other players, completely bypassing their chat mute.

### 5. Position Validation Bypassed (`min_victim_position`)
- **Source Context**: `pkg/game/socials.go#L4-L9` (`Social`), `pkg/session/cmd_social.go`
- **Logic Gap**: In legacy C, each social defines a `min_victim_position`. If the target is below this position (e.g., dead or sleeping), the social fails with `"$N is not in a proper position for that."`
- **Fidelity Bug**: 
  - The Go `Social` struct lacks the `MinVictimPosition` field, and the hardcoded database in `socials.go` completely discarded this metadata.
  - Neither `DoAction` nor `cmdSocial` checks the victim's position, allowing players to hug, spank, tickle, or dance with sleeping, resting, or dead characters as if they were standing upright.

### 6. Dangerous Substring-Based Target Matching
- **Source Context**: `pkg/session/cmd_social.go#L95-L101` (`cmdSocial`)
- **Fidelity Bug**: When looking up target players in the room, `cmdSocial` checks `strings.Contains(strings.ToLower(p.Name), strings.ToLower(targetName))`. This is a non-standard MUD convention that matches arbitrary substrings. For example, typing `smile exa` will match a player named `Alexander` (since "exa" is in "Alexander"), whereas standard Diku prefix-matching (`strings.HasPrefix`) would correctly ignore it.

### 7. Bypassed Invisibility and Blindness Checks
- **Source Context**: `pkg/session/cmd_social.go#L61-L66` (`sendToRoom`)
- **Fidelity Bug**: The session-layer `cmdSocial` broadcasts raw formatted bytes directly to the room via `BroadcastToRoom`. This completely bypasses all MUD sensory constraints. Invisible players' emotes are seen by everyone, and blind players still "see" visual smile/grin emotes. (The core `Act()` engine in `pkg/game/act.go` correctly handles these, but is bypassed here).

### 8. Unregistered / Unimplemented Core Commands (`insult`, `dream`)
- **Source Context**: `pkg/game/act_social.go#L104-L170`
- **Fidelity Bug**: 
  - The custom command `do_insult` is ported to `DoInsult` in `pkg/game/act_social.go`, but it is **never registered** in `commands.go`, making the `/insult` command completely unavailable to players.
  - The sleep-based social `do_dream` is ported to `DoDream`, but is **never registered**, making `/dream` unavailable.

---

## 3. Secondary Discrepancies & Stubs

### 1. Static Configuration File Bypassed
- **Fidelity Gap**: Legacy C loads and sorts socials dynamically from the authoritative `lib/misc/socials` file at boot via `boot_social_messages()`. In Go, the socials database is hardcoded into `pkg/game/socials.go`. While this makes execution fast, it prevents administrators from modifying MUD emotes at runtime without editing source code and recompiling the server.

### 2. Misnamed Source Files
- **Fidelity Gap**:
  - `pkg/session/act_social.go` contains `cmdAlias` (alias management).
  - `pkg/game/player_social.go` contains general player attributes (AFK, AutoGold, conditions, ignores).
  These files have nothing to do with socials and represent significant naming drift.

---

## 4. Concurrency & Thread Safety

- **Session and World Boundary**:
  - `cmdSocial` operates at the session layer, performing target lookups on the world state. These reads are concurrent with the game ticker. 
  - Because `GetPlayersInRoom` and `GetMobsInRoom` are accessed concurrently, proper locks must be acquired inside world state lookups to prevent race conditions during room mutations.

---

## 5. Summary of Recommended Fixes

1. **Retire `cmdSocial` and Route to `DoAction`**:
   Update `commands.go` to remove the buggy, duplicate session-layer `cmdSocial` implementation. Route all social command lookups directly to `DoAction` in `pkg/game/act_social.go` so that the MUD uses the correct, fully-featured `Act()` engine (which automatically handles gender pronouns, visibility, blindness, and `$E` replacements).
2. **Restore Mute checks**:
   Add a `PlrNoshout` flag check to `DoAction` to prevent muted players from spamming emotes.
3. **Restore `MinVictimPosition` to `Socials`**:
   Add a `MinVictimPosition` field to the `Social` struct, restore this metadata to the hardcoded `Socials` map, and check it against the victim's position before executing the social.
4. **Register Custom Commands**:
   Register `insult` and `dream` commands in `commands.go` and route them to `DoInsult` and `DoDream` respectively.
5. **Rename Files**:
   Rename `pkg/session/act_social.go` to `pkg/session/cmd_alias.go`, and `pkg/game/player_social.go` to `pkg/game/player_settings.go` to restore logical folder architecture.
