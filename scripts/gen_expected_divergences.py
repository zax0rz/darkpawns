#!/usr/bin/env python3
"""Generate the expected-divergence baseline from the depth manifests.

Every entry MUST cite a blocked or excluded manifest row whose proof names
the scenario. Divergence with no ledger row backing it is a FAIL, never a
baseline entry; a baseline entry that stops diverging is STALE. The baseline
is NEVER minted from observed behavior — that would certify today's bugs.
"""
import csv
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
MANIFEST_DIR = ROOT / "docs" / "fidelity" / "depth"
SCENARIO_DIR = ROOT / "cmd" / "dp-oracle-diff" / "scenarios"
OUT = ROOT / "cmd" / "dp-oracle-diff" / "expected_divergences.tsv"

def check_pins(pins_path: pathlib.Path) -> int:
    """Oracle-free integrity check: every pins row must cite a ledger row that
    exists in expected_divergences.tsv (same scenario, citation present)."""
    with pins_path.open(encoding="utf-8") as stream:
        pin_lines = [line.rstrip("\n").split("\t") for line in stream][1:]
    ledger = {}
    with OUT.open(encoding="utf-8") as stream:
        for line in stream:
            scenario, manifest, case_id, status = line.rstrip("\n").split("\t")
            ledger.setdefault(scenario, set()).add(f"{manifest}:{case_id}:{status}")
    bad = 0
    for scenario, _label, _sha, citations in pin_lines:
        for citation in citations.split(";"):
            if citation not in ledger.get(scenario, set()):
                print(f"pin cites unknown ledger row: {scenario}: {citation}", file=sys.stderr)
                bad += 1
    return bad


def main() -> int:
    if "--check-pins" in sys.argv:
        pins = ROOT / "cmd" / "dp-oracle-diff" / "expected_divergence_pins.tsv"
        bad = check_pins(pins)
        print(f"expected_divergence_pins: {'OK' if bad == 0 else f'{bad} dangling citations'}")
        return 1 if bad else 0
    rows = []
    for path in sorted(MANIFEST_DIR.glob("*.tsv")):
        with path.open(encoding="utf-8", newline="") as stream:
            reader = csv.DictReader(stream, delimiter="\t")
            for row in reader:
                status = (row.get("status") or "").strip()
                proof = (row.get("proof") or "").strip()
                if status not in ("blocked", "excluded") or not proof or proof == "-":
                    continue
                scenario = proof.split("@", 1)[0]
                if not (SCENARIO_DIR / f"{scenario}.txt").exists():
                    continue
                rows.append((scenario, path.name, row["case_id"], status))
    rows.sort()
    kept_scenarios = sorted({r[0] for r in rows})
    no_proof = 0
    unresolved = []
    for path in sorted(MANIFEST_DIR.glob("*.tsv")):
        with path.open(encoding="utf-8", newline="") as stream:
            for row in csv.DictReader(stream, delimiter="\t"):
                status = (row.get("status") or "").strip()
                proof = (row.get("proof") or "").strip()
                if status not in ("blocked", "excluded"):
                    continue
                if not proof or proof == "-":
                    no_proof += 1
                    continue
                scenario = proof.split("@", 1)[0]
                if not (SCENARIO_DIR / f"{scenario}.txt").exists():
                    unresolved.append(f"{path.name}:{row['case_id']}:{scenario}")
    print(f"expected_divergences: {len(rows)} rows across {len(kept_scenarios)} scenarios "
          f"({no_proof} blocked/excluded rows carry no scenario proof; "
          f"{len(unresolved)} proofs did not resolve to a scenario file)")
    for item in unresolved:
        print(f"  unresolved: {item}", file=sys.stderr)
    with OUT.open("w", encoding="utf-8", newline="") as stream:
        stream.write("scenario\tmanifest\tcase_id\tstatus\n")
        for row in rows:
            stream.write("\t".join(row) + "\n")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
