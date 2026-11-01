#!/usr/bin/env python3
"""Assemble slop.tsv for the slop track (research question 2).

Two row sources:
  * the 100-commit random sample of slop-tagged git leads (sample.json)
  * the 101 slop-tagged screening records (slop-track-register.jsonl)

Mechanical fields (fix commit/date, PR, branch, birth) are computed from git.
Judgment fields (category, catch method, visibility, summary) are authored in
the dicts below and every one is checked against the source in-session.
"""
import json, subprocess, re, os, datetime

REPO = '/home/zach/darkpawns'
OUT = '/home/zach/dp-research/slop-track/v2'
ORACLE = datetime.date(2026, 7, 12)  # first oracle harness commit, ba05d9906


def git(*args, timeout=180):
    return subprocess.run(['git', '-C', REPO] + list(args), capture_output=True,
                          text=True, errors='replace', timeout=timeout).stdout


def commit_date(c):
    return git('show', '-s', '--format=%ad', '--date=short', c).strip()


def parents(c):
    return git('rev-parse', c + '^@').split()


PRS = json.load(open('/tmp/prs.json'))  # number -> {ref, merged_at, merge, title}


def fix_from_pr(pr):
    meta = PRS[str(pr)]
    merge = meta['merge']
    ps = parents(merge)
    fix = ps[1] if len(ps) == 2 else merge
    return fix, meta['ref'], meta['merged_at'][:10]


def ancestry_merge(c):
    out = git('log', '--ancestry-path', '--merges', '--reverse',
              '--format=%H|%s', c + '..origin/main').splitlines()
    if not out:
        return None, None
    m = re.search(r'#(\d+) from [^/]+/(\S+)', out[0])
    return (m.group(1), m.group(2)) if m else (None, None)


def days(a, b):
    da = datetime.date.fromisoformat(a)
    db = datetime.date.fromisoformat(b)
    return (db - da).days


