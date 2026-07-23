#!/usr/bin/env python3
"""
gen_scenario_coverage.py — Deterministic "scenario coverage" analyzer for Dark Pawns.

Parses every oracle-diff scenario file in cmd/dp-oracle-diff/scenarios/*.txt
and determines, for each command in the C command table, whether any scenario
has ever probed it.

A "probe" is a line inside a [probe] section.  Setup / fixture / creation /
warmup sections are never treated as commands — they are server-creation
keystrokes that are drained before the probe phase.

Resolution rule (mirrors C command_interpreter, interpreter.c:909-912):
  The first row in cmd_info[] whose name has the typed word as a PREFIX
  AND whose min_level <= PLAYER_LEVEL wins.

  Assumption: PLAYER_LEVEL = 1 (scenarios run as fresh mortals).

This is DOCUMENTED here because it is load-bearing for correctness:
  - "go hello"  → gossip  (goto is L31, skipped; gossip L0 wins)
  - "qui"       → qui     (the abbreviation stub, not quit)
  - "l"         → look    (first L0 entry whose name starts with "l")
  - "in"        → inventory (first L0 "in*" entry)
  - "sa"        → say     (first L0 "sa*" entry)
  - "da"        → dance   (first L0 "da*" entry)

Unresolved probe words (no matching C table row at level 1) are collected
and reported loudly — they are either scenario typos or resolution-rule bugs.

Usage:
    python3 scripts/gen_scenario_coverage.py
    python3 scripts/gen_scenario_coverage.py --out /path/to/report.tsv
"""

import argparse
import datetime
import sys
from collections import OrderedDict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# ---------------------------------------------------------------------------
# Step A: Parse the C command resolution table (command_order.tsv)
# ---------------------------------------------------------------------------

def parse_command_order(path: Path) -> list[dict]:
    """Parse pkg/session/command_order.tsv.

    Returns a list of dicts with keys: seq, name, min_level.
    Preserves source order (the TSV is already in cmd_info[] order).
    """
    rows = []
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split("\t")
        if len(parts) < 3:
            continue
        seq = int(parts[0].strip())
        name = parts[1].strip().lower()
        min_level = int(parts[2].strip())
        rows.append({"seq": seq, "name": name, "min_level": min_level})
    return rows


# ---------------------------------------------------------------------------
# Step B: Parse the reachability report for command metadata
# ---------------------------------------------------------------------------

def parse_reachability(path: Path) -> dict[str, str]:
    """Parse docs/reports/reachability-2026-07-22.tsv.

    Returns dict mapping command_name (lowercase) -> reach_status.
    """
    result = OrderedDict()
    text = path.read_text(encoding="utf-8").splitlines()
    if not text:
        return result
    header = text[0].strip().split("\t")
    # Expected: command, c_handler, c_min_position, c_min_level, c_subcmd, status, go_evidence
    for line in text[1:]:
        line = line.strip()
        if not line:
            continue
        parts = line.split("\t")
        if len(parts) < 7:
            continue
        cmd = parts[0].strip().lower()
        status = parts[5].strip()
        result[cmd] = status
    return result


# ---------------------------------------------------------------------------
# Step C: Parse scenario files and extract probe words
# ---------------------------------------------------------------------------

def extract_probe_words(scenarios_dir: Path) -> dict[str, set[str]]:
    """Parse every *.txt scenario file in scenarios_dir.

    Returns dict mapping probe_word (lowercased, tokenized) -> set of scenario basenames.
    """
    probe_word_scenarios: dict[str, set[str]] = {}

    for scenario_path in sorted(scenarios_dir.glob("*.txt")):
        basename = scenario_path.name
        text = scenario_path.read_text(encoding="utf-8")

        in_probe = False
        for line in text.splitlines():
            stripped = line.strip()

            # Track section boundaries
            if stripped.startswith("["):
                if stripped.startswith("[probe]"):
                    in_probe = True
                else:
                    in_probe = False
                continue

            # Only process lines inside [probe] sections
            if not in_probe:
                continue

            # Skip comment lines and <ENTER> (empty lines)
            if not stripped or stripped.startswith("#"):
                continue
            if stripped.upper() == "<ENTER>":
                continue

            # Tokenize: extract the command word
            word = tokenize_command(stripped)
            if word is None:
                continue

            if word not in probe_word_scenarios:
                probe_word_scenarios[word] = set()
            probe_word_scenarios[word].add(basename)

    return probe_word_scenarios


