#!/usr/bin/env python3
"""Generate deterministic handler-depth coverage from fidelity case manifests."""

from __future__ import annotations

import argparse
import csv
import pathlib
import re
import sys
from collections import Counter, defaultdict


ROOT = pathlib.Path(__file__).resolve().parents[1]
MANIFEST_DIR = ROOT / "docs" / "fidelity" / "depth"
SCENARIO_DIR = ROOT / "cmd" / "dp-oracle-diff" / "scenarios"
CASE_RE = re.compile(r"^\s*#\s*depth-case:\s*(\S+)\s*$", re.MULTILINE)
FIELDS = ("handler", "command", "case_id", "depth", "scope", "status", "proof", "c_site", "notes")
VALID_STATUSES = {
    "oracle-green",
    "oracle-green-multiseed",
    "unit-green",
    "blocked",
    "excluded",
    "delegated",
}
PROVEN = {"oracle-green", "oracle-green-multiseed", "unit-green", "delegated"}


def load_rows() -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for path in sorted(MANIFEST_DIR.glob("*.tsv")):
        with path.open(encoding="utf-8", newline="") as stream:
            reader = csv.DictReader(stream, delimiter="\t")
            if tuple(reader.fieldnames or ()) != FIELDS:
                raise ValueError(f"{path}: fields {reader.fieldnames!r}, want {FIELDS!r}")
            for line_no, row in enumerate(reader, 2):
                row = {key: (value or "").strip() for key, value in row.items()}
                if not row["case_id"]:
                    raise ValueError(f"{path}:{line_no}: empty case_id")
                if row["status"] not in VALID_STATUSES:
                    raise ValueError(f"{path}:{line_no}: invalid status {row['status']!r}")
                row["manifest"] = str(path.relative_to(ROOT))
                rows.append(row)
    return rows


def scenario_cases() -> dict[str, set[str]]:
    result: dict[str, set[str]] = {}
    for path in sorted(SCENARIO_DIR.glob("*.txt")):
        result[path.stem] = set(CASE_RE.findall(path.read_text(encoding="utf-8")))
    return result


def validate(rows: list[dict[str, str]], annotations: dict[str, set[str]]) -> list[str]:
    errors: list[str] = []
    seen: set[str] = set()
    for row in rows:
        case_id = row["case_id"]
        if case_id in seen:
            errors.append(f"duplicate case_id: {case_id}")
        seen.add(case_id)
        if row["status"].startswith("oracle-green"):
            scenario = row["proof"].split("@", 1)[0]
            if scenario not in annotations:
                errors.append(f"{case_id}: missing scenario {scenario}")
            elif case_id not in annotations[scenario]:
                errors.append(f"{case_id}: scenario {scenario} lacks depth-case annotation")
        if row["status"] == "unit-green":
            symbol = row["proof"]
            if not symbol or not any(symbol in path.read_text(encoding="utf-8", errors="ignore") for path in ROOT.rglob("*_test.go")):
                errors.append(f"{case_id}: unit proof symbol {symbol!r} not found")
    declared = {row["case_id"] for row in rows}
    for scenario, cases in annotations.items():
        for case_id in cases - declared:
            errors.append(f"{scenario}: annotated case {case_id} is absent from manifests")
    return errors


def render(rows: list[dict[str, str]]) -> str:
    status_counts = Counter(row["status"] for row in rows)
    by_handler: dict[str, list[dict[str, str]]] = defaultdict(list)
    for row in rows:
        by_handler[row["handler"]].append(row)
    proven = sum(row["status"] in PROVEN for row in rows)
    actionable = sum(row["status"] not in {"excluded"} for row in rows)
    output = [
        "FIDELITY DEPTH REPORT",
        "=====================",
        f"Cases: {len(rows)} total, {proven} proven/delegated, {status_counts['blocked']} blocked, {status_counts['excluded']} excluded",
        f"Actionable completion: {proven}/{actionable} = {(100 * proven / actionable if actionable else 100):.1f}%",
        "",
    ]
    for handler in sorted(by_handler):
        cases = by_handler[handler]
        handler_proven = sum(row["status"] in PROVEN for row in cases)
        output.append(f"{handler}: {handler_proven}/{len(cases)}")
        for row in cases:
            output.append(f"  {row['depth']}  {row['status']:<24} {row['case_id']}  [{row['proof']}]")
        output.append("")
    return "\n".join(output)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=pathlib.Path, help="optional report output path")
    args = parser.parse_args()
    try:
        rows = load_rows()
        annotations = scenario_cases()
        errors = validate(rows, annotations)
    except (OSError, ValueError) as exc:
        print(f"fidelity-depth: {exc}", file=sys.stderr)
        return 1
    if errors:
        for error in errors:
            print(f"fidelity-depth: {error}", file=sys.stderr)
        return 1
    report = render(rows)
    print(report)
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(report + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
