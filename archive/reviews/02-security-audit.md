# Security Audit — Dark Pawns MUD

**Date:** 2026-05-03
**Auditor:** BRENDA69 (subagent, security-audit)
**Scope:** Authentication, input handling, Lua sandbox, WebSocket, admin commands, persistence, concurrency, file access
**Severity Scale:** Critical / High / Medium / Low / Info

---

## Executive Summary

Dark Pawns has clear evidence of recent hardening (rate limiters, IP lockout, JWT
required-secret, Origin checks, WebSocket message size cap, force-command
denylist, Lua sandbox neutering `os`, `io`, `package`, `debug`, `dofile`,
`loadstring`, etc.). The Lua sandbox in particular is in good shape, the SQL
layer is uniformly parameterized, and the JWT signer requires a 32+ byte
`JWT_SECRET` with no fallback.

**However, the authentication layer is catastrophically broken.** Two
independent flaws combine to produce a complete account-takeover and
admin-escalation primitive that an unauthenticated attacker can chain in a few
seconds:

1. **Passwords for new accounts are never persisted** — `CreatePlayer`
   silently drops `password_hash`, and the login path treats an empty stored
   hash as "no password required."
2. **Admin authority is decided by player name string match** — registering
   any account named `gm`, `moderator`, or (post-takeover) `zax0rz` grants
   `mute`/`kick`/`ban`/`warn`/etc.

Net result: anyone can become any user (including the operator) and seize
moderation power without ever knowing a password. A second admin path
(`zax0rz`) is currently protected only by name reservation, but the password
bug means knowing the name is sufficient anyway.

There are also several concurrency hazards (unsynchronized `manager.sessions`
reads), a path-traversal exposure in `RunScript` (mitigated only because the
filename source is currently trusted), and a few smaller hardening gaps.

