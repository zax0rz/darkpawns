#!/usr/bin/env python3
"""
gen_reachability.py — Deterministic "port reachability" report for Dark Pawns.

Parses the C command table in src/interpreter.c and determines Go-side
reachability for every command. Outputs a TSV to docs/reports/.

Usage:
    python3 scripts/gen_reachability.py
    python3 scripts/gen_reachability.py --out /path/to/report.tsv
"""

import argparse
import re
import sys
from pathlib import Path
from collections import OrderedDict

ROOT = Path(__file__).resolve().parent.parent

# ---------------------------------------------------------------------------
# Step A: Parse C command table from src/interpreter.c
# ---------------------------------------------------------------------------

# Known abbreviation stubs that exist as separate safety entries alongside
# the real command. These are not real commands but prefix-abbreviation
# entries that CircleMUD uses for prefix matching.
ABBREV_STUBS = {"qui", "shutdow"}

# The "whod  " entry has trailing spaces in the C source — it's a typo/duplicate
# of the real "whod" command (registered in Go). We flag it as a stub.
TYPO_STUBS = {"whod  "}


def parse_c_command_table(path: Path) -> list[dict]:
    """Parse the cmd_info[] array in interpreter.c.

    Returns a list of dicts with keys:
        command, c_handler, c_min_position, c_min_level, c_subcmd,
        is_stub
    Skips the RESERVED entry and the sentinel "\\n" entry.
    """
    text = path.read_text(encoding="utf-8", errors="replace")

    # Find the cmd_info array body: from "cmd_info[] = {" to the closing "};"
    start_marker = "const struct command_info cmd_info[] = {"
    start_idx = text.find(start_marker)
    if start_idx == -1:
        raise ValueError("Could not find cmd_info[] array in interpreter.c")

    # Find the sentinel entry which marks the end
    sentinel_marker = '{ "\\n"'
    end_idx = text.find(sentinel_marker, start_idx)
    if end_idx == -1:
        raise ValueError("Could not find sentinel entry in interpreter.c")

    body = text[start_idx + len(start_marker) : end_idx]

    # Strip comments: C-style /* ... */ and //
    # Do a simple pass: remove /* ... */ blocks and // line comments
    body = re.sub(r'/\*.*?\*/', '', body, flags=re.DOTALL)
    body = re.sub(r'//[^\n]*', '', body)

    entries = []

    # Pattern to match a single entry: { "name", POS_CONST, handler, min_level, subcmd }
    # The entry format varies; some have trailing TRUE instead of subcmd (e.g., shadow).
    # Structure: { "command" , POS_XXX , do_xxx , LEVEL_CONST , VALUE }
    #
    # We use a multi-step approach: find each opening brace, then capture the contents.
    # More robust: split on "}," and parse each segment.

    # Collect raw entry strings between { and },
    brace_depth = 0
    current_entry = ""
    raw_entries = []

    for ch in body:
        if ch == '{':
            brace_depth += 1
            if brace_depth == 1:
                current_entry = ""
                continue
        if ch == '}':
            brace_depth -= 1
            if brace_depth == 0:
                raw_entries.append(current_entry.strip())
                continue
        if brace_depth >= 1:
            current_entry += ch

    for raw in raw_entries:
        # Skip empty or RESERVED
        if not raw or "RESERVED" in raw:
            continue

        # Parse fields: command, position, handler, min_level, subcmd
        # Fields are separated by commas, but we need to be careful with
        # string literal commas (none in these entries) and nested parens (none).
        #
        # Standard format: "name", POS_XXX, do_xxx, LEVEL, SCMD_XXX
        # Some entries:   "name", POS_XXX, do_xxx, 0, 0
        # Some entries:   "name", POS_XXX, do_xxx, 0, SCMD_XXX
        # Some entries:   "name", POS_XXX, do_xxx, 0, TRUE   (shadow)
        # Special:        "name", POS_XXX, do_xxx, LVL_XXX, LVL_XXX (luaedit)

        fields = [f.strip() for f in raw.split(",")]

        if len(fields) < 4:
            continue

        # Field 0: command name (quoted string)
        cmd_name = fields[0].strip('"').strip()

        # Field 1: minimum position
        c_min_position = fields[1].strip()

        # Field 2: handler function
        c_handler = fields[2].strip()

        # Field 3: minimum level
        c_min_level = fields[3].strip()

        # Field 4: subcommand (may be absent)
        c_subcmd = fields[4].strip() if len(fields) >= 5 else "0"

        # Determine if this is an abbreviation stub
        is_stub = cmd_name in ABBREV_STUBS or cmd_name in TYPO_STUBS

        entries.append({
            "command": cmd_name,
            "c_handler": c_handler,
            "c_min_position": c_min_position,
            "c_min_level": c_min_level,
            "c_subcmd": c_subcmd,
            "is_stub": is_stub,
        })

    return entries


