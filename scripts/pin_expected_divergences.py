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
    with BASE.open(encoding="utf-8") as stream:
        next(stream)
        for line in stream:
            scenario, manifest, case_id, status = line.rstrip("\n").split("\t")
            citations.setdefault(scenario, []).append(f"{manifest}:{case_id}:{status}")
    for scenario in sorted(citations):
        proc = subprocess.run(
            [str(harness), "--scenario", scenario, "--seed", "1"],
            capture_output=True, timeout=600,
        )
        # Some baselined scenarios diverge BECAUSE the C oracle emits non-text
        # bytes (the accuse pointer anomaly); decode with replacement.
        stdout = proc.stdout.decode("utf-8", "replace")
        for line in stdout.splitlines():
            m = FP.match(line)
            if m:
                rows.append((scenario, m.group(1), m.group(2), ";".join(citations[scenario])))
    with OUT.open("w", encoding="utf-8") as stream:
        stream.write("scenario\tlabel\tsha256\tcitations\n")
        for row in sorted(rows):
            stream.write("\t".join(row) + "\n")
    print(f"expected_divergence_pins: {len(rows)} pinned blocks across {len(citations)} scenarios")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
