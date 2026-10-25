# Harness vehicles for the unproven lifecycle systems — design

Status: **design only, stop tier.** This PR adds this file alone. It changes no
harness code, no scenario, no seam, and no oracle bytes. Rules: R1, R2, R3a–R3d,
R4, R5e, R5f, R5g, R5h. A separate build brief implements whatever Zach approves
here.

Scope: the blocked `lifecycle` and `shops` rows of
`docs/fidelity/depth/surface-inventory.tsv` (terminal audit 2026-09-04) plus the
login/creation fixture parked as harness stop tier in
`docs/fidelity/depth/handoff/2026-10-21-deepseek-round12-final-draw-rows.md` §3.

**Headline conclusion: no C seam change is needed.** The `~dpclock pulse` seam
already reaches mud-hour scale (630 pulses = one mud hour, proven green at 31
hours), `<RESTART>`/`<CRASH>`/`<RELOGIN>` already reach persistence, and peers +
`keep-prompts` already reach audiences and prompt bytes. The remaining blockers
are (a) two small **harness-side** fixtures (a Go player-store seed and a
per-scenario `DP_FIXED_TIME` override) and (b) scenario breadth. The single
exception the design cannot close by pumping: mud-year rollover (§2.1).

---

## 1. Baseline: what the harness already does

Citations from `src/` are R5g; the seam lives only in the oracle tree
(`~/darkpawns-c-oracle/src/`, checked out on `dp-oracle-seam`) and is cited as
such. Harness citations are from this branch (`4fc5c8ab2`).

### 1.1 Clock

- The heartbeat runs the hourly block every
  `SECS_PER_MUD_HOUR * PASSES_PER_SEC` pulses: `weather_and_time(1)`,
  `affect_update()`, `point_update()`, `hunt_items()`
  (`src/comm.c:824-830`). `SECS_PER_MUD_HOUR = 63` (`src/utils.h:135`),
  `PASSES_PER_SEC = 10` (`src/structs.h:629`), so **one mud hour = 630 pulses**;
  one mud day = 15,120; one mud month = 529,200; one mud year = 8,996,400.
- Pulse cadence: `PULSE_ZONE` 100, `PULSE_MOBILE` 40, `PULSE_VIOLENCE 20`
  (`src/structs.h:632-634`); heartbeat ordering is
  events → extract → zone (100) → idle-password check (150) →
  mobile/room/object activity (40) → violence (20) → **hourly block (630)** →
  autosave (600) (`src/comm.c:800-840`).
- The seam: `DP_CLOCK=1` gates the heartbeat on a pumped queue;
  `~dpclock pulse N` queues up to `DP_CLOCK_MAX_PENDING = 100000` pulses per
  line (oracle-tree `src/comm.c:100`, `:1899-1920`), drained back-to-back in the
  loop's heartbeat slot after descriptor I/O. `DP-1368` made `WAIT_STATE`
  decrements count only in pumped pulses (oracle-tree `src/comm.c:489-530`), so
  waits are deterministic under the frozen clock (R3c).
- The Go mirror is `pkg/session/clock_control.go:12-55`.
- **Boot calendar:** the harness pins both engines with
  `DP_FIXED_TIME=1770838461` ⇒ hour 14, day 17, month 8, year 1245
  (`cmd/dp-oracle-diff/main.go:31-44`; C reads it in `db.c` boot, oracle-tree
  `src/db.c:419-431`). It pins the calendar only; time still advances by pumped
  pulses only.
- **Proven ceiling:** `lifecycle-idle-terminal-weather` and five siblings each
  pump **19,530 pulses (31 mud hours) in 31 control lines** and are
  `oracle-green-multiseed` (`docs/fidelity/depth/idle.tsv` rows 11–15). A
  control line costs one send + one ~300 ms quiescent read regardless of N, so
  cost scales with line count, not pulse count. 311 of 1,610 scenarios already
  use `~dpclock`.

### 1.2 Persistence, reconnect, audiences, prompts

- Scenario sections: `[setup:oracle]`/`[setup:port]`, `[creation:*]` (diffed),
  `[relogin:*]`, `[warmup]` (discarded), `[probe]` (diffed), per-peer setups,
  `[fixture]` (`internal/oraclediff/scenario.go:314-729`).
- Probe control steps: `<RELOGIN>` (same process, new connection),
  `<RESTART>` (SIGINT→SIGKILL bounce), `<CRASH>` (immediate SIGKILL, no grace)
  — each at most once, all requiring the relogin sections
  (`scenario.go:688-717`). `send:<peer> <line>` directs input to one peer
  (`scenario.go:908-947`); `peer-drop` drops a peer mid-scenario.
- Prompt bytes: the ordinary normalizer drops prompt lines, so prompt parity
  needs `keep-prompts` scenarios, and the first game-entry prompt needs
  `no-settle` (R5f, DP-1326; `RULEBOOK.md:174-178`). Both flags exist.