# ---------------------------------------------------------------- git leads
# idx -> (pan_category, proposed_category, caught_by, player_visible, summary,
#         status, notes). Categories are Pan et al. names, verbatim.
GIT = {
 0: ("B4", "", "unknown", "yes",
     "The port batched do_advance's per-level rise lines and suppressed them inside the gain, so rise and autowiz lines did not interleave as C's per-level loop does.",
     "coded", "Same commit also fixed a doubled CRLF on the rise line (output formatting)."),
 1: ("B3", "", "unknown", "yes",
     "Loot moved corpse contents in Go map order rather than C's list traversal order.", "coded", ""),
 2: ("B5", "", "audit", "yes",
     "The oedit disk-save mudlog producer was never ported.", "coded", "DP-1371 mudlog producer train."),
 3: ("B5", "", "unknown", "yes",
     "The ordered-NPC report lane, the stable-horse block and the mount name cache were missing or mis-modelled, and the port invented a 300 MV value C never has.",
     "coded", "Composite; also carries an invented constant."),
 4: ("B5", "", "oracle-depth", "no",
     "checkload reported from prototypes instead of reading throwaway instances, so its RNG draws did not match C's.",
     "coded", "Draw-parity only; player bytes unchanged."),
 5: ("B3", "", "oracle-depth", "yes",
     "damage()'s pain-scream draw ran only on the charge attack type, so a headbutt left the shared RNG stream one draw short.",
     "coded", "Reported by a returning player as colour weirdness."),
 6: ("B5", "", "unknown", "yes",
     "identify's character output and retaliation were missing.", "coded", ""),
 7: ("B3", "", "audit", "yes",
     "Mob socials indexed the social message table directly instead of do_action's guarded slots, panicking on socials with fewer messages than the branch expects.",
     "coded", "Security pass (BUG-R2-C2-A1-H1)."),
 8: ("B5", "", "audit", "yes", "The stable mudlog boundary producer was missing.", "coded", ""),
 9: ("C3", "", "oracle-depth", "yes",
     "galeru_alive's line went through act() instead of send_to_char, printing an extra blank line.", "coded", ""),
 10: ("B5", "", "oracle-depth", "yes",
     "Only a partial damage pre-message rescue tail was ported; the full neutral-rescue tail was missing.", "coded", ""),
 11: ("B5", "", "audit", "yes", "The assassinpc mudlog boundary producer was missing.", "coded", ""),
 12: ("B5", "", "audit", "yes",
     "The object_activity weight-log-and-repair arm was documented as unported and never ran.", "coded", ""),
 13: ("B3", "", "unknown", "yes", "AGGR24 NPC targets were not gated by visibility.", "coded", ""),
 14: ("B5", "", "unknown", "yes",
     "The show shops listing was incomplete (boot index, profit columns, customer mask, headers).", "coded", ""),
 15: ("B5", "", "unknown", "yes", "The protection-from-good death log was missing.", "coded", ""),
 16: ("B3", "", "audit", "no",
     "Reset equipment lost C's occupied-slot refusal and canonical ownership.", "coded", ""),
 17: ("B3", "", "unknown", "yes",
     "The take-to-jail breed caller and its world-people ordering were missing.", "coded", ""),
 18: ("B5", "", "audit", "yes", "The loot mudlog boundary producer was missing.", "coded", ""),
 19: ("B5", "", "unknown", "yes", "C's RawKill teardown and corpse transfer were incomplete.", "coded", ""),
 20: ("B3", "", "audit", "no",
     "Unknown reset commands were not preserved/disabled the way C does.", "coded", ""),
 21: ("B5", "", "audit", "yes", "The purge-player mudlog producer was missing.", "coded", ""),
 22: ("B5", "", "audit", "no", "Bounded file-editor open failures were not logged.", "coded", ""),
 23: ("B4", "", "audit", "no",
     "The port drove point updates from a standalone ticker in addition to the hourly heartbeat.",
     "coded", "Invented extra driver."),
 24: ("B5", "", "unknown", "no",
     "Old rent and aliases were not deleted before a rename save.", "coded", ""),
 25: ("B5", "", "audit", "no",
     "Pending session input was unbounded where C's queue is bounded.", "coded", "Security/vuln sweep."),
 26: ("B3", "", "unknown", "yes",
     "Reports used the wrong mob attack type instead of the prototype's.", "coded", ""),
 27: ("B3", "", "unknown", "yes",
     "CAN_GO was implemented as \"the exit exists\", so wandering mobs walked through closed and locked doors.", "coded", ""),
 28: ("B5", "", "unknown", "no",
     "Show-rent snapshots were not independent as C's are.", "coded", ""),
 29: ("B5", "", "audit", "yes", "The zedit memory-commit mudlog producer was missing.", "coded", ""),
 30: ("C3", "", "unknown", "yes", "The Lua cast refusal wording did not match C.", "coded", ""),
 31: ("B5", "", "playtest", "yes",
     "The DEX arm's TYPE_SUFFERING line was dropped because the body had no registered session during Crash_load.", "coded", ""),
 32: ("C3", "", "unknown", "yes", "Mage reagent output did not match C.", "coded", ""),
 33: ("B3", "", "audit", "no",
     "Zone puts used a local object order instead of C's global order.", "coded", ""),
 34: ("B5", "", "audit", "no",
     "APPLY_RACE_HATE from player equipment was not applied.", "coded", ""),
 35: ("B5", "", "playtest", "yes",
     "process_input's ! repeat and ^old^new substitution were not ported, so '!' answered Huh?!?.",
     "coded", "Found in Zach's playtest 2026-10-15."),
 36: ("B5", "", "audit", "yes", "The force-single mudlog producer was missing.", "coded", ""),
 37: ("B5", "", "audit", "yes", "The medit disk-save mudlog producer was missing.", "coded", ""),
 38: ("B5", "", "audit", "no", "oedit open-parent failures were not logged after save.", "coded", ""),
 39: ("B5", "", "audit", "yes",
     "New-room reset diagnostics at publication were missing.", "coded", ""),
 40: ("B5", "", "audit", "yes",
     "The password check was guarded on a non-empty hash, so a row with a NULL/empty hash authenticated any password.",
     "coded", "Security pass; Go-only fail-open."),
 41: ("B5", "", "audit", "yes", "The oedit memory-commit mudlog producer was missing.", "coded", ""),
 42: ("B5", "", "audit", "yes", "The redit memory-commit mudlog producer was missing.", "coded", ""),
 43: ("B3", "", "audit", "yes",
     "do_retreat's gate checked only exit presence and destination, not CAN_GO's closed-door clause.", "coded", ""),
 44: ("B4", "", "audit", "yes",
     "The port invented CON and hit-point floors C does not have, making C's croak and too-unhealthy entry gate unreachable.",
     "coded", ""),
 45: ("B5", "", "unknown", "yes", "Wizard mob stat fields were missing.", "coded", ""),
 46: ("B3", "", "oracle-depth", "yes",
     "The attack loop kept swinging through a stale FIGHTING pointer that damage() had cleared mid-loop.",
     "coded", "Draw-parity on play-lab-m3-death-in-combat."),
 47: ("B3", "", "oracle-depth", "yes",
     "do_shoot validated direction before the projectile, the reverse of C's order.", "coded", ""),
 48: ("C3", "", "audit", "yes",
     "do_auto_exits emitted no C_CMP door colour, joined entries differently and placed the closing-bracket space wrongly.",
     "coded", "ANSI colour audit."),
 49: ("C3", "", "unknown", "yes",
     "Shop stock lists were rendered and paged differently from C's bytes.", "coded", ""),
 50: ("C3", "", "audit", "yes",
     "print_group emitted no colour at all; C wraps every stat at C_CMP.", "coded", "ANSI colour audit."),
 51: ("B5", "", "audit", "yes", "The inventory mudlog boundary producer was missing.", "coded", ""),
 52: ("B5", "", "audit", "no",
     "The readable-file deletion boundary was not reproduced.", "coded", ""),
 53: ("B3", "", "unknown", "yes",
     "wiznet did not list NOWIZ gods offline or skip non-playing gods as C does.", "coded", ""),
 54: ("C3", "", "audit", "yes",
     "The colour-confirmation ANSI after a level change did not match C.", "coded", ""),
 55: ("B5", "", "audit", "yes",
     "Score's equipment AC and pack weight omitted worn gear.", "coded", ""),
 56: ("B4", "", "oracle-depth", "yes",
     "SPECIAL(beholder) had been invented rather than ported, so its draws and arms did not match C.",
     "coded", "Draw-parity."),
 57: ("B5", "", "oracle-depth", "yes",
     "The mobile wander applied only mobact's gate list, not perform_move/do_simple_move's gates, so a boatless mob left a WATER_NOSWIM room.",
     "coded", "Draw-parity on shop-buy-count-capacity."),
 58: ("B5", "", "audit", "yes", "The reset-world mudlog producer was missing.", "coded", ""),
 59: ("B5", "", "audit", "yes", "The load-mob mudlog producer was missing.", "coded", ""),
 60: ("B4", "", "oracle-depth", "yes",
     "An invented AWAKE gate was present in shared visibility that C's CAN_SEE/PERS/CAN_SEE_OBJ do not have.", "coded", ""),
 61: ("B3", "", "unknown", "yes", "NPC social target order did not match C.", "coded", ""),
 62: ("B5", "", "audit", "no",
     "Five session producers sent without the close guard, so a link-dead takeover could panic the process.",
     "coded", "Security pass."),
 63: ("B3", "", "audit", "yes", "Force notifications did not use C's Act gates.", "coded", ""),
 64: ("B5", "", "oracle-depth", "yes",
     "perform_move's post-entry mp_greet and greet-script order was missing for non-sneaking mobiles.", "coded", ""),
 65: ("B5", "", "audit", "yes",
     "Only close_socket's CON_PLAYING arm was ported; the losing-player and losing-descriptor producers were missing.", "coded", ""),
 66: ("B5", "", "unknown", "yes",
     "Movement guards answered SpecialUnsupported for NPC actors instead of running C's bodies, and the player lane lacked C's immortal/hunting gates.",
     "coded", ""),
 67: ("C2", "", "unknown", "yes",
     "DoUnban trimmed and lowercased the whole argument instead of using one_argument, so \"unban foo bar\" looked up the wrong site.",
     "coded", ""),
 68: ("B5", "", "unknown", "no",
     "Lua set_skill did not return C's stack values.", "coded", ""),
 69: ("B5", "", "unknown", "yes", "Shared arena act broadcasts were missing.", "coded", ""),
 70: ("B3", "", "audit", "yes",
     "lua_log's success arm was sent through the file-FALSE sink, so script log() lines never reached the file.",
     "coded", "DP-1371 mudlog train."),
 71: ("B5", "", "oracle-depth", "yes",
     "do_shoot used the wrong room lookup and lacked C's fallback.", "coded", ""),
 72: ("B5", "", "audit", "yes", "The sedit mudlog producer was missing.", "coded", ""),
 73: ("B5", "", "audit", "yes", "The equipment mudlog boundary producer was missing.", "coded", ""),
 74: ("B5", "", "oracle-depth", "yes",
     "The port retired do_kill's flagged body from combat immediately and refused its second death, where C flags without POS_DEAD and runs the full death branch again; it also invented a '[N]' room-list count.",
     "coded", "Composite; the invented '[N]' suffix is a second root cause."),
 75: ("B5", "", "audit", "yes", "The zonedefault mudlog boundary producer was missing.", "coded", ""),
 76: ("B5", "", "code-review", "yes",
     "Fourteen call sites built objects with the bare constructor, so objects were unregistered (ID 0) and world-wide moves failed.",
     "coded", "Found in #1839 review."),
 77: ("C3", "", "playtest", "yes",
     "do_review wrote plain text with CRLF where C wraps each entry in colour and ends it with C's LF-CR.",
     "coded", "Found in Zach's playtest 2026-10-15."),
 78: ("B5", "", "audit", "yes",
     "CAN_SEE/CAN_SEE_OBJ lacked C's LIGHT_OK room-light arm, so dark rooms rendered wrongly.", "coded", ""),
 79: ("B5", "", "audit", "yes", "The medit memory-commit mudlog producer was missing.", "coded", ""),
 80: ("B5", "", "audit", "yes", "The remort mudlog boundary producer was missing.", "coded", ""),
 81: ("B5", "", "audit", "no",
     "Duplicate-name selection was not serialized with body admission and retirement.", "coded", ""),
 82: ("B5", "", "audit", "yes", "The zedit disk-save mudlog producer was missing.", "coded", ""),
 83: ("B5", "", "unknown", "yes",
     "Shop pet names and detailed descriptions were not preserved as C's are.", "coded", ""),
 84: ("B5", "", "code-review", "yes",
     "say_spell hand-pasted real names for every listener instead of rendering through perform_act, revealing invisible casters and bypassing blindness/darkness.",
     "coded", "Found reviewing #2007."),
 85: ("B5", "", "oracle-depth", "yes",
     "SPECIAL(butler)'s do_gen_door ok_pick draw was not preserved on container transitions.", "coded", ""),
 86: ("B5", "", "unknown", "yes",
     "Counted shop sales and trade refusals were not preserved.", "coded", ""),
 87: ("B5", "", "unknown", "yes",
     "shop_keeper's four shopping arms were missing for NPC actors.",
     "coded", "Commit subject is feat:; a missing port surface."),
 88: ("B5", "", "unknown", "yes",
     "Prompts without an attached player were not preserved.", "coded", ""),
 89: ("B5", "", "audit", "yes", "The oedit mudlog producer was missing.", "coded", ""),
 90: ("B5", "", "audit", "no",
     "Held offline identities were not guarded during file rename.", "coded", ""),
 91: ("B5", "", "audit", "yes",
     "Three combat-round special arms (snake, magic_user, breed_killer) were missing or wrong.",
     "coded", "Class audit."),
 92: ("B5", "", "audit", "no",
     "redit open-parent failures were not logged after save.", "coded", ""),
 93: ("B5", "", "unknown", "yes",
     "Crash_load aborted entry on any object-save read error instead of C's NOTICE-and-admit arm.", "coded", ""),
 94: ("B5", "", "audit", "yes", "The force-all mudlog producer was missing.", "coded", ""),
 95: ("B5", "", "audit", "yes", "The sedit disk-save mudlog producer was missing.", "coded", ""),
 96: ("B5", "", "oracle-depth", "yes",
     "Returning login used the wrong colour flags for the raw MOTD.", "coded", ""),
 97: ("B5", "", "oracle-depth", "yes",
     "C's deferred extraction window and die_with_killer's dying spec arm were not mirrored.", "coded", ""),
 98: ("B5", "", "audit", "yes", "The new-zone zon mudlog producer was missing.", "coded", ""),
 99: ("B3", "", "unknown", "no",
     "C's shop expression pop underflow semantics were not preserved.", "coded", ""),
}


