#!/usr/bin/env python3
"""Weekly port-reachability snapshot: run the generator, diff against the
previous snapshot, append to the historical JSONL log, print a one-line summary.

Produces (per docs/fidelity methodology; see docs/port-reachability-map.md):
  1. docs/reports/reachability-<today>.tsv           — current state
  2. delta vs the previous snapshot                  — printed, and stored in the log
  3. one-line summary on stdout                      — suitable for posting to a channel
  4. docs/research/metrics/reachability-history.jsonl — append-only time series

Deterministic given the same source tree + previous snapshots. Re-running on
the same day replaces that day's log entry instead of duplicating it.

Usage:
    python3 scripts/reachability_weekly.py [--commit]

--commit stages and commits the new TSV + JSONL (data files only).
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
HISTORY = ROOT / "docs" / "research" / "metrics" / "reachability-history.jsonl"

# Statuses a player can actually use vs. not. Movement between these two sets
# is what improved/regressed means.
REACHABLE = {"registered", "social", "specproc"}
UNREACHABLE = {"implemented-unwired", "missing", "missing-social"}


def run_generator(out_path: Path) -> None:
    subprocess.run(
        [sys.executable, str(ROOT / "scripts" / "gen_reachability.py"),
         "--out", str(out_path)],
        check=True,
        cwd=ROOT,
        stdout=subprocess.DEVNULL,
    )


def load_tsv(path: Path) -> dict[str, str]:
    """command -> status"""
    result = {}
    lines = path.read_text().splitlines()
    for line in lines[1:]:  # skip header
        fields = line.split("\t")
        if len(fields) >= 6:
            result[fields[0]] = fields[5]
    return result


def find_previous(today_path: Path) -> Path | None:
    """Most recent dated snapshot other than today's."""
    pattern = re.compile(r"reachability-(\d{4}-\d{2}-\d{2})\.tsv$")
    candidates = sorted(
        p for p in REPORTS.glob("reachability-*.tsv")
        if pattern.search(p.name) and p != today_path
    )
    return candidates[-1] if candidates else None


def compute_delta(prev: dict[str, str], curr: dict[str, str]) -> dict:
    improved = []   # unreachable -> reachable
    regressed = []  # reachable -> unreachable  (the signal that pays the rent)
    changed = []    # any other status movement
    for cmd in sorted(set(prev) & set(curr)):
        a, b = prev[cmd], curr[cmd]
        if a == b:
            continue
        entry = {"command": cmd, "from": a, "to": b}
        if a in UNREACHABLE and b in REACHABLE:
            improved.append(entry)
        elif a in REACHABLE and b in UNREACHABLE:
            regressed.append(entry)
        else:
            changed.append(entry)
    return {
        "improved": improved,
        "regressed": regressed,
        "changed": changed,
        "added": sorted(set(curr) - set(prev)),
        "removed": sorted(set(prev) - set(curr)),
    }


def status_counts(snapshot: dict[str, str]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for status in snapshot.values():
        counts[status] = counts.get(status, 0) + 1
    return dict(sorted(counts.items()))


def append_history(entry: dict) -> None:
    """Append, replacing any existing entry for the same date."""
    HISTORY.parent.mkdir(parents=True, exist_ok=True)
    lines = []
    if HISTORY.exists():
        lines = [
            l for l in HISTORY.read_text().splitlines()
            if l.strip() and json.loads(l).get("date") != entry["date"]
        ]
    lines.append(json.dumps(entry, sort_keys=True))
    HISTORY.write_text("\n".join(lines) + "\n")


def summary_line(date: str, counts: dict[str, int], delta: dict | None) -> str:
    reachable = sum(v for k, v in counts.items() if k in REACHABLE)
    unreachable = sum(v for k, v in counts.items() if k in UNREACHABLE)
    line = (
        f"reachability {date}: {reachable} reachable / {unreachable} unreachable "
        f"({counts.get('registered', 0)} reg, "
        f"{counts.get('implemented-unwired', 0)} unwired, "
        f"{counts.get('missing', 0)} missing)"
    )
    if delta is None:
        return line + " | first snapshot, no delta"
    parts = []
    if delta["improved"]:
        parts.append(f"+{len(delta['improved'])} improved "
                     f"({', '.join(e['command'] for e in delta['improved'][:5])}"
                     f"{'…' if len(delta['improved']) > 5 else ''})")
    if delta["regressed"]:
        parts.append(f"⚠ {len(delta['regressed'])} REGRESSED "
                     f"({', '.join(e['command'] for e in delta['regressed'])})")
    if delta["added"] or delta["removed"]:
        parts.append(f"{len(delta['added'])} added / {len(delta['removed'])} removed")
    return line + " | Δ " + ("; ".join(parts) if parts else "no changes")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--commit", action="store_true",
                        help="git-commit the new TSV + history log")
    args = parser.parse_args()

    today = datetime.date.today().isoformat()
    tsv_path = REPORTS / f"reachability-{today}.tsv"

    run_generator(tsv_path)
    curr = load_tsv(tsv_path)
    prev_path = find_previous(tsv_path)
    delta = compute_delta(load_tsv(prev_path), curr) if prev_path else None
    counts = status_counts(curr)

    append_history({
        "date": today,
        "counts": counts,
        "total": len(curr),
        "previous": prev_path.name if prev_path else None,
        "delta": delta,
        "source": "generator",
    })

    line = summary_line(today, counts, delta)
    print(line)
    if delta:
        for e in delta["regressed"]:
            print(f"  REGRESSION: {e['command']} {e['from']} -> {e['to']}")

    if args.commit:
        subprocess.run(
            ["git", "add", str(tsv_path), str(HISTORY)], check=True, cwd=ROOT)
        # Nothing staged (same-day rerun, no changes) → skip the commit.
        staged = subprocess.run(
            ["git", "diff", "--cached", "--quiet"], cwd=ROOT)
        if staged.returncode != 0:
            subprocess.run(
                ["git", "commit", "-m", f"data: weekly reachability snapshot {today}\n\n{line}"],
                check=True, cwd=ROOT)

    # Non-zero exit on regression so CI/cron wrappers can alert on it.
    return 1 if delta and delta["regressed"] else 0


if __name__ == "__main__":
    sys.exit(main())
