#!/usr/bin/env python3
"""CI reachability ratchet — fail when unreachable commands increase.

Runs the deterministic generator (``scripts/gen_reachability.py``) against the
current tree, compares the count of *unreachable* commands to the newest
committed snapshot in ``docs/reports/``, and exits non-zero if the count went
UP. The floor only ever moves in one direction; same spirit as the Go coverage
ratchet in ``.github/workflows/ci.yml``.

Cite: rules **R2** / **R5c** (``docs/fidelity/RULEBOOK.md``) — R5c says make
the audit deterministic and rerunnable; this gate turns the deterministic
report into a blocking CI signal.

Which statuses count as "unreachable" here:

    {"implemented-unwired", "missing"}

NOTE: this is deliberately NARROWER than ``reachability_weekly.py``'s
``UNREACHABLE`` set, which also includes ``missing-social``. The weekly report
treats unported socials as regressions; the CI gate does not, because socials
are bulk-imported and their count moves for data reasons unrelated to whether
the code surface is wired. The committed floor (61 = 25 unwired + 36 missing,
as of 2026-07-22) reflects this narrower set. Do not silently widen the set —
the floor number and the set must stay in lockstep.

The status constants are duplicated here (rather than imported from
``reachability_weekly.py``) because that module imports ``subprocess`` at
module top for its ``--commit`` path, and this gate must stay stdlib-only with
no git/subprocess side effects beyond running the generator.

Exit codes:
    0  — unreachable count is equal to or below the baseline (healthy)
    1  — unreachable count increased (regression); offending names printed
    2  — no committed baseline found, or generator/parse failure (infra)

Usage:
    python3 scripts/reachability_ci_gate.py                 # CI invocation
    python3 scripts/reachability_ci_gate.py --baseline /tmp/doctored.tsv
        # override the auto-discovered baseline (local both-directions test:
        # doctor a COPY to claim fewer unreachable → expect exit 1)
"""

import argparse
import re
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
REPORTS = ROOT / "docs" / "reports"
GENERATOR = ROOT / "scripts" / "gen_reachability.py"

# Statuses this gate ratchets on. See module docstring for why this set is
# narrower than reachability_weekly.py's and why it is duplicated, not shared.
UNREACHABLE = {"implemented-unwired", "missing"}

_STATUS_COL = 5  # TSV is 1-indexed in the header; "status" is column 6 → fields[5]
_DATE_RE = re.compile(r"reachability-(\d{4}-\d{2}-\d{2})\.tsv$")


def find_newest_baseline(reports_dir: Path) -> Path | None:
    """Newest ``reachability-<date>.tsv`` in *reports_dir*, by filename date.

    Date is parsed from the filename, not mtime, so a fresh checkout (identical
    mtimes) and a re-cloned repo both resolve to the same file.
    """
    candidates = [
        (m.group(1), p)
        for p in reports_dir.glob("reachability-*.tsv")
        if (m := _DATE_RE.search(p.name))
    ]
    if not candidates:
        return None
    candidates.sort(key=lambda kv: kv[0])
    return candidates[-1][1]


def load_unreachable(path: Path) -> set[str]:
    """Return the set of command names whose status is in UNREACHABLE."""
    names: set[str] = set()
    lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    for line in lines[1:]:  # skip header
        fields = line.split("\t")
        if len(fields) <= _STATUS_COL:
            continue
        if fields[_STATUS_COL] in UNREACHABLE:
            names.add(fields[0])
    return names


def generate_ci_snapshot(out_path: Path) -> None:
    """Run the deterministic generator into *out_path*. Stdlib-only, ~0.1s."""
    subprocess.run(
        [sys.executable, str(GENERATOR), "--out", str(out_path)],
        check=True,
        cwd=ROOT,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.STDOUT,
    )


def main() -> int:
    parser = argparse.ArgumentParser(
        description="CI reachability ratchet — fail when unreachable commands increase.",
    )
    parser.add_argument(
        "--baseline",
        type=Path,
        default=None,
        help="override the auto-discovered baseline TSV (local testing only)",
    )
    parser.add_argument(
        "--reports-dir",
        type=Path,
        default=REPORTS,
        help="dir to search for committed baselines (default: docs/reports)",
    )
    args = parser.parse_args()

    baseline = args.baseline or find_newest_baseline(args.reports_dir)
    if baseline is None:
        print(
            "::error::no committed reachability-<date>.tsv baseline found in "
            f"{args.reports_dir} — cannot establish a ratchet floor. "
            "Run `python3 scripts/gen_reachability.py` and commit the TSV."
        )
        return 2
    if not baseline.is_file():
        print(f"::error::baseline not found: {baseline}")
        return 2

    baseline_unreachable = load_unreachable(baseline)

    with tempfile.TemporaryDirectory(prefix="reachability-ci-") as tmp:
        ci_tsv = Path(tmp) / "reachability-ci.tsv"
        try:
            generate_ci_snapshot(ci_tsv)
        except subprocess.CalledProcessError as exc:
            print(f"::error::reachability generator failed (exit {exc.returncode})")
            return 2
        ci_unreachable = load_unreachable(ci_tsv)

    ci_count = len(ci_unreachable)
    baseline_count = len(baseline_unreachable)
    print(f"reachability ratchet: baseline {baseline.name} = {baseline_count} unreachable")
    print(f"reachability ratchet: current tree = {ci_count} unreachable")

    if ci_count > baseline_count:
        # Newly unreachable = unreachable now but not unreachable in baseline.
        # Always non-empty when the count rose (proven: |A|>|B| ⇒ A∖B ≠ ∅), so
        # the message is never empty on the failure path.
        newly = sorted(ci_unreachable - baseline_unreachable)
        print(
            f"::error::reachability ratchet REGRESSED: {ci_count} > {baseline_count} "
            f"(+{ci_count - baseline_count}). Commands now unreachable:"
        )
        for name in newly:
            print(f"  - {name}")
        print(
            "Restore these, or — if the new floor is intended — regenerate the "
            "baseline with `python3 scripts/reachability_weekly.py --commit`."
        )
        return 1

    print(f"reachability ratchet: OK ({ci_count} <= {baseline_count} floor)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
