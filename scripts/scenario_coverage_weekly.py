#!/usr/bin/env python3
"""Weekly scenario-coverage snapshot: run the analyzer, diff against the
previous snapshot, append to the historical JSONL log, print a one-line summary.

Sibling of reachability_weekly.py — same artifacts, different instrument:
  1. docs/reports/scenario-coverage-<today>.tsv        — current state
  2. delta vs the previous snapshot                    — printed + stored
  3. one-line summary on stdout                        — embed-ready
  4. docs/research/metrics/scenario-coverage-history.jsonl — time series

IMPORTANT: "probed" is a STATIC claim — some oracle scenario exercises the
command. It says nothing about whether the suite currently passes. Pass/fail
belongs to actual oracle runs, not this analyzer.

Run AFTER reachability_weekly.py: the analyzer cross-references the newest
reachability TSV for per-command status.

Exit codes: 0 clean; 1 = coverage regression (a command went probed→never,
i.e. a scenario was deleted/renamed or resolution changed — suite hygiene,
not a live break); >1 = broken.

Usage:
    python3 scripts/scenario_coverage_weekly.py [--commit]
"""

import argparse
import datetime
import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
REPORTS = ROOT / "docs" / "reports"
HISTORY = ROOT / "docs" / "research" / "metrics" / "scenario-coverage-history.jsonl"


def run_analyzer(out_path: Path) -> None:
    subprocess.run(
        [sys.executable, str(ROOT / "scripts" / "gen_scenario_coverage.py"),
         "--out", str(out_path)],
        check=True,
        cwd=ROOT,
        stdout=subprocess.DEVNULL,
    )


def load_tsv(path: Path) -> dict[str, str]:
    """command -> coverage (probed | never-probed)"""
    result = {}
    for line in path.read_text().splitlines():
        if line.startswith("#") or line.startswith("command\t") or not line.strip():
            continue
        fields = line.split("\t")
        if len(fields) >= 3:
            result[fields[0]] = fields[2]
    return result


def find_previous(today_path: Path) -> Path | None:
    pattern = re.compile(r"scenario-coverage-(\d{4}-\d{2}-\d{2})\.tsv$")
    candidates = sorted(
        p for p in REPORTS.glob("scenario-coverage-*.tsv")
        if pattern.search(p.name) and p != today_path
    )
    return candidates[-1] if candidates else None


def compute_delta(prev: dict[str, str], curr: dict[str, str]) -> dict:
    newly_probed = sorted(
        c for c in set(prev) & set(curr)
        if prev[c] == "never-probed" and curr[c] == "probed"
    )
    unprobed = sorted(  # the hygiene regression: lost coverage
        c for c in set(prev) & set(curr)
        if prev[c] == "probed" and curr[c] == "never-probed"
    )
    return {
        "newly_probed": newly_probed,
        "unprobed": unprobed,
        "added": sorted(set(curr) - set(prev)),
        "removed": sorted(set(prev) - set(curr)),
    }


def append_history(entry: dict) -> None:
    HISTORY.parent.mkdir(parents=True, exist_ok=True)
    lines = []
    if HISTORY.exists():
        lines = [
            l for l in HISTORY.read_text().splitlines()
            if l.strip() and json.loads(l).get("date") != entry["date"]
        ]
    lines.append(json.dumps(entry, sort_keys=True))
    HISTORY.write_text("\n".join(lines) + "\n")


def summary_line(date: str, curr: dict[str, str], delta: dict | None) -> str:
    probed = sum(1 for v in curr.values() if v == "probed")
    never = sum(1 for v in curr.values() if v == "never-probed")
    line = f"coverage {date}: {probed} probed / {never} never-probed"
    if delta is None:
        return line + " | first snapshot, no delta"
    parts = []
    if delta["newly_probed"]:
        parts.append(f"+{len(delta['newly_probed'])} newly probed "
                     f"({', '.join(delta['newly_probed'][:5])}"
                     f"{'…' if len(delta['newly_probed']) > 5 else ''})")
    if delta["unprobed"]:
        parts.append(f"⚠ {len(delta['unprobed'])} LOST COVERAGE "
                     f"({', '.join(delta['unprobed'])})")
    if delta["added"] or delta["removed"]:
        parts.append(f"{len(delta['added'])} added / {len(delta['removed'])} removed")
    return line + " | Δ " + ("; ".join(parts) if parts else "no changes")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--commit", action="store_true",
                        help="git-commit the new TSV + history log")
    args = parser.parse_args()

    today = datetime.date.today().isoformat()
    tsv_path = REPORTS / f"scenario-coverage-{today}.tsv"

    run_analyzer(tsv_path)
    curr = load_tsv(tsv_path)
    prev_path = find_previous(tsv_path)
    delta = compute_delta(load_tsv(prev_path), curr) if prev_path else None

    append_history({
        "date": today,
        "probed": sum(1 for v in curr.values() if v == "probed"),
        "never_probed": sum(1 for v in curr.values() if v == "never-probed"),
        "total": len(curr),
        "previous": prev_path.name if prev_path else None,
        "delta": delta,
        "source": "analyzer",
    })

    line = summary_line(today, curr, delta)
    print(line)
    if delta:
        for c in delta["unprobed"]:
            print(f"  LOST COVERAGE: {c}")

    if args.commit:
        subprocess.run(["git", "add", str(tsv_path), str(HISTORY)],
                       check=True, cwd=ROOT)
        staged = subprocess.run(["git", "diff", "--cached", "--quiet"], cwd=ROOT)
        if staged.returncode != 0:
            subprocess.run(
                ["git", "commit", "-m",
                 f"data: weekly scenario-coverage snapshot {today}\n\n{line}"],
                check=True, cwd=ROOT)

    return 1 if delta and delta["unprobed"] else 0


if __name__ == "__main__":
    sys.exit(main())
