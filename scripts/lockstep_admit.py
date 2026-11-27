#!/usr/bin/env python3
"""Acquire a single host-budget worker permit for a lockstep session.

Calls directly into scripts/census_budget.py without modifying census admission
logic: shares the exact host worker budget, memory floor, release/exclusive
draining barriers, subreaper process-group cleanup, and crash reconciliation.
"""
from datetime import datetime
from pathlib import Path
import argparse
import hashlib
import json
import os
import shutil
import sys
import tempfile
import time

sys.path.insert(0, str(Path(__file__).resolve().parent))
import census_budget as budget


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(
        prog="lockstep_admit.py",
        description="Run a lockstep dp-oracle-diff process under the shared census worker budget.",
    )
    parser.add_argument("--name", required=True, help="session label for admission logs")
    parser.add_argument(
        "--oracle",
        default=os.environ.get("DP_ORACLE_BIN", "/home/zach/darkpawns-c-oracle/bin/circle"),
        help="path to C oracle binary (default: DP_ORACLE_BIN)",
    )
    parser.add_argument(
        "--run-dir",
        default="",
        help="directory for admission state/events (default: temporary directory removed on exit)",
    )
    parser.add_argument("command", nargs=argparse.REMAINDER, help="command to execute after '--'")
    args = parser.parse_args(argv)

    cmd = list(args.command)
    if cmd and cmd[0] == "--":
        cmd = cmd[1:]
    if not cmd:
        parser.error("missing command after '--'")

    oracle = Path(args.oracle).resolve()
    if not oracle.is_file():
        print(f"lockstep-admit: oracle binary not found: {oracle}", file=sys.stderr)
        return 2
    oracle_sha = sha256_file(oracle)

    temp_dir = None
    if args.run_dir:
        run_dir = Path(args.run_dir).resolve()
        run_dir.mkdir(parents=True, exist_ok=True)
    else:
        temp_dir = tempfile.mkdtemp(prefix=f"dp-lockstep-{args.name}-")
        run_dir = Path(temp_dir).resolve()

    key = str(run_dir)
    started = datetime.now().astimezone().isoformat(timespec="seconds")
    record = dict(
        status="queued",
        pid=os.getpid(),
        birth=budget.identity(os.getpid()),
        submitted=time.time(),
        started=started,
        jobs=1,
        release=False,
        exclusive=False,
        oracle=str(oracle),
        oracle_sha=oracle_sha,
        budget=budget.LIMIT,
        memory_floor_kib=budget.FLOOR_KIB,
        coordinator_version=budget.VERSION,
        kind="lockstep",
        name=args.name,
    )

    registered = False
    try:
        with budget.transaction() as state:
            state["runs"][key] = record
            budget.atomic(run_dir / "state", f"queued pid={record['pid']} started={started}\n")
            (run_dir / "admission.json").write_text(json.dumps(record, indent=2))
            budget.event(key, "registered", budget=budget.LIMIT, jobs=1, release=False, kind="lockstep")
            registered = True

        os.environ["CENSUS_RUN_DIR"] = key
        return budget.execute("worker", cmd)
    finally:
        if registered:
            try:
                with budget.transaction() as state:
                    if key in state["runs"]:
                        state["runs"][key]["status"] = "done"
                        budget.atomic(run_dir / "state", "done\n")
                        budget.event(key, "finished", kind="lockstep")
            except Exception as exc:
                print(f"lockstep-admit: failed to mark run done: {exc}", file=sys.stderr)
        if temp_dir is not None:
            shutil.rmtree(temp_dir, ignore_errors=True)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (RuntimeError, InterruptedError, OSError) as exc:
        print(f"lockstep-admit: {exc}", file=sys.stderr)
        sys.exit(125)