- **Go player store:** scenarios with relogin lines (or `stat file`/`set file`
  steps) get a real SQLite store seeded with one placeholder player
  (`Zzplaceholder`, level 1, room 8004) so the first real login is not crowned
  God (`main.go:1174-1189`, `main.go:2066-2083`). All other scenarios run fully
  in-memory (`DP_ALLOW_NO_DB=1`, dead store path, `main.go:37-44`). C's pfile is
  not seeded by the harness: `Testor` (immortal Warrior, player ID 1) ships in
  the C checkout's `lib/etc/players`; `empty-players` truncates the copy
  (`main.go:1030-1037`).
- Already-green lifecycle proofs (the idle lane, `docs/fidelity/depth/idle.tsv`):
  void on the ninth hour and its audiences (`lifecycle-idle-void`,
  `lifecycle-idle-observer`), return-on-command, command timer reset, idle
  force-rent — link-loss act, transport close, no-menu handoff, post-close
  relogin from the rent file (`lifecycle-idle-force-rent-*`), the terminal
  weather byte at close (`lifecycle-idle-terminal-weather[-burst]`), and rent
  round-trip inventory/equip-NORENT as unit-green session tests
  (`TestIdleRentRoundtripInventory`, `TestIdleRentRoundtripEquippedNoRent`).

### 1.3 Census cost envelope

- Full census: 1,610 scenarios, seed 1, 36 workers, **1,102 s wall, CLEAN** —
  `~/Archives/darkpawns/oracle-runs/2026-10-25/sol-parallel-profile-full/`.
- Combined census: 5,288 claim pairs deduplicated to 1,448 + full leg,
  **4,153 s**, CLEAN_AFTER_RECHECK —
  `~/Archives/darkpawns/oracle-runs/2026-10-23/v014-release-6fbe6fa9e/`.
- Per-scenario census timeout: **240 s** (`scripts/oracle_regression.sh:16`,
  worker `--kill-after=10s`). This bounds pumping: ~40 control lines
  (~25k pulses ≈ 40 mud hours) is comfortably inside; the proven max is 31
  lines. Longer travel must use the `fixed-time` fixture (§2.1), not more
  lines.

---

## 2. Per-row design

Each section answers the brief's six questions: what C does (cited), why the
harness can't reach it today, the vehicle, how it can fail (R5h), its cost, and
its place in the order.

### 2.1 Weather and time pulses

**What C does.** `weather_and_time(1)` runs `another_hour` then
`weather_change` per mud hour (`src/weather.c:31-37`).

- `another_hour` advances `time_info.hours` and, at hours 5/6/21/22, sets
  sunlight and sends the outdoor world: `"The suns rise in the east and
  north."` (hour 5, `src/weather.c:58`), `"The day has begun."` (6,
  `src/weather.c:64`), `"The suns slowly disappear in the west and south."`
  (21, `src/weather.c:68`), `"The night has begun."` (22,
  `src/weather.c:80`); hour 21 also loads the night gate / ghost ship,
  and days 22–25 with a full moon fire `full_moon()`/`lunar_hunter()`
  (`src/weather.c:56-77`). Calendar rollover: hours>23 → day++, day>34 →
  month++, month>16 → year++ (`src/weather.c:86-124`). Recipients are all
  awake outdoor players (`send_to_outdoor`, `src/comm.c:2353-2365`).
- `weather_change` draws **5 `number()` per mud hour** —
  `dice(1,4)*diff + dice(2,6) - dice(2,6)` (`src/weather.c:137`) — plus at
  most one conditional `dice(1,4)` (`src/weather.c:155-188`), and emits the
  six sky-transition strings ("The sky starts to get cloudy." …
  "The lightning stops.", `src/weather.c:198-228`). Seasonal pressure bias:
  months 9–16 threshold 985, else 1015 (`src/weather.c:132-135`).
- Boot rolls the initial pressure `dice(1,50)`/`dice(1,80)`
  (`src/db.c:449-451`).

**Why today's harness can't reach it.** It can pump hours (§1.1), and the idle
lane proved one terminal weather byte. What no green scenario covers: the four
sunlight strings, sky transitions, moon phases, calendar rollovers, and the
seasonal pressure bias. The only hard wall is the **year rollover**: 17 months
of pumping ≈ 8.99M pulses ≈ 14,280 control lines — impossible inside the 240 s
scenario timeout, and `DP_FIXED_TIME` is currently harness-global, not
per-scenario. Month/day/hour rollovers are reachable either by pumping (a day
is 24 lines) or cheaply by booting near the boundary.

**Vehicle.** New scenarios only, plus one **harness fixture**:

- `fixed-time <unix-seconds>` — a `[fixture]` directive setting `DP_FIXED_TIME`
  for that scenario on both engines (the env var is already read by both;
  `main.go:31-44` default remains for scenarios that don't set it). Harness-only
  change; default-off, so existing scenarios are untouched.
- `weather-sun-cycle`: outdoor actor (goto an outdoor room), pump 630-pulse
  steps from hour 14 to hour 6 of the next day (16 steps, 10,080 pulses),
  comparing each hour block separately — the discipline already proven by
  `lifecycle-idle-terminal-weather`. Covers sunset 21, night 22, rollover at
  24, sunrise 5, day 6.
- `weather-sky-transitions`: multiseed (1,2,3,5,8); per-hour blocks until each
  of the six sky strings has appeared at least once across seeds; draw-parity
  claim counting the 5/hour stream draws (R3a).
- `weather-calendar-day-month`: `fixed-time` boots at day 34 (and a second
  scenario at month 16); pump 1–3 hours to cross day→month and month→year
  boundaries; the receiving strings are the sun/hour bytes around the rollover
  plus `time`-command output in the probe.
- `weather-seasonal-bias`: `fixed-time` at month 8 vs month 16, each pumping
  N hours with draw-parity on, showing the 985/1015 threshold divergence in
  the draw stream (the strings themselves may coincide; R5a says the draw
  stream is the proof).

Year rollover is **not proposed**: unreachable by pumping within timeout;
would need a seam calendar verb, which is Zach's call and explicitly not
requested here.

**R5h.** Plant: change Go's `weather_change` draw order (swap the two
`dice(2,6)` terms) or one sky string. The per-hour block compare reddens at the
first affected hour in every seed; the draw-parity claim catches the
value-preserving swap that transcript-only comparison would miss (R5a/R3a).

**Cost.** 4–6 new scenarios, all ≤ 24 control lines; roughly +1–2% full
census. No existing scenario changes (`fixed-time` is opt-in).

### 2.2 Regeneration, hunger, thirst pulses

**What C does.** All in `point_update` (`src/limits.c:460`), per mud hour, in
order: hunger/thirst/drunk −1 (`gain_condition`, `src/limits.c:476-480`);
regen applies `hit_gain`/`mana_gain`/`move_gain` gated on position ≥ stunned
(`src/limits.c:493-502`); poison ticks `damage(i,i,10,SPELL_POISON)`
(`src/limits.c:503-504`); incap/bleed-out ticks 1/2 damage
(`src/limits.c:509-512`). Strings: `"Your stomach growls with hunger."` /
`"You feel a bit parched."` at value 1, `"You are hungry."` /
`"You are thirsty."` at 0 (`src/limits.c:386-406`); immortals have cond −1 and
never decay (`src/limits.c:368-370`, set at `src/class.c:705-707`). Zero
FULL/THIRST quarters the gains (`src/limits.c:182-183` etc.) — no starvation
death exists. The gain formulas themselves (`src/limits.c:59-243`) contain
**no RNG**; the only point_update draws are NPC memory `number(0,98)`
(`src/limits.c:515-518`) and corpse-rot flavor `number(0,6)`
(`src/limits.c:600-641`), both avoidable with `quiet-mobs` and no corpses.

**Why today's harness can't reach it.** Nothing blocks it but scenario
absence: the actor must be a **mortal** (immortals never decay), pumped ≥ 25
hours to walk FULL 24→0, with per-hour blocks. No existing scenario drives a
mortal through hunger/thirst messages or observes regen state.

**Vehicle.** New scenarios only:

- `regen-hunger-thirst-cycle`: mortal (demote after creation, the pattern the
  zen fixture already uses), `quiet-mobs`, pump 26 hourly steps; hour blocks
  carry the growls/parched/hungry/thirsty lines in order; probe `score`/`stat`
  after to pin the regen-quartered state (state claim via command surface).
- `regen-position-gains`: unit tests pin the exact `hit_gain`/`mana_gain`/
  `move_gain` arithmetic per position (sleeping/resting/sitting/standing),
  hunger-quartering, and class multipliers (D4 — state not fully visible),
  plus one oracle scenario proving the sleeping-vs-standing byte difference
  via `score` deltas across two pumped hours.
- `regen-poison-bleed-ticks`: a poisoned mortal pumped one hour (poison 10
  damage line), and an incap actor pumped to the 1/2-damage bleed-out
  (`src/limits.c:509-512`) — the death edge of this row deliberately stops
  before death (owned by §2.6).

**R5h.** Plant: change one hunger string, or make Go's `hit_gain` use
standing gain while sleeping. The hour-block compare reddens on the string;
the unit test reddens on the arithmetic (revert-watched per R5h).

**Cost.** 2–3 scenarios (≤ 30 lines each) + unit tests; ≈ +1% census. No seam,
no harness change.

### 2.3 Player save/load round-trip