# ---------------------------------------------------------------------------
# Step B: Determine Go-side reachability
# ---------------------------------------------------------------------------

def parse_go_registry(session_dir: Path) -> set[str]:
    """Parse registerCommand calls from all pkg/session/*.go files.

    Returns a set of lowercase registered names (primary names + aliases).
    """
    registered = set()

    go_files = sorted(session_dir.glob("*.go"))
    for go_file in go_files:
        if go_file.name.endswith("_test.go"):
            continue
        text = go_file.read_text(encoding="utf-8", errors="replace")
        lines = text.split("\n")

        for i, line in enumerate(lines):
            if "registerCommand(" not in line:
                continue

            # Collect continuation lines until we see a closing paren
            full_call = line
            if ");" not in line:
                for j in range(i + 1, min(i + 5, len(lines))):
                    full_call += " " + lines[j].strip()
                    if ");" in lines[j]:
                        break

            # Extract string literals
            strings = re.findall(r'"([^"]*)"', full_call)

            if not strings:
                continue

            # First string is the primary command name
            primary = strings[0].strip().lower()
            if primary and " " not in primary and len(primary) < 50:
                registered.add(primary)

            # Remaining strings could be help text or aliases.
            # Heuristic: aliases are short, no spaces, not help-like.
            for s in strings[1:]:
                s = s.strip().lower()
                if not s:
                    continue
                # Skip help text (long, contains spaces, starts with capital or has punctuation)
                if " " in s and len(s) > 15:
                    continue
                if len(s) > 50:
                    continue
                # Skip obvious help text
                if s.startswith("show ") or s.startswith("list ") or s.startswith("set "):
                    continue
                if any(c in s for c in ".!?,") and len(s) > 10:
                    continue
                # Multi-word aliases like "pick lock" are legit
                if len(s) <= 20:
                    registered.add(s)

    return registered


def parse_go_socials(path: Path) -> set[str]:
    """Parse pkg/game/socials.txt for social names.

    Format: name min_pos min_level (tab or space separated)
    """
    socials = set()
    text = path.read_text(encoding="utf-8", errors="replace")
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split()
        if parts:
            socials.add(parts[0].lower())
    return socials


def parse_specproc_intercepts(glob_pattern: str) -> dict[str, str]:
    """Scan spec_proc*.go files for command keywords they intercept.

    Returns dict mapping command_name -> source_file.
    Looks for patterns like: cmd == "commandname" or cmd != "commandname"
    (both imply interception; != is used for early-return guards).
    """
    intercepts = {}
    for f in sorted(Path(ROOT).glob(glob_pattern)):
        if f.name.endswith("_test.go"):
            continue
        text = f.read_text(encoding="utf-8", errors="replace")
        # Find all cmd == "xyz" and cmd != "xyz" patterns
        matches = re.findall(r'cmd\s*[!=]=\s*"([^"]+)"', text)
        for cmd in matches:
            cmd_lower = cmd.strip().lower()
            if cmd_lower and cmd_lower not in intercepts:
                intercepts[cmd_lower] = f"pkg/game/{f.name}"
    return intercepts