# ------------------------------------------------------------ screening rows
# record_id -> (fix_ref, pan_category, proposed, caught_by, player_visible,
#               summary, c_truth, status, notes)
# fix_ref: PR number (int), commit (str) or None if not established.
SCR = {
 "look-first": (None, "C3", "", "oracle-breadth", "yes",
   "The first live oracle look scenario exposed three player-visible C/Go divergences.",
   "", "coded", "PR #243 is the harness, not the fix; the individual fixes are not established here."),
 "pf-002": ('980560edb', "none-fits", "semantic paraphrase", "code-review", "yes",
   "The Go yank handler was plausible but reworded C's messages and differed in branches, position gating, pronouns and mob handling.",
   "", "coded", "Canonical paraphrase case (DP-1214)."),
 "pf-003": ('1dad2c527', "B5", "", "code-review", "no",
   "A fidelity fix was tested in a function with no live callers, creating verified dead code.",
   "", "coded", ""),
 "pf-005": (601, "B3", "", "oracle-depth", "yes",
   "Flee was breadth-green but depth work still exposed canonical movement, audience, XP, wait-state and force divergences.",
   "", "coded", ""),
 "whirlpool-vehicle": ('1bd91e71c', "B3", "", "oracle-depth", "yes",
   "The whirlpool blocked cluster was root-caused to a wrong-room assumption, MOB_RANDZON scatter and a prototype-only misread.",
   "", "coded", ""),
 "login-crlf": (None, "C3", "", "oracle-depth", "yes",
   "A restart/login CRLF explanation was refuted by raw bytes.",
   "", "excluded", "Refuted hypothesis; no port bug established."),
 "saved-player": (1586, "B5", "", "audit", "yes",
   "Logout/relogin lost durable player state.",
   "", "coded", ""),
 "entry-room": (1659, "B3", "", "oracle-depth", "yes",
   "Login room selection differed from C.",
   "", "coded", ""),
 "idle-clock": (1774, "B3", "", "oracle-depth", "yes",
   "The connected-player idle clock was too short and was not reset.",
   "", "coded", ""),
 "pf-010": (None, "none-fits", "semantic paraphrase", "audit", "yes",
   "The Go couch spec proc paraphrased C's 'the mimic attacks you!' as 'the couch attacks you!', removing the joke.",
   "src/spec_procs2.c:282-310", "coded", "Flagged by the first string census (PR #1587)."),
 "pf-011": ('32896d536', "none-fits", "correcting the source's typo", "code-review", "yes",
   "The port corrected C's typo 'incapacitated an will slowly die' to 'and will'.",
   "", "coded", "Fix commit restores C's typo (DP-1318)."),
 "flag-overlap": ('1a6ce85b9', "C1", "", "audit", "yes",
   "PRF and PLR bit ranges collided, so one flag store overwrote the other.",
   "", "coded", ""),
 "flag-two-stores": ('1a6ce85b9', "C1", "", "audit", "yes",
   "Lycanthropy wrote one PLR store and transform read another.",
   "", "coded", "Same root cause as flag-overlap; one fix, two symptoms."),
 "gsay-neighbor": (1592, "B5", "", "oracle-breadth", "yes",
   "A delegate's PR passed its own new scenario but regressed gsay-depth, which only a full census ran.",
   "", "excluded", "Process/regression incident; the port bug was the missing act() delivery."),
 "browser-renderer": (1606, "B5", "", "audit", "yes",
   "Driving the port through the real browser client turned a green census into 104 web-only failures: the telnet listener had become the byte renderer and the browser client reimplemented none of it.",
   "", "coded", "Found by the new WebSocket oracle driver."),
 "agent-group": (1611, "B4", "", "audit", "yes",
   "A group auto-follow rule invented for agent sessions (no C path) survived every census because every scenario logged in as a human.",
   "", "coded", ""),
 "level-heal": ('369edafcc', "B4", "", "oracle-depth", "yes",
   "The port's AdvanceLevel refilled HP/mana/move, which C's advance_level never does.",
   "src/class.c:698-712", "coded", "Same commit as victim-death-output."),
 "shopkeeper-name-parse": (1627, "B6", "", "oracle-depth", "yes",
   "ok_damage_shopkeeper's typed GET_NAME argument was replaced by a direct slap, diverging when a mob attacker tripped the gate.",
   "src/shop.c:1016-1018", "coded", ""),
 "victim-death-output": ('369edafcc', "B4", "", "oracle-depth", "yes",
   "The victim saw the room's R.I.P. line and two invented death-penalty lines (exp, constitution).",
   "src/fight.c:589-628", "coded", ""),
 "reconnect-identity": (1632, "B5", "", "oracle-depth", "yes",
   "Old-session teardown unregistered the replacement login.",
   "src/interpreter.c:1528-1659", "coded", ""),
 "reconnect-echo": (1632, "B5", "", "oracle-depth", "yes",
   "Reconnect never restored telnet echo.",
   "src/comm.c:954-967", "coded", "Same fix PR as reconnect-identity."),
 "lua-attachments": ('b72c9436a', "B4", "", "audit", "yes",
   "World data attached Lua scripts to 323 mobs where C attaches 5; 318 attachments were keyword-matched from C's unattached archive.",
   "", "coded", "DP-1333 world/script inventory."),
 "lua-inventions": (1637, "B4", "", "code-review", "yes",
   "The port carried Go-only Lua bindings and reconstructed scripts C never had.",
   "src/scripts.c:1609-1664", "coded", ""),
 "lua-shims": (1637, "A3", "", "code-review", "yes",
   "Lua-4 shims copied C's stack API into gopher-lua.",
   "", "coded", ""),
 "lua-format": (1637, "C3", "", "code-review", "yes",
   "Lua-4 format and the %u incompatibility were missing.",
   "", "coded", ""),
 "lua-writeback": (1637, "B3", "", "code-review", "yes",
   "C's run_script write-back reads stack slot 1 and pops nothing, so ch is applied twice and me never; the port did not reproduce that.",
   "src/scripts.c:1805-1815", "coded", ""),
 "ongive-invented": (1637, "B4", "", "code-review", "yes",
   "The port sent an invented \"You can't give that here.\" after a false ongive; no C source has the string.",
   "", "coded", ""),
 "lua-nesting": (1637, "B3", "", "code-review", "no",
   "C nests run_script; the port's engine mutex turned that into a game-loop deadlock.",
   "", "coded", ""),
 "mob-flags": ('cadc62469', "B3", "", "oracle-depth", "yes",
   "Runtime mob flags were read from the prototype instead of the instance.",
   "", "coded", ""),
 "object-flags": ('cadc62469', "B3", "", "oracle-depth", "yes",
   "Changing object flags dropped the prototype's flags.",
   "", "coded", "Same fix commit as mob-flags."),
 "look-auras": ('2e90c2c84', "B5", "", "oracle-depth", "yes",
   "Character look omitted the aura lines.",
   "src/act.informative.c:494-626", "coded", ""),
 "look-space": ('2e90c2c84', "none-fits", "scenario-fitted special case", "oracle-depth", "yes",
   "do_description's trailing-space rule had been reproduced three times as scenario-fitted special cases rather than ported as a rule.",
   "src/act.informative.c:2757-2803", "coded", "PF-040; the census caught the galeru hack doubling the space."),
 "lua-wait-field": ('2e90c2c84', "C1", "", "oracle-depth", "no",
   "PR #1637 added a second GET_MOB_WAIT field for script timer writes although MobInstance.WaitState already was GET_MOB_WAIT, so combat read one copy and scripts wrote the other.",
   "", "coded", "Found by the same author one day later while porting cast_spell's gates."),
 "legacy-spell": ('2e90c2c84', "B4", "", "code-review", "yes",
   "The legacy spell binding invented effects; no C source has the string.",
   "", "coded", "PF-037."),
 "legacy-gossip": ('2e90c2c84', "B4", "", "code-review", "yes",
   "The legacy gossip binding invented a format.",
   "", "coded", "PF-037."),
 "mallory": (None, "B5", "", "audit", "yes",
   "Mallory's revenge hunt was missing (DP-493 reopened 2026-10-09).",
   "", "coded", "Fix not established from the available leads."),
 "mudlog-provider": (1644, "B5", "", "code-review", "yes",
   "The mudlog was implemented but its session provider was never registered, so no immortal ever saw a syslog line.",
   "", "coded", ""),
 "extra-commands": (1646, "B4", "", "audit", "yes",
   "The port answered 36 command names C's table does not have, plus 18 dead aliases shadowed by C's prefix match.",
   "", "coded", "PF-042."),
 "skill-storage": (1649, "C1", "", "audit", "yes",
   "Skill names reached handler keys through two disagreeing paths, so practised martial-arts skills never worked.",
   "", "coded", ""),
 "mp-give": (1652, "B5", "", "audit", "yes",
   "C's mp_give object-give trigger was never ported; performGiveToMob printed the give acts and ran only the ongive script.",
   "src/mobprog.c:112-170", "coded", ""),
 "wait-rounds": ('48e3dabd0', "C1", "", "oracle-depth", "yes",
   "C's +N pulse wait was translated as whole rounds.",
   "src/act.offensive.c:127", "coded", ""),
 "stand-gate": ('48e3dabd0', "B3", "", "oracle-depth", "yes",
   "Already-fighting victims stood unconditionally.",
   "src/fight.c:1443", "coded", ""),
 "position-float": ('0210c7ff9', "A4", "", "oracle-depth", "yes",
   "Position damage used a float multiplier instead of C's integer arithmetic.",
   "src/fight.c:1855", "coded", "Fixed 2026-05-27 as \"restore developer intent\" (DP-515)."),
 "creation-saves": (None, "B5", "", "audit", "yes",
   "C's creation/fresh-save boundaries were missing in Go.",
   "", "coded", "Fix not established from the available leads."),
 "ws-name-hold": (1748, "B5", "", "code-review", "yes",
   "A WebSocket disconnect failed to release the entry-name hold.",
   "", "coded", ""),
 "deleted-admin": (1749, "B4", "", "code-review", "yes",
   "A deleted character retained Go-only admin authorization.",
   "", "coded", ""),
 "audit-input": (1751, "B4", "", "code-review", "yes",
   "A dormant audit logger would have persisted name-prompt input.",
   "", "coded", ""),
 "movement-lock": (1757, "none-fits", "not a fidelity mistake (internal locking)", "code-review", "no",
   "MovementLook took the lifecycle lock unnecessarily.",
   "", "excluded", "Efficiency/robustness, no player-facing C divergence."),
 "mob-wear-slot": (1765, "C1", "", "audit", "yes",
   "Mobs were equipped under C's slot numbers but read under the Go enum.",
   "", "coded", ""),
 "mob-wear-effects": (1765, "B5", "", "audit", "yes",
   "Mobile loot/wear skipped equip effects and reversed the order.",
   "", "coded", ""),
 "foreign-level": (1775, "C1", "", "playtest", "yes",
   "Level thresholds from other DikuMUD derivatives (50/60/61/100) were ported into a game whose ladder ends at LVL_IMPL=40.",
   "src/boards.c:93-98", "coded", "PF-055; each carried a plausible comment."),
 "followers-death": (1774, "B5", "", "playtest", "yes",
   "Followers remained attached after their master died.",
   "", "coded", ""),
 "idle-timeout": (1774, "B4", "", "playtest", "yes",
   "An invented five-minute telnet read timeout was present.",
   "", "coded", ""),
 "body-room-order": ('ded5392c3', "B3", "", "oracle-depth", "yes",
   "Duplicate-body room lookup used the wrong ordinal order.",
   "", "coded", ""),
 "body-consumers": ('ded5392c3', "B3", "", "code-review", "yes",
   "Residual combat consumers compared names instead of bodies.",
   "", "coded", ""),
 "restore-ordinal": ('d1e6626e7', "B3", "", "playtest", "yes",
   "A Go lookup bypassed C's ordinal-aware get_char_vis, so numbered restore targets were refused.",
   "src/act.wizard.c:1584-1593", "coded", ""),
 "ban-bytes": ('31e0e4398', "C3", "", "audit", "yes",
   "A banned telnet connection omitted C's refusal bytes.",
   "", "coded", ""),
 "dominate": ('e84b2b1ac', "B4", "", "audit", "yes",
   "Dominate invented a not-implemented line instead of charming.",
   "", "coded", ""),
 "rent-code": ('34a8f82db', "B5", "", "audit", "no",
   "Durable storage omitted C's rent outcome code.",
   "", "coded", ""),
 "lua-return": ('814d8c5ee', "B3", "", "code-review", "yes",
   "Lua bindings reported success for invalid arguments.",
   "", "coded", ""),
 "bad-stats": (None, "B5", "", "audit", "yes",
   "Player bad-stat equipment effects were missing.",
   "", "coded", "DP-493 reopened; fix not established from the available leads."),
 "host-padding": ('31e0e4398', "C1", "", "audit", "yes",
   "C stores d->host as a zero-padded dotted quad; the port printed raw IPs in ~9 syslog producers and inverted ban matching.",
   "src/comm.c:1536-1548", "coded", "Visible in an oracle transcript 11 days before it was fixed."),
 "ambient-draw": (1974, "B5", "", "oracle-depth", "yes",
   "A blocked scenario was one missing ambient RNG draw: loud_mobs' number(0,20) gate for a noisy mob.",
   "src/spec_procs2.c:1536", "coded", ""),
 "combat-entry": (1982, "B3", "", "oracle-depth", "yes",
   "Combat was enrolled before damage's refusal checks.",
   "src/fight.c:1404-1408", "coded", ""),
 "mob-speech": (1974, "B5", "", "audit", "yes",
   "Mobile speech and kill-brag sinks were unwired or misrouted.",
   "src/fight.c:1693-1697", "coded", ""),
 "headbutt-audience": ('53a751987', "B3", "", "oracle-depth", "yes",
   "do_headbutt's refusal lines were sent to the wrong audience; both go to ch, the descriptor-less mobile.",
   "src/new_cmds.c:413-418", "coded", "Correction to PF-082."),
 "butler-door-args": ('e4a87f803', "B5", "", "audit", "yes",
   "Six butler do_gen_door calls used an inventory-first generic_find path instead of C's literals.",
   "src/spec_procs3.c:172-192", "coded", ""),
 "hunter-doors": (2146, "B3", "", "returning-player", "yes",
   "Hunters bypassed closed/locked door checks.",
   "", "coded", "Returning-player report, Oct 24-25."),
 "lantern": ('6e245a2ba', "B5", "", "returning-player", "yes",
   "Equipping a held lantern failed room-light accounting.",
   "", "coded", "Independently rediscovered by the play lab."),
 "death-affects": (2158, "B5", "", "returning-player", "yes",
   "Player death skipped affect stripping.",
   "", "coded", "Returning-player report."),
 "mana-regen": (None, "B3", "", "returning-player", "yes",
   "Mana equipment regen reversed C's sleeping-only condition.",
   "", "coded", "Fix not established from the available leads."),
 "practice-keys": (2160, "C1", "", "audit", "yes",
   "The practice list read display skill names rather than storage keys.",
   "", "coded", ""),
 "save-totals": (1586, "B3", "", "audit", "no",
   "Reload double-counted saved AC/hitroll/damroll totals.",
   "", "coded", ""),
 "ws-linkdead": (None, "B5", "", "audit", "yes",
   "Closing a browser tab extracted the character instead of leaving it linkdead.",
   "", "coded", "Fix not established from the available leads."),
 "prompt-race": ('c9d2da764', "none-fits", "internal concurrency (no C counterpart)", "code-review", "yes",
   "Prompt state clearing raced the writer under typeahead.",
   "", "coded", "Detected before anyone relied on it (project favour)."),
 "immortal-kill-type": (None, "C1", "", "oracle-depth", "yes",
   "Immortal kill supplied attack type 0 instead of TYPE_SLASH.",
   "", "coded", "Fix not established from the available leads."),
 "house-contents": ('2fe592c99', "B5", "", "audit", "no",
   "House container contents were lost at boot.",
   "", "coded", ""),
 "vampire-line": (1741, "B4", "", "code-review", "yes",
   "Vampirism added a player-only refusal absent from C.",
   "", "coded", ""),
 "identify-preamble": ('3aa6fe6c9', "B4", "", "code-review", "yes",
   "Identify added an invented preamble.",
   "", "coded", ""),
 "cityguard-bribe": ('57a1d30d1', "B3", "", "code-review", "yes",
   "The cityguard bribe probability was inverted.",
   "", "coded", ""),
 "class-table": (None, "C1", "", "code-review", "yes",
   "set class swapped the Magus/Avatar mapping.",
   "", "coded", "Fix not established from the available leads."),
 "shop-audience": (2174, "B4", "", "code-review", "yes",
   "Shop room and keeper messages were replaced by summaries.",
   "", "coded", ""),
 "mail-constants": (None, "C1", "", "code-review", "yes",
   "Mail level/stamp-price constants differed from C.",
   "", "coded", "Fix not established from the available leads."),
 "held-light-timer": ('60be0951f', "B5", "", "code-review", "yes",
   "Held lights never burned: wrong slot and an unwired update.",
   "", "coded", ""),
 "board-remove": (None, "B4", "", "code-review", "yes",
   "Board removal added a read-level permission gate C does not have.",
   "", "coded", "Fix not established from the available leads."),
 "meteor-proof": ('263cbeaf5', "B5", "", "oracle-depth", "yes",
   "The meteor-swarm proof stopped at the peaceful gate, missing C's draw order.",
   "", "coded", ""),
 "unindexed-mob-proof": (2017, "none-fits", "not a fidelity mistake (unrunnable fixture)", "audit", "no",
   "A scenario could not run: unsupported fixture / unindexed mob.",
   "", "excluded", "Tooling gap, not a port defect."),
 "zero-bribe": ('8a9392ff0', "B3", "", "code-review", "yes",
   "A zero-coin mob bribe bypassed perform_give_gold's checks.",
   "", "coded", ""),
 "conjure-ub": ('01d51f054', "none-fits", "certifies source undefined behaviour", "audit", "no",
   "An ordinary fixture certified C NULL-pointer UB.",
   "", "excluded", "Not an LLM-introduced mistake; the C source is undefined."),
 "color-level": ('c1937babe', "B3", "", "code-review", "yes",
   "The colour gate wrongly required both preference bits.",
   "", "coded", ""),
 "affect-index": (None, "B3", "", "code-review", "no",
   "An unchecked affect bit index could panic.",
   "", "coded", "Fix not established from the available leads."),
 "blindness-neighbors": ('dbd23d2ba', "B5", "", "oracle-depth", "yes",
   "Correct smokescreen blindness exposed visibility bypasses.",
   "", "coded", ""),
 "deadcode-alarm": (None, "none-fits", "not a fidelity mistake (tooling false alarm)", "audit", "no",
   "A dead-code alarm (DP-1198) was a false positive.",
   "", "excluded", "Tooling, not a port defect."),
 "god-draws": ('120c5fc5a', "B4", "", "oracle-depth", "no",
   "God character creation consumed two phantom AdvanceLevel RNG draws.",
   "", "coded", "DP-1212."),
 "aiko-entry": ('0f6373303', "B5", "", "playtest", "yes",
   "Aiko's browser entry dialogue and persistence escaped creation coverage.",
   "", "coded", ""),
 "mob-deadlock": (None, "none-fits", "internal concurrency (no C counterpart)", "unknown", "no",
   "A mob-AI self-deadlock froze commands.",
   "", "excluded", "Go-only concurrency defect, no C divergence."),
 "creation-reskin": (None, "B4", "", "oracle-depth", "yes",
   "The character-creation dialogue was re-skinned rather than ported.",
   "", "coded", "Fix not established from the available leads."),
 "tokenizer": (420, "B3", "", "oracle-breadth", "yes",
   "The tokenizer pre-split input, altering C's command semantics.",
   "", "coded", ""),
 "emote-echo": (420, "B4", "", "oracle-breadth", "yes",
   "An invented cmdEmote echo survived review and unit checks.",
   "", "coded", ""),
 "teleport-credited-false-proof": (None, "none-fits", "not a fidelity mistake (process: false proof credited)", "code-review", "no",
   "A teleport false proof was credited in a manifest and handoff before withdrawal.",
   "", "excluded", "Checking-study material, not an LLM-introduced port mistake."),
 "spell-name-withdrawn-diagnostics": (None, "none-fits", "not a fidelity mistake (process: withdrawn proof)", "code-review", "no",
   "Wrong-name mutation diagnostics for Shift Reality / COC were caught before a corrected proof.",
   "", "excluded", "Checking-study material, not an LLM-introduced port mistake."),
}


