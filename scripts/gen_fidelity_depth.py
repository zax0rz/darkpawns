#!/usr/bin/env python3
"""Generate deterministic handler-depth coverage from fidelity case manifests."""

from __future__ import annotations

import argparse
import pathlib
import re
import sys
from collections import Counter, defaultdict


ROOT = pathlib.Path(__file__).resolve().parents[1]
MANIFEST_DIR = ROOT / "docs" / "fidelity" / "depth"
SCENARIO_DIR = ROOT / "cmd" / "dp-oracle-diff" / "scenarios"
CASE_RE = re.compile(r"^\s*#\s*depth-case:\s*(\S+)\s*$", re.MULTILINE)
from fidelity_manifest import load_rows, UNIT_PROOF_STATUSES
from unit_proofs import validate_unit_rows, test_index, proof_symbols, resolve
from manifest_claims import claimed_pairs

PROVEN = {"oracle-green", "oracle-green-multiseed", "unit-green", "delegated"}
NOT_ACTIONABLE = {"excluded", "divergent-approved"}

# A cited proof is any whitespace/;/,-separated token spelled like a Go test
# (Test followed by at least one identifier character). Prose tokens, scenario
# names, and bare seed numbers in oracle-green proof cells are not citations.
TEST_TOKEN = re.compile(r"^Test[A-Za-z0-9_]*$")


def validate_citations(rows, index):
    """Every row, in any status, must cite only Test names a *_test.go declares.

    R5h: a renamed or deleted proof test must turn this gate red instead of
    leaving the row citing something that no longer exists. Unit-status rows
    are checked again — with Run: subclaim execution — by validate_unit_rows,
    which uses the same index.
    """
    errors = []
    for row in rows:
        if row["status"] in UNIT_PROOF_STATUSES:
            continue  # validate_unit_rows re-checks these, with Run: execution
        location = f"{row['manifest']}:{row['line']} ({row['case_id']})"
        for symbol in proof_symbols(row["proof"]):
            if not TEST_TOKEN.match(symbol):
                continue
            _declaration, problem, detail = resolve(symbol, index)
            if problem:
                errors.append(f"{location}: {symbol}: {problem}: {detail}")
    return errors


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
            # Parse the proof exactly as the claims census does, so a seed list
            # it would reject fails this gate before review.
            try:
                claimed_pairs([row])
            except ValueError as error:
                errors.append(f"{case_id}: {error}")
            scenario = row["proof"].split("@", 1)[0]
            if scenario not in annotations:
                errors.append(f"{case_id}: missing scenario {scenario}")
            elif case_id not in annotations[scenario]:
                errors.append(f"{case_id}: scenario {scenario} lacks depth-case annotation")
    declared = {row["case_id"] for row in rows}
    for scenario, cases in annotations.items():
        for case_id in cases - declared:
            errors.append(f"{scenario}: annotated case {case_id} is absent from manifests")
    index = test_index(ROOT)
    errors.extend(validate_citations(rows, index))
    errors.extend(validate_unit_rows(rows, index=index))
    return errors


def render(rows: list[dict[str, str]]) -> str:
    status_counts = Counter(row["status"] for row in rows)
    by_handler: dict[str, list[dict[str, str]]] = defaultdict(list)
    for row in rows:
        by_handler[row["handler"]].append(row)
    proven = sum(row["status"] in PROVEN for row in rows)
    # Completion measures C parity: excluded cases are unreachable, and
    # divergent-approved cases are deliberately not C, so neither counts.
    actionable = sum(row["status"] not in NOT_ACTIONABLE for row in rows)
    output = [
        "FIDELITY DEPTH REPORT",
        "=====================",
        f"Cases: {len(rows)} total, {proven} proven/delegated, {status_counts['blocked']} blocked, {status_counts['excluded']} excluded, {status_counts['divergent-approved']} divergent-approved",
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



def self_test() -> int:
    """R5h control for validate_citations, stubbed and fast.

    The citation check must fail a renamed proof that only an oracle-green
    row cites, and must not misfire on scenario names, bare seeds, prose, or
    unit-status rows (which validate_unit_rows owns).
    """
    index = {"tests": [{"name": "TestReal", "package": "./pkg/game", "valid": True,
                        "vacuous": False, "vacuous_subtests": []}],
             "files": {"pkg/game/real_test.go": "func TestReal(t *testing.T) {"}}
    rows = [
        {"manifest": "self.tsv", "line": 1, "case_id": "renamed-proof",
         "status": "oracle-green", "proof": "show@1,2;TestReal;TestGone"},
        {"manifest": "self.tsv", "line": 2, "case_id": "prose-is-not-a-citation",
         "status": "oracle-green-multiseed",
         "proof": "12 act() sites; prose naming TestReal is not a citation"},
        {"manifest": "self.tsv", "line": 3, "case_id": "unit-row-owned-elsewhere",
         "status": "unit-green", "proof": "TestGone"},
    ]
    errors = validate_citations(rows, index)
    if len(errors) != 1 or "TestGone" not in errors[0] or "(renamed-proof)" not in errors[0]:
        print("fidelity-depth self-test: expected exactly one missing-citation "
              f"error on the renamed proof, got: {errors}", file=sys.stderr)
        return 1
    print("fidelity-depth self-test: citation gate fails a renamed oracle-green "
          "proof; scenario tokens, seeds, prose, and unit rows are exempt")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=pathlib.Path, help="optional report output path")
    parser.add_argument("--self-test", action="store_true",
                        help="run the R5h citation-gate control instead of the report")
    args = parser.parse_args()
    if args.self_test:
        return self_test()
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
