#!/usr/bin/env python3
"""Print distinct claimed oracle pairs: seed, scenario, sorted case IDs."""
from __future__ import annotations
import argparse
from collections import defaultdict
import pathlib
import re
import sys
from typing import NamedTuple
from fidelity_manifest import ROOT, MANIFEST_DIR, load_rows


class OracleProof(NamedTuple):
    scenario: str
    seeds: tuple[str, ...]
    transport: str | None
    has_seed_list: bool


def strip_transport_suffix(token: str) -> tuple[str, str | None]:
    """Remove the supported transport qualifier from a scenario proof."""
    token = token.strip()
    if "[" not in token and "]" not in token:
        return token, None
    match = re.fullmatch(r"(?P<proof>[^\[\]\r\n]+)\[(?P<transport>[^\[\]\r\n]+)\]", token)
    if not match:
        raise ValueError(f"invalid transport suffix in {token!r}; expected one [ws] suffix after the seed list")
    transport = match["transport"]
    if transport != "ws":
        raise ValueError(f"unsupported transport suffix [{transport}]; only [ws] is allowed")
    proof = match["proof"]
    if "@" not in proof:
        raise ValueError("[ws] suffix must follow a scenario seed list: <scenario>@<seeds>[ws]")
    return proof, transport


def parse_oracle_proof(token: str) -> OracleProof:
    """Parse a scenario proof and its optional seed and transport qualifiers."""
    token, transport = strip_transport_suffix(token)
    scenario, marker, seeds = token.partition("@")
    if not re.fullmatch(r"[A-Za-z0-9_-]+", scenario):
        raise ValueError(f"invalid scenario {scenario!r}")
    seed_list = tuple(seeds.split(",")) if marker else ("1",)
    if any(not re.fullmatch(r"[1-9][0-9]*", seed) for seed in seed_list):
        raise ValueError(f"invalid seeds {seeds!r}")
    return OracleProof(scenario, seed_list, transport, bool(marker))


def proof_scenario(token: str) -> str:
    """Return the scenario name after validating and removing [ws], if present."""
    proof, _transport = strip_transport_suffix(token)
    if _transport == "ws":
        return parse_oracle_proof(token).scenario
    return proof.partition("@")[0]


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
            try:
                parsed = parse_oracle_proof(proof)
            except ValueError as exc:
                raise ValueError(f"{row['manifest']}:{row['line']}: {exc}") from exc
            # The claims census is telnet-only. A WS proof is validated by the
            # depth gate but is never scheduled as an ordinary telnet claim.
            if parsed.transport == "ws":
                continue
            for seed in parsed.seeds:
                pairs[int(seed), parsed.scenario].add(row["case_id"])
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