PAN = {
    "A1": "Violating target language requirements",
    "A2": "Duplicating source syntax to the target language",
    "A3": "Mismatch of API behaviors in the source and target",
    "A4": "Incorrect use of operators",
    "B1": "Missing library imports",
    "B2": "Missing definition",
    "B3": "Incorrect loop and conditional statements",
    "B4": "Inclusion of logic not in the source",
    "B5": "Removal of logic in the source code",
    "B6": "Mismatch of behavior after replacing API call",
    "C1": "Incorrect data type",
    "C2": "Incorrect input parsing",
    "C3": "Output formatting",
    "D": "Model specific constraints",
    "E": "Others",
    "none-fits": "none-fits",
}


def verify_ctruth(ct):
    """Return the citation if it resolves on origin/main, else raise."""
    if not ct:
        return ""
    m = re.match(r'^src/([A-Za-z0-9_.\-]+\.(?:c|h)):(\d+)(?:-(\d+))?$', ct)
    if not m:
        raise ValueError('bad c_truth format: ' + ct)
    f, ln = m.group(1), int(m.group(2))
    blob = git('show', 'origin/main:src/' + f)
    if not blob:
        raise ValueError('missing file: ' + ct)
    if len(blob.splitlines()) < ln:
        raise ValueError('line out of range: ' + ct)
    return ct


