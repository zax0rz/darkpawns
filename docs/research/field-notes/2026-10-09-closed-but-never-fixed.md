# 2026-10-09: Closed but never fixed, and other things the inventory caught

## Scope

- **Tracks:** legacy-port fidelity; multi-agent software engineering
- **Window:** 2026-10-08 to 2026-10-09, DP-1371 case D7 (`mudlog.unported-sites`), PRs 2 and 3
- **Primary artifacts:** zax0rz/darkpawns#1870 (merged; handoff `docs/fidelity/depth/handoff/2026-10-09-dp-1371-mudlog-admission.md`) and #1877 (open at the time of writing; handoff `…-mudlog-closeout.md` on its branch); oracle runs `~/Archives/darkpawns/oracle-runs/2026-10-08/dp-1371-mudlog-2/` and `2026-10-09/dp-1402-damage-tail-review-fixes/`; Linear DP-493, DP-1413, DP-1414
- **Status:** artifact-grounded field note. It continues [the 2026-10-08 note](2026-10-08-exhaustive-inventory-as-a-detector.md) (PF-060).

## 1. A tracker ticket closed as Done with no fix behind it

PR 3 ported `utils.c:724`, the `%s started hunting %s` line that C's `set_hunting` logs when a mob starts hunting a player. To place that producer, the agent had to list every C caller of `set_hunting` (`grep -n "set_hunting(" src/*.c`) and find each one's Go counterpart. Three callers had **no Go counterpart at all**. Each is a missing game behaviour, not just a missing log line:

| C caller | Behaviour | Go state on `3e771df2a` |
|---|---|---|
| `hunt_items` (`src/new_cmds2.c:766`, hourly from `src/comm.c:829`) | Mobs hunt players wearing particular gear | `OnHuntItems` is declared (`pkg/engine/gameloop.go:90`) and invoked (`:355`) but **never assigned**. No Go implementation exists, and `pkg/session/wiz_system.go:770` still carries a `TODO(port)`. |
| `SPECIAL(mallory)` (`src/spec_procs.c:1412-1419`) | Mallory sends Mickey after her attacker | `specMallory` (`pkg/game/spec_procs.go:1217`) only barks. Its comment says "calls mickey for revenge". |
| `check_for_bad_stats` via `equip_char` (`src/handler.c:640-675`, `:749`) | Equipment that zeroes a stat: messages, a Pestilence hunt, 40 trip damage | Only the mobile path is ported (`pkg/game/mobile_equipment.go:201-232`). |

**The first one already had a ticket.** Linear DP-493, "OnHuntItems callback declared but never wired", described this exact defect, with the fix (`cb.OnHuntItems = func() { world.HuntItems() }`). It was marked **Done on 2026-05-27**. No commit in the repository mentions DP-493, `hunt_items` or a `HuntItems` implementation (`git log --all -i --grep="DP-493\|hunt_items"`; `git log -S OnHuntItems`). The ticket was closed with no code behind it, and the feature stayed dead for four and a half months while the tracker reported it fixed. It was reopened on 2026-10-09.

This is the project's recurring "registered but never called" shape, one layer up. The subsystem looked wired because a callback slot existed. Then the *tracker* looked resolved because a status changed. Neither the callback slot nor the status proved the code ran. What found it was an exhaustive sweep that had to name a Go counterpart for every C caller. The sweep couldn't accept "there's a ticket for that" as an answer.

The second entry also shows a code comment asserting behaviour the body doesn't have. That's the same family as the "plausible paraphrase" cases (PF-002).

## 2. One producer exposed a whole byte class

Porting `comm.c:1574`'s `Connection attempt denied from [%s]` meant reading what C puts in `%s`. In C, `d->host` is the **zero-padded** dotted quad (`%03u.%03u.%03u.%03u`, `src/comm.c:1536-1539`) whenever no name was resolved. Because `nameserver_is_slow` ships YES (`src/config.c:206`), that is every connection. The port printed the raw `127.0.0.1` in about nine syslog lines (connected, reconnected, Bad PW, new player and the entry refusals), and its ban matching was inverted relative to C (`ban all 127.000.000.*` banned nothing; a raw-IP ban matched where C's wouldn't).

The same divergence had been **observed in an oracle transcript 11 days earlier** (`docs/fidelity/depth/handoff/2026-09-27-idle-lifecycle-expansion.md:121`, `[127.000.000.001] has connected.` vs `[127.0.0.1]`). It was recorded as "not part of the idle lifecycle" and left. The census never failed on it, because no scenario's assertions covered an immortal's view of a connect line. #1870 fixed the whole class with one helper.

## 3. An agent reported another agent's census as its own

AGENTS.md has agents start a census and later call `scripts/census.sh wait`, which prints one result line. DeepSeek reported PR 2's census with this line: `pairs=3442 … elapsed=2050.567s verdict=CLEAN_AFTER_RECHECK`. It also flagged an "off-by-one in census.sh", because its manifest said 3441 pairs.

The quoted line was character-for-character the result of **Sol's** run `dp-1402-damage-tail-review-fixes`, which had finished more recently. DeepSeek's own run (`dp-1371-mudlog-2/summary.txt`) said `pairs=3441 … elapsed=2047.581s`. Its verdict happened to be the same. The "off-by-one" was two runs being compared. The reviewer caught it because the elapsed time matched another run to the millisecond, and then checked the run's own `summary.txt` and `go-head.txt`.

The cause is the instrument, not the agent. `wait`'s line names neither the run nor the commit, so in a shared single-slot queue, "the latest result" and "my result" can't be told apart from the output alone. Briefs now say to read the run's own `summary.txt`. A proposed tool fix (put the run name and `go-head` on the line) is pending.

## 4. Reviewer rulings made from a summary, inverted by reading C

Three rulings Claude issued on 2026-10-08 for D7 were reversed on 2026-10-09, once the C call path was read before writing the next brief (R5e):

| Ruling as first issued | What the C showed |
|---|---|
| `close_socket`'s `Losing player` / `Losing descriptor` (`comm.c:2138`, `:2142`): classify blocked-reachability unless the audit finds a path | C creates `d->character` on the first input at the name prompt (`src/interpreter.c:1743-1750`). A drop before input, an empty line, or menu `0` after `quit` all reach it in ordinary play, and the oracle had already seen the menu case. Ported in #1870. |
| `comm.c:1554` DNS: report Go's always-on lookup as a divergence, don't fix | It's the same mechanism as the host string. Fixed in the same commit in #1870. |
| `limits.c:281` autowiz: no autowiz in Go, so don't build it, classify | The *log* is reachable through `do_advance` → `gain_exp_regardless` → `check_autowiz` (`src/limits.c:323-360`, `src/act.wizard.c:1569-1571`) with `use_autowiz = YES`. It needs one line at a call site Go already had. Ported in #1877. |

The first versions of these rulings live in the operator's private briefs, not the repository. The reversals are recorded in the two PRs' handoffs and review comments. The pattern repeats the DP-1401 lesson (a ruling built on a wrong premise cost a PR): a ruling built on a summary of the C costs more than reading the C.

## Ledger rows

PF-061, PF-062, RO-014 and RO-015.
