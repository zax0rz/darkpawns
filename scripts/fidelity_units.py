#!/usr/bin/env python3
"""Run manifest-referenced Go unit proofs, retaining classified JSON events."""
from __future__ import annotations

import argparse
from collections import Counter
import csv
import json
from pathlib import Path
import subprocess
import sys
import time

from fidelity_manifest import ROOT, load_rows
from unit_proofs import test_index, proof_symbols, resolve, execution_batches, outcome


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT, help="main checkout whose manifests and tests are measured")
    parser.add_argument("--out", type=Path, required=True, help="retained evidence directory (must be new or empty)")
    args = parser.parse_args()
    root = args.root.resolve()
    out = args.out.resolve()
    if out.exists() and any(out.iterdir()):
        parser.error(f"evidence directory is not empty: {out}")
    out.mkdir(parents=True, exist_ok=True)
    started = time.monotonic()
    head = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], text=True).strip()
    tool_head = subprocess.check_output(["git", "-C", str(ROOT), "rev-parse", "HEAD"], text=True).strip()
    test_state = subprocess.check_output(["git", "-C", str(root), "status", "--porcelain"], text=True).strip()
    tool_state = subprocess.check_output(["git", "-C", str(ROOT), "status", "--porcelain"], text=True).strip()
    go_version = subprocess.check_output(["go", "version"], text=True).strip()
    index = test_index(root)
    (out / "test-index.json").write_text(json.dumps(index, sort_keys=True))
    rows = [row for row in load_rows(root / "docs/fidelity/depth", root) if row["status"] == "unit-green"]
    requests = []
    resolved = {}
    for row in rows:
        symbols = proof_symbols(row["proof"]) or [""]
        for symbol in symbols:
            declaration, problem, detail = resolve(symbol, index)
            requests.append((row, symbol, declaration, problem, detail))
            if declaration:
                name = symbol.partition(":")[2] if ":" in symbol else symbol
                resolved[symbol] = (declaration, name)
    runs = execution_batches(list(resolved.values()), root, out)
    records = []
    for row, symbol, declaration, problem, detail in requests:
        if not problem:
            declaration, name = resolved[symbol]
            problem, detail = outcome(declaration, name, runs[declaration["package"]])
        records.append({"manifest": row["manifest"], "line": row["line"], "case_id": row["case_id"],
                        "symbol": symbol, "problem": problem, "detail": detail,
                        "test_file": declaration["file"] if declaration else "",
                        "test_line": declaration["line"] if declaration else ""})
    with (out / "unit-results.tsv").open("w") as stream:
        writer = csv.DictWriter(stream, fieldnames=["manifest", "line", "case_id", "symbol", "problem", "detail", "test_file", "test_line"], delimiter="\t", lineterminator="\n")
        writer.writeheader(); writer.writerows(records)
    counts = Counter(record["problem"] for record in records)
    summary = f"fidelity-units: rows={len(rows)} symbols={len({r['symbol'] for r in records})} claims={len(records)} packages={len(runs)} " + " ".join(f"{k}={v}" for k,v in sorted(counts.items())) + f" elapsed={time.monotonic()-started:.3f}s"
    (out / "summary.txt").write_text(summary + "\n")
    (out / "MANIFEST.md").write_text(f"# Unit proof evidence\n\n- Main/test root: `{root}`\n- Test and manifest HEAD: `{head}`\n- Tooling HEAD: `{tool_head}`\n- Test tree at start: `{test_state or "clean"}`\n- Tooling tree at start: `{tool_state or "clean"}`\n- Go version: `{go_version}`\n- Command: `{sys.argv}`\n- Summary: `{summary}`\n\n## Package commands\n\n" + "\n".join(f"- `{r['command']}` → exit {r['status']}; `{r['log']}`" for r in runs.values()) + "\n")
    print(summary)
    for record in records:
        if record["problem"] != "PASS":
            print(f"{record['manifest']}:{record['line']} {record['symbol']}: {record['problem']}: {record['detail']}")
    return 1 if any(record["problem"] != "PASS" for record in records) else 0

if __name__ == "__main__":
    raise SystemExit(main())
