# Modernization Phase 6.4 audit — 2026-09-13

## Disposition and stop boundary

This is a documentation-only audit of the four original Phase 6.4 global-to-struct candidates on a fresh origin/main. It does not implement injection, claim Phase 6.4 implemented or complete, change a save/storage format, or reopen Phase 6.3’s deferred combat/RNG and broader authority candidates. The branch is intentionally left for human review.

No family is ready for an unqualified implementation slice. The explicit dispositions are:

| family | disposition | reason |
|---|---|---|
| mail subsystem state and hooks | proof-first work required | The production boot call to InitMailSystem is not present; restart/index behavior and the C-versus-Go mail storage contract are therefore not proven. |
| weather weatherWorld wiring | concrete locking defect requires proof-first triage; retain the canonical weather/tick boundary | The live heartbeat can re-enter weatherMu through event helpers at hours 5 and 21 and block indefinitely. Injection is not a safe next step; the defect must be separately characterized and resolved without changing tick or RNG order. |
| merge_bridge.go ban manager | deferred with evidence | banManager and World.Bans are genuinely separate live authorities today, with different callers and file paths. Consolidation would first need a fidelity decision and authority proof. |
| spec_assign registries | retain as-is pending a bounded proof task | Assignment maps are C-derived registration data, while handler registries are startup-mutated maps and exported assignment maps are mutated by tests and read directly by production code. No single owning runtime struct is established without changing dispatch seams. |

The historical −250–400 line estimate is not used as a disposition or a current measurement. The four original candidates remain YELLOW and unimplemented.

The smallest useful next task is therefore a proof-first weather lock
re-entry defect task, not an implementation PR or an injection refactor. It
must precede the mail lifecycle proof because it can stop the live heartbeat
on an ordinary scheduled tick. The mail proof remains the next modernization
candidate after this weather defect has its own reviewed disposition.

## Audit vehicle and governing boundary

The audit ran in /home/zach/darkpawns-audit-phase6-4, branch glm/audit-phase6-4, from fresh origin/main at 57414fb998f2d1257afd17a62eecb5894a79a482, the merge of #1454. The primary checkout’s existing docs/specs/tui-setup-wizard.md edit was preserved. The GitHub open-PR query returned no open modernization PRs on 2026-09-13, so no overlap was found and nothing was merged. Local branches were not treated as open PRs and were not used as input.

The governing AGENTS.md, amended standing charter /home/zach/dp-audit-PICKUP.md, docs/modernization/06-roadmap.md, /home/zach/dp-handoff-brief-2026-09-05.md, docs/fidelity/RULEBOOK.md, and docs/fidelity/DEPTH_TESTING.md were read. The recovered original reports were read from /home/zach/dp-modernization. Verbatim excerpts and report hashes are preserved in the companion original-source evidence. src/ and darkpawns-c-oracle/ were read-only throughout.

The current source hashes at the audit base are recorded in the companion evidence README. The exact original report hashes and original clone checkpoint are not being treated as current source facts.

## Census provenance and reusable boundary

The accepted Phase 6.3 provenance record provides the only reusable whole-corpus census. Its census checkpoint was `ab916e4b21d4b99a1740521846c21b9b90b7dd23`. The verified diff from that checkpoint to this audit base contains only documentation/evidence paths:

    M  docs/fidelity/depth/handoff/2026-09-13-modernization-phase6-3-provenance.md
    A  docs/fidelity/depth/handoff/2026-09-13-modernization-phase6-4-audit.md
    M  docs/fidelity/evidence/2026-09-13-phase6-3-provenance/original-excerpts.md
    A  docs/fidelity/evidence/2026-09-13-phase6-3-provenance/recovered-census-output.txt
    A  docs/fidelity/evidence/2026-09-13-phase6-4/README.md
    A  docs/fidelity/evidence/2026-09-13-phase6-4/original-excerpts.md
    M  docs/modernization/06-roadmap.md

The subsequent merge to current origin/main and this audit’s commits add only
documentation. Production files, scenario files, fixture inputs, and runner
scripts are identical to the actual census checkpoint. Reuse is therefore
justified for the unchanged corpus, but only as aggregate evidence.

The preserved aggregate is:

    scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0 elapsed=7066.739s

The recovered output preserves 856 status lines (847 PASS, 8 EXPECTED, 1 UNPINNABLE) and the final aggregate, while 85 scenario status lines and all temporary per-attempt logs are absent. It cannot independently reconcile all 941 scenario identities or inspect the deleted per-attempt logs. No missing output is reconstructed. The one UNPINNABLE row is the established human-cleared accuse-noarg-depth baseline; there are no failed, infra, timed-out, or stale rows in the preserved aggregate. Fresh focused runs below are the evidence for the families in this audit, not a claim that the recovered status stream is complete.

## Candidate inventory

### 1. Mail subsystem state and hooks

#### Finite current state inventory