def c_truth_quote(ct):
    """The exact C line(s) at the citation, copied verbatim from src/ at
    origin/main (leading whitespace preserved). Multiple lines join with the
    separator ' ~~~ ', which cannot occur in C source."""
    if not ct:
        return ''
    m = re.match(r'^src/([A-Za-z0-9_.\-]+\.(?:c|h)):(\d+)(?:-(\d+))?$', ct)
    if not m:
        raise ValueError('bad c_truth format: ' + ct)
    f, a = m.group(1), int(m.group(2))
    b = int(m.group(3)) if m.group(3) else a
    lines = git('show', 'origin/main:src/' + f).splitlines()[a - 1:b]
    return ' ~~~ '.join(lines)


def src_cite_from_message(msg):
    for c in re.findall(r'src/[A-Za-z0-9_.\-]+\.c:\d+(?:-\d+)?', msg):
        try:
            return verify_ctruth(c)
        except ValueError:
            continue
    return ""


AGENTS = {'glm', 'codex', 'sol', 'deepseek', 'claude', 'cline', 'gemini', 'kimi'}

# Recognised Co-Authored-By names -> agent label. Only these are attributed;
# anything else stays unknown (never guessed from style).
TRAILER_AGENT = {
    'claude': 'claude', 'glm': 'glm', 'codex': 'codex', 'deepseek': 'deepseek',
    'gemini': 'gemini', 'kimi': 'kimi', 'cline': 'cline',
}