**Prioritized fix order:** auth-bypass (#1) → admin-by-name (#2) → reserved
names (#3) → race conditions (#4) → everything else.

---

## Findings

### [CRITICAL] Finding: New-account passwords are never written to the database (auth bypass)

- **File(s):** `pkg/db/player.go:222-232` (`CreatePlayer`),
  `pkg/db/player.go:243-260` (`SavePlayer`),
  `pkg/session/session_login.go:113-128`,
  `pkg/session/char_creation.go:122`,
  `pkg/db/player.go:194-209` (`GetPlayer` with `COALESCE(password_hash,'')`)
- **Description:**
  - The `players` table defines `password_hash VARCHAR(255)`.
  - `CreatePlayer`'s INSERT lists 25 columns and binds 25 values — none of
    them is `password_hash`. The hash that `session_login.go` computed and
    assigned via `r.Password = string(hashedPwd)` is silently discarded.
  - `SavePlayer`'s UPDATE also never touches `password_hash`, so subsequent
    saves never repair the gap.
  - `GetPlayer` reads `COALESCE(password_hash,'')` into `p.Password`, so the
    stored hash is always the empty string.
  - The login path then conditionally enforces password verification:
    ```go
    if rec.Password != "" {
        // bcrypt compare
    }
    ```
    With `rec.Password == ""` the entire compare is skipped.
- **Impact:** Anyone who connects can authenticate as ANY existing
  player by sending only `{"player_name": "<victim>"}` — no password
  required, no bcrypt comparison invoked, full session granted including a
  signed JWT tied to the victim's identity. Combined with finding #2 and #3
  this yields full server compromise.
- **Proof of Concept:**
  1. Open a WebSocket to `/ws`.
  2. Send `{"type":"login","data":{"player_name":"zax0rz"}}` (no password).
  3. Login succeeds; `JWT` is issued for `zax0rz`; you now act as the operator.
- **Recommendation:**
  1. Add `password_hash` to `CreatePlayer`'s INSERT column list and bind
     `p.Password`. Add a corresponding column to `SavePlayer` only if you
     intend to overwrite it (otherwise leave it to the dedicated
     `cmd_account.go` path which is correct).
  2. In `session_login.go`, **fail closed**: if the loaded record has an
     empty `password_hash`, refuse the login (or force a password reset),
     never silently bypass.
  3. In `char_creation.go:completeCharCreation`, generate the bcrypt hash from
     a password collected during creation (the current flow has no password
     prompt at all) and persist it in the same INSERT.
  4. Add a unit test that creates a player, fetches the row, and asserts
     `password_hash` is non-empty.

---

### [CRITICAL] Finding: Admin authority hard-coded by player name (privilege escalation)

- **File(s):** `pkg/command/admin_commands.go:692-705`
- **Description:** `isAdmin()` returns true if and only if the lowercased
  player name is in a static map `{"admin","zax0rz","gm","moderator"}`. There
  is no level check, no DB-backed role, no audit, and no out-of-band proof
  that the session actually belongs to that operator. The gate is the
  account-name string itself.
- **Impact:** Combined with finding #1, an attacker can simply register —
  or, if already taken, log in to — `gm` or `moderator` and immediately gain
  `warn`, `mute`, `kick`, `ban`, `investigate`, `reports`, `penalties`,
  `filter`, `spamconfig`. Even without finding #1, "admin" is reserved but
  `gm` and `moderator` are NOT in the reserved-names list (see #3) and are
  freely registerable.
- **Proof of Concept:**
  1. Connect to `/ws`, send `{"type":"login","data":{"player_name":"gm","password":"x","new_char":true}}`.
  2. Run `ban <victim> permanent harassment`.
  3. Victim is banned; `notifyAdmins` confirms you as an admin.
- **Recommendation:**
  - Replace the name map with an explicit role/level check sourced from the
    DB (e.g., a `players.is_admin` column or a separate `admin_roles` table),
    AND require `getEffectiveLevel(s) >= LVL_GOD` (or similar) at the same
    time. Two independent gates — name → DB role → effective level — would
    be ideal.
  - Reject `isAdmin` checks while `IsForced` or `isSwitched` if the
    underlying privilege source has been altered.

---

### [HIGH] Finding: Reserved-names list is incomplete; admin-equivalent names are registerable

- **File(s):** `pkg/validation/validation.go:27-32`
- **Description:** `IsValidPlayerName` reserves only `admin`, `system`, `root`,
  `server`, `null`, `undefined`. The hardcoded admin map in
  `admin_commands.go` includes `gm`, `moderator`, and `zax0rz`, none of
  which are reserved. The 2-char minimum allows `gm` exactly. Also, the
  case-insensitive admin lookup pairs with the case-sensitive registration
  path, so `Gm`, `GM`, `gM` all collapse to admin authority on login.
- **Impact:** Trivial admin acquisition (see #2). The same list also fails
  to reserve names that look like wizards (`god`, `implementor`, `imp`),
  staff (`staff`, `dev`), or system actors (`bot`, `agent`).
- **Recommendation:**
  - Drop the name-based admin model entirely (preferred — see #2).
  - If keeping it short-term, add `gm`, `moderator`, `god`, `implementor`,
    `imp`, `staff`, `dev`, `system`, `bot`, and the operator's handle to the
    reserved list, and lift the minimum length to ≥3 to kill `gm`.

---

### [HIGH] Finding: Concurrent reads of `Manager.sessions` without `mu.RLock` (data race / use-after-free)

- **File(s):**
  - `pkg/session/wizard_cmds.go:43-50` (`findSessionByName`)
  - `pkg/session/wiz_communication.go:24-31` (`cmdGecho`),
    lines 122-145 (`cmdForce all`), lines 213-227 (`cmdWiznet @list`)
  - `pkg/session/wiz_system.go:135-145` (`cmdDc all`),
    line 326 (`cmdLast`-adjacent loops)
  - `pkg/session/cmd_info.go:46`, `:79-80`, `:131-132`, `:168` (`cmdWho`,
    `cmdWhere`, `cmdReview`)
  - `pkg/session/comm_cmds.go:169`, `:224` (some gossip/shout paths)
  - `pkg/session/wiz_info.go:25`, `:30` (`Sessions: %d`)
  - `pkg/session/act_comm.go:52`, `:90`, `:136`
- **Description:** `Manager.mu` is a `sync.RWMutex` documented to protect
  `sessions`. `Register`/`Unregister`/`BroadcastToRoom` honor it. Many
  read-side loops do not. Concurrent map iteration with concurrent writes
  in Go is **undefined behavior** — runtime panic
  (`fatal error: concurrent map iteration and map write`) is the documented
  outcome.
- **Impact:**
  - Crash the server by triggering any of the above commands while
    other players are connecting/disconnecting (high-traffic times,
    or an attacker who opens/closes sockets in a tight loop).
  - Stale snapshots are also possible (player listed who has just left,
    being skipped who has just joined), producing inconsistent
    moderation/ban behavior.
- **Proof of Concept:**
  1. Open ~50 short-lived WebSocket connections per second from one IP
     (under the connection cap of 5 means several IPs, but a single client
     opening/closing fast is enough).
  2. From a second client, run `who` repeatedly.
  3. The Go runtime detects concurrent map ops and panics.
- **Recommendation:** Either (a) take `m.mu.RLock()` around every read of
  `m.sessions`, or (b) provide a `Manager.SnapshotSessions()` helper that
  returns a copied slice under the lock, and use it from all callers. Static
  analysis (`go vet -race` in CI, plus `errcheck`/lint pass) should pin this.

---

### [HIGH] Finding: `cmdForce all` holds `m.mu.RLock` while executing arbitrary commands (deadlock risk)

- **File(s):** `pkg/session/wiz_communication.go:120-145`
- **Description:** `cmdForce` (target == "all") acquires `s.manager.mu.RLock()`
  with `defer RUnlock`, then calls `ExecuteCommand` for every session inside
  the loop. Any command that needs a write lock on `m.mu`
  (`Register`/`Unregister`/`UnregisterAndClose`) — including the cleanup
  triggered by a connection drop or `quit` — will block forever, deadlocking
  the server.
- **Impact:** A single `force all quit` (denylist allows `quit`) deadlocks
  the entire MUD until process kill.
- **Recommendation:** Snapshot the session list under the lock, release it,
  then iterate and execute. Same pattern fix as #4.

---

### [HIGH] Finding: `RunScript` script path is concatenated, not joined and bounded

- **File(s):** `pkg/scripting/engine.go:247` (`scriptPath := e.scriptsDir + "/" + fname`)
- **Description:** Only the `dofile` Lua-side helper sanitizes the path
  (`filepath.Clean` + `filepath.Rel` "../" check). The Go-side `RunScript`
  builds the path with raw string concatenation. `fname` is currently
  sourced from `mob.Prototype.ScriptName` (parsed from world files) so
  this is not directly reachable by an unauth'd attacker today, but:
  - if any future code path passes user-influenced data here (e.g., builder
    OLC, mob naming, an `oncmd` redirect), it becomes
    `../../etc/passwd`-style file read via `L.DoFile`, which will
    interpret the file as Lua and at minimum crash/leak via error messages.
- **Impact:** Latent path traversal; hardening required as defense-in-depth.
- **Recommendation:**
  ```go
  full := filepath.Clean(filepath.Join(e.scriptsDir, fname))
  rel, err := filepath.Rel(e.scriptsDir, full)
  if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
      return false, fmt.Errorf("invalid script path: %s", fname)
  }
  ```
  Apply the same to `loadGlobalsOn` (which is fixed-string but consistency
  is cheap).

---

### [MEDIUM] Finding: Communication messages are not stripped of control characters / terminal escapes

- **File(s):** `pkg/session/comm_cmds.go` (all `cmdTell/cmdReply/cmdShout/cmdGossip`),
  `pkg/game/act_comm.go` (whisper)
- **Description:** Player-supplied message text is passed verbatim into
  `Sprintf("...'%s'", message)` and emitted to other sessions inside a JSON
  envelope. The JSON layer escapes byte values, so XSS-on-server is not
  the issue; however, the WebSocket reference client (and any telnet-style
  client) renders the text. There is no filtering of:
  - ANSI/VT100 escape sequences (`\x1b[`...) — terminal injection in raw
    clients (move cursor, clear screen, set window title).
  - Embedded `\r`/`\n` — multi-line spoofing (e.g., faking an admin line
    break before injected text).
  - Long-message DoS (no per-message length cap on `cmdTell`/`cmdReply`;
    only `gecho`/`echo` cap at 500 chars).
- **Impact:** An attacker can craft messages that, when rendered by another
  player's terminal client, redraw the screen, hide content, or impersonate
  server output (e.g., fake "Admin: you have been banned" on the victim's
  display).
- **Recommendation:** Centralize in `filterCommMessage`:
  - Reject or strip `\x1b`, `\x07`, `\x00`, and other C0 controls (allow
    only printable + space).
  - Strip embedded newlines and replace with a space (or reject).
  - Enforce a max length (e.g., 1024 chars) on every comm path.

---

### [MEDIUM] Finding: `/metrics` endpoint exposed without authentication

- **File(s):** `cmd/server/main.go:146`
- **Description:** `http.HandleFunc("/metrics", metrics.Handler().ServeHTTP)`
  serves Prometheus metrics with no auth and no IP allowlist. Metrics are
  not directly sensitive credentials but typically leak: connection counts,
  per-player session counts, error rates, internal queue sizes — useful
  reconnaissance for timing attacks and active-user enumeration.
- **Impact:** Information disclosure; not exploitable on its own.
- **Recommendation:** Either bind `/metrics` to a separate internal listener
  (e.g., `127.0.0.1:9090`) or wrap with `AuthMiddleware` or an
  `internal-IP` middleware. The `nginx.conf` in `web/` should also block
  `/metrics` from public traffic.

---

### [MEDIUM] Finding: Login rate limiter is per-IP only; password attempts not actually limited to victim

- **File(s):** `pkg/session/session_login.go:24-40`,
  `pkg/auth/ratelimit.go`
- **Description:** Both the IP rate limiter and the `LoginAttemptTracker`
  key on IP, not on `(IP, player_name)` or `player_name` alone. An
  attacker with a botnet (or a TOR exit node pool) gets effectively
  unlimited password attempts against one victim, because each IP gets
  its own 10-attempt budget. The `LoginAttemptConfig{Threshold:10,
  Lockout:15min}` is also generous — 10 attempts per 15 minutes per IP
  against any user.
- **Impact:** Online password brute-force feasible for low-entropy
  passwords (the password floor is 4 chars, see `cmd_account.go:33`).
- **Recommendation:**
  - Track failures by `(player_name)` as the primary key with a much
    tighter threshold (e.g., 5 in 15 min), with the IP-level limiter as
    a secondary defense.
  - Raise minimum password length to ≥8 and add a basic character-class
    requirement (or just force ≥10 chars).

---

### [MEDIUM] Finding: Bcrypt hashing on the readPump goroutine — login DoS

- **File(s):** `pkg/session/session_login.go:88-95, 113-120`
- **Description:** Both `bcrypt.CompareHashAndPassword` and
  `bcrypt.GenerateFromPassword` run synchronously inside `handleLogin`,
  which executes on the per-session `readPump` goroutine. Bcrypt at
  `DefaultCost` is intentionally ~80–250ms. The login rate limit (5/sec
  per IP, burst 10) caps a single IP, but **multiple IPs** can each
  pin a CPU core on bcrypt simultaneously.
- **Impact:** Tens of attackers can saturate available cores and starve
  legitimate gameplay (the same goroutines also pump game commands once
  authenticated).
- **Recommendation:**
  - Cap the in-flight bcrypt count via a buffered semaphore channel
    (e.g., `sem := make(chan struct{}, runtime.NumCPU())`).
  - Consider lowering bcrypt cost to 10 (vs the default 10 in
    `golang.org/x/crypto/bcrypt`) only after you understand current
    timings. Don't reduce below 10.

---

### [MEDIUM] Finding: `db.Exec` is exported and accepts arbitrary SQL strings

- **File(s):** `pkg/db/player.go:241-243`
- **Description:** `func (db *DB) Exec(query string, args ...interface{})`
  is exported. Currently the only caller is `cmd_account.go` which uses a
  static parameterized query — safe today. However, the API surface
  invites unsafe use (callers might `fmt.Sprintf` user input into the
  query).
- **Impact:** Latent SQL injection if any future caller composes a
  query with user input.
- **Recommendation:** Replace the generic `Exec` with task-specific
  typed methods (`UpdatePassword(id, hash)`), or at minimum keep `Exec`
  unexported / package-private and add a lint rule banning string
  concatenation/`Sprintf` against the SQL signature.

---

### [LOW] Finding: WebSocket send-channel drops messages silently on full

- **File(s):** `pkg/session/manager.go` (`MessageSink`, `BroadcastToRoom`),
  `pkg/session/comm_cmds.go` (`cmdShout`)
- **Description:** When a session's `send` channel (capacity 256) is full,
  the broadcaster does `default:` and drops the message. A misbehaving or
  slow client can therefore cause other players to miss combat damage,
  death notifications, or moderation actions. There's no per-connection
  backpressure or kick on slow consumers.
- **Impact:** Reliability + minor moderation evasion (a victim's client
  could legitimately not see a `mute` notice).
- **Recommendation:** When a send drop happens, log + increment a metric;
  if drops cross a threshold within a window, close the connection and
  unregister the session.

---

### [LOW] Finding: `fmt.Sprintf` in user-facing messages without bounds on inputs

- **File(s):** `pkg/session/comm_cmds.go` (Tell/Reply unbounded `args`),
  `pkg/session/cmd_account.go:113` (`Sprintf("Prompt set to: %s", arg)` —
  prompt string can include `%h/%m/%v` macro tokens that are
  later format-substituted; needs review for double-interpretation).
- **Description:** The custom prompt string is stored as-is and later
  expanded; no cap on length or rejection of unmatched `%` tokens. Also,
  a player's `PromptStr` is inherent to the session and shown to
  themselves, so impact is mostly self-inflicted, but worth reviewing.
- **Recommendation:** Cap prompt length, reject unknown `%X` codes, and
  generally cap user-supplied free-text inputs.

---

### [LOW] Finding: `wizlock` permission check uses `s.player.Level` instead of `getEffectiveLevel(s)`

- **File(s):** `pkg/session/wiz_system.go:103-105`
- **Description:** The level-cap on `wizlock <N>` reads `s.player.Level`
  directly. A wizard who is `switched` retains their original level for
  permission checks via `getEffectiveLevel`, but `wizlock` instead caps
  to the *current* (possibly lower) level after switch. Not a clear
  vulnerability — the gate `LVL_IMPL` already protects entry — but the
  inconsistency invites future bugs.
- **Recommendation:** Use `getEffectiveLevel(s)` everywhere a level
  comparison happens.

---

### [LOW] Finding: `force` denylist is exact-match; alias-resolved commands bypass

- **File(s):** `pkg/session/wiz_communication.go:108-117`
- **Description:** The denylist checks the literal first token against
  `force/shutdown/purge/set/advance/switch/wiznet`. The command registry
  supports aliases (`registerWithAliases`), so a player-defined alias or
  a built-in alias for any of those would slip past. Also, the registry
  lookup is case-insensitive but the denylist comparison is case-folded
  via `ToLower` — that part is fine.
- **Recommendation:** After resolving the command via `cmdRegistry.Lookup`,
  test the canonical command name against the denylist (and also test
  for `social` matches that might trigger emote spam at a scale a wizard
  shouldn't be able to deputize).

---

### [INFO] Finding: Initial `password_hash VARCHAR(255)` is fine for bcrypt today, but no migration tracking

- **File(s):** `pkg/db/player.go:75-110`
- **Description:** Schema migrations are inline `ALTER TABLE ... ADD
  COLUMN IF NOT EXISTS` with no version table. Hard to roll back; hard to
  reason about state in shared envs.
- **Recommendation:** Adopt `goose`/`migrate`/`atlas` and version the
  schema explicitly.

---

### [INFO] Finding: Lua sandbox is reasonably tight

- **File(s):** `pkg/scripting/engine.go:35-100`
- **Positive:** `os.execute`, `os.exit`, `os.getenv`, `os.remove`,
  `os.rename`, `os.tmpname`, `string.dump`, `math.randomseed`,
  `package`, `debug`, `io`, `dofile` (raw), `loadfile`, `load`,
  `loadstring` are all nilled. `dofile` is re-exposed only via a
  sandboxed wrapper that path-checks via `filepath.Rel`. Script
  execution is wrapped in a 5s `context.WithTimeout` and the LState is
  recreated on panic.
- **Residual concerns:**
  - The single shared `LState` is mutex-serialized, so a script that
    panics before `recover` can still leak globals across runs (the
    code does recreate the state, which is correct — keep doing so).
  - `string.rep` is not capped — `string.rep("a", 1e9)` will allocate
    and OOM the process. `lua-gopher` doesn't currently expose a
    memory limit; consider replacing `string.rep` with a sandboxed
    version.

---

## Positive Findings

- **JWT signing requires a 32+ byte secret**, no default fallback (`pkg/auth/jwt.go`).
- **Origin checking on WebSocket upgrades** in production with
  documented allowlist (`pkg/session/manager.go`).
- **Per-IP connection cap of 5** + per-IP login rate limit + per-IP
  failure lockout (15 min after 10 failures).
- **`SetReadLimit(16384)` and `SetReadDeadline` on every read**, plus a
  54s ping/pong heartbeat — good DoS posture for the WebSocket itself.
- **All SQL queries** in `pkg/db/`, `pkg/moderation/`, and
  `pkg/storage/sqlite.go` use parameterized placeholders (`$1`, `?`).
  No `Sprintf`-into-SQL anywhere in the active code.
- **Force command** has thoughtful safety: denylist, no-transitive,
  3s cooldown, level-gate via `getEffectiveLevel`.
- **Switch/return** preserves the wizard's original level for permission
  checks; auto-return on disconnect prevents stuck-in-mob exploits.
- **House save filenames** use integer vnums, not user names.
- **`sanitizeName` for player save files** strips path separators and
  dots — defense-in-depth even though the validation regex `^[a-zA-Z0-9_]+$`
  already forbids them.
- **HTTP server has explicit `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`** —
  Slowloris is mitigated.
- **Trusted-proxy parsing** for `X-Forwarded-For` is correctly bounded
  to the configured CIDRs only.

---

## Recommendations Summary (priority order)

1. **(Critical, ship today)** Fix `CreatePlayer` to persist `password_hash`,
   add a `password_hash != ''` precondition on returning-player login,
   and add a regression test. Also add a one-off migration to flag every
   existing row with an empty hash and require those users to reset
   passwords on next login.
2. **(Critical)** Replace name-based `isAdmin` with a DB-backed role +
   level check; revoke `gm` / `moderator` / `zax0rz` from the static map.
3. **(High)** Expand `validation.IsValidPlayerName` reserved list and
   raise minimum length to 3.
4. **(High)** Add `m.mu.RLock()` (or a `Snapshot()` helper) to every
   `Manager.sessions` read site; add `go test -race` to CI.
5. **(High)** Snapshot sessions in `cmdForce all` and outside of the
   manager lock.
6. **(High)** Path-clean and bound `RunScript` script paths.
7. **(Medium)** Strip C0/escape characters and cap length on all comm
   commands.
8. **(Medium)** Move `/metrics` to a separate internal listener or
   protect with `AuthMiddleware`.
9. **(Medium)** Track login failures per `(player_name)`; bump min
   password length; cap concurrent bcrypt operations.
10. **(Medium)** Replace `db.Exec` with typed methods or restrict its
    use via lint.
11. **(Low)** Drop or close slow-consumer sessions instead of silently
    dropping their broadcasts; cap prompt length; align `wizlock` and
    other gates to `getEffectiveLevel`.
12. **(Info)** Adopt a real schema migration tool; consider sandboxed
    `string.rep` to cap Lua-side memory.

---

*End of audit.*