**What C does.** Saves: `save_char` writes the fixed-size `char_file_u` record
into `lib/etc/players` (`src/db.c:2366-2406`); equipment/inventory goes to
`plrobjs/<bucket>/<name>.objs` via the `Crash_*` family — `Crash_crashsave`
(player `save`, `src/act.other.c:186-203`), `Crash_rentsave` (quit in a save
room / idle force-rent, `src/objsave.c:911-956`), `Crash_cryosave`; autosave
every 10 min (`src/comm.c:832-838`). Load: `store_to_char` restores, and
**full-heals if the player was away ≥ 1 real hour** (`src/db.c:2487-2493`);
`Crash_load` restores objects, auto-equips worn items, rebuilds container
nesting, debits rent days, and on unaffordable rent rewrites the file and
returns the "Salvation Army" total-loss code 2 (`src/objsave.c:455-670`,
message at `src/interpreter.c:2226-2229`). Boot prunes stale rent/crash files
(`update_obj_file`, `src/objsave.c:203-271`). Login of an existing player sets
`wait = 1` after the name (`src/interpreter.c:1766`).

**Why today's harness can't reach it.** Mostly reached already, narrowly:
`<RESTART>`/`<CRASH>`/`<RELOGIN>` exist, and the idle lane proved the
force-rent→menu→relogin round-trip from the rent file. Unproven: the pfile
round-trip across a **restart** (stats/affects/loadroom through
`char_to_store`→`store_to_char`), the crash-file restore path after `<CRASH>`,
the away-≥1-real-hour full-heal rule, boot-time rent-file pruning, and the
player-visible `save`/`quit` branches (save-room quit vs `REALLYQUIT` eq loss,
`src/act.other.c:87-182`). Note `free_rent = YES` (`src/config.c:106`) makes
rent cost 0 on every reachable save path, so the **unaffordable-rent /
Salvation Army branch is dead in the shipped config** — the vehicle cannot and
should not reach it (R4; §3).

**Vehicle.** New scenarios only, on the existing restart machinery (the store
appears automatically because these scenarios have relogin sections,
`main.go:2066-2083`):

- `saveload-restart-pfile`: create mortal with known stats/eq, `save`, probe,
  `<RESTART>`, relogin, probe `stat self`/`inventory` — byte compare of the
  relogin menu (`config.c:209-219` MENU) and restored state.
- `saveload-crash-restore`: same, but `<CRASH>` (no grace) — the `.objs` file
  stays `RENT_CRASH`, restores on relogin; pins that the boot prune
  (`no_rent_check` unset) does not delete a fresh crash file.
- `saveload-away-heal`: damage self to low HP, `save`, `<RESTART>`, relogin —
  the away-≥1-real-hour rule needs **real** elapsed time on C's `time(0)`;
  the seam freezes pulses, not `time(0)` (oracle-tree `src/db.c:419-431`
  pins the calendar only). Design: run the relogin after the scenario's own
  restart (boot gap ≈ seconds) to prove the **non**-heal branch (fresh
  relogin keeps low HP); the heal branch is a documented unit test
  (`store_to_char` arithmetic) because the oracle cannot wait an hour — the
  boundary is stated, not papered over (R5f).
- `quit-save-room` and `quit-reallyquit-lose-eq`: command-surface scenarios
  (no clock), pinning `do_quit`'s three exits (`src/act.other.c:87-182`).

**R5h.** Plant: drop one restored inventory item, or heal on a short-gap
relogin. The relogin probe block reddens; the unit test reddens on the
threshold.

**Cost.** 4–5 scenarios, all light (≤ 4 control lines each; restarts cost
~20 s of the static scheduler weight each per `oracle_regression.sh:64-77`).
No seam, no harness change.

### 2.4 Descriptor link/prompt lifecycle

**What C does.** Link loss in `close_socket` for a playing character: save,
room act `"$n has lost $s link."`, `desc = NULL` — **the body stays in the
world** (`src/comm.c:2129-2134`); there is **no link-dead purge timer** — the
idle path (§1.2, already green) is the only remover. Reconnect before removal
re-attaches via the dupe check (`"Reconnecting."` / `"You take over your own
body, already in use!"`, `src/interpreter.c:1528-1660`). Login-state timeouts
fire from `check_idle_passwords` every 150 pulses: `"\r\nTimed out...
goodbye.\r\n"` (`src/comm.c:2170-2187`) — under `DP_CLOCK` this is **pumped,
deterministic time**. Bad passwords: 3 tries then disconnect
(`src/interpreter.c:1882-1884`). Prompts: `make_prompt` composes `<n>H <n>M
<n>V > ` gated on PRF flags, `i<n>` for invis, AFK/INACTIVE tags, empty for
all non-playing states (`src/comm.c:1028-1207`). **Zero RNG** on this whole
surface.

