#!/usr/bin/env python3
"""Prove flee-death-exit reaches the C death-room gate with a fixture control."""

import argparse
import os
from pathlib import Path
import subprocess


SCENARIO = Path("cmd/dp-oracle-diff/scenarios/flee-death-exit.txt")
ON = "set-room-flag 8161 1 on"
OFF = "set-room-flag 8161 1 off"
PANIC = "PANIC!  You couldn't escape!"
MOVED = "You flee head over heels."


def assert_death_exit_witness(text: str) -> None:
    assert PANIC in text, "death-exit refusal witness missing"
    assert MOVED not in text, "flee unexpectedly moved through a death exit"


def run(stage: str, output: Path, seed: int, expected: str) -> None:
    dump = output / f"{stage}-C"
    dump.mkdir(parents=True, exist_ok=True)
    command = [
        "go",
        "run",
        "./cmd/dp-oracle-diff",
        "--scenario",
        "flee-death-exit",
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
    transcript = (dump / "flee-death-exit.txt").read_text()
    if expected == "death":
        assert_death_exit_witness(transcript)
    else:
        assert MOVED in transcript, "safe-exit control did not reach movement"
        assert PANIC not in transcript, "death-exit witness survived after clearing ROOM_DEATH"
        try:
            assert_death_exit_witness(transcript)
        except AssertionError:
            pass
        else:
            raise AssertionError("removing ROOM_DEATH did not invalidate the proof assertion")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    parser.add_argument("--seed", type=int, default=1)
    args = parser.parse_args()

    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    original = SCENARIO.read_text()
    assert original.count(ON) == 1, "expected exactly one death-room flag fixture"
    (output / "HEAD.txt").write_text(subprocess.check_output(["git", "rev-parse", "HEAD"], text=True))
    try:
        for stage, source, expected in (
            ("green", original, "death"),
            ("control", original.replace(ON, OFF), "safe"),
            ("restore", original, "death"),
        ):
            SCENARIO.write_text(source)
            run(stage, output, args.seed, expected)
            print(f"flee-death-exit {stage}: PASS", flush=True)
    finally:
        SCENARIO.write_text(original)


if __name__ == "__main__":
    main()
