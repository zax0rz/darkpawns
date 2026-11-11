#!/usr/bin/env python3
"""Generate the expected-divergence baseline from the depth manifests.

Every entry MUST cite a blocked or excluded manifest row whose proof names
the scenario. Divergence with no ledger row backing it is a FAIL, never a
baseline entry; a baseline entry that stops diverging is STALE. The baseline
is NEVER minted from observed behavior — that would certify today's bugs.
"""
import csv
import pathlib
import re
import sys
from manifest_claims import proof_scenario

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
            parts = line.rstrip("\n").split("\t")
            scenario, manifest, case_id, status = parts[:4]
            ledger.setdefault(scenario, set()).add(f"{manifest}:{case_id}:{status}")
    bad = 0
    for scenario, _label, _sha, citations in pin_lines:
        for citation in citations.split(";"):
            if citation not in ledger.get(scenario, set()):
                print(f"pin cites unknown ledger row: {scenario}: {citation}", file=sys.stderr)
                bad += 1
    return bad


def proof_scenarios(proof: str, where: str) -> list[str]:
    scenarios = []
    for part in proof.split(";"):
        try:
            scenarios.append(proof_scenario(part.strip()))
        except ValueError as exc:
            raise ValueError(f"{where}: {exc}") from exc
    return scenarios


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
                # Only Zach-approved exclusions become baseline rows, plus a
                # blocked row whose notes cite a filed divergence issue
                # ("divergence: DP-nnnn"): that citation says the scenario is
                # red right now with a known, filed shape. A blocked row
                # WITHOUT that citation is analytical debt -- gates a probe has
                # not exercised yet -- and its scenario is usually green;
                # baselining it would flip the census row to STALE (taught by
                # #2178: three blocked rows swept in by a regeneration, all
                # three scenarios green on the branch).
                notes = (row.get("notes") or "").strip()
                if status == "blocked" and not re.search(r"divergence: DP-\d+", notes):
                    continue
                if status not in ("excluded", "blocked") or not proof or proof == "-":
                    continue
                # stability is the optional divergence-shape class: "" (default)
                # pins a stable shape; "run-varying" marks a C-side divergence
                # whose bytes differ every run, which the census accepts as
                # EXPECTED_UNSTABLE instead of demanding an impossible pin.
                stability = (row.get("stability") or "").strip()
                if stability not in ("", "run-varying"):
                    print(f"unknown stability {stability!r} in {path.name}:{row['case_id']}", file=sys.stderr)
                    return 1
                # A case may cite several proofs separated by ";"; every
                # scenario among them carries the expected divergence.
                try:
                    scenarios = proof_scenarios(proof, f"{path.name}:{row['case_id']}")
                except ValueError as exc:
                    print(f"gen_expected_divergences: {exc}", file=sys.stderr)
                    return 1
                for scenario in scenarios:
                    if not (SCENARIO_DIR / f"{scenario}.txt").exists():
                        continue
                    rows.append((scenario, path.name, row["case_id"], status, stability))
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
                try:
                    scenarios = proof_scenarios(proof, f"{path.name}:{row['case_id']}")
                except ValueError as exc:
                    print(f"gen_expected_divergences: {exc}", file=sys.stderr)
                    return 1
                if not any((SCENARIO_DIR / f"{scenario}.txt").exists() for scenario in scenarios):
                    unresolved.append(f"{path.name}:{row['case_id']}:{scenarios[0]}")
    print(f"expected_divergences: {len(rows)} rows across {len(kept_scenarios)} scenarios "
          f"({no_proof} blocked/excluded rows carry no scenario proof; "
          f"{len(unresolved)} proofs did not resolve to a scenario file)")
    for item in unresolved:
        print(f"  unresolved: {item}", file=sys.stderr)
    with OUT.open("w", encoding="utf-8", newline="") as stream:
        stream.write("scenario\tmanifest\tcase_id\tstatus\tstability\n")
        for row in rows:
            stream.write("\t".join(row) + "\n")
    return 0

def self_test() -> int:
    """A blocked row without a 'divergence: DP-' citation must stay OUT of the
    baseline (R5h): #2178 taught that bare blocked rows sweep green scenarios
    in and flip them STALE. Only a filed-issue citation admits them."""
    import tempfile

    global MANIFEST_DIR, SCENARIO_DIR, OUT
    with tempfile.TemporaryDirectory() as tmp:
        root = pathlib.Path(tmp)
        depth = root / "depth"
        depth.mkdir()
        (depth / "t.tsv").write_text(
            "handler\tcommand\tcase_id\tdepth\tscope\tstatus\tproof\tc_site\tnotes\n"
            "do_x\tx\tx.red-with-issue\tD1\tcommand\tblocked\tx-scenario@1\tsrc/x.c:1\t"
            "DP-1456 repair exposed a real divergence (divergence: DP-1460)\n"
            "do_x\tx\tx.blocked-no-issue\tD1\tcommand\tblocked\tx-scenario@1\tsrc/x.c:1\t"
            "DP-1456: demoted until re-proven\n"
            "do_x\tx\tx.excluded\tD1\tcommand\texcluded\tx-scenario@1\tsrc/x.c:1\t-\n",
            encoding="utf-8",
        )
        scenarios = root / "scenarios"
        scenarios.mkdir()
        (scenarios / "x-scenario.txt").write_text("[probe]\nlook\n", encoding="utf-8")
        out = root / "expected.tsv"
        MANIFEST_DIR, SCENARIO_DIR, OUT = depth, scenarios, out
        if main() != 0:
            return 1
        lines = out.read_text(encoding="utf-8").splitlines()
        cases = {line.split("\t")[2] for line in lines[1:]}
        if "x.red-with-issue" not in cases:
            print("self-test: blocked row with 'divergence: DP-' citation was not baselined", file=sys.stderr)
            return 1
        if "x.blocked-no-issue" in cases:
            print("self-test: blocked row without a divergence issue citation was baselined", file=sys.stderr)
            return 1
        if "x.excluded" not in cases:
            print("self-test: excluded row missing from baseline", file=sys.stderr)
            return 1
        print("expected_divergences self-test: unbacked blocked rows stay out; issue-backed and excluded rows are baselined")
        return 0


if __name__ == "__main__":
    if "--self-test" in sys.argv:
        raise SystemExit(self_test())
    raise SystemExit(main())
