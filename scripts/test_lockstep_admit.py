#!/usr/bin/env python3
"""Regression tests for scripts/lockstep_admit.py (R5h)."""
from pathlib import Path
import json
import os
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPTS))
import census_budget as budget


class LockstepAdmitTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name)
        self.memory = self.base / "memory"
        self.memory.write_text(str(32 * 1024 * 1024))
        self.oracle = self.base / "circle"
        self.oracle.write_text("#!/bin/sh\nexit 0\n")
        self.oracle.chmod(0o755)
        self.env = dict(
            os.environ,
            CENSUS_TESTING="1",
            CENSUS_TEST_RUNTIME=str(self.base / "runtime"),
            CENSUS_TEST_MEMORY=str(self.memory),
            DP_ORACLE_BIN=str(self.oracle),
        )
        self.patcher = patch.dict(os.environ, self.env)
        self.patcher.start()

    def tearDown(self):
        self.patcher.stop()
        self.temp.cleanup()

    def test_acquires_and_releases_worker_permit(self):
        run_dir = self.base / "run-ok"
        res = subprocess.run(
            [
                sys.executable,
                str(SCRIPTS / "lockstep_admit.py"),
                "--name",
                "smoke",
                "--oracle",
                str(self.oracle),
                "--run-dir",
                str(run_dir),
                "--",
                sys.executable,
                "-c",
                "import os; assert os.environ.get('CENSUS_LEASED') == '1'; print('ok')",
            ],
            env=self.env,
            text=True,
            capture_output=True,
        )
        self.assertEqual(res.returncode, 0, res.stderr)
        self.assertEqual(res.stdout.strip(), "ok")
        events = [json.loads(line) for line in (run_dir / "admission-events.jsonl").read_text().splitlines()]
        names = [e["event"] for e in events]
        self.assertIn("registered", names)
        self.assertIn("granted", names)
        self.assertIn("released", names)
        self.assertIn("finished", names)
        with budget.transaction() as state:
            self.assertEqual(state["leases"], {})
            self.assertEqual(state["runs"][str(run_dir)]["status"], "done")

    def test_sigkill_reaps_child_and_permit_on_reconcile(self):
        run_dir = self.base / "run-kill"
        proc = subprocess.Popen(
            [
                sys.executable,
                str(SCRIPTS / "lockstep_admit.py"),
                "--name",
                "crash",
                "--oracle",
                str(self.oracle),
                "--run-dir",
                str(run_dir),
                "--",
                "sleep",
                "60",
            ],
            env=self.env,
        )
        pgid = 0
        for _ in range(200):
            reg = self.base / "runtime" / "registry.json"
            if reg.exists():
                data = json.loads(reg.read_text())
                leases = [l for l in data["leases"].values() if l.get("pgid")]
                if leases:
                    pgid = leases[0]["pgid"]
                    break
            time.sleep(0.02)
        self.assertTrue(pgid, "lockstep_admit did not acquire a live lease")
        proc.send_signal(signal.SIGKILL)
        proc.wait(timeout=5)
        for _ in range(200):
            with budget.transaction() as state:
                if not state["leases"] and not budget.group_alive(pgid):
                    break
            time.sleep(0.02)
        with budget.transaction() as state:
            self.assertEqual(state["leases"], {})
            self.assertFalse(budget.group_alive(pgid))


if __name__ == "__main__":
    unittest.main()
