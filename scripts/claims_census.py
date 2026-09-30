#!/usr/bin/env python3
"""Detached claims driver, called only by census_runner.sh under its lock."""
from __future__ import annotations
from collections import Counter, defaultdict
import os
from pathlib import Path
import re
import subprocess
import time

GOOD = {"PASS", "EXPECTED", "EXPECTED_UNSTABLE"}
KINDS = GOOD | {"FAIL", "STALE", "UNPINNABLE", "INFRA", "TIMEOUT"}


def atomic(path, text):
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text(text)
    tmp.replace(path)


def read_results(path, names):
    rows = {}
    if path.exists():
        for line in path.read_text().splitlines():
            fields = line.split("\t")
            if len(fields) >= 2 and fields[0] in KINDS and fields[1] in names:
                if fields[1] in rows:
                    raise ValueError(f"duplicate result: {fields[1]}")
                seconds = fields[2] if len(fields) > 2 else "0"
                if not seconds.isdigit():
                    raise ValueError(f"invalid seconds: {line}")
                rows[fields[1]] = (fields[0], seconds)
            else:
                raise ValueError(f"unexpected result: {line}")
    # A scheduling/build failure cannot become a green missing pair.
    return {name: rows.get(name, ("FAIL", "0")) for name in names}


def invoke(run, runner, seed, names, jobs, recheck=False):
    suffix = "recheck" if recheck else "initial"
    directory = run / f"seed{seed}"
    directory.mkdir(exist_ok=True)
    results = directory / ("recheck-results.tsv" if recheck else "results.tsv")
    env = dict(os.environ, ORACLE_REGRESSION_SEED=str(seed),
               ORACLE_REGRESSION_SCENARIOS=",".join(names),
               ORACLE_REGRESSION_JOBS=str(min(jobs, len(names))),
               ORACLE_REGRESSION_RESULTS=str(results),
               ORACLE_REGRESSION_DUMP=str(directory / f"{suffix}-dump"),
               ORACLE_REGRESSION_LOG_ROOT=str(directory / f"{suffix}-logs"))
    seen = set()
    with (directory / f"{suffix}.log").open("w") as log:
        process = subprocess.Popen([runner], env=env, stdout=subprocess.PIPE,
                                   stderr=subprocess.STDOUT, text=True)
        for line in process.stdout:
            log.write(line)
            log.flush()
            match = re.match(r"^(PASS|FAIL|EXPECTED|EXPECTED_UNSTABLE|UNPINNABLE|STALE|TIMEOUT|INFRA) (\S+)", line)
            if match and match[2] in names:
                seen.add(match[2])
                if not recheck:
                    atomic(run / "claims-progress", f"{seed}\t{len(seen)}\t{len(names)}\n")
        status = process.wait()
    if status and not results.exists():
        raise RuntimeError(f"seed {seed}: runner exited {status} before writing results; see {directory / (suffix + '.log')}")
    return read_results(results, names)


def summarize(rows, seed, elapsed):
    counts = Counter(kind for kind, _ in rows.values())
    return (f"oracle-claims: seed={seed} pairs={len(rows)} " +
            " ".join(f"{kind.lower()}={counts[kind]}" for kind in sorted(KINDS)) +
            f" elapsed={elapsed:.3f}s")


def main():
    run = Path(os.environ["CENSUS_RUN_DIR"])
    runner = os.environ["CENSUS_RUNNER"]
    groups = defaultdict(list)
    for line in (run / "claims.tsv").read_text().splitlines():
        seed, name, _ = line.split("\t")
        groups[int(seed)].append(name)
    started = time.monotonic()
    all_rows = {}
    rechecked = False
    with (run / "claims-results.tsv").open("w") as merged, (run / "claims-recheck-results.tsv").open("w") as checks:
        for seed, names in sorted(groups.items()):
            before = time.monotonic()
            atomic(run / "claims-progress", f"{seed}\t0\t{len(names)}\n")
            rows = invoke(run, runner, seed, names, int(os.environ["CENSUS_JOBS"]))
            for name in sorted(rows):
                kind, seconds = rows[name]
                merged.write(f"{kind}\t{name}\t{seed}\t{seconds}\n")
            merged.flush()
            retry = [name for name in names if rows[name][0] in {"INFRA", "TIMEOUT"}]
            if retry:
                rechecked = True
                repeats = invoke(run, runner, seed, retry, 1, recheck=True)
                for name in sorted(repeats):
                    kind, seconds = repeats[name]
                    checks.write(f"{kind}\t{name}\t{seed}\t{seconds}\n")
                checks.flush()
                rows.update(repeats)
            verdict = "CLEAN" if all(kind in GOOD for kind, _ in rows.values()) else "NOT_CLEAN"
            (run / f"summary-seed{seed}.txt").write_text(summarize(rows, seed, time.monotonic()-before) + f" verdict={verdict}\n")
            all_rows.update({(seed, name): value for name, value in rows.items()})
    verdict = "NOT_CLEAN" if any(kind not in GOOD for kind, _ in all_rows.values()) else ("CLEAN_AFTER_RECHECK" if rechecked else "CLEAN")
    (run / "claims-summary.txt").write_text(summarize(all_rows, "all", time.monotonic()-started) + "\n")
    (run / "claims-verdict").write_text(verdict + "\n")
    return 1 if verdict == "NOT_CLEAN" else 0


if __name__ == "__main__":
    raise SystemExit(main())
