#!/usr/bin/env python3
"""Pin the SHAPE of ledger-expected divergences.

Runs each baselined scenario once at seed 1 against the oracle and records
the per-block divergence fingerprints (label + sha256 of the unified diff),
joined back to the ledger citations. The census self-verifies these pins:
a divergence whose shape differs from its pin is a FAIL, never EXPECTED.
Requires DP_ORACLE_BIN; local-only target (CI has no C oracle).
"""
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
BASE = ROOT / "cmd" / "dp-oracle-diff" / "expected_divergences.tsv"
OUT = ROOT / "cmd" / "dp-oracle-diff" / "expected_divergence_pins.tsv"
FP = re.compile(r"^divergence-fingerprint\t(.+)\t([0-9a-f]{64})$")

def main() -> int:
    oracle = subprocess.run(["printenv", "DP_ORACLE_BIN"], capture_output=True, text=True).stdout.strip()
    if not oracle:
        print("pin_expected_divergences: DP_ORACLE_BIN is required", file=sys.stderr)
        return 2
    import tempfile
    harness = pathlib.Path(tempfile.mkdtemp(prefix="dp-pins-")) / "dp-oracle-diff"
    build = subprocess.run(["/usr/local/go/bin/go", "build", "-C", str(ROOT), "-o", str(harness), "./cmd/dp-oracle-diff"])
    if build.returncode != 0:
        print("pin_expected_divergences: harness build failed", file=sys.stderr)
        return 2
    rows = []
    citations = {}
    unstable = set()
    with BASE.open(encoding="utf-8") as stream:
        next(stream)
        for line in stream:
            parts = line.rstrip("\n").split("\t")
            scenario, manifest, case_id, status = parts[:4]
            stability = parts[4] if len(parts) > 4 else ""
            citations.setdefault(scenario, []).append(f"{manifest}:{case_id}:{status}")
            if stability == "run-varying":
                unstable.add(scenario)
    # Run-varying divergences (declared unstable in the manifest) must not be
    # pinned: their bytes differ every run, so any pin is dead on arrival and
    # the census classifies them as EXPECTED_UNSTABLE without a pin.
    pinnable = {s: c for s, c in citations.items() if s not in unstable}
    for scenario in sorted(pinnable):
        proc = subprocess.run(
            [str(harness), "--scenario", scenario, "--seed", "1"],
            capture_output=True, timeout=600,
        )
        stdout = proc.stdout.decode("utf-8", "replace")
        saw = 0
        for line in stdout.splitlines():
            m = FP.match(line)
            if m:
                rows.append((scenario, m.group(1), m.group(2), ";".join(pinnable[scenario])))
                saw += 1
        if saw == 0:
            # A baselined scenario producing no fingerprints would otherwise
            # silently lose its pins (this is exactly how force-mob's pin went
            # missing): fail loudly so the operator retries instead of the next
            # census reporting a misleading "shape differs from pinned baseline".
            print(f"expected_divergence_pins: {scenario} produced no divergence fingerprints; "
                  f"not writing pins", file=sys.stderr)
            return 1
    with OUT.open("w", encoding="utf-8") as stream:
        stream.write("scenario\tlabel\tsha256\tcitations\n")
        for row in sorted(rows):
            stream.write("\t".join(row) + "\n")
    print(f"expected_divergence_pins: {len(rows)} pinned blocks across {len(pinnable)} scenarios "
          f"({len(unstable)} unstable skipped)")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
