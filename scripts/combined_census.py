#!/usr/bin/env python3
"""Full census and claims projections of one deduplicated, bounded worker pool.

Only census_runner.sh starts this driver, under the existing census lock.
The original worker owns all attempt classification and retained artifacts.
"""
from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor, as_completed
from collections import Counter
import os
from pathlib import Path
import re
import shutil
import subprocess
import time

from claims_census import GOOD, atomic, read_results, summarize


def scenario_cost(path):
    lines = path.read_text().splitlines()
    steps = sum(bool(line.strip()) and not line.lstrip().startswith(("#", "[")) for line in lines)
    peers = {m[1] for line in lines if (m := re.match(r"^\[setup:(?:oracle|port):([^]]+)\]", line))}
    bounces = sum(line.startswith(("<RESTART>", "<CRASH>")) for line in lines)
    return steps * (1 + len(peers)) + 20 * bounces


def prior_timings(run):
    """Latest completed timing per pair, with a scenario fallback across seeds.

    Only scheduling uses history. Verdicts always come from newly executed workers.
    Retain the selected estimates so the queue is reproducible after history grows.
    """
    timings = {}
    if os.environ.get("CENSUS_TIMINGS_FILE"):
        for line in Path(os.environ["CENSUS_TIMINGS_FILE"]).read_text().splitlines():
            seed, name, seconds = line.split("\t")
            if not seed.isdigit() or int(seed) < 1 or not seconds.isdigit():
                raise ValueError(f"invalid scheduling timing: {line}")
            timings[int(seed), name] = int(seconds)
        return timings
    root = run.parent.parent
    previous = sorted((p for p in root.glob("*/*") if (p / "summary.txt").exists()),
                      key=lambda p: (p / "summary.txt").stat().st_mtime)
    for directory in previous:
        for filename, claimed in (("results.tsv", False), ("claims-results.tsv", True)):
            path = directory / filename
            if not path.exists():
                continue
            for line in path.read_text().splitlines():
                fields = line.split("\t")
                if len(fields) != (4 if claimed else 3):
                    continue
                kind, name = fields[:2]
                seed, seconds = (fields[2], fields[3]) if claimed else ("1", fields[2])
                if kind in GOOD and seed.isdigit() and seconds.isdigit():
                    timings[(int(seed), name)] = int(seconds)
    return timings


def order_pairs(pairs, costs, timings):
    by_name = {}
    for (_, name), seconds in timings.items():
        by_name[name] = max(by_name.get(name, 0), seconds)
    estimate = lambda pair: timings.get(pair, by_name.get(pair[1], costs[pair[1]]))
    return sorted(pairs, key=lambda pair: (-estimate(pair), -costs[pair[1]], pair[1], pair[0]))


def enumerate_pairs(full, claims):
    return {(1, name) for name in full} | set(claims)


def verdict(rows, repeats):
    final = dict(rows)
    final.update(repeats)
    if any(kind not in GOOD for kind, _ in final.values()):
        return "NOT_CLEAN"
    return "CLEAN_AFTER_RECHECK" if repeats else "CLEAN"


def write_rows(path, rows, claims=False):
    atomic(path, "".join(
        f"{kind}\t{name}\t{seed}\t{seconds}\n" if claims else f"{kind}\t{name}\t{seconds}\n"
        for (seed, name), (kind, seconds) in sorted(rows.items())
    ))


def full_summary(rows, elapsed):
    counts = Counter(kind for kind, _ in rows.values())
    labels = {"passed": "PASS", "expected": "EXPECTED", "unpinnable": "UNPINNABLE",
              "stale": "STALE", "failed": "FAIL", "infra": "INFRA",
              "timed_out": "TIMEOUT", "unstable": "EXPECTED_UNSTABLE"}
    return (f"oracle-regression: scenarios={len(rows)} " +
            " ".join(f"{label}={counts[kind]}" for label, kind in labels.items()) +
            f" elapsed={elapsed:.3f}s")