**Why today's harness can't reach it.** Link-loss bytes, re-attach, password
timeouts, bad-password disconnect, menu options 0/2–5, and prompt bytes have no
green scenarios. `lifecycle-reconnect-linkdead` covers one reconnect slice;
prompts are normalized away by default (R5f/DP-1326).

**Vehicle.** New scenarios only, using existing machinery (`peer-drop`,
peers, `keep-prompts`, `no-settle`, pumping):

- `descriptor-linkloss-world`: actor + same-room peer; `peer-drop` the actor;
  peer's block proves `"$n has lost $s link."`; the link-dead body is then
  attacked/killed by a spawned mob to prove it persists in-world (death bytes
  owned by §2.6; this scenario stops at the corpse).
- `descriptor-reconnect-reattach`: two connections, second login of the same
  name usurps: `"Reconnecting."` to the new connection, `"You take over your
  own body, already in use!"` to the old (peers; `send:<peer>`).
- `descriptor-password-timeout`: connect, give name only, pump ≥ 300 pulses —
  `Timed out... goodbye.` arrives deterministically at the second 150-pulse
  check (`src/comm.c:2178-2185`).
- `descriptor-badpassword`: wrong password ×3 → mudlog + disconnect bytes.
- `descriptor-menu-options`: after death or quit-to-menu, options 0, 2, 3, 4,
  5 (invalid-choice re-print at `src/interpreter.c:2284-2285`).
- `descriptor-prompt-bytes`: `keep-prompts` + `no-settle`; PRF toggles
  (`preferences`) walk the H/M/V/invis/AFK compositions; compare raw prompt
  lines.

**R5h.** Plant: drop the `i<n>` invis tag or change the timeout grace tick.
The prompt compare / pumped-timeout block reddens on the exact bytes.

**Cost.** 5–6 light scenarios; < +1% census. No seam, no harness change.

### 2.5 Rent and house lifecycle (+ rent/storage shop-adjacent)

**What C does — and what is dead.** Four R5e reachability findings bound this
row (all verified in `src/`):

1. **The receptionist/cryogenicist specials are never assigned.** Declared
   (`src/spec_assign.c:89-90`) but no `ASSIGNMOB` exists for them, so
   `gen_receptionist` (`src/objsave.c:1123-1196`), `do_rent`/`do_offer`
   prices, and their strings are unreachable; `rent`/`offer` fall through to
   `do_not_here` ("Sorry, but you cannot do that here.",
   `src/act.other.c:210`).
2. **`free_rent = YES`** (`src/config.c:106`): the reachable rent paths
   (save-room quit, idle force-rent) cost 0, so charge/loss arithmetic
   (`Crash_offer_rent`, `Crash_load` debits) never reaches a player.
3. **House contents never persist:** every `House_crashsave` caller is
   commented out and `House_save_all` is a stub (`src/house.c:709-721`;
   `src/comm.c:836`, `src/act.other.c:201-202`) — `house/<vnum>.house` files
   are read at boot (`src/house.c:70-107`) but nothing writes them.
4. **House entry gating is commented out:** `House_can_enter`
   (`src/house.c:725-744`) has no live caller; `do_simple_move` has no house
   check. Only NPC guards and `key_seller` enforce ownership
   (`src/spec_procs2.c:1845-1932`).

Live surface: `hcontrol build/destroy/pay/show/key` (immortal,
`src/house.c:323-598`), the owner `house guest/transfer` command
(`src/house.c:603-702`), quit/REALLYQUIT eq handling (§2.3), idle force-rent
(already green), and the key_seller shop special (blocked on the login
fixture).

**Why today's harness can't reach it.** House ownership needs **player ID 1**
(`house-control <vnum> <exit> <player>` resolves the name to an id,
`src/house.c:386-471`); today C logs in the pre-seeded ID-1 `Testor` while Go
creates a fresh character — the login fixture (§2.9). The harness has a
`house-control` fixture (`scenario.go:472-491`, `main.go:1081-1166`) with a
port-owner override, but the draw streams still diverge at creation.

**Vehicle.**

- The **login fixture** (§2.9) is the prerequisite; it unblocks the
  house/rent owner surface.
- `house-owner-guests`: seeded Testor owns a house (house-control fixture);
  `house guest` add/list/delete/cap bytes (`src/house.c:603-702`); a second
  mortal peer proves guest vs non-guest visibility where any exists.
- `house-transfer-key-hcontrol`: immortal `hcontrol build/show/pay/key`
  report bytes (the `-O0` restored forms, per the PR #1682 history) +
  `house transfer`.
- `house-contents-not-saved`: drop an item in a house, `<RESTART>`, relogin —
  **prove C's non-persistence byte-for-byte** (the item is gone after reboot
  because nothing wrote a house file). This is R4 fidelity to dead behavior,
  not an invention.