def parse_go_handlers(commands_dir: Path) -> dict[str, str]:
    """Scan pkg/command/*.go for exported/unexported handler functions.

    Returns dict mapping handler_name_lower -> filename.
    Looks for: func CmdXxx(...) and func cmdXxx(...)
    """
    handlers = {}
    for f in sorted(commands_dir.glob("*.go")):
        if f.name.endswith("_test.go"):
            continue
        text = f.read_text(encoding="utf-8", errors="replace")
        matches = re.findall(r'func\s+(Cmd\w+|cmd\w+)\s*\(', text)
        for h in matches:
            h_lower = h.lower()
            if h_lower not in handlers:
                handlers[h_lower] = f"pkg/command/{f.name}"
    return handlers


def parse_session_handlers(session_dir: Path) -> dict[str, str]:
    """Scan pkg/session/*.go for handler functions.

    Returns dict mapping handler_name_lower -> filename.
    """
    handlers = {}
    for f in sorted(session_dir.glob("*.go")):
        if f.name.endswith("_test.go"):
            continue
        text = f.read_text(encoding="utf-8", errors="replace")
        matches = re.findall(r'func\s+(Cmd\w+|cmd\w+)\s*\(', text)
        for h in matches:
            h_lower = h.lower()
            if h_lower not in handlers:
                handlers[h_lower] = f"pkg/session/{f.name}"
    return handlers


def handler_to_command_name(handler: str) -> str:
    """Convert a C handler name like do_move to a plausible command name stem.

    e.g. do_move -> move, do_dragon_kick -> dragon_kick, do_gen_comm -> gen_comm
    """
    if handler.startswith("do_"):
        return handler[3:]
    # adjust_mobs -> admobs (special case in C table)
    return handler


def find_plausible_go_handler(
    c_cmd: str, c_handler: str,
    go_cmd_handlers: dict[str, str],
    go_session_handlers: dict[str, str]
) -> str | None:
    """Heuristically check if a Go handler exists for a given C command.

    Returns evidence string like "pkg/command/skill_commands.go:CmdBearhug" or None.
    """
    cmd_lower = c_cmd.lower()
    handler_stem = handler_to_command_name(c_handler)

    # Build candidate Go function names
    candidates = []

    # Direct: CmdBearhug for "bearhug"
    title = cmd_lower.replace("_", "").replace("-", "")
    candidates.append(f"cmd{title}")

    # Handler stem based: do_dragon_kick -> cmddragonkick
    stem_title = handler_stem.replace("_", "")
    candidates.append(f"cmd{stem_title}")

    # Special cases
    special_map = {
        "aid": "cmdfirstaid",
        "alter": "cmdfleshalter",
        "flesh": "cmdfleshalter",
        "serpent": "cmdserpentkick",
        "search": "cmddetect",
        "detect": "cmddetect",
        "dragon": "cmddragonkick",
        "tiger": "cmdtigerpunch",
        "abils": "cmdabils",
        "abilities": "cmdabils",
        "first_aid": "cmdfirstaid",
        "flesh_alter": "cmdfleshalter",
        "dragon_kick": "cmddragonkick",
        "tiger_punch": "cmdtigerpunch",
        "serpent_kick": "cmdserpentkick",
        "kuji_kiri": "cmdkujikiri",
        "gen_comm": "cmdgencomm",
        "gen_door": "cmdgendoor",
        "gen_ps": "cmdgenps",
        "gen_tog": "cmdgentog",
        "gen_write": "cmdgenwrite",
        "spec_comm": "cmdspeccomm",
        "not_here": "cmdnothere",
        "first_aid": "cmdfirstaid",
        "whois": "cmdwhois",  # do_whois for "finger"
    }
    if cmd_lower in special_map:
        candidates.append(special_map[cmd_lower])
    if handler_stem in special_map:
        candidates.append(special_map[handler_stem])

    # Also try Cmd + title case
    title_case = "".join(
        word.capitalize() for word in cmd_lower.replace("_", " ").replace("-", " ").split()
    )
    candidates.append(f"cmd{title_case}")

    # Also for handler stem
    stem_title_case = "".join(
        word.capitalize() for word in handler_stem.replace("_", " ").split()
    )
    candidates.append(f"cmd{stem_title_case}")

    # Search
    all_handlers = {**go_cmd_handlers, **go_session_handlers}
    for cand in candidates:
        if cand in all_handlers:
            return f"{all_handlers[cand]}:{cand}"

    return None