def author_from_branch(branch):
    """Agent only from a recognised branch prefix (never guessed from style)."""
    if not branch:
        return 'unknown', ''
    head = branch.split('/')[0]
    if head in AGENTS:
        return head, branch
    return 'unknown', branch


def trailer_agent(msg):
    """Return (agent, evidence) from a Co-Authored-By trailer, else (None, None)."""
    m = re.search(r'Co-Authored-By: ([^\n]+)', msg)
    if not m:
        return None, None
    t = m.group(1).strip()
    low = t.lower()
    for key, agent in TRAILER_AGENT.items():
        if key in low:
            return agent, 'Co-Authored-By: ' + t
    return None, 'Co-Authored-By: ' + t


def author_from_birth(birth):
    """Attribute the birth: its own trailer, else the first PR merge that
    brought it into origin/main, else unknown. Never guessed from style."""
    if not birth:
        return 'unknown', ''
    msg = git('show', '-s', '--format=%B', birth)
    agent, ev = trailer_agent(msg)
    if agent:
        return agent, ev
    merges = git('log', 'origin/main', '--merges', '--ancestry-path',
                 birth + '..origin/main', '--reverse', '--format=%H|%s').splitlines()
    if merges:
        m = re.search(r'#(\d+) from [^/]+/(\S+)', merges[0])
        if m:
            a, ev = author_from_branch(m.group(2))
            return a, ev
    return 'unknown', ''