- Dead-code labeling (no scenarios): receptionist rent surface, rent-charge
  branches, and `House_can_enter` recorded in the manifest as `excluded`
  (unreachable; owning surface: unassigned special / commented-out caller),
  per DEPTH_TESTING's explicit-status rule — **Zach approves the labels**.

**R5h.** Plant: make Go persist house contents across restart, or gate entry
Go-side. The `house-contents-not-saved` restart block reddens (items present
after reboot); the entry-gate plant reddens any movement scenario into the
house. A label-only row has no green to fake.

**Cost.** 3–4 scenarios + labels; light. Prerequisite: §2.9 harness change.

### 2.6 Death cycle and corpse teardown

**What C does.** Kill branch in `damage()`: `"$n is dead!  R.I.P."` to the
room and `"You are dead!  Sorry..."` to the victim (`src/fight.c:1582-1583`);
exp/loot bookkeeping; `die_with_killer` takes `exp/37`, rolls CON loss
`number(0,3)` (level>5) and nested `number(0,5)` (level>20, only if the first
hit) (`src/fight.c:586-610`); `raw_kill` strips affects/tattoo/forms, emits
the death cry to the room and neighbors (`src/fight.c:506-529`), builds the
corpse — `"<name> corpse"`, `ITEM_CONTAINER`, `val3=1`, no `WEAR_TAKE`,
`ITEM_NODONATE`, inventory+equipment+gold moved inside, timer 10 mud hours
(PC) / 5 (NPC) (`src/fight.c:258-428`, timers at `src/config.c:85-86`) — then
`extract_char`. For a PC, `extract_char_final` saves the pfile, deletes the
crash file, and hands the descriptor `CON_MENU` + MENU bytes
(`src/handler.c:1160-1174`). Menu '1' re-enters: `reset_char` floors HP/MA/MV
at 1 and stands the body up (`src/db.c:2936-2970`), `Crash_load` (empty),
`WELC_MESSG`, placement at `GET_LOADROOM`, `"$n has entered the game."`,
`look_at_room` (`src/interpreter.c:2173-2222`). Corpse teardown: one
`number(0,6)` flavor draw per decaying corpse at timer 0, contents dumped to
the floor (`src/limits.c:590-650`). The CON-death lockout
("You are too unhealthy to play!") is at `src/interpreter.c:2176-2181`.

**Why today's harness can't reach it.** Nothing structural: mob deaths and
combat-deep lanes are heavily green; what's missing is the **player** death →
menu → re-entry arc (needs a mortal killed deterministically and prompt-frame
comparison through the menu) and corpse decay (needs ~10 mud hours of pumping
plus the `number(0,6)` draw pinned by seed). Kill-blow RNG (retarget/dismount
rolls, `src/fight.c:1411-1552`) makes these draw-sensitive → multiseed.

**Vehicle.** New scenarios only:

- `death-player-menu-reentry`: mortal with known eq fights a spawned stronger
  mob (`spawn-mob`, quiet zone) until death; blocks: death bytes → MENU → '1'
  → welcome/entry/look; probe `score` for the exp/37 loss and HP floor;
  CON-loss draw parity via multiseed claims (1,2,3,5,8).
- `death-corpse-loot`: killer loots inventory/eq/gold from the corpse
  (`get all corpse`), audiences included; AUTOGOLD/AUTOLOOT toggles on/off
  change which gold path the corpse sees (`src/fight.c:1654-1711`).
- `death-corpse-decay-pc` / `-npc`: pump 5/10 mud hours from the kill; the
  `number(0,6)` flavor line is seed-pinned and draw-claimed; contents dump
  proven with an item in the corpse.
- `death-con-loss-lockout`: repeated deaths until CON hits 0 (unit-test the
  arithmetic; one oracle scenario for the `"You are too unhealthy to play!"`
  disconnect if cheaply reachable — otherwise unit-only, boundary stated).
- `death-neutral-room-save`: a killing blow in a `ROOM_NEUTRAL` room:
  `"You are saved by the gods!"` + teleport to 8004, no corpse
  (`src/fight.c:1496-1519`).

**R5h.** Plant: floor HP at 0 instead of 1 on re-entry, or drop the corpse's
gold coin object. The re-entry `score`/look and loot blocks redden; the decay
scenario reddens on the wrong flavor line or a missing contents dump.

**Cost.** 4–5 scenarios; the decay pair is the heaviest (≤ 16 lines each).
Multiseed claims add ~10 census pairs. No seam, no harness change.

### 2.7 Core shop transaction + special-procedure shop surfaces

**What C does.** `do_list/buy/sell/value` and keeper gates
(`src/shop.c:293-925`); registered specials (tattoo, pet, banker, key_seller,
…) across `src/spec_procs*.c`. RNG: `slide_obj` retention `number(0,5)`
(`src/shop.c:679`).

