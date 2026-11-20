# DP-1583 dispositions — all 413 C functions settled

Status: **complete**. Companion tables:
[`dp1583-dispositions.tsv`](dp1583-dispositions.tsv) (one row per C function)
and [`dp1583-missing-cites.tsv`](dp1583-missing-cites.tsv). Docs-only: no Go
code, no manifests, no rulebook edits.

**What `ported-other-name` means:** a Go function does this C function's job.
It does **not** mean the port is faithful — that is what depth coverage (R5h,
the oracle) settles; "305 ported" is not "305 proven".

## Counts per disposition (all 413)

| disposition | rows |
|---|---:|
| ported-other-name | 305 |
| go-native | 51 |
| inlined | 27 |
| c-unreachable | 17 |
| gap (candidate) | 13 |
| **total** | **413** |

Missing-cites rows: **95** (Go functions whose doc comment carries no
machine-resolvable C citation).

## Missing-cite forms

| form | rows |
|---|---:|
| no C reference of any kind in the doc comment | 79 |
| C named in prose the index cannot resolve (no `src/` prefix, or no `C name()` / `file.c:name()` form) | 13 |
| banner names a different function (stale copy-paste) | 4 |

The four stale banners: `cmdTeleport` (says "load"), `cmdWizlock` (says
"cmdStat"), `cmdDate` (says "cmdHome"), `cmdHome` (says "heal"). Cleanup is
Zach's call after review; nothing was edited.

## Candidate gaps — five issues filed, 13 rows

| issue | class | rows |
|---|---|---:|
| DP-1601 | C POSIX signal interface (SIGUSR1 reread_wizlists, SIGUSR2 unrestrict_game, SIGHUP, SIGVTALRM watchdog, signal_setup) — Go registers only SIGINT/SIGTERM | 5 |
| DP-1602 | record_usage socket-usage log ported but unwired (OnRecordUsage never set; CountSessions has no caller) | 1 |
| DP-1603 | etc/date_record game-clock persistence not ported — in-game calendar resets on restart; documented non-port at pkg/game/weather.go:186-188 with no cited ruling | 2 |
| DP-1604 | DNS hash cache not consulted/populated on the connection path when !nameserver_is_slow | 2 |
| DP-1605 | RFC1413 ident login lookup not ported — the "ident" toggle exists and stores the flag, but no worker consumes it | 3 |

## c-unreachable (17) — dead on the C side, with greps

- **show_mult_obj_to_char** — only caller list_obj_in_heap is itself commented
  out (act.informative.c:825).
- **do_skills** — forward declaration only (interpreter.c:237); never in
  cmd_info[].
- **black_horn, ninelives, mirror** — declared in spec_assign.c but never
  ASSIGNOBJ'd/ASSIGNMOB'd; absent from lib/world; no Go registration.
- **Crash_offer_rent / gen_receptionist / receptionist / cryogenicist /
  Crash_report_rent / Crash_report_unrentables / Crash_rent_deadline** (7) —
  the rent-NPC chain: receptionist/cryogenicist are declared
  (spec_assign.c:89-90) but never ASSIGNMOB'd, so the whole offer path is
  dead.
- **Crash_idlesave, Crash_extract_expensive** — only reached under
  `free_rent = NO` (limits.c:448 → objsave.c:865); free_rent ships YES
  (config.c:106) and is never reassigned.
- **Crash_calculate_rent** — callers all sit in the two dead arms above.
- **get_bad_guy** — only caller kill_bad_guy (mobprog.c:424) is itself
  unreferenced.
- **clean_up** — prototype (modify.c:55) and definition (modify.c:776) only.

## Notable verified ports

- **random.c (R3):** pkg/dprng `Generator` is a verbatim CMWC port — a=123471786,
  c=362436, i=1023, r=0xfffffffe, xorshift seed 13/17/5, float32 uniform scale —
  and the server draws through `dprng.Number`. Draw parity holds by construction.
- **scripts.c (47 rows, one group):** one binding table
  (`pkg/scripting/engine.go` SetGlobal block, :740-:820) covers all 44 named
  lua_* globals; table_to_obj/table_to_room fold into the bridge write-backs;
  clear_stack is gopher-lua's stack. **Five spot-checks against the C bodies
  passed**: lua_say, lua_oload, lua_mob_flagged, lua_tport, lua_echo (same arg
  shapes, dispatch arms, and "[Lua] Invalid…" mudlog strings). `bridgeOLoad`
  matches the location case-insensitively (`equalFoldASCII`), as C does:
  `str_cmp` lowercases both sides (`src/utils.c:112`).
- **objsave.c:** the live rent path (rentsave → extract_norents →
  extract_objs → load) ports through pkg/game/quit.go
  RentOut/extractNorents/ExtractRentedObjects with the rent-code storage in
  pkg/db.

## Method

- Every C citation in the tables was grepped or read this session from `src/`
  in this worktree (R5g); every Go citation likewise from `pkg/`, `pkg/db`,
  `pkg/telnet`, `pkg/events`, `pkg/scripting`, `pkg/combat`, `pkg/spells`,
  `pkg/olc`, `pkg/validation`, `cmd/server`.
- Evidence names a shared message string with both line numbers, or the state
  a branch sets, so a reviewer can check each row in one step. Per the
  checkpoint review, every `ported-other-name` row carries main-path evidence
  (success output or state change), not only a refusal.
- Missing-cites criterion: the fnindex citation grammar
  (`tools/fnindex/main.go:45-51`). Inlined rows are not listed there (the
  brief attaches the list to `ported-other-name` only).

## Decisions for Zach

- **Five candidate-gap issues** (DP-1601..DP-1605 above) need triage: real gap,
  accepted divergence, or intentional absence.
- **The date_record non-port** (DP-1603) has an in-code rationale but no cited
  ruling — decide whether to bless it (add the DP- id) or close the gap.
- **Missing-cites cleanup (95 rows)**: bulk normalization pass vs folding into
  each function's next depth pass.