def tokenize_command(line: str) -> str | None:
    """Extract the command word from a probe line using the C tokenization rule.

    C rule (interpreter.c:883-907):
      - Strip leading whitespace.
      - If the first character is NOT a letter (a-z/A-Z), the command is
        that single character.
      - Otherwise the command is the text up to the first whitespace.
      - Lowercase the result.

    Returns None if the line has no tokenizable content.
    """
    stripped = line.lstrip()
    if not stripped:
        return None

    first_char = stripped[0]

    # Non-alpha first char → command is that single character
    if not (("a" <= first_char <= "z") or ("A" <= first_char <= "Z")):
        return first_char.lower()

    # Alpha first char → consume until whitespace
    end = 0
    for i, ch in enumerate(stripped):
        if ch in (" ", "\t"):
            break
        end = i + 1

    return stripped[:end].lower()


# ---------------------------------------------------------------------------
# Step D: Resolve probe words to canonical C commands
# ---------------------------------------------------------------------------

def resolve_probe_words(
    probe_word_scenarios: dict[str, set[str]],
    command_order: list[dict],
    player_level: int = 1,
) -> tuple[dict[str, set[str]], dict[str, set[str]]]:
    """Resolve each probe word to its canonical C command.

    Resolution rule (mirrors C command_interpreter):
      Scan the command_order rows in order; the first row whose name has
      the typed word as a PREFIX AND whose min_level <= player_level wins.

    Returns (resolved: canonical_cmd -> scenario_set,
             unresolved: probe_word -> scenario_set).
    """
    resolved: dict[str, set[str]] = {}
    unresolved: dict[str, set[str]] = {}

    for word, scenarios in sorted(probe_word_scenarios.items()):
        match = None
        for row in command_order:
            if row["name"].startswith(word) and row["min_level"] <= player_level:
                match = row["name"]
                break

        if match is not None:
            if match not in resolved:
                resolved[match] = set()
            resolved[match].update(scenarios)
        else:
            if word not in unresolved:
                unresolved[word] = set()
            unresolved[word].update(scenarios)

    return resolved, unresolved


# ---------------------------------------------------------------------------
# Step E: Build the coverage report
# ---------------------------------------------------------------------------

def build_report(
    reachability: dict[str, str],
    resolved: dict[str, set[str]],
) -> list[dict]:
    """Cross-reference resolved probes against the reachability TSV.

    Returns a list of dicts with keys: command, reach_status, coverage, scenarios.
    Sorted by command name.
    """
    rows = []
    for cmd, status in reachability.items():
        if cmd in resolved:
            coverage = "probed"
            scenarios = ",".join(sorted(resolved[cmd]))
        else:
            coverage = "never-probed"
            scenarios = ""
        rows.append({
            "command": cmd,
            "reach_status": status,
            "coverage": coverage,
            "scenarios": scenarios,
        })

    # Sort by command name (deterministic)
    rows.sort(key=lambda r: r["command"])
    return rows


# ---------------------------------------------------------------------------
# Step F: Self-verification
# ---------------------------------------------------------------------------

