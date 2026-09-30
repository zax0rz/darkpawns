#!/usr/bin/env python3
"""Print distinct claimed oracle pairs: seed, scenario, sorted case IDs."""
from __future__ import annotations
import argparse
from collections import defaultdict
import pathlib
import re
import sys
from fidelity_manifest import ROOT, MANIFEST_DIR, load_rows


def claimed_pairs(rows):
    pairs = defaultdict(set)
    for row in rows:
        if row["status"] not in {"oracle-green", "oracle-green-multiseed"}:
            continue
        for proof in row["proof"].split(";"):
            proof = proof.strip()
            # Composite oracle rows may append a supplemental Go unit symbol.
            if re.fullmatch(r"Test[A-Za-z0-9_]+", proof):
                continue
            scenario, marker, seeds = proof.partition("@")
            if not re.fullmatch(r"[A-Za-z0-9_-]+", scenario):
                raise ValueError(f"{row['manifest']}:{row['line']}: invalid scenario {scenario!r}")
            seed_list = seeds.split(",") if marker else ["1"]
            if any(not re.fullmatch(r"[1-9][0-9]*", seed) for seed in seed_list):
                raise ValueError(f"{row['manifest']}:{row['line']}: invalid seeds {seeds!r}")
            for seed in seed_list:
                pairs[int(seed), scenario].add(row["case_id"])
    return [(seed, scenario, ",".join(sorted(ids))) for (seed, scenario), ids in sorted(pairs.items())]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest-dir", type=pathlib.Path, default=MANIFEST_DIR)
    parser.add_argument("--root", type=pathlib.Path, default=ROOT)
    args = parser.parse_args()
    try:
        for pair in claimed_pairs(load_rows(args.manifest_dir, args.root)):
            print("\t".join(map(str, pair)))
    except (OSError, ValueError) as exc:
        print(f"manifest-claims: {exc}", file=sys.stderr)
        return 1
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