**Why today's harness can't reach it.** This row is **not harness-blocked**:
`shop.tsv` already proves list/buy/sell slices green at zone 121 with stocked
M/G-reset fixtures, and the shop setup-draw rows were cleared in round 11
(`handoff/2026-10-21-deepseek-round12-final-draw-rows.md` §5 rows 3–5). The
residue is breadth: `value`, closed/sleeping keeper gates, gold/state
transitions, room audiences, and the named specials — all reachable with
existing verbs today.

**Vehicle.** None new. Scenario breadth on the existing pattern
(`shop.tsv` zone-121 keeper fixtures, spec-proc manifests), owned by the shops
lane. The only shared dependency is the key_seller owner gate (§2.9).

**R5h.** Inherited from the existing shop scenarios' discipline (e.g. the
`slide_obj` revert control in `spec-procs.tsv:582`).

**Cost.** No census change from this design; counted in the shops lane.

### 2.8 Mobile room object pulse order

**What C does.** Heartbeat order at PULSE_MOBILE: `mobile_activity` →
`room_activity` → `object_activity` (`src/comm.c:814-819`), with the hourly
block after violence (`src/comm.c:824-830`). The inventory row's own notes say
the C prepend order is restored and unit-pinned; the residual blockers are
**engine draw bugs**, not harness gaps: combat-entry draws (C's mobile `hit()`
vs Go's `StartCombat`, DP-1435) and god-vs-mob round order in long soaks
(`handoff/2026-10-15-dp1438-mobile-callers.md` §"Follow-up findings").

**Why today's harness can't reach it.** It can reach it — the living-world
soaks reach the divergences; they are **parked** because the first mismatch is
an engine bug, and long soaks are expensive to iterate on. The login fixture
also pollutes some soaks' setup-phase draws.

**Vehicle.** No new machinery. Once the engine fixes land (separate lane),
add minimal reproducer scenarios: one spawned mob, quiet zone, pumped pulses,
draw-parity on — the `mobact-aggr24-weaker-mobile-grin` pattern. The
`seed-player` fixture (§2.9) removes the setup-phase draw divergence that
forces today's soaks to be long before they become informative.

**R5h.** Each reproducer scenario names the draw it pins; reverting the engine
fix reddens exactly that stream index (the DP-1438 acceptance-evidence
pattern).

**Cost.** 3–5 scenarios post-engine-fix; light. No seam, no harness change.

### 2.9 The login/creation fixture (round-12 §3)

**What C does.** New-player creation runs `roll_real_abils` — 24 ×
`number(1,6)` — plus weight/height draws and `do_start` (`src/class.c:379-402`,
`:501-590`); logging in an existing player runs none of it (`load_char` +
`store_to_char`, `src/interpreter.c:1762-1766`). The first player is crowned
`LVL_IMPL` and skips `advance_level` (R3d, DP-1212).

**Why today's harness can't reach it.** The harness pre-seeds nothing: C's
`lib/etc/players` ships `Testor` (ID 1), so scenarios that need an existing
owner log C in (zero creation draws) while the Go side — which has no seeded
player — runs the full creation nanny (24 stat draws at the hometown 'K' step,
measured at draw index 4036/4066, round-12 §3). Result: `house-depth` ×5
(seeds 1,2,3,5,8) and `spec-proc-key-seller`(-non-owner) are **red under
draw parity** — 7 of the 36 release draw rows, parked as "the harness's to
fix" (round-12 §3, §5).

**Vehicle — the one harness change that matters.** A new opt-in `[fixture]`
directive:

- `seed-player` — the harness seeds the Go scenario store with a **Testor
  twin**: name `Testor`, player ID 1, password/class/level/stats/loadroom
  dumped from C's shipped `char_file_u` record in the build brief, so both
  engines log in an existing ID-1 player and roll identical (zero) creation
  draws. This generalizes the existing `Zzplaceholder` seeding
  (`main.go:1174-1189`) and keeps the ID-1/house-owner alignment the
  `house-control` fixture already relies on. Default-off: scenarios that
  don't set it behave exactly as today.
- New replacement scenarios (`house-owner-depth-v2`,
  `key-seller-owner-depth-v2`, …) written login-style on both sides with
  `seed-player`; the seven red rows are then **retired** — by superseding
  scenario files (preferred; the old files' claims move to the new IDs) or by
  amending the originals in the build brief with a full census — Zach picks
  (§4).
- Creation-surface bytes stay proven separately: existing creation-diffing
  scenarios (`[creation:*]` sections, `empty-players` fixtures) already drive
  both engines through the nanny symmetrically; this fixture changes none of
  them.

**R5h.** (a) Reverting the seeding re-reddens the seven rows — demonstrated
red today. (b) The harness change itself: a no-`seed-player` scenario must be
byte-identical before/after the change — the build brief runs a targeted
census over the relogin/house/key-seller scenario set plus the batch combined
census and confirms all 1,610 existing verdicts unchanged.

**Cost.** Harness change + ~3–6 new scenarios. Unblocks: 7 red draw rows, the
house/rent owner surface (§2.5), clean soak setup phases (§2.8), and the
existing-player login half of save/load (§2.3).

---

## 3. Cross-cutting findings for the manifests (R5e/R4)

These bound what any vehicle can prove and belong in the row notes as
`excluded`/`blocked` labels — **Zach approves each label**:

| C surface | Evidence | Consequence |
|---|---|---|
| Receptionist/cryo rent specials | declared `src/spec_assign.c:89-90`, never assigned | rent price/offer/store strings unreachable; label `excluded` (owning surface: unassigned special) |
| Rent charge/loss branches | `free_rent = YES` (`src/config.c:106`) ⇒ cost 0 on all reachable saves; "Salvation Army" arm needs `cost > gold` | unreachable in shipped config; label `excluded` (owning surface: compiled config) |
| House contents persistence | `House_save_all` stub `src/house.c:709-721`; all `House_crashsave` callers commented | nothing writes house files; prove **non**-persistence (§2.5) |
| House entry gating | `House_can_enter` `src/house.c:725-744` has no live caller | no trespassing bytes exist; movement scenarios must not expect them |
| Year rollover | 8,996,400 pulses (`src/utils.h:135-138`) | out of scope without a calendar seam (Zach's call, not proposed) |

---

## 4. Build order (argued from the inventory)

Units unblocked per vehicle (blocked units in `lifecycle`+`shops` = 13, plus
the 7 red draw rows):

1. **Login fixture (§2.9) first.** It is the only harness change with
   dependents: 7 currently-red draw rows, the house/rent owner units (§2.5),
   and clean soak setup phases (§2.8). Nothing else unblocks cross-row work,
   and its R5h is already demonstrated (the rows are red without it).
2. **Weather/time (§2.1) and regen/hunger (§2.2) next.** Pure scenario
   additions on proven pumping — no dependencies, immediate blocked-unit
   closure (2 of the 10 lifecycle units each, the largest green-able chunk),
   and they establish the per-hour compare discipline the other pulse rows
   reuse. (The brief guessed "steppable clock first"; the clock is already
   steppable at hour scale — §1.1 — so its vehicles reduce to scenarios.)
3. **Save/load (§2.3) and descriptor lifecycle (§2.4)** — light, independent,
   exercise existing restart/peer/prompt machinery.
4. **Death cycle (§2.6)** — heaviest draw-sensitivity; benefits from the
   multiseed discipline established by 1–3.
5. **Pulse order (§2.8) last** — gated on engine fixes (DP-1435 lane), not on
   this design.
6. **Shops (§2.7)** — no vehicle; stays with the shops lane's own breadth
   work.

## 5. Cost summary

- New scenarios: ≈ 25–35 total (≈ 10 heavy pumpers ≤ 40 control lines, rest
  light). Expected full-census growth ≈ +3–5% (today 1,102 s / 1,610
  scenarios); every scenario fits the 240 s timeout by design (§1.3 budget).
- Existing 1,610 scenarios: **must not change** — both new fixtures are
  default-off; the build brief proves no verdict flips (targeted relogin-set
  census + batch combined census).
- Harness changes: `seed-player` and `fixed-time` fixtures (§2.9, §2.1).
- **C seam changes: none proposed.** The reference binary and
  `dp-oracle-seam` are untouched; no promotion procedure (DEV-SETUP) is
  triggered.

## Decisions for Zach

1. **`seed-player` fixture (harness-only):** seed the Go store with a Testor
   twin (fields dumped from C's pfile) so both engines log in an existing
   ID-1 player. Approve / modify / reject.
2. **`fixed-time` fixture (harness-only):** per-scenario `DP_FIXED_TIME`
   override for calendar-boundary boots. Approve / modify / reject.
3. **Retiring the 7 red rows:** supersede `house-depth` and
   `spec-proc-key-seller(-non-owner)` with v2 scenarios (preferred), or amend
   the originals in the build brief with a full census. Pick one.
4. **Dead-code labels (§3):** approve `excluded` labels for the receptionist
   rent surface, rent charge/loss branches, and house-entry gating, and the
   prove-non-persistence stance for house contents.
5. **Year rollover:** accept as out of scope (no calendar seam proposed), or
   ask for a seam proposal.
6. **Build order (§4):** login fixture → weather/time + regen → save/load +
   descriptor → death → pulse order (engine-gated); shops stay in their lane.
7. **New fixture verbs, complete list:** `seed-player`, `fixed-time`. Nothing
   else; no changes to existing verbs.