| current global | type / owner | initialization and reads | writes, reset, shutdown, lifetime | locking and persistence |
|---|---|---|---|---|
| mailIndex (pkg/game/mail.go:72) | *MailIndex; package-owned recipient-to-block index | Zero value at package init; findCharInIndex and hasMail read it; scanFile rebuilds it from the file; indexMail inserts records; readDelete removes the delivered record | Mutated by indexMail, readDelete, and indirectly scanFile; no explicit reset before a second scan | storeMail/readDelete hold mailGlobalMu; scanFile, hasMail, and lookup helpers do not. Memory-only; intended to rebuild at boot. |
| freeList (:73) | *MailPosList; package-owned free-block list | Zero value; scanFile pushes deleted blocks; popFreeList reads/removes the head | Mutated by pushFreeList and popFreeList; deleted markers are written immediately; no shutdown persistence | Same mailGlobalMu limitation as mailIndex; reconstructed from the fixed block file. |
| fileEndPos (:74) | int64; package-owned file allocation cursor | Zero value; scanFile sets it from f.Stat; writeToFile refreshes it after writes; popFreeList returns it when no free block exists | Mutated by scan and writes; no shutdown hook | Normally reached under mailGlobalMu from store/read, but scanFile is not covered. Not separately persisted. |
| worldNameFunc, worldIDFunc (:75-76) | function hooks for ID/name translation | Set only by InitMailSystem; read by GetNameByID/GetIDByName | No production reset or shutdown reset found; process lifetime; fallback is Player(<id>) / -1 when nil | No lock. No production caller of InitMailSystem was found by the current call-path sweep, so the hooks are not proven wired in the live server. |
| mailGlobalMu (:81) | sync.Mutex; file/index critical section | Package initialization | Held by storeMail and readDelete; no shutdown action | Protects seek/read/write sequences in those two paths. It does not make hasMail, scanFile, or all hook reads a transaction. |
| mailWriteMu and mailWriteEntries (:85-86) | sync.Mutex plus map[int]*mailWriteEntry; process-local composition state | Map allocated at package init; PostmasterSendMail creates an entry; HandleMailInput reads/deletes or appends; CancelMailWriting deletes | PostmasterSendMail writes recipient state; input appends up to MailMaxSize; @ deletes and calls storeMail; disconnect cleanup deletes; process shutdown has no mail-specific flush | Map operations are protected by mailWriteMu. Buffered composition is not persisted and is discarded on disconnect/process exit; completed mail is written immediately. |

The current fixed-format helpers are marshalMailHeader/unmarshalMailHeader and marshalMailData/unmarshalMailData at pkg/game/mail.go:581-612. The current Go block size and file path are MailBlockSize=512 and MailFile=data/mail at :22-35. This audit does not change or bless that format.

#### Call paths and ownership

- Boot intent is InitMailSystem → hook assignment → scanFile at pkg/game/mail.go:95-100. The current production entrypoint calls ResetTime, parses the world, constructs World, and creates the session manager (cmd/server/main.go:164-218), but no production call to InitMailSystem exists. This is a verified current observation, not the original report’s assumption.
- The postmaster registration is pkg/game/postmaster.go:7-32; C’s src/spec_assign.c assigns postmasters at approximately :497-503. Go command dispatch reaches World.PostmasterSendMail, PostmasterCheckMail, and PostmasterReceiveMail for mail, check, and receive.
- PostmasterSendMail is a World method, but recipient lookup uses the package hook GetIDByName (pkg/game/mail.go:433-475). Composition input is intercepted in pkg/session/session_login.go:363-378 and disconnect cleanup calls game.CancelMailWriting in pkg/session/manager.go:1191-1194.
- storeMail and readDelete serialize file access, but hasMail is called by World.PostmasterCheckMail/ReceiveMail without taking mailGlobalMu. scanFile is also not synchronized. This is an ownership and concurrency proof boundary, not an authorization to add a lock here.
- The Go process shutdown path stops world tickers, stops the heartbeat, drains sessions, and saves dynamic world state (cmd/server/main.go:545-595). No mail-index flush or mail-composition persistence occurs there. Completed mail writes are already direct file writes; in-progress compositions are canceled during session cleanup.
- No existing World field owns the mail index, free list, file cursor, or ID/name hooks. World owns the player/object delivery side, but not the file subsystem. WorldPath does not currently control MailFile.

#### Player behavior and C call-path proof

C’s src/mail.c:218-251 scans and indexes the mail file once at boot; src/db.c:366-374 calls it before load_banned and Read_Invalid_List. src/mail.c:261-361 stores fixed blocks, and :372-467 reads, formats, marks deleted, and returns one message. The C postmaster special procedure at src/mail.c:476-500 gates descriptor/NPC state, recognizes only mail/check/receive, and checks no_mail; the send helper begins at :503.

The observable Go paths are the level/stamp gates and postmaster messages at pkg/game/mail.go:433-505, the @ completion/abort messages at :511-547, and the formatted receipt at :390-400. The C source is the authority under R1/R2/R5; any future injection must keep these bytes and the existing chosen storage format unchanged unless a separate human-authorized fidelity task says otherwise.

#### Existing proof and gaps

Named current test:

- pkg/game/mail_test.go:TestMailReadWriteSharedFilePositioning passes. It changes to a temporary directory, creates data, writes one 512-byte Go block through writeToFile, reads it back, and compares bytes. It proves only same-process helper positioning for the current Go block size; it does not exercise scan/index, free blocks, multi-block chaining, hooks, postmaster output, restart, or C parity.
- No mail-specific depth row or oracle scenario exists. The durable depth inventory row docs/fidelity/depth/surface-inventory.tsv:27 is the 11-call-site src/mail.c activity-output family, status blocked, with entry, reading, composition, and delivery explicitly residual.

Fresh focused result: TestMailReadWriteSharedFilePositioning passed in the game package run recorded at /home/zach/dp-phase6-4-focused-game-2026-09-13.log. This is not restart or player-output proof.

#### Separate fidelity defect recorded, not fixed