def self_verify(
    resolved: dict[str, set[str]],
    unresolved: dict[str, set[str]],
    scenario_files: list[str],
    rows: list[dict],
) -> bool:
    """Run sanity checks and print results. Returns True if all pass."""
    ok = True

    def check(condition: bool, msg: str) -> None:
        nonlocal ok
        if condition:
            print(f"  OK: {msg}")
        else:
            print(f"  FAIL: {msg}")
            ok = False

    print()
    print("  --- Self-verification ---")

    # 1. look must be probed (look-start-room.txt)
    look_row = next((r for r in rows if r["command"] == "look"), None)
    check(
        look_row is not None and look_row["coverage"] == "probed" and "look-start-room.txt" in look_row["scenarios"],
        "'look' is probed (look-start-room.txt)",
    )

    # 2. grats must be probed (command-surface-punctuation.txt)
    grats_row = next((r for r in rows if r["command"] == "grats"), None)
    check(
        grats_row is not None and grats_row["coverage"] == "probed" and "command-surface-punctuation.txt" in grats_row["scenarios"],
        "'grats' is probed (command-surface-punctuation.txt)",
    )

    # 3. gratz must be unresolved (NOT in C table — deliberate negative probe)
    check(
        "gratz" in unresolved,
        "'gratz' is unresolved (deliberate negative probe — not in C table)",
    )

    # 4. 'go hello' must resolve to 'gossip' (level-1 skip of goto)
    gossip_row = next((r for r in rows if r["command"] == "gossip"), None)
    check(
        gossip_row is not None and "command-abbreviations.txt" in gossip_row["scenarios"],
        "'go hello' resolves to 'gossip' (level-1 skip of goto)",
    )

    # 5. ' must resolve to ' (the say shorthand row)
    apostrophe_row = next((r for r in rows if r["command"] == "'"), None)
    check(
        apostrophe_row is not None and apostrophe_row["coverage"] == "probed",
        "''' resolves to ' (say shorthand row)",
    )

    # 6. zz must be unresolved (deliberate miss probe)
    check(
        "zz" in unresolved,
        "'zz' is unresolved (deliberate miss probe)",
    )

    # 7. Scenario file count should be ~33-34
    check(
        30 <= len(scenario_files) <= 40,
        f"Scenario file count is {len(scenario_files)} (~33-34 expected)",
    )

    return ok


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Generate deterministic scenario-coverage report."
    )
    parser.add_argument(
        "--out",
        default=str(
            ROOT / "docs" / "reports"
            / f"scenario-coverage-{datetime.date.today().isoformat()}.tsv"
        ),
        help="Output TSV path (default: docs/reports/scenario-coverage-<today>.tsv)",
    )
    args = parser.parse_args()

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    scenarios_dir = ROOT / "cmd" / "dp-oracle-diff" / "scenarios"
    command_order_path = ROOT / "pkg" / "session" / "command_order.tsv"
    reachability_path = ROOT / "docs" / "reports" / "reachability-2026-07-22.tsv"

    PLAYER_LEVEL = 1  # Scenarios run as fresh mortals

    # --- Parse inputs ---
    command_order = parse_command_order(command_order_path)
    print(f"Parsed {len(command_order)} rows from {command_order_path.relative_to(ROOT)}")

    reachability = parse_reachability(reachability_path)
    print(f"Parsed {len(reachability)} commands from {reachability_path.relative_to(ROOT)}")

    # --- Extract probe words from scenarios ---
    probe_word_scenarios = extract_probe_words(scenarios_dir)

    scenario_files = sorted(p.name for p in scenarios_dir.glob("*.txt"))
    print(f"Parsed {len(scenario_files)} scenario files from {scenarios_dir.relative_to(ROOT)}")

    # --- Resolve probe words ---
    resolved, unresolved = resolve_probe_words(
        probe_word_scenarios, command_order, PLAYER_LEVEL
    )

    # --- Build report ---
    rows = build_report(reachability, resolved)

    # --- Write TSV ---
    tsv_header = "command\treach_status\tcoverage\tscenarios"
    tsv_lines = [tsv_header]
    # Add a comment line documenting the assumption
    tsv_lines.append(
        "# Assumption: player level = 1 (scenarios run as fresh mortals)."
    )
    tsv_lines.append(
        f"# Resolution table: pkg/session/command_order.tsv ({len(command_order)} rows)."
    )
    tsv_lines.append(
        f"# Scenario files parsed: {len(scenario_files)}."
    )
    for r in rows:
        tsv_lines.append(
            f"{r['command']}\t{r['reach_status']}\t{r['coverage']}\t{r['scenarios']}"
        )

    out_path.write_text("\n".join(tsv_lines) + "\n", encoding="utf-8")
    try:
        display = out_path.relative_to(ROOT)
    except ValueError:
        display = out_path
    print(f"\nWrote {len(rows)} rows to {display}")

    # --- Summary statistics ---
    total = len(rows)
    probed = [r for r in rows if r["coverage"] == "probed"]
    never_probed = [r for r in rows if r["coverage"] == "never-probed"]

    # Count by reach_status bucket
    status_buckets: dict[str, dict[str, int]] = {}
    for r in rows:
        s = r["reach_status"]
        if s not in status_buckets:
            status_buckets[s] = {"probed": 0, "never-probed": 0}
        status_buckets[s][r["coverage"]] += 1

    print()
    print("=" * 72)
    print("SCENARIO COVERAGE REPORT —", datetime.date.today().isoformat())
    print("=" * 72)
    print()
    print(f"  Total C commands (reachability TSV): {total}")
    print(f"  Total probed:   {len(probed):>4}")
    print(f"  Total never-probed: {len(never_probed):>4}")
    print(f"  Unresolved probe words: {len(unresolved)}")
    print(f"  Scenario files:  {len(scenario_files)}")
    print()
    print(f"  {'Status bucket':<25} {'Probed':>7} {'Never':>7} {'Total':>7}")
    print(f"  {'-'*25} {'-'*7} {'-'*7} {'-'*7}")

    # Sort buckets: registered, social, specproc, implemented-unwired, missing, abbrev-stub, then any others
    bucket_order = [
        "registered", "social", "specproc", "implemented-unwired",
        "missing", "missing-social", "abbrev-stub",
    ]
    for bucket in bucket_order:
        if bucket in status_buckets:
            b = status_buckets[bucket]
            t = b["probed"] + b["never-probed"]
            print(f"  {bucket:<25} {b['probed']:>7} {b['never-probed']:>7} {t:>7}")

    # Any remaining buckets not in the predefined order
    for bucket in sorted(status_buckets):
        if bucket not in bucket_order:
            b = status_buckets[bucket]
            t = b["probed"] + b["never-probed"]
            print(f"  {bucket:<25} {b['probed']:>7} {b['never-probed']:>7} {t:>7}")

    # --- Unresolved probe words ---
    if unresolved:
        print()
        print(f"  --- UNRESOLVED PROBE WORDS ({len(unresolved)}) ---")
        print("  These typed words had no matching C table row at level 1.")
        print("  They are either deliberate negative probes or typos.")
        for word in sorted(unresolved):
            sc_list = ",".join(sorted(unresolved[word]))
            print(f"    {word:<20}  ({sc_list})")
    else:
        print()
        print("  No unresolved probe words.")

    # --- Scenarios with zero probe lines ---
    probed_scenarios: set[str] = set()
    for word_scenarios in probe_word_scenarios.values():
        probed_scenarios.update(word_scenarios)

    zero_probe = [s for s in scenario_files if s not in probed_scenarios]
    if zero_probe:
        print()
        print(f"  --- SCENARIOS WITH ZERO PROBE LINES ({len(zero_probe)}) ---")
        print("  These files have no [probe] section (creation-only, etc.):")
        for s in sorted(zero_probe):
            print(f"    - {s}")

    # --- Top-level bucket table for patching into the report header ---
    print()
    print(f"  {'='*50}")
    print(f"  Coverage rate: {len(probed)}/{total} = {len(probed)/total*100:.1f}%")
    print(f"  {'='*50}")

    # --- Self-verification ---
    self_verify(resolved, unresolved, scenario_files, rows)

    return 0


if __name__ == "__main__":
    sys.exit(main())