# ---------------------------------------------------------------------------
# Step C & D: Classification
# ---------------------------------------------------------------------------

def classify_command(
    entry: dict,
    go_registry: set[str],
    go_socials: set[str],
    specproc_intercepts: dict[str, str],
    go_cmd_handlers: dict[str, str],
    go_session_handlers: dict[str, str],
) -> tuple[str, str]:
    """Classify a single C command entry.

    Returns (status, go_evidence).
    """
    cmd = entry["command"].lower()
    handler = entry["c_handler"]
    is_stub = entry["is_stub"]

    # Abbreviation stubs
    if is_stub:
        return ("abbrev-stub", "abbreviation-only entry in interpreter.c")

    # Socials (do_action handler)
    if handler == "do_action":
        if cmd in go_socials:
            return ("social", "pkg/game/socials.txt")
        else:
            return ("missing-social", "not found in pkg/game/socials.txt")

    # Check Go registry (exact match)
    if cmd in go_registry:
        return ("registered", f"pkg/session/commands.go:registerCommand(\"{cmd}\", ...)")

    # Check specproc interception
    if cmd in specproc_intercepts:
        return ("specproc", specproc_intercepts[cmd])

    # Check for plausible Go handler
    impl = find_plausible_go_handler(
        entry["command"], handler,
        go_cmd_handlers, go_session_handlers
    )
    if impl:
        return ("implemented-unwired", impl)

    return ("missing", "no implementation found")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Generate deterministic port reachability report."
    )
    parser.add_argument(
        "--out",
        default=str(ROOT / "docs" / "reports" / "reachability-2026-07-22.tsv"),
        help="Output TSV path (default: docs/reports/reachability-2026-07-22.tsv)",
    )
    args = parser.parse_args()

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    # Parse C command table
    c_entries = parse_c_command_table(ROOT / "src" / "interpreter.c")

    # Parse Go registry (all session/*.go files)
    go_registry = parse_go_registry(ROOT / "pkg" / "session")

    # Parse Go socials
    go_socials = parse_go_socials(ROOT / "pkg" / "game" / "socials.txt")

    # Parse specproc intercepts
    specproc_intercepts = parse_specproc_intercepts("pkg/game/spec_proc*.go")

    # Parse Go handlers
    go_cmd_handlers = parse_go_handlers(ROOT / "pkg" / "command")
    go_session_handlers = parse_session_handlers(ROOT / "pkg" / "session")

    # Classify each entry
    results = []
    for entry in c_entries:
        status, evidence = classify_command(
            entry, go_registry, go_socials, specproc_intercepts,
            go_cmd_handlers, go_session_handlers
        )
        results.append({
            **entry,
            "status": status,
            "go_evidence": evidence,
        })

    # Sort by command name (deterministic)
    results.sort(key=lambda r: r["command"].lower())

    # Write TSV
    tsv_header = "command\tc_handler\tc_min_position\tc_min_level\tc_subcmd\tstatus\tgo_evidence"
    tsv_lines = [tsv_header]
    for r in results:
        tsv_lines.append(
            f"{r['command']}\t{r['c_handler']}\t{r['c_min_position']}"
            f"\t{r['c_min_level']}\t{r['c_subcmd']}\t{r['status']}\t{r['go_evidence']}"
        )

    out_path.write_text("\n".join(tsv_lines) + "\n", encoding="utf-8")
    print(f"Wrote {len(results)} rows to {out_path}")

    # Summary statistics
    total = len(results)
    socials = [r for r in results if r["status"] in ("social", "missing-social")]
    non_socials = [r for r in results if r["status"] not in ("social", "missing-social", "abbrev-stub")]
    stubs = [r for r in results if r["status"] == "abbrev-stub"]

    status_counts: dict[str, int] = {}
    for r in results:
        s = r["status"]
        status_counts[s] = status_counts.get(s, 0) + 1

    print()
    print("=" * 72)
    print("PORT REACHABILITY REPORT — 2026-07-22")
    print("=" * 72)
    print()
    print(f"  {'Category':<30} {'Count':>6}")
    print(f"  {'-'*30} {'-'*6}")
    print(f"  {'Total C entries':<30} {total:>6}")
    print(f"  {'  Socials':<30} {len(socials):>6}")
    print(f"  {'  Non-socials':<30} {len(non_socials):>6}")
    print(f"  {'  Abbreviation stubs':<30} {len(stubs):>6}")
    print()
    print(f"  {'Status':<30} {'Count':>6}")
    print(f"  {'-'*30} {'-'*6}")
    for status in sorted(status_counts.keys()):
        label = f"  {status}"
        print(f"  {label:<30} {status_counts[status]:>6}")
    print()
    print(f"  TSV rows: {len(results)}")

    # Sanity checks
    print()
    print("  --- Sanity checks ---")
    sanity_ok = True

    # Check expected ballpark
    if not (480 <= total <= 530):
        print(f"  WARNING: total entries ({total}) outside expected ~503 ballpark")
        sanity_ok = False
    if not (160 <= len(socials) <= 200):
        print(f"  WARNING: social count ({len(socials)}) outside expected ~183 ballpark")
        sanity_ok = False
    if not (290 <= len(non_socials) <= 350):
        print(f"  WARNING: non-social count ({len(non_socials)}) outside expected ~320 ballpark")
        sanity_ok = False

    # Check specific commands
    must_be_registered = {"north", "look", "kill", "cast", "inventory"}
    for cmd in must_be_registered:
        found = [r for r in results if r["command"].lower() == cmd]
        if not found:
            print(f"  FAIL: '{cmd}' not found in parsed entries")
            sanity_ok = False
        elif found[0]["status"] not in ("registered", "alias"):
            print(f"  FAIL: '{cmd}' status is '{found[0]['status']}' (expected registered/alias)")
            sanity_ok = False
        else:
            print(f"  OK: '{cmd}' → {found[0]['status']}")

    # dragon, tiger, flesh should be registered/alias
    for cmd in ("dragon", "tiger", "flesh"):
        found = [r for r in results if r["command"].lower() == cmd]
        if not found:
            print(f"  FAIL: '{cmd}' not found in parsed entries")
            sanity_ok = False
        elif found[0]["status"] not in ("registered", "alias"):
            print(f"  FAIL: '{cmd}' status is '{found[0]['status']}' (expected registered/alias)")
            sanity_ok = False
        else:
            print(f"  OK: '{cmd}' → {found[0]['status']}")

    # dig and mold should be unwired or missing
    for cmd in ("dig", "mold"):
        found = [r for r in results if r["command"].lower() == cmd]
        if not found:
            print(f"  FAIL: '{cmd}' not found in parsed entries")
            sanity_ok = False
        elif found[0]["status"] not in ("implemented-unwired", "missing"):
            print(f"  NOTE: '{cmd}' status is '{found[0]['status']}' (expected unwired/missing)")
        else:
            print(f"  OK: '{cmd}' → {found[0]['status']}")

    if sanity_ok:
        print("\n  All sanity checks passed.")
    else:
        print("\n  Some sanity checks FAILED — review above.")

    # List stub entries
    if stubs:
        print(f"\n  Abbreviation stubs ({len(stubs)}):")
        for r in stubs:
            print(f"    - {r['command']}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