def agent_from_trailer(msg):
    return trailer_agent(msg)


def row(fields):
    return '\t'.join(str(x).replace('\t', ' ').replace('\n', ' ') for x in fields)


COLS = ['id', 'source', 'summary', 'fix_commit', 'fix_pr', 'fix_date',
        'birth_commit', 'birth_date', 'birth_confidence', 'lifetime_days',
        'born_before_oracle', 'caught_by', 'author_agent', 'author_evidence',
        'pan_category', 'proposed_category', 'c_truth', 'player_visible',
        'status', 'notes', 'fixer_agent', 'fixer_evidence', 'c_truth_quote']

# Hand-verified C citations for rows whose fix commit message carries none.
# Each line was opened in-session and checked to show the behaviour claimed
# (R5g: cite src/, never the oracle checkout).
CTRUTH = {
 'G001': 'src/act.item.c:246', 'G002': 'src/olc.c:203', 'G006': 'src/spells.c:489',
 'G008': 'src/spec_procs2.c:315', 'G011': 'src/spec_procs2.c:879',
 'G013': 'src/mobact.c:199', 'G015': 'src/magic.c:1166', 'G018': 'src/fight.c:1157',
 'G019': 'src/act.offensive.c:50', 'G022': 'src/file-edit.c:48', 'G025': 'src/comm.c:1861',
 'G026': 'src/act.wizard.c:854', 'G030': 'src/spell_parser.c:965', 'G032': 'src/magic.c:465',
 'G034': 'src/handler.c:205', 'G037': 'src/olc.c:210', 'G038': 'src/oedit.c:349',
 'G039': 'src/redit.c:256', 'G045': 'src/act.wizard.c:936', 'G049': 'src/shop.c:868',
 'G051': 'src/act.other.c:432', 'G052': 'src/file-edit.c:40', 'G060': 'src/utils.h:515',
 'G061': 'src/act.social.c:102', 'G063': 'src/act.wizard.c:1864', 'G064': 'src/act.movement.c:311',
 'G068': 'src/scripts.c:1365', 'G072': 'src/sedit.c:1167', 'G073': 'src/act.other.c:475',
 'G075': 'src/zedit.c:1272', 'G076': 'src/db.c:1928', 'G078': 'src/act.informative.c:759',
 'G080': 'src/spec_procs2.c:833', 'G082': 'src/olc.c:196', 'G083': 'src/spec_assign.c:574',
 'G086': 'src/shop.c:753', 'G088': 'src/comm.c:613', 'G089': 'src/oedit.c:1545',
 'G091': 'src/comm.c:136', 'G092': 'src/redit.c:293', 'G095': 'src/olc.c:217',
 'S001': 'src/act.other.c:1629', 'S003': 'src/act.offensive.c:387', 'S004': 'src/spec_procs2.c:244',
 'S006': 'src/db.c:127', 'S007': 'src/comm.c:620', 'S008': 'src/limits.c:438',
 'S010': 'src/fight.c:1572', 'S011': 'src/structs.h:221', 'S012': 'src/structs.h:221',
 'S014': 'src/act.comm.c:824', 'S015': 'src/act.other.c:722', 'S023': 'src/scripts.c:129',
 'S024': 'src/scripts.c:129', 'S026': 'src/act.item.c:704', 'S027': 'src/act.item.c:704',
 'S028': 'src/handler.c:1197', 'S029': 'src/handler.c:1197', 'S032': 'src/utils.h:434',
 'S033': 'src/utils.h:434', 'S034': 'src/utils.h:434', 'S036': 'src/comm.c:784',
 'S037': 'src/interpreter.c:310', 'S038': 'src/act.other.c:543', 'S044': 'src/comm.c:111',
 'S045': 'src/interpreter.c:2341', 'S048': 'src/act.item.c:1461', 'S049': 'src/act.item.c:1461',
 'S052': 'src/limits.c:424', 'S053': 'src/handler.c:1303', 'S054': 'src/handler.c:1303',
 'S057': 'src/spell_parser.c:132', 'S058': 'src/objsave.c:488', 'S059': 'src/scripts.c:90',
 'S067': 'src/act.movement.c:618', 'S068': 'src/handler.c:679', 'S069': 'src/act.offensive.c:50',
 'S071': 'src/act.other.c:543', 'S072': 'src/db.c:2412', 'S077': 'src/act.informative.c:420',
 'S078': 'src/spells.c:489', 'S079': 'src/spec_procs.c:1025', 'S083': 'src/handler.c:1057',
 'S085': 'src/spells.c:1109', 'S087': 'src/act.item.c:740', 'S091': 'src/constants.c:1223',
 'S093': 'src/interpreter.c:69', 'S094': 'src/interpreter.c:1883', 'S097': 'src/interpreter.c:883',
 'S098': 'src/interpreter.c:883',
 'G040': 'src/interpreter.c:1860', 'S000': 'src/act.informative.c:725',
 'S035': 'src/spec_procs.c:1374', 'S043': 'src/act.other.c:198',
 'S056': 'src/comm.c:1567', 'S060': 'src/handler.c:640', 'S070': 'src/limits.c:90',
 'S075': 'src/act.offensive.c:160', 'S080': 'src/class.c:73', 'S082': 'src/mail.c:522',
 'S084': 'src/boards.c:398', 'S089': 'src/act.informative.c:2489', 'S090': 'src/db.c:3068',
}

# Rows retained in the table but not coded: the claim is real (slop-track
# material) but no single src/ line establishes it in this pass, so they cannot
# carry a checkable c_truth. Listed in slop-notes.md open questions.
NOCTRUTH = {'G062', 'G069', 'G099', 'S002', 'S021', 'S046', 'S051', 'S073',
            'S074', 'S076', 'S081', 'S096'}