def execute_pool(pairs, jobs, worker, progress):
    """Bound concurrency across seeds; collect every task, including missing results."""
    rows = {}
    with ThreadPoolExecutor(max_workers=jobs) as pool:
        futures = {pool.submit(worker, pair): pair for pair in pairs}
        for future in as_completed(futures):
            pair = futures[future]
            rows[pair] = future.result()
            progress(len(rows), len(pairs))
    return rows


def run_worker(repo, run, binaries, pair, recheck=False):
    seed, name = pair
    suffix = "recheck" if recheck else "initial"
    directory = run / f"seed{seed}"
    # One retained attempt directory per seed; each scenario's names are unique.
    log_dir = directory / f"{suffix}-logs" / "attempt.combined"
    result_dir = log_dir / "results"
    dump = directory / f"{suffix}-dump"
    result_dir.mkdir(parents=True, exist_ok=True)
    dump.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, repo_root=str(repo), go_bin=os.environ.get("ORACLE_REGRESSION_GO", "/usr/local/go/bin/go"),
               oracle_bin=os.environ["CENSUS_ORACLE_BIN"],
               scenario_timeout=os.environ.get("ORACLE_REGRESSION_TIMEOUT", "240s"),
               seed=str(seed), log_dir=str(log_dir), result_dir=str(result_dir),
               harness_bin=str(binaries / "dp-oracle-diff"), server_bin=str(binaries / "dp-server-prebuilt"),
               ORACLE_REGRESSION_SERVER=str(binaries / "dp-server-prebuilt"), ORACLE_REGRESSION_DUMP=str(dump),
               EXPECTED_DIVERGENCES_FILE=str(repo / "cmd/dp-oracle-diff/expected_divergences.tsv"),
               EXPECTED_DIVERGENCE_PINS_FILE=str(repo / "cmd/dp-oracle-diff/expected_divergence_pins.tsv"))
    process = subprocess.run([str(repo / "scripts/oracle_regression_worker.sh"), name + ".txt"],
                             env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    (log_dir / f"{name}.worker.log").write_text(process.stdout)
    # Worker output is retained and replayed into the existing per-seed log.
    return read_results(result_dir / name, [name])[name]


def recheck_pairs(ordered, initial, full, claims):
    full_bad = any(initial[(1, name)][0] in {"FAIL", "STALE", "UNPINNABLE"} for name in full)
    return [pair for pair in ordered if initial[pair][0] in {"INFRA", "TIMEOUT"}
            and (pair in claims or (pair[0] == 1 and not full_bad))]


def project(run, full, claims, initial, repeats, elapsed):
    full_pairs = {(1, name) for name in full}
    write_rows(run / "combined-results.tsv", initial, True)
    write_rows(run / "combined-recheck-results.tsv", repeats, True)
    for label, selection in (("full", full_pairs), ("claims", claims)):
        rows = {pair: initial[pair] for pair in selection}
        checks = {pair: repeats[pair] for pair in selection if pair in repeats}
        # Full census does not recheck infra if it has content failures.
        if label == "full" and any(kind in {"FAIL", "STALE", "UNPINNABLE"} for kind, _ in rows.values()):
            checks = {}
        prefix = "claims-" if label == "claims" else ""
        write_rows(run / f"{prefix}results.tsv", rows, label == "claims")
        write_rows(run / f"{prefix}recheck-results.tsv", checks, label == "claims")
        final = rows | checks
        summary = summarize(final, "all", elapsed) if label == "claims" else full_summary(rows, elapsed)
        outcome = verdict(rows, checks)
        atomic(run / f"{label}-summary.txt", summary + "\n")
        atomic(run / f"{label}-verdict", outcome + "\n")
    for seed in sorted({seed for seed, _ in initial}):
        directory = run / f"seed{seed}"
        directory.mkdir(exist_ok=True)
        rows = {pair: value for pair, value in initial.items() if pair[0] == seed}
        checks = {pair: value for pair, value in repeats.items() if pair[0] == seed}
        write_rows(directory / "combined-results.tsv", rows)
        write_rows(directory / "results.tsv", {pair: value for pair, value in rows.items() if pair in claims})
        write_rows(directory / "combined-recheck-results.tsv", checks)
        write_rows(directory / "recheck-results.tsv", {pair: value for pair, value in checks.items() if pair in claims})
        for suffix in ("initial", "recheck"):
            logs = directory / f"{suffix}-logs" / "attempt.combined"
            atomic(directory / f"{suffix}.log", "".join(path.read_text() for path in sorted(logs.glob("*.worker.log"))))
        claim_rows = {pair: value for pair, value in (initial | repeats).items() if pair in claims and pair[0] == seed}
        atomic(run / f"summary-seed{seed}.txt", summarize(claim_rows, seed, elapsed) +
               f" verdict={verdict(claim_rows, {})}\n")
    if (run / "seed1/recheck.log").exists():
        shutil.copy2(run / "seed1/recheck.log", run / "recheck.log")
    # Seed-1 normalized C blocks remain available at the full census path too.
    dump = run / "seed1/initial-dump"
    if dump.exists():
        shutil.copytree(dump, run / "census-dump", dirs_exist_ok=True)


def main():
    started = time.monotonic()
    run = Path(os.environ["CENSUS_RUN_DIR"])
    repo = Path(os.environ["CENSUS_REPO_ROOT"])
    full = {path.stem for path in (repo / "cmd/dp-oracle-diff/scenarios").glob("*.txt")}
    claims = {(int(seed), name) for seed, name, _ in
              (line.split("\t") for line in (run / "claims.tsv").read_text().splitlines())}
    pairs = enumerate_pairs(full, claims)
    costs = {name: scenario_cost(repo / f"cmd/dp-oracle-diff/scenarios/{name}.txt") for _, name in pairs}
    timings = prior_timings(run)
    ordered = order_pairs(pairs, costs, timings)
    atomic(run / "scheduling-timings.tsv", "".join(f"{seed}\t{name}\t{seconds}\n"
           for (seed, name), seconds in sorted(timings.items()) if (seed, name) in pairs))
    atomic(run / "pairs.tsv", "".join(f"{seed}\t{name}\n" for seed, name in ordered))
    binaries = run / "binaries"
    binaries.mkdir()
    go = os.environ.get("ORACLE_REGRESSION_GO", "/usr/local/go/bin/go")
    for package, name in (("./cmd/dp-oracle-diff", "dp-oracle-diff"), ("./cmd/server", "dp-server-prebuilt")):
        subprocess.run([go, "build", "-C", str(repo), "-o", str(binaries / name), package], check=True)
    progress = lambda done, total: atomic(run / "claims-progress", f"all\t{done}\t{total}\n")
    initial = execute_pool(ordered, int(os.environ["CENSUS_JOBS"]),
                           lambda pair: run_worker(repo, run, binaries, pair), progress)
    retry = recheck_pairs(ordered, initial, full, claims)
    repeats = execute_pool(retry, 1, lambda pair: run_worker(repo, run, binaries, pair, True),
                           lambda done, total: None)
    project(run, full, claims, initial, repeats, time.monotonic() - started)
    outcomes = [(run / f"{label}-verdict").read_text().strip() for label in ("full", "claims")]
    outcome = "NOT_CLEAN" if "NOT_CLEAN" in outcomes else (
        "CLEAN_AFTER_RECHECK" if "CLEAN_AFTER_RECHECK" in outcomes else "CLEAN")
    atomic(run / "combined-summary.txt", f"oracle-combined: pairs={len(pairs)} deduplicated={len(full)+len(claims)-len(pairs)} "
           f"full={outcomes[0]} claims={outcomes[1]} elapsed={time.monotonic()-started:.3f}s\n")
    atomic(run / "combined-verdict", outcome + "\n")
    print((run / "full-summary.txt").read_text().strip(), flush=True)
    print((run / "claims-summary.txt").read_text().strip(), flush=True)
    print((run / "combined-summary.txt").read_text().strip(), flush=True)
    return 1 if outcome == "NOT_CLEAN" else 0


if __name__ == "__main__":
    raise SystemExit(main())