Current source differs from the recovered C mail contract at multiple player/storage boundaries: C src/mail.h:40-50 defines minimum level 2, stamp price 25, maximum 4096, and block size 100; Go pkg/game/mail.go:22-35 uses level 5, price 50, maximum 4096, and block 512. C src/db.h:73-75 names the mail and ban files under etc/ (etc/plrmail and etc/badsites), while Go uses data/mail and separately wires ban paths. The Go production path also does not call InitMailSystem, so the C boot scan and no_mail failure boundary are not established in the live Go path. This is a separate fidelity/storage issue. No constant, path, format, boot call, or production code was changed here.

### 2. Weather weatherWorld and associated wiring

#### Finite current state inventory

| current global | type / owner | initialization and reads | writes, reset, shutdown, lifetime | locking and RNG boundary |
|---|---|---|---|---|
| nowFunc (pkg/game/weather.go:90) | func() int64; time test seam | Defaults to time.Now().Unix; ConfigureNowFromEnv replaces it when DP_FIXED_TIME is set | Set-once by environment configuration; no shutdown reset; process lifetime | Configuration is before boot; resetTimeLocked reads it under weatherMu. |
| weatherInitNumber (:156) | function seam defaulting to dprng.Number | Used by resetTimeLocked for initial pressure | Tests may replace it; no production reset | It controls the first pressure draw; do not move or alter this draw boundary. |
| timeInfo (:157-163) | TimeInfoData; package canonical clock | ResetTime derives it; snapshots, AnotherHour, moon/calendar logic, and consumers read it | resetTimeLocked and AnotherHour mutate it; no world shutdown persistence | Protected by weatherMu on public combined paths; direct tests mutate/inspect package state. |
| weatherInfo (:164-169) | WeatherData; package canonical weather | ResetTime, WeatherChange, snapshots, sunlight/moon consumers read it | resetTimeLocked, AnotherHour, WeatherChange, ModifyWeatherChange mutate it | Same global lock boundary; WeatherChange consumes fixed and conditional RNG draws. |
| weatherMu (:170) | sync.RWMutex | Package initialization | Held by snapshots, reset, weather/time, events, and weather mutation helpers | WeatherAndTime holds the write lock across AnotherHour and WeatherChange; those bodies assume the caller’s lock. |
| weatherWorld (:171) | *World; event broadcast back-pointer | Set by SetWeatherWorld from cmd/server/main.go:218; six event helpers read it | No reset or shutdown clear; process lifetime; no World field owns it | Assignment/read is protected by weatherMu, but it provides one process-wide world target, not per-World isolation. |

The original candidate is specifically the last row and its associated SetWeatherWorld/event wiring. Injecting only that pointer would not make canonical timeInfo/weatherInfo state per-world.

#### Call paths, event order, and ownership

- cmd/server/main.go:164-170 calls ResetTime before ParseWorld, preserving C’s first boot pressure draw. :218 sets the weather world after the session manager exists. The main heartbeat callback at :324-326 calls game.WeatherAndTime(true, manager.SendToOutdoor).
- pkg/session/wiz_system.go:739 is the manual tick path and calls the same weather/time function before affect/point updates. This corresponds to src/act.wizard.c:3501-3510; the normal C heartbeat is src/comm.c:822-829.
- WeatherAndTime (pkg/game/weather.go:310-320) advances AnotherHour and, when mode is true, calls WeatherChange under one weatherMu write lock. At hours 5 and 21, AnotherHour emits outdoor text and calls the special event helpers. The helpers copy weatherWorld under a read lock and broadcast after releasing it (:533-603).
- Weather consumers use snapshots or sunlight under the same canonical state: pkg/session time/weather commands, pkg/game/world.go:969, pkg/game/skill_stealth.go:125, pkg/game/item_consumable.go:113,215, and pkg/game/spec_procs3.go:274.
- World has no weather field. Existing MessageSink/session manager ownership can deliver messages, but does not own the clock, weather state, or event side effects. Multiple World instances therefore share package state and the one event target.

#### Separate concrete locking defect: weather lock re-entry

The live Go call path contains a blocking lock re-entry at the scheduled event
hours:

1. `pkg/engine/gameloop.go:325-328` invokes the registered
   `OnWeatherAndTime` callback every 63 real seconds.
2. `cmd/server/main.go:324-326` supplies the production callback, which calls
   `game.WeatherAndTime(true, manager.SendToOutdoor)`.
3. `pkg/game/weather.go:313-319` takes `weatherMu.Lock()` and keeps that write
   lock while calling `AnotherHour`.
4. When the incremented hour is 5, `AnotherHour` calls
   `ghostShipDisappear` and `removeNightGate` (`:329-335`). When it is 21, it
   calls `ghostShipAppear` and `loadNightGate` and, for days 21-24, also
   `fullMoon` and `lunarHunter` (`:341-351`).
5. Each helper takes `weatherMu.RLock()` before checking `weatherWorld`
   (`:533-603`). `weatherMu` is the same `sync.RWMutex` at `:170`.

Go's `sync.RWMutex` is not re-entrant: a goroutine holding its write lock
cannot acquire the read lock. Therefore a normal production heartbeat blocks
at the first helper when `timeInfo.Hours` is 4 or 20 before the increment. At
hour 5 both sunrise helpers are reachable regardless of `weatherWorld`; a nil
world does not avoid the lock because the read lock precedes the nil check. At
hour 21 the ghost-ship and night-gate helpers are always reachable, and the
full-moon/lunar-hunter pair is additionally reachable when the pre-increment
day is 21-24. The immortal `tick` command reaches the same path at
`pkg/session/wiz_system.go:732-740`, so it can trigger the defect immediately
when manually invoked at those hours.

