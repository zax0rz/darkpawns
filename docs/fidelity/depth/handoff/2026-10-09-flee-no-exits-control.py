#!/usr/bin/env python3
"""Prove flee-no-exits reaches six failed CAN_GO samples with a fixture control."""

import argparse
import os
from pathlib import Path
import subprocess


SCENARIO = Path("cmd/dp-oracle-diff/scenarios/flee-no-exits.txt")
NO_EXITS = "replace-room-exits 8162 none"
SAFE_EXITS = "replace-room-exits 8162 all 8161 0"
PANIC = "PANIC!  You couldn't escape!"
MOVED = "You flee head over heels."


def assert_no_exits_witness(text: str) -> None:
    assert PANIC in text, "no-exits terminal witness missing"
    assert MOVED not in text, "flee unexpectedly moved from the no-exits fixture"


def run(stage: str, output: Path, seed: int, expected: str) -> None:
    dump = output / f"{stage}-C"
    dump.mkdir(parents=True, exist_ok=True)
    command = [
        "go",
        "run",
        "./cmd/dp-oracle-diff",
        "--scenario",
        "flee-no-exits",
        "--seed",
        str(seed),
        "--show-oracle",
        "--dump-oracle",
        str(dump),
    ]
    result = subprocess.run(
        command,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        env={
            **os.environ,
            "DP_ORACLE_BIN": os.environ.get(
                "DP_ORACLE_BIN", "/home/zach/darkpawns-c-oracle/bin/circle"
            ),
        },
    )
    (output / f"{stage}.log").write_text(result.stdout + f"\nEXIT={result.returncode}\n")
    assert "[build failed]" not in result.stdout and "undefined:" not in result.stdout, result.stdout
    assert result.returncode == 0, result.stdout
    transcript = (dump / "flee-no-exits.txt").read_text()
    if expected == "none":
        assert_no_exits_witness(transcript)
    else:
        assert MOVED in transcript, "safe-exit control did not reach movement"
        assert PANIC not in transcript, "no-exits witness survived after adding safe exits"
        try:
            assert_no_exits_witness(transcript)
        except AssertionError:
            pass
        else:
            raise AssertionError("adding safe exits did not invalidate the proof assertion")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    parser.add_argument("--seed", type=int, default=1)
    args = parser.parse_args()

    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    original = SCENARIO.read_text()
    assert original.count(NO_EXITS) == 1, "expected exactly one no-exits fixture"
    (output / "HEAD.txt").write_text(subprocess.check_output(["git", "rev-parse", "HEAD"], text=True))
    try:
        for stage, source, expected in (
            ("green", original, "none"),
            ("control", original.replace(NO_EXITS, SAFE_EXITS), "safe"),
            ("restore", original, "none"),
        ):
            SCENARIO.write_text(source)
            run(stage, output, args.seed, expected)
            print(f"flee-no-exits {stage}: PASS", flush=True)
    finally:
        SCENARIO.write_text(original)


if __name__ == "__main__":
    main()
