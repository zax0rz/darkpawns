#!/usr/bin/env python3
"""Replay assertion-only 0/1/0 tooling mutations in the supplied checkout."""
from pathlib import Path
import subprocess
import sys

repo = Path(sys.argv[1]).resolve()
logs = Path(sys.argv[2]).resolve()
logs.mkdir(parents=True, exist_ok=True)
source = repo / 'scripts/combined_census.py'
original = source.read_text()
mutations = [
 ('deduplicate', "return {(1, name) for name in full} | set(claims)", "return [(1, name) for name in full] + list(claims)", 'test_union'),
 ('global-limit', 'ThreadPoolExecutor(max_workers=jobs)', 'ThreadPoolExecutor(max_workers=jobs * 2)', 'test_global_pool'),
 ('slowest-first', '(-estimate(pair), -costs[pair[1]], pair[1], pair[0])', '(estimate(pair), -costs[pair[1]], pair[1], pair[0])', 'test_order'),
 ('verdict', 'if any(kind not in GOOD for kind, _ in final.values()):', 'if False:', 'test_rechecks_only_infrastructure_once'),
 ('retry-content', 'initial[pair][0] in {"INFRA", "TIMEOUT"}', 'initial[pair][0] in {"INFRA", "TIMEOUT", "STALE"}', 'test_rechecks_only_infrastructure_once'),
 ('projection', 'checks = {}', 'checks = dict(checks)', 'test_projection_and_independent_recheck'),
 ('seed-isolation', 'directory = run / f"seed{seed}"', 'directory = run / "seed1"', 'test_worker_retention_and_seed_isolation'),
]
rows=[]
try:
 for name, before, after, test in mutations:
  assert before in original, name
  statuses=[]
  for stage in ('baseline','reverted','restored'):
   source.write_text(original.replace(before, after) if stage=='reverted' else original)
   # Avoid stale bytecode after rapid same-size edits.
   for cache in (repo/'scripts/__pycache__').glob('combined_census.*.pyc'):
    cache.unlink()
   p=subprocess.run([sys.executable,'-m','unittest',f'test_combined_census.CombinedTest.{test}'],cwd=repo/'scripts',text=True,capture_output=True)
   output=p.stdout+p.stderr
   (logs/f'{name}-{stage}.log').write_text(output)
   statuses.append(p.returncode)
   if stage=='reverted':
    assert 'AssertionError' in output and 'FAIL:' in output and 'ERROR:' not in output, output
  assert statuses==[0,1,0], (name,statuses)
  rows.append(f'{name}\t{test}\t0\t1\t0\n')
finally:
 source.write_text(original)
 for cache in (repo/'scripts/__pycache__').glob('combined_census.*.pyc'):
  cache.unlink()
(logs/'revert-triples.tsv').write_text('mutation\ttest\tbefore\treverted\trestored\n'+''.join(rows))
print(f'{len(rows)} assertion-only triples passed')