This is a separate concrete fidelity/availability defect, not an authorization
to refactor weather, RNG, scheduler, or injection code. The C call path is
`src/comm.c:825-831` → `weather_and_time(1)` and
`src/weather.c:41-80` → event helpers inside `another_hour`; C has no analogous
Go `RWMutex` re-entry boundary. The defect can stop subsequent heartbeat work
and therefore player-visible time, weather, and world updates. No production
code is changed in this PR.

#### RNG and player-visible boundaries

C src/weather.c:31-37 defines the order another_hour(mode) then conditional weather_change(). Its hour event order is :41-125; pressure and sky changes, including conditional dice(1,4) branches, are :128-229. Go mirrors those draw sites at pkg/game/weather.go:239-257 and :395-521, including conditional branch draws. The boot draw is explicitly called out in cmd/server/main.go:164-170 and C src/db.c:415-462.

No proposed or accepted Phase 6.4 change may alter draw order, draw conditions, scheduler order, or the SendToOutdoor callback sequence. The original risk report’s RED boundary includes the weather five-draw/branch chain and the boot-order chain; this audit leaves both untouched.

#### Existing proof and gaps

Named focused tests run and passed:

- TestAnotherHour_AdvancesTimeAndSunlight
- TestInitializeWeatherConsumesCPressureRoll
- TestAnotherHour_AdvancesMoonsAndMonths
- TestWeatherChange_AdjustsPressureAndSky
- TestWeatherEvents_BroadcastToWorld
- TestTimeWeatherSnapshotTracksCanonicalClock
- TestWorldIsOutsideMatchesCMacro
- TestMudTimePassedMatchesC
- TestMudTimePassedSubHourDoesNotCarry
- TestResetTimeDerivesClockFromEpoch
- TestResetTimeSunlightBands
- TestResetTimeMoonCascade
- pkg/session/time_weather_test.go: the four named formatter/snapshot tests TestFormatMudTimeHourWording, TestFormatMudTimeDateAndOrdinal, TestFormatMudTimeMoonOnlyWhenDark, TestFormatMudWeatherBranches, and TestTimeAndWeatherCommandsUseGameSnapshots

The depth manifest rows are time.clock-variants, weather.boot-state, and weather.sky-variants in docs/fidelity/depth/info.tsv. They use info-pulse-variants or info-basic, seeds 1,2,3,5,8 as listed in the manifest, and C informative weather/time readers at src/act.informative.c:1501-1529,1568-1600. The fresh matrix matched with no normalized divergence for all listed runs. The durable full focused output is /home/zach/dp-phase6-4-oracle-focused-2026-09-13.log; the seed-1 info-basic show-oracle capture is /home/zach/dp-phase6-4-oracle-smoke-info-basic-seed1-2026-09-13.log.

The proof gap is the separate lifecycle row docs/fidelity/depth/surface-inventory.tsv:66, weather and time pulses, which remains blocked: progression, rendered output, clock, and draw dependencies require a dedicated frozen-time vehicle. The command rows prove selected renders and variants; they do not prove every hour event, concurrent world target, manual tick interleave, or all C world side effects. In particular, `TestAnotherHour_AdvancesTimeAndSunlight` calls `AnotherHour` directly without the enclosing write lock, `TestWeatherEvents_BroadcastToWorld` calls each helper directly, and `TestTimeWeatherSnapshotTracksCanonicalClock` exercises only `WeatherAndTime(false, nil)` from hour 8. No named test calls `WeatherAndTime(true, ...)` from pre-increment hour 4 or 20, and the selected `info-pulse-variants` vehicle advances from hour 14 to 15, so these tests and scenarios do not reach the live re-entry defect.

#### Separate fidelity defects recorded, not fixed

The current Go event helpers are not equivalent to their C call paths:

- C full_moon (src/new_cmds.c:1302-1329) transforms eligible vampire/werewolf players and emits per-player text; Go fullMoon (pkg/game/weather.go:533-543) only broadcasts a custom global message.
- C lunar_hunter (src/new_cmds.c:2662-2682) conditionally consumes number(0,5), creates a mob, and tells the target; Go lunarHunter (:545-555) only broadcasts, so its draw/state path is absent.
- C ghost_ship_appear/ghost_ship_disappear (src/new_cmds.c:2684-2740) mutate room exits and emit room-specific text; Go ghostShipAppear/ghostShipDisappear (:581-603) only broadcast.
- C load_night_gate/remove_night_gate (src/gate.c:171-225) create and extract portal objects based on moon phase; Go loadNightGate/removeNightGate (:557-579) only broadcast.

These are fidelity defects and RNG/state proof boundaries, not a license to change the weather scheduler or draw sequence in this audit. They must be resolved or explicitly bounded before any weather-world injection is claimed behavior-preserving.

### 3. merge_bridge.go banManager and World.Bans

#### Finite authority and wiring inventory

