# Strict L1 Phase C Bottom-Up Batch 1 (`src/` Queue Orders 550–571)

- **Queue**: `docs/research/l1-phasec-queue-2026-11-16.tsv` (571 open sites total)
- **Batch scope**: 22 sites across the 10 lowest-count C files (`medit.c`, `sedit.c`, `weather.c`, `act.display.c`, `boards.c`, `act.other.c`, `act.social.c`, `file-edit.c`, `modify.c`, `utils.c`; queue orders 550–571)
- **Baseline clean census (restored Go)**: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-16/hydrogen-phasec1-clean` (`scenarios=7 passed=7 failed=0 verdict=CLEAN | draw_divergent=0`)
- **Mutant census (one-byte Go mutations)**: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-16/hydrogen-phasec1-mutant`
- **Additional retained witness runs**:
  - `quit-incap-depth`: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-16/hydrogen-phasec1-quit3` (witnesses `You die before your time...` on both C and Go; exposes `HandleNonCombatDeath` missing room `death_cry`)
  - `medit-script-unsaved-depth`: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-16/hydrogen-phasec1-baseline2` (witnesses `Cannot assign a script until the mob is saved at least once.` on both C and Go; exposes `medit` new-mob `Hitroll: 20` vs `0` divergence)

## Settled Queue Sites (Orders 550–571, 22 sites)

| Order | C Site | Disposition | Manifest Row | Evidence / Call-Path Proof |
|---:|---|---|---|---|
| 550 | `src/act.other.c:120#1` | C-first witness (divergence filed) | `quit.tsv` (`quit.incapacitated-death`) | `quit-incap-depth` witnesses `You die before your time...` on C & Go; exposes Go `HandleNonCombatDeath` omitting room `death_cry`. |
| 551 | `src/act.other.c:483#1` | C-first witness | `peek.tsv` (`peek.kender-mob-followup`) | `peek-kender-depth@3` (`kender_steal` -> `do_steal(..., 0, 1)` `"Somehow $p makes it's way into your pack."`). |
| 552 | `src/act.other.c:1124#1` | C-unreachable (OS fault-only) | `gen-write.tsv` (`gen-write.open-failure`) | `fopen(filename, "a")` failure in `do_gen_write`; `TestDoGenWriteEntryGates`. |
| 553 | `src/act.social.c:229#1` | C-unreachable (boot-time) | `socials.tsv` (`boot-social-messages.boot-errors-unreachable`) | `fopen(SOCMESS_FILE, "r")` boot error (`log()`/`exit(1)` only); unreachable under shipped `lib/misc/socials`. |
| 554 | `src/act.social.c:246#1` | C-unreachable (boot-time) | `socials.tsv` (`boot-social-messages.boot-errors-unreachable`) | `find_command(next_soc) < 0` boot error; unreachable under shipped `lib/misc/socials`. |
| 555 | `src/act.social.c:246#2` | C-unreachable (boot-time) | `socials.tsv` (`boot-social-messages.boot-errors-unreachable`) | Same `boot_social_messages` boot error (`' in social file`). |
| 556 | `src/file-edit.c:40#1` | C-unreachable (OS fault-only) | `mudlog.tsv` (`mudlog.file-editor-delete-error`) | `kill_file` failure (`SYSERR: Can't delete file '`); `TestFileEditorDeleteErrorMudlog`. |
| 557 | `src/file-edit.c:43#1` | C-first witness + mutant red | `mudlog.tsv` (`mudlog.file-editor-delete`) | `tedit-delete-mudlog-depth` (`OLC: Teditactor deletes 'scripts/mob/dog.lua'.`). |
| 558 | `src/file-edit.c:48#1` | C-unreachable (OS fault-only) | `mudlog.tsv` (`mudlog.file-editor-open-parent`) | `fopen(storage, "w")` failure (`SYSERR: Can't write file '`); `TestFileEditorOpenParentMudlog`. |
| 559 | `src/modify.c:225#1` | C-first witness + mutant red | `mail.tsv` (`mail.abort-output`) | `mail-abort-depth` (`@` on empty mail buffer -> `Mail aborted.`); also exposes `HandleMailInput` missing `/a` abort support (filed in Linear). |
| 560 | `src/modify.c:236#1` | C-first witness + mutant red | `boards.tsv` (`boards.post-abort-output`) | `board-post-abort-depth` (`/a` in board write -> `Post not aborted, use REMOVE <post #>.`). |
| 561 | `src/modify.c:713#1` | C-first witness + mutant red | `string.tsv` (`string.extradesc-delete-keyword-required`) | `string-extradesc-delete-empty-depth` (`You must supply a field name.`). |
| 562 | `src/utils.c:250#1` | C-unreachable (64-bit) | `follow.tsv` (`sprintbit.invalid-bitvector-unreachable`) | `if (bitvector < 0)` on `long bitvector` in `sprintbit` is unreachable on 64-bit Linux where bitvectors use bits 0..31. |
| 563 | `src/utils.c:412#1` | C-first witness + mutant red | `follow.tsv` (`follow.charm-stop-jerk-act`) | `charm-stop-follower-depth` (`You realize that Charmmaster is a jerk!`). |
| 564 | `src/utils.c:412#2` | C-first witness + mutant red | `follow.tsv` (`follow.charm-stop-jerk-act`) | `charm-stop-follower-depth` (`Charmpet realizes that Charmmaster is a jerk!`). |
| 565 | `src/act.display.c:127#1` | C-unreachable | `infobar.tsv` (`infobar.unknown-state-unreachable`) | `GET_INFOBAR(ch)` is only ever assigned `INFOBAR_OFF` (0) or `INFOBAR_ON` (1) in `src/`. |
| 566 | `src/act.display.c:128#1` | C-unreachable | `infobar.tsv` (`infobar.unknown-state-unreachable`) | Same `default:` branch in `do_infobar`. |
| 567 | `src/boards.c:254#1` | C-unreachable (OOM) | `boards.tsv` (`boards.write-heading-allocation-failure`) | `CREATE` calls `calloc` and exits on OOM, so NULL `msg_storage[slot]` after allocation cannot occur in C. |
| 568 | `src/boards.c:419#1` | C-first witness + mutant red | `boards.tsv` (`boards.remove-active`) | `boards-remove-active-depth` (`At least wait until the author is finished before removing it!`). |
| 569 | `src/medit.c:685#1` | C-first witness (divergence filed) | `medit.tsv` (`medit.new-mob-script-rejected`) | `medit-script-unsaved-depth` witnesses `Cannot assign a script until the mob is saved at least once.` on C & Go; exposes Go `medit` new-mob `Hitroll: 20` vs `0`. |
| 570 | `src/sedit.c:866#1` | C-first witness + mutant red | `sedit.tsv` (`sedit.keeper-validation`) | `sedit-keeper-prompt-depth` (`Enter virtual number of shop keeper : `). |
| 571 | `src/weather.c:223#1` | C-first witness | `weather.tsv` (`weather.sky-transition-strings`) | `weather-sky-transitions@1,2,3,5,8` (`weather_change` case 6 `The lightning stops.`). |
