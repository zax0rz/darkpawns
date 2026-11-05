#!/usr/bin/env python3
"""Measure scratch-server readiness and peak RSS for the idiom experiment."""

from __future__ import annotations

import os
import re
import signal
import socket
import statistics
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[4]
RUN = Path(__file__).resolve().parent
BINARY = Path(sys.argv[1]).resolve()
LABEL = sys.argv[2]
SCRATCH = RUN / f"efficiency-{LABEL}"
SCRATCH.mkdir(parents=True, exist_ok=True)
(SCRATCH / "world").symlink_to(ROOT / "lib/world", target_is_directory=True)

FORBIDDEN = {4000, 7777, 7778, 7779}
TIME_BIN = "/usr/bin/time"
READY_TIMEOUT = 180.0
ROWS: list[tuple[int, float, int]] = []


def free_port() -> int:
    while True:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
        sock.close()
        if port not in FORBIDDEN:
            return port


def child_pid(parent_pid: int) -> int | None:
    children_file = Path(f"/proc/{parent_pid}/task/{parent_pid}/children")
    try:
        children = children_file.read_text().split()
    except FileNotFoundError:
        return None
    return int(children[0]) if children else None


def terminate_only(pid: int, wrapper: subprocess.Popen[bytes]) -> None:
    try:
        os.kill(pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        wrapper.wait(timeout=30)
    except subprocess.TimeoutExpired:
        try:
            os.kill(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        wrapper.wait(timeout=10)


for trial in range(1, 6):
    trial_dir = SCRATCH / f"trial-{trial:02d}"
    trial_dir.mkdir(parents=True, exist_ok=True)
    port = free_port()
    time_file = trial_dir / "time.txt"
    log_file = trial_dir / "server.log"
    database = trial_dir / "darkpawns.db"
    args = [
        TIME_BIN,
        "-v",
        "-o",
        str(time_file),
        str(BINARY),
        "-world",
        str(SCRATCH / "world"),
        "-db",
        str(database),
        "-port",
        str(port),
        "-http-bind",
        "127.0.0.1",
        "-telnet-port",
        "0",
        "-telnet-tls-port",
        "0",
        "-whod-port",
        "0",
    ]
    env = os.environ.copy()
    env["JWT_SECRET"] = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    env["DP_SEED"] = "1"
    env.pop("DP_CLOCK", None)
    env.pop("DP_FIXED_TIME", None)
    with log_file.open("wb") as log:
        started = time.monotonic_ns()
        proc = subprocess.Popen(args, cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT)
        server_pid = None
        deadline = time.monotonic() + READY_TIMEOUT
        connected = False
        while time.monotonic() < deadline:
            if proc.poll() is not None:
                break
            server_pid = child_pid(proc.pid)
            try:
                with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                    connected = True
                    break
            except OSError:
                time.sleep(0.005)
        ready_ms = (time.monotonic_ns() - started) / 1_000_000
        if server_pid is None:
            server_pid = child_pid(proc.pid)
        if server_pid is not None:
            terminate_only(server_pid, proc)
        else:
            proc.wait(timeout=10)
    if not connected:
        print(f"trial {trial}: server failed to accept TCP connections; see {log_file}", file=sys.stderr)
        sys.exit(1)
    timed = time_file.read_text()
    match = re.search(r"Maximum resident set size \(kbytes\):\s+(\d+)", timed)
    if not match:
        print(f"trial {trial}: peak RSS missing from {time_file}", file=sys.stderr)
        sys.exit(1)
    rss_bytes = int(match.group(1)) * 1024
    ROWS.append((port, ready_ms, rss_bytes))
    print(f"trial\t{trial}\tport\t{port}\tready_ms\t{ready_ms:.3f}\tpeak_rss_bytes\t{rss_bytes}")

print(f"binary_bytes\t{BINARY.stat().st_size}")
print(f"median_ready_ms\t{statistics.median(row[1] for row in ROWS):.3f}")
print(f"median_peak_rss_bytes\t{int(statistics.median(row[2] for row in ROWS))}")
print(f"raw_dir\t{SCRATCH}")