| authority/path | current owner and lifecycle | production reads/writes | locking, lifetime, persistence |
|---|---|---|---|
| package banManager (pkg/game/merge_bridge.go:15) | Lazy-created by LoadBanned, ReadInvalidList, AddBan, and RemoveBan; process-global pointer | ValidName/ValidNameNoActive read it for invalid names; Manager.NewManager calls the two load wrappers at :382-388; other wrapper APIs have no production caller found | BanManager has no mutex; pointer/path/callback globals have no synchronization. Loaded slices are process memory and are not World.Bans. |
| package paths (:19-30) | banFilePath/invalidFilePath, default lib/etc/banned and lib/text/xnames, changed by SetBanFilePaths in main at :205-206 | Only bridge load wrappers use these paths | Mutable process globals with no lock; current load methods discard underlying errors and wrappers return nil. |
| HasActiveCharacter (:33-35) | Package callback assigned by each session manager in pkg/session/manager.go:369-380 | ValidName checks it after the package ban manager; merge bridge tests replace/restore it | Callback closure takes m.mu.RLock; function variable itself has no owner/reset and is overwritten by another manager. |
| World.Bans (pkg/game/world.go:118-119,288-290) | One *BanManager created in each NewWorld; returned through Manager.GetBanManager | Telnet/WebSocket BanAll and BanNew/BanSelect checks read it (pkg/telnet/listener.go:167-209, pkg/session/manager.go:988-998); World.ExecBan/ExecUnban and session commands mutate it (pkg/game/act_other_bridge.go:107-123, pkg/session/cmd_misc.go:184-195) | BanManager has no mutex and World.mu does not guard its slices. Admin persistence is direct ./data/badsites in ExecBan/ExecUnban. Instance lifetime follows World; no reload bridge joins it. |

The two authorities are genuinely separate live state, not merely a duplicate field awaiting a mechanical move:

1. Manager.NewManager loads the package-global manager, while NewWorld creates World.Bans and does not load it.
2. Login name validation uses package-global ValidName/ValidNameNoActive; connection host checks and GetBanManager use World.Bans.
3. Admin ban/unban mutates World.Bans and writes ./data/badsites, while bridge load uses worldDir/etc/banned after SetBanFilePaths.
4. The package AddBan/RemoveBan/ListBans wrappers are not reached by current production admin commands in the repository search.

#### C authority and player call paths

C has one ban_list at src/ban.c:41, one invalid-name list at :254-255, and one file authority. load_banned is :52-83; isbanned is :86-104; admin list/add/write is :132-210; unban is :213-244; Valid_Name and invalid-list loading are :257-312. C boot calls both loaders at src/db.c:366-374. Connection BanAll is checked at src/comm.c:1566-1576; name validation and new-name flow are src/interpreter.c:1743-1808, with the BanSelect check at :1891-1904.

Observable boundaries include BanAll connection rejection, BanNew and BanSelect new-character restrictions, invalid-name rejection, exact ban list format/order, and admin add/unban messages. C’s single authority and paths are the R5 comparison target.

#### Existing proof and gaps

Named focused tests run and passed:

- TestIsBannedLiteralAsterisk, TestIsBannedSubstring, and TestIsBannedEmptyHostname exercise the BanManager matching contract.
- TestManager_GetBanManager proves a World.Bans instance is exposed by the manager.
- TestValidNameOnlineDuplicate, TestValidNameNilCallback, and TestHasActiveCharacter_CaseInsensitive exercise the package callback seam.
- TestSpecBank and TestSpecHornObjectReceiverAudienceAndGates were run in the same focused setup but are spec-family evidence, not ban-authority proof.

The depth manifest docs/fidelity/depth/ban.tsv:2-11 names ban-depth for no-bans, add, list format/order, parsing, duplicate, invalid flag, missing site, and truncation, plus unit TestIsBannedLiteralAsterisk. The fresh oracle matrix ran ban-depth with seeds 1,2,3,5,8, all with no normalized divergence. It proves the selected scenario vehicle’s command outputs and state; it does not prove that live manager/load/admin paths share one authority or that a process restart preserves the same list.

The durable focused unit and oracle outputs are recorded in the evidence README and outside self-cleaning directories. The C/Go authority mismatch itself is not cleared by green ban-depth rows because that scenario does not cross both load and admin authorities in one restart-aware vehicle.

#### Separate fidelity defect recorded, not fixed

The current split authority is a reachable fidelity defect: C’s isbanned uses the list loaded by load_banned and the same list is changed by do_ban/do_unban; current Go host checks use World.Bans, while login name checks use the package-global manager. Current file paths also diverge between bridge load and world admin writes. This audit does not choose an authority, repair paths, add synchronization, or consolidate the managers. Any such change is a separate behavior-changing fidelity task.

### 4. spec_assign registries

#### Finite data and registry inventory

Current assignment data in pkg/game/spec_assign.go is:

| object | current declaration | measured entries | semantic role |
|---|---|---:|---|
| MobSpecAssign | :17-326 | 228 | C assign_mobiles VNum → procedure-name data; final duplicate assignment wins in the recovered map. |
| ObjSpecAssign | :328-358 | 27 | C assign_objects VNum → procedure-name data. |
| RoomSpecAssign | :360-392 | 25 | C assign_rooms VNum → procedure-name data. |

The current registration maps are:

| registry | declaration and mutation | measured registrations | lifetime |
|---|---|---:|---|
| SpecRegistry | :394-413; RegisterSpec writes it | 119 unique RegisterSpec calls across spec_procs.go, spec_procs2.go, spec_procs3.go, spec_procs4.go, and spec_procs_missing.go | Empty map allocation, then writes during package init; reads after package initialization are expected to be concurrent-read-only. |
| ObjSpecRegistry | :405-419; RegisterObjSpec writes it | 1 unique object-aware registration (horn) | Same startup lifetime; GetObjSpecForObject falls back through SpecRegistry with a nil-object adapter. |