# ---- v2: citations that v1 pointed at a prototype, blank, brace or comment
# line, or that did not show the behaviour named in the summary. Each new line
# was opened in-session and checked to establish the claim.
CTRUTH_V2 = {
 # prototypes / declaration headers -> the definition or the body
 'G019': 'src/fight.c:573-580',        # death_cry / make_corpse / extract_char
 'G091': 'src/fight.c:2030-2031',      # the per-round MOB_SPEC dispatch
 'S006': 'src/db.c:2531',              # char_to_store definition
 'S069': 'src/fight.c:544-545',        # raw_kill's affect strip
 'S093': 'src/interpreter.c:2214-2216',  # `if (!GET_LEVEL(...)) do_start(...)`
 # comment / blank / brace-leading ranges -> the lines that carry the behaviour
 'G003': 'src/spec_procs2.c:359-360',  # the 230/230 horse pool C really has
 'S004': 'src/spec_procs2.c:258-266',  # the scatter do/while + teleport
 'S008': 'src/limits.c:438-441',       # the IDLE_DISCONNECT arm
 'S016': 'src/class.c:699-703',        # advance_level bumps maxima only
 # SPECIAL(name) declaration lines -> the behaviour inside the special
 'G008': 'src/spec_procs2.c:370-371',  # "Mount not loaded in stable." + mudlog
 'G083': 'src/spec_procs.c:1855',      # pet listing uses GET_NAME
 'S035': 'src/spec_procs.c:1419',      # set_hunting(mickey, ...) revenge mallory
 # a line that did not show the behaviour
 'G025': 'src/comm.c:1863-1868',       # MAX_RAW_INPUT_LENGTH bound + refusal
 'G061': 'src/act.social.c:132',       # get_char_room_vis target order
 'G087': 'src/shop.c:981-996',         # the four CMD_IS shopping arms
 'G088': 'src/comm.c:646-648',         # make_prompt emission behind has_prompt
 'S048': 'src/structs.h:390-395',      # C's WEAR_ slot numbers
 'S057': 'src/spell_parser.c:505-507',  # SPELL_DOMINATE -> MANUAL_SPELL(spell_charm)
 'S079': 'src/mobprog.c:221-225',      # the cityguard bribe probability + hit
 'S080': 'src/class.c:68-74',          # pc_class_types, Magus then Avatar
 'S091': 'src/utils.h:515',            # LIGHT_OK: AFF_BLIND gates sight
}

# ---- v2: rows whose behaviour has no C counterpart at all (Go-only), so no
# src/ line can establish it. Excluded with the reason.
NOCTRUTH_V2 = {
 'S014': 'Go-only browser/WebSocket client: the port had no byte renderer for '
         'the /play transport, and C has no such surface to cite.',
 'S044': 'Go-only WebSocket surface: the entry-name hold is released by the '
         'browser transport close, which C has no counterpart for.',
}

CTRUTH.update(CTRUTH_V2)
NOCTRUTH = NOCTRUTH | set(NOCTRUTH_V2)



def earliest_birth(b):
    """Pick the earliest candidate birth by date (birth_find stored 'hash date')."""
    if not b or not b.get('candidates'):
        return None
    return min(b['candidates'], key=lambda c: c['birth'].split()[1])['birth']


def birth_fields(b, fix_date):
    """Return (commit, date, confidence, lifetime_days, born_before_oracle)."""
    eb = earliest_birth(b)
    if not eb:
        return '', '', 'none', '', ''
    bc, bd = eb.split()[0], eb.split()[1]
    conf = b['confidence']
    life = days(bd, fix_date) if fix_date else ''
    bbo = 'yes' if datetime.date.fromisoformat(bd) < ORACLE else 'no'
    return bc, bd, conf, life, bbo


def build():
    sample = json.load(open(os.path.join(OUT, 'sample.json')))
    drawn = sample['drawn_commits']
    births = json.load(open('/tmp/birth_out.json'))
    scr_births = json.load(open('/tmp/birth_scr.json')) if os.path.exists('/tmp/birth_scr.json') else {}
    out = []

    def finish(rid, source, summ, fixc, pr, fdate, b, caught, cat, prop, ct,
               vis, status, note, fixer_agent, fixer_ev):
        bc, bd, conf, life, bbo = birth_fields(b, fdate)
        author, aev = author_from_birth(bc)
        if rid in NOCTRUTH:
            ct = ''
            status = 'excluded'
            why = NOCTRUTH_V2.get(rid,
                'no single src/ line establishes the claim; see slop-notes open questions.')
            note = (note + ' ' if note else '') + 'Not coded: ' + why
        return row([rid, source, summ, fixc, pr, fdate, bc, bd, conf, life, bbo,
                    caught, author, aev, PAN[cat], prop, verify_ctruth(ct), vis,
                    status, note, fixer_agent, fixer_ev, c_truth_quote(ct)])

    for idx, c in enumerate(drawn):
        cat, prop, caught, vis, summ, status, note = GIT[idx]
        rid = 'G%03d' % idx
        fix_date = commit_date(c)
        pr, branch = ancestry_merge(c)
        fixer, fixer_ev = author_from_branch(branch)
        msg = git('show', '-s', '--format=%B', c)
        if fixer == 'unknown':
            a2, e2 = agent_from_trailer(msg)
            if a2:
                fixer, fixer_ev = a2, e2
        ct = CTRUTH.get(rid) or src_cite_from_message(msg)
        out.append(finish(rid, c, summ, c, pr or '', fix_date, births[c], caught,
                          cat, prop, ct, vis, status, note, fixer, fixer_ev))
    for i, rid in enumerate(SCR):
        fix_ref, cat, prop, caught, vis, summ, ct, status, note = SCR[rid]
        sid = 'S%03d' % i
        fixc = pr = ref = fdate = ''
        if isinstance(fix_ref, int):
            fixc, ref, fdate = fix_from_pr(fix_ref)
            pr = str(fix_ref)
        elif isinstance(fix_ref, str):
            fixc = git('rev-parse', fix_ref).strip()
            fdate = commit_date(fixc)
            p2, b2 = ancestry_merge(fixc)
            pr, ref = p2 or '', b2 or ''
        fixer, fixer_ev = author_from_branch(ref)
        ct = CTRUTH.get(sid) or ct
        sb = scr_births.get(fixc) if fixc else None
        out.append(finish(sid, rid, summ, fixc, pr, fdate, sb, caught, cat, prop,
                          ct, vis, status, note, fixer, fixer_ev))
    return out


if __name__ == '__main__':
    lines = [row(COLS)] + build()
    open(os.path.join(OUT, 'slop.tsv'), 'w').write('\n'.join(lines) + '\n')
    print('rows', len(lines) - 1)
