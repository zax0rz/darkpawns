#!/usr/bin/env python3
"""Fail-capable combined gate contracts, with real worker retention smoke test."""
import os
import shutil
import subprocess
from pathlib import Path
import tempfile
import threading
import time
import unittest
from unittest.mock import patch

import combined_census as gate


class CombinedTest(unittest.TestCase):
    def test_union(self):
        self.assertEqual(gate.enumerate_pairs({'a', 'b'}, {(1, 'a'), (2, 'a')}),
                         {(1, 'a'), (1, 'b'), (2, 'a')})

    def test_global_pool(self):
        lock = threading.Lock()
        active = maximum = 0
        calls = []
        def worker(pair):
            nonlocal active, maximum
            with lock:
                active += 1
                maximum = max(maximum, active)
                calls.append(pair)
            time.sleep(.02)
            with lock:
                active -= 1
            return ('PASS', '1')
        pairs = [(seed, name) for seed in (1, 2, 3) for name in ('a', 'b')]
        rows = gate.execute_pool(pairs, 2, worker, lambda *_: None)
        self.assertEqual(set(rows), set(pairs))
        self.assertCountEqual(calls, pairs)
        self.assertEqual(maximum, 2)

    def test_order(self):
        pairs = {(1, 'a'), (2, 'a'), (1, 'b')}
        self.assertEqual(gate.order_pairs(pairs, {'a': 1, 'b': 100}, {(1, 'a'): 80, (2, 'a'): 90, (1, 'b'): 2}),
                         [(2, 'a'), (1, 'a'), (1, 'b')])

    def test_frozen_timings(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'timings.tsv'
            path.write_text('1\ta\t55\n2\ta\t44\n')
            with patch.dict(os.environ, CENSUS_TIMINGS_FILE=str(path)):
                self.assertEqual(gate.prior_timings(Path(tmp)/'runs/date/name'), {(1, 'a'): 55, (2, 'a'): 44})
                path.write_text('1\ta\tbad\n')
                with self.assertRaises(ValueError):
                    gate.prior_timings(Path(tmp)/'runs/date/name')

    def test_cost(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp)/'scenario.txt'
            p.write_text('# skip\n[setup:oracle:peer]\n[setup:port:peer]\nlook\n<RELOGIN>\n<CRASH>\n')
            self.assertEqual(gate.scenario_cost(p), 26)

    def test_projection_and_independent_recheck(self):
        with tempfile.TemporaryDirectory() as tmp:
            run = Path(tmp)
            full = {'a', 'b', 'c'}
            claims = {(1, 'a'), (2, 'a')}
            initial = {(1, 'a'): ('INFRA', '4'), (1, 'b'): ('FAIL', '5'),
                       (1, 'c'): ('PASS', '6'), (2, 'a'): ('PASS', '7')}
            repeats = {(1, 'a'): ('PASS', '8')}
            gate.project(run, full, claims, initial, repeats, 10)
            self.assertEqual((run/'full-verdict').read_text(), 'NOT_CLEAN\n')
            self.assertEqual((run/'claims-verdict').read_text(), 'CLEAN_AFTER_RECHECK\n')
            self.assertEqual((run/'recheck-results.tsv').read_text(), '')
            self.assertEqual((run/'claims-recheck-results.tsv').read_text(), 'PASS\ta\t1\t8\n')
            self.assertEqual((run/'seed1/results.tsv').read_text(), 'INFRA\ta\t4\n')
            self.assertEqual((run/'claims-results.tsv').read_text(), 'INFRA\ta\t1\t4\nPASS\ta\t2\t7\n')
            self.assertEqual((run/'results.tsv').read_text(), 'INFRA\ta\t4\nFAIL\tb\t5\nPASS\tc\t6\n')

    def test_rechecks_only_infrastructure_once(self):
        pairs = [(1, 'a'), (1, 'b'), (2, 'a'), (2, 'b'), (2, 'c')]
        rows = dict(zip(pairs, [('TIMEOUT', '1'), ('FAIL', '1'), ('INFRA', '1'), ('STALE', '1'), ('UNPINNABLE', '1')]))
        self.assertEqual(gate.recheck_pairs(pairs, rows, {'a', 'b'}, {(2, 'a'), (2, 'b'), (2, 'c')}), [(2, 'a')])
        self.assertEqual(gate.verdict({(1, 'a'): ('TIMEOUT', '1')}, {(1, 'a'): ('TIMEOUT', '1')}), 'NOT_CLEAN')

    def test_census_cli(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp)/'repo'; (repo/'scripts').mkdir(parents=True)
            source = Path(__file__).resolve().parent
            for name in ('census.sh', 'census_runner.sh', 'combined_census.py', 'claims_census.py', 'manifest_claims.py', 'fidelity_manifest.py'):
                shutil.copy2(source/name, repo/'scripts'/name)
            scenarios = repo/'cmd/dp-oracle-diff/scenarios'; scenarios.mkdir(parents=True)
            for name in ('alpha', 'beta'):
                (scenarios/f'{name}.txt').write_text('look\n')
            manifests = repo/'manifests'; manifests.mkdir()
            (manifests/'test.tsv').write_text('handler\tcommand\tcase_id\tdepth\tscope\tstatus\tproof\tc_site\tnotes\nh\tc\tA\tD1\ta\toracle-green-multiseed\talpha@1,2\tsrc/x:1\t\n')
            stub = repo/'go'; stub.write_text('#!/bin/sh\nexit 0\n'); stub.chmod(0o755)
            worker = repo/'scripts/oracle_regression_worker.sh'
            worker.write_text('''#!/bin/sh
name=${1%.txt}
printf '%s %s\\n' "$seed" "$name" >> "$CENSUS_CALLS"
printf 'PASS\\t%s\\t1\\n' "$name" > "$result_dir/$name"
printf 'PASS %s\\n' "$name"
''')
            worker.chmod(0o755)
            rt = repo/'rt'; rt.mkdir()
            env = dict(os.environ, XDG_RUNTIME_DIR=str(rt), ORACLE_RUNS_ROOT=str(repo/'runs'),
                       DP_ORACLE_BIN=str(stub), ORACLE_REGRESSION_GO=str(stub), CENSUS_ALLOW_NONREFERENCE='1',
                       CENSUS_MANIFEST_DIR=str(manifests), CENSUS_POLL_SECONDS='.05', CENSUS_CALLS=str(repo/'calls'))
            script = str(repo/'scripts/census.sh')
            start = subprocess.run([script, 'start', '--combined', '--name', 'test', '--jobs', '2'], env=env, capture_output=True, text=True)
            self.assertEqual(start.returncode, 0, start.stderr)
            wait = subprocess.run([script, 'wait', '--max-seconds', '10'], env=env, capture_output=True, text=True)
            self.assertEqual(wait.returncode, 0, wait.stdout + wait.stderr)
            self.assertIn('pairs=3 deduplicated=1 full=CLEAN claims=CLEAN', wait.stdout)
            self.assertCountEqual((repo/'calls').read_text().splitlines(), ['1 alpha', '1 beta', '2 alpha'])
            run = next((repo/'runs').glob('*/*'))
            self.assertEqual((run/'full-verdict').read_text(), 'CLEAN\n')
            self.assertEqual((run/'claims-verdict').read_text(), 'CLEAN\n')
            self.assertIn('Combined gate', (run/'MANIFEST.md').read_text())
            bad = subprocess.run([script, 'start', '--combined', '--claims', '--name', 'bad'], env=env, capture_output=True)
            self.assertEqual(bad.returncode, 2)

    def test_fail_closed_results(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp)/'results'
            self.assertEqual(gate.read_results(p, ['a']), {'a': ('FAIL', '0')})
            for text in ('PASS\ta\t1\nPASS\ta\t2\n', 'PASS\tother\t1\n', 'BOGUS\ta\t1\n', 'PASS\ta\tbad\n'):
                p.write_text(text)
                with self.assertRaises(ValueError):
                    gate.read_results(p, ['a'])

    def test_worker_retention_and_seed_isolation(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp)/'repo'; run = Path(tmp)/'run'; bins = run/'binaries'
            (repo/'scripts').mkdir(parents=True); bins.mkdir(parents=True)
            worker = repo/'scripts/oracle_regression_worker.sh'
            worker.write_text('''#!/bin/sh
name=${1%.txt}
printf 'PASS\\t%s\\t1\\n' "$name" > "$result_dir/$name"
printf '%s\\n' "$seed" > "$log_dir/$name.attempt1.log"
printf '%s\\n' "$seed" > "$ORACLE_REGRESSION_DUMP/$name.txt"
printf 'PASS %s\\n' "$name"
''')
            worker.chmod(0o755)
            with patch.dict(os.environ, CENSUS_ORACLE_BIN='/oracle'):
                for seed in (1, 2):
                    self.assertEqual(gate.run_worker(repo, run, bins, (seed, 'a')), ('PASS', '1'))
                    self.assertTrue((run/f'seed{seed}/initial-logs/attempt.combined/a.attempt1.log').exists(), 'each seed must retain its own attempt')
                    self.assertEqual((run/f'seed{seed}/initial-logs/attempt.combined/a.attempt1.log').read_text(), f'{seed}\n')
                    self.assertEqual((run/f'seed{seed}/initial-dump/a.txt').read_text(), f'{seed}\n')
                    self.assertEqual((run/f'seed{seed}/initial-logs/attempt.combined/a.worker.log').read_text(), 'PASS a\n')


if __name__ == '__main__':
    unittest.main()