The maps are package globals, but their roles differ. Assignment maps are static C-derived registration data in production; because they are exported, tests mutate MobSpecAssign temporarily for fixture VNums (pkg/game/mobact_test.go:232-238, world_test.go:223-229, groinrip_depth_test.go:82-88, spec_conjured_test.go:115-121, and neckbreak_depth_test.go:41-47). Production code also reads the assignment map directly in pkg/game/mobprogs.go:26,204, pkg/game/mobact.go:103-110,341, pkg/game/combat_wire.go:204, pkg/game/spec_procs2.go:483,2263, pkg/game/skill_special.go:627, pkg/game/skill_stealth.go:486, pkg/session/combat_cmds.go:290, and pkg/game/world.go:1061-1063.

Handler registration occurs in package init functions at spec_procs.go:1269-1292, spec_procs2.go:17-73, spec_procs3.go:60-92, spec_procs4.go:20-40, and spec_procs_missing.go:202-207, plus postmaster registration at pkg/game/postmaster.go:31-32 and other unrelated registrations. NewWorld is called from cmd/server/main.go only after package initialization. There is no production late-registration path found, and no startup call to AllSpecNames in NewWorld.

#### Dispatch and C call paths

The lookup functions at spec_assign.go:421-462 resolve VNum → name → handler. GetObjSpecForObject preserves the concrete object receiver for the one ObjSpecRegistry handler and uses a compatibility adapter for ordinary SpecFunc handlers. This receiver distinction is part of the player-visible dispatch contract.

The live Go dispatch paths are finite and distinct:

- player command specials: pkg/session/commands.go:694-761, in order mob, room, room objects, equipped objects, inventory objects, with a handled true return stopping the path;
- recursive follower movement specials: pkg/game/act_movement.go:465-507;
- room activity/pulse specials: pkg/game/room_activity.go:100-105;
- autonomous mobile activity: pkg/game/mobact.go:103-110,133-163;
- the World.specRooms fast-path cache and rebuild: pkg/game/world.go:1010-1067; and
- direct assignment-name checks in the production readers listed above.

C ASSIGNMOB, ASSIGNOBJ, and ASSIGNROOM install function pointers into loaded prototype/world records at src/spec_assign.c:45-77; the finite live assignments are in :85-511, :515-563, and :567-642. C command dispatch order is src/interpreter.c:1407-1480 (room, equipment, inventory, mobs, room objects), and autonomous mobile dispatch is src/mobact.c:68-93 with cmd=0/empty argument. The recovered C source, not a registration-name assumption, determines which assignments exist; commented/declaration-only procedures are not invented.

#### Existing proof and gaps

Named focused tests run and passed:

- TestGetMobVNumSpecUsesCFinalAssignments checks final duplicate-assignment results at VNums 8014 and 11024 and non-nil handler lookup.
- TestUnregisteredSpecProcs checks every name in the three assignment maps is present in SpecRegistry.
- TestSpecProc_SmokeAll invokes every assigned name through benign mob, object, and room-shaped calls and checks no panic.
- TestSpecBank and TestSpecBankObjectCallContract cover object/mob bank behavior and receiver/audience contracts.
- TestSpecHornObjectReceiverAudienceAndGates covers the concrete object registry path, equipment gate, audience, and fallthrough.

The fresh oracle matrix covered:

| scenario | seeds | named depth proof |
|---|---|---|
| spec-proc-bank | 1,2,3,5,8 | Object #8034 bank balance/deposit/withdraw, fallthrough, audience, and state rows in docs/fidelity/depth/spec-procs.tsv:156-164. |
| spec-proc-bank-kir-oshi | 1,2,3,5,8 | Second registered #18224 ATM registration and same object-special path, row :165. |
| spec-proc-whirlpool | 1,2,3,5,8 | Autonomous registered mob #12200, commandless pulse dispatch, random destination, output, and state rows :180-185. |
| spec-proc-elements-master-column-none | 1,2,3 | Registered room #1315 no-talisman branch, relocation, and audience rows :414-415. |

All listed fresh oracle runs completed with no normalized divergence. The current depth ledger has the historical aggregate pkg/game/spec_procs*.go: 241 scenario-proven / 243 unit / 7 blocked / 26 excluded in docs/fidelity/depth/spec-procs.tsv and the risk report; those counts are not closure proof for a registry refactor. The named scenarios prove selected dispatch reaches and player/state branches. They do not prove all 228/27/25 assignments, package initialization order across every build context, late-registration rejection, data-race behavior, or direct-reader equivalence after maps are moved.

The existing file itself records two possible improvements at spec_assign.go:464-476: resolving handlers onto MobInstance during world load and startup validation. They remain documentation-only notes and are not implemented here.

#### Disposition rationale

Retain the registry representation as-is for this phase. The static assignment maps could eventually be made private or loaded into a registration object, and the two handler maps could eventually be held by an explicit registry owner. But a behavior-preserving struct injection must first account for all direct production map readers, the exported test mutations, the object-aware adapter, C assignment order/absence, World.specRooms cache rebuild, and the package-init-before-NewWorld lifetime. Moving only SpecRegistry would not remove the actual global authority; resolving handlers onto instances would change timing and fixture seams. There is no proof-backed smallest slice in the current candidate set.

## Cross-family proof boundaries

The named unit and oracle runs establish reachability and selected bytes, not whole-family closure:

- A passing info-basic or weather variant proves selected command rendering after a particular boot/tick setup; it does not prove every event branch, scheduler/draw interleave, or the lock-reentry path.
- A passing ban-depth proves its isolated scenario’s manager and output behavior; it does not prove the current load/login/admin authorities are one instance across restart.
- A passing spec-proc scenario proves the named VNum and dispatch topology; it does not prove all assignment rows or registration mutation safety.
- The one mail helper test proves a Go block round trip only. It does not prove the live postmaster, boot scan, fixed-file compatibility, or restart.

Under R5, the proof stops exactly at those boundaries. No historical estimate, green breadth row, or unit-only seam is treated as authorization to move state.

## Recommended next task: weather lock re-entry proof and triage

Because the live weather path has a concrete heartbeat-blocking defect, the
recommended first task is a bounded proof/triage task for that defect. This is
not a production fix or a weather injection refactor, and it is not part of
this PR. The mail lifecycle proof below remains the next modernization proof
task only after this weather boundary has a reviewed disposition.

### Exact weather scope

Trace and prove only the existing lock boundary and its two event-hour paths:

- Go: `pkg/game/weather.go:313-357,533-603`,
  `pkg/engine/gameloop.go:325-328`, `cmd/server/main.go:324-326`, and
  `pkg/session/wiz_system.go:732-740`;
- C reference: `src/weather.c:41-80`, `src/comm.c:825-831`, and
  `src/act.wizard.c:3501-3510`; and
- the future focused proof owner: `pkg/game/weather_test.go` plus a dedicated
  weather event/lifecycle oracle vehicle if the existing scenarios cannot
  observe the boundary.

The proof must exercise `WeatherAndTime(true, ...)` when the pre-increment
hour is 4 and 20, with both a nil and a live `weatherWorld` target as
applicable, and must establish whether the call returns at the scheduled
heartbeat and manual `tick` entry points. It must separately record the event
order and player-visible output at hour 5 and hour 21, including the day
21-24 full-moon/lunar-hunter condition. Any eventual fix must preserve C/Go
event order, conditional RNG draws, `SendToOutdoor` ordering, and the 63-second
scheduler boundary; do not combine it with weather-world injection.

### Weather stop conditions

Stop before any implementation if the lock ownership remains ambiguous, the
event-hour return behavior is not proven, output or draw order changes, the
proposal requires scheduler/RNG refactoring, or the work expands into the
separate C-vs-Go weather side-effect defects listed above. A separate human
review must authorize any production fix.

## Subsequent task: mail lifecycle proof

After the weather defect has its own reviewed disposition, the next bounded
proof vehicle remains the mail lifecycle task. It is a gate for a later
implementation slice, not part of this PR.

### Exact scope

Read/trace only at first, then add only the minimum future proof artifacts to:

- Go call paths: pkg/game/mail.go, pkg/game/postmaster.go, pkg/game/world.go, pkg/session/session_login.go, pkg/session/manager.go, and cmd/server/main.go;
- C authority paths: src/mail.c, src/mail.h, src/db.c, src/db.h, src/interpreter.c, and the C postmaster/assignment path; and
- future evidence/depth ownership for the named mail vehicle.

The first implementation slice, if and only if the proof clears, should be one explicit mail subsystem owner connected to the existing lifecycle owner (likely World or the server/session bootstrap, choice to be made after the path proof). It must include the index, free list, file cursor, ID/name hooks, and lock as one ownership decision or document why a state member remains process-global. It must not silently leave a second mail authority behind.

### Required coverage before implementation

The proof must name exact checkpoints and compare C/Go behavior for:

1. fresh boot with absent/empty mail storage;
2. boot scan with one header, deleted/free blocks, a multi-block message, and corrupt/truncated non-aligned storage;
3. postmaster mail, check, and receive entry/fallthrough gates, exact actor/room output, ID/name hooks, and multi-block receipt;
4. process restart/reopen of the same file, including index reconstruction and free-block reuse;
5. composition buffer completion, empty @ abort, maximum length, and disconnect cancellation; and
6. concurrent send/check/receive access at the existing file/index lock boundary, without changing ordering or bytes.

The proof must explicitly resolve the C-versus-Go constants/path/storage discrepancy and state which already-deployed Go format is preserved. It must not change the format as an incidental part of dependency injection.

### Exclusions and stop conditions

Exclude weather/tick/RNG refactoring, weather-world injection, ban authority consolidation, spec registry movement, player-save format changes, database schema changes, production behavior changes without C evidence, and any edit to src/ or darkpawns-c-oracle. The weather lock-reentry defect is recorded and prioritized here, but not fixed here.

Stop before implementation if any of these remains unresolved: no production boot owner for InitMailSystem; no authoritative file/path decision; no restart/index proof; any output or block-byte divergence; an unclassified concurrent access; or any need to change the save/storage format. A future implementation PR should be a separate reviewable slice with its own focused oracle vehicle and no automatic merge.

## Validation and durable evidence

### Fresh focused units

The combined selected game tests passed (0.079s), the selected session tests passed (0.024s), and the supplemental spec registry/bank smoke run passed (0.021s). Complete outputs are preserved outside self-cleaning directories:

- /home/zach/dp-phase6-4-focused-game-2026-09-13.log
- /home/zach/dp-phase6-4-focused-session-2026-09-13.log
- /home/zach/dp-phase6-4-focused-spec-2026-09-13.log

### Fresh focused oracle matrix

The input scripts and tested scenario files were frozen during the runs. The
preserved base output is `/home/zach/dp-phase6-4-oracle-focused-2026-09-13.log`.
It contains 32 completed PASS report blocks with these exact identities:

| scenario | completed seeds in preserved base log | count |
|---|---|---:|
| `info-basic` | 1,2,3,5,8 | 5 |
| `info-pulse-variants` | 1,2,3,5,8 | 5 |
| `ban-depth` | 1,2,3,5,8 | 5 |
| `spec-proc-bank` | 1,2,5,8 | 4 |
| `spec-proc-bank-kir-oshi` | 1,2,3,5,8 | 5 |
| `spec-proc-whirlpool` | 1,2,3,5,8 | 5 |
| `spec-proc-elements-master-column-none` | 1,2,3 | 3 |
| **total** | **unique completed report identities** | **32** |

The base log also contains one incomplete C-oracle readiness diagnostic at
lines 86-228: the seed-3 bank attempt exited before readiness because of
`SYSERR: bind: Address already in use`. It has no result row, was manually
inspected, and is not counted as a pass or as a duplicate. The missing unique
identity was `spec-proc-bank` seed 3. It was rerun with the same harness and
unchanged scenario input using:

```text
PATH=/usr/local/go/bin:$PATH \
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
/home/zach/dp-phase6-4-oracle-diff-2026-09-13 --scenario spec-proc-bank --seed 3
```

The complete rerun output is
`/home/zach/dp-phase6-4-oracle-spec-proc-bank-seed3-2026-09-13.log`; it
returned `result: no normalized divergence`. The reconciled matrix therefore
contains 33 unique scenario/seed identities, all passing, with no duplicate
completed identity. The separate seed-1 `info-basic --show-oracle` capture
does not replace the missing bank seed.

| family | scenarios and seeds | runs | passed | failed | infra | timed out | stale |
|---|---|---:|---:|---:|---:|---:|---:|
| weather | info-basic, info-pulse-variants × 1,2,3,5,8 | 10 | 10 | 0 | 0 | 0 | 0 |
| bans | ban-depth × 1,2,3,5,8 | 5 | 5 | 0 | 0 | 0 | 0 |
| spec registries | spec-proc-bank, spec-proc-bank-kir-oshi, spec-proc-whirlpool × 1,2,3,5,8; spec-proc-elements-master-column-none × 1,2,3 | 18 | 18 | 0 | 0 | 0 | 0 |
| mail | no mail oracle scenario exists; unit only | 0 | — | — | — | — | — |
| total | reconciled unique selected matrix | 33 | 33 | 0 | 0 | 0 | 0 |

One seed-1 `info-basic` run was repeated with `--show-oracle`; the normalized
C blocks are preserved at
/home/zach/dp-phase6-4-oracle-smoke-info-basic-seed1-2026-09-13.log. That
repeat is not included as an additional matrix identity.

### Required repository gates

All required gates passed at the final documentation checkpoint. Complete
outputs are preserved outside self-cleaning directories:

| gate | result | durable output |
|---|---|---|
| `gofumpt -l .` | PASS; no files listed | `/home/zach/dp-phase6-4-gofumpt-review-2026-09-13.log` |
| `git diff --check` | PASS | `/home/zach/dp-phase6-4-diff-check-review-2026-09-13.log` |
| `/usr/local/go/bin/go build ./...` | PASS | `/home/zach/dp-phase6-4-build-review-2026-09-13.log` |
| `/usr/local/go/bin/go vet ./...` | PASS | `/home/zach/dp-phase6-4-vet-review-2026-09-13.log` |
| `/usr/local/go/bin/go test ./...` | PASS | `/home/zach/dp-phase6-4-test-all-review-2026-09-13.log` |
| `/usr/local/go/bin/go test ./pkg/game/...` | PASS | `/home/zach/dp-phase6-4-test-game-review-2026-09-13.log` |
| `golangci-lint run ./...` | PASS; 0 issues | `/home/zach/dp-phase6-4-lint-review-2026-09-13.log` |
| `make fidelity-depth` | PASS; 4816 total, 4697 proven/delegated, 68 blocked, 51 excluded | `/home/zach/dp-phase6-4-fidelity-depth-review-2026-09-13.log` |
| `make expected-divergences-check` | PASS; 26 rows across 10 scenarios; pins OK | `/home/zach/dp-phase6-4-expected-divergences-review-2026-09-13.log` |

The first lint invocation lacked `/usr/local/go/bin` on PATH and stopped at
environment discovery without code diagnostics; the recorded rerun with the
required Go path passed with zero issues.

The reusable census aggregate is documented above and in the evidence README; it is not rerun because production/scenario/fixture/runner identity was verified against its checkpoint. No INFRA row requires manual exception review in that aggregate; the only non-PASS baseline is the known human-cleared unpinnable accuse-noarg-depth.

## Changed files and human-review request

This PR’s intended changed-file set is documentation/evidence only:

- docs/fidelity/evidence/2026-09-13-phase6-4/original-excerpts.md — exact original report excerpts and hashes;
- docs/fidelity/evidence/2026-09-13-phase6-4/README.md — current hashes, census reuse boundary, and durable focused-run references;
- this dated handoff; and
- docs/modernization/06-roadmap.md — Phase 6.4 tracking update.

No production, test, scenario, fixture, runner, CI, deploy, website, src/, oracle, or save-format file is changed. Human review is requested on the four dispositions, the separate fidelity defects, and the bounded mail proof task. Do not treat this audit as implementation authorization. The branch and resulting PR remain unmerged; stop here for review.
