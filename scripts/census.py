#!/usr/bin/env python3
"""Register immutable census inputs, launch detached, report one run-bound line."""
from pathlib import Path
import argparse
from datetime import datetime
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import time

import census_budget as budget

REPO = Path(__file__).resolve().parent.parent


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def git(*args):
    return subprocess.check_output(['git', '-C', str(REPO), *args])


def snapshot(run):
    head = git('rev-parse', 'HEAD')
    status = git('status', '--porcelain')
    (run/'go-head.txt').write_bytes(head)
    (run/'git-status.txt').write_bytes(status)
    source = run/'source'
    source.mkdir()
    files = sorted(set(git('ls-files', '-c', '-o', '--exclude-standard', '-z').decode().split('\0'))-{''})
    hashes = {}
    for name in files:
        src = REPO/name
        if not src.exists() and not src.is_symlink():
            continue  # A tracked deletion belongs to this dirty snapshot.
        dst = source/name
        dst.parent.mkdir(parents=True, exist_ok=True)
        if src.is_symlink():
            target = os.readlink(src)
            if os.path.isabs(target) or not (src.parent/target).resolve().is_relative_to(REPO):
                raise RuntimeError(f'snapshot symlink escapes checkout: {name}')
            dst.symlink_to(target)
            hashes[name] = hashlib.sha256(target.encode()).hexdigest()
        else:
            shutil.copy2(src, dst)
            hashes[name] = digest(dst)
    if head != git('rev-parse', 'HEAD') or status != git('status', '--porcelain'):
        raise RuntimeError('checkout changed during census snapshot; register again under a new name')
    for name, expected in hashes.items():
        src = REPO/name
        actual = hashlib.sha256(os.readlink(src).encode()).hexdigest() if src.is_symlink() else digest(src)
        if actual != expected:
            raise RuntimeError(f'checkout changed during snapshot: {name}')
    (run/'source-hashes.json').write_text(json.dumps(hashes, sort_keys=True))
    for path in sorted(source.rglob('*'), reverse=True):
        if not path.is_symlink():
            path.chmod(path.stat().st_mode & ~0o222)
    source.chmod(0o555)
    return source


def start(args):
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]{0,127}', args.name):
        raise RuntimeError('run name must use letters, digits, dots, underscores or hyphens')
    if not 1 <= args.jobs <= 36:
        raise RuntimeError('--jobs must be between 1 and the approved per-run maximum 36')
    if args.release and not args.combined:
        raise RuntimeError('--release requires --combined')
    if sum([args.combined, args.claims, bool(args.scenarios)]) > 1:
        raise RuntimeError('--combined, --claims and --scenarios are mutually exclusive')
    seed = os.environ.get('ORACLE_REGRESSION_SEED', '1')
    if not seed.isdigit() or int(seed) < 1 or (args.combined and seed != '1'):
        raise RuntimeError('positive seed required; --combined requires seed 1')
    oracle = Path(os.environ.get('DP_ORACLE_BIN', '/home/zach/darkpawns-c-oracle/bin/circle')).resolve()
    if not os.access(oracle, os.X_OK):
        raise RuntimeError(f'oracle binary is not executable: {oracle}')
    oracle_sha = digest(oracle)
    nonreference = oracle_sha != (REPO/'cmd/dp-oracle-diff/reference-oracle.sha256').read_text().strip()
    if nonreference and os.environ.get('CENSUS_ALLOW_NONREFERENCE') != '1':
        print(f'census: oracle sha256 {oracle_sha} is not the reference', file=sys.stderr)
        return 6
    names = args.scenarios.split(',') if args.scenarios else []
    for name in names:
        if not name or '/' in name or not (REPO/f'cmd/dp-oracle-diff/scenarios/{name}.txt').is_file():
            raise RuntimeError(f'scenario not found: {name}')
    if len(names) != len(set(names)):
        raise RuntimeError('duplicate scenario in selection')
    rt = budget.runtime()
    with budget.legacy_guard(allow_busy=True) as guard:
        run = Path(os.environ.get('ORACLE_RUNS_ROOT', str(Path.home()/'Archives/darkpawns/oracle-runs'))).resolve()/datetime.now().strftime('%Y-%m-%d')/args.name
        if run.is_relative_to(REPO):
            raise RuntimeError('census evidence root must be outside the checkout')
        run.parent.mkdir(parents=True, exist_ok=True)
        run.mkdir()  # Atomic identity reservation; never reuse even an empty run.
        child = None
        try:
            source = snapshot(run)
            if args.claims or args.combined:
                command = [sys.executable, str(source/'scripts/manifest_claims.py')]
                if os.environ.get('CENSUS_MANIFEST_DIR'):
                    fixture = Path(os.environ['CENSUS_MANIFEST_DIR']).resolve()
                    frozen = run/'manifest-input'
                    shutil.copytree(fixture, frozen)
                    command += ['--manifest-dir', str(frozen), '--root', str(frozen)]
                with (run/'claims.tsv').open('w') as out:
                    subprocess.run(command, check=True, stdout=out)
                if not (run/'claims.tsv').stat().st_size:
                    raise RuntimeError('no oracle claims')
                shutil.copy2(run/'go-head.txt', run/'manifests-head.txt')
            count = len(names) if names else len(list((source/'cmd/dp-oracle-diff/scenarios').glob('*.txt')))
            if args.claims:
                count = len((run/'claims.tsv').read_text().splitlines())
            if args.combined:
                sys.path.insert(0, str(source/'scripts'))
                from combined_census import enumerate_pairs
                full = {p.stem for p in (source/'cmd/dp-oracle-diff/scenarios').glob('*.txt')}
                claims = {(int(s), n) for s,n,_ in (l.split('\t') for l in (run/'claims.tsv').read_text().splitlines())}
                count = len(enumerate_pairs(full, claims))
            (run/'total').write_text(f'{count}\n')
            jobs = min(args.jobs, len(names)) if names else args.jobs
            runner = source/'scripts/oracle_regression.sh'
            if os.environ.get('CENSUS_RUNNER'):
                runner = run/'custom-runner'
                shutil.copy2(os.environ['CENSUS_RUNNER'], runner)
            started = datetime.now().astimezone().isoformat(timespec='seconds')
            record = dict(status='queued', pid=0, birth='', submitted=time.time(), started=started,
                          jobs=jobs, release=args.release, exclusive=args.exclusive,
                          oracle=str(oracle), oracle_sha=oracle_sha, source=str(source),
                          budget=budget.LIMIT, memory_floor_kib=budget.FLOOR_KIB, coordinator_version=budget.VERSION,
                          seed=seed, draw_parity=args.draw_parity, selection=args.scenarios or ("combined" if args.combined else "claims" if args.claims else "full"),
                          runner_sha=digest(runner),
                          command=sys.argv, goflags=os.environ.get('GOFLAGS',''))
            key = str(run)
            with budget.transaction() as state:
                state['runs'][key] = record
                candidate = dict(run=key, kind='build')
                immediate = (not guard.busy and not state['waiting'] and budget.eligible(state,candidate)
                             and sum(l['kind']=='worker' for l in state['leases'].values()) < budget.LIMIT)
                if args.no_queue and not immediate:
                    del state['runs'][key]
                    budget.atomic(run/'state','refused\n')
                    print(f'census refused: {run} budget full or release draining eta=unknown')
                    return 5
                env = dict(os.environ, CENSUS_RUN_DIR=key, CENSUS_RUNNER=str(runner), CENSUS_REPO_ROOT=str(source),
                           CENSUS_ORACLE_BIN=str(oracle), DP_ORACLE_BIN=str(oracle), CENSUS_ORACLE_SHA=oracle_sha,
                           CENSUS_ALLOW_NOTE=' (NOT THE REFERENCE ORACLE)' if nonreference else '',
                           CENSUS_SCENARIOS=args.scenarios or '', CENSUS_JOBS=str(jobs), CENSUS_SEED=seed,
                           CENSUS_CLAIMS=str(int(args.claims)), CENSUS_COMBINED=str(int(args.combined)),
                           CENSUS_DRAW_PARITY=str(int(args.draw_parity)), CENSUS_CUSTOM_RUNNER=str(int(bool(os.environ.get('CENSUS_RUNNER')))))
                with (run/'census.log').open('ab') as log:
                    child = subprocess.Popen(['bash', str(source/'scripts/census_runner.sh')], env=env, cwd=source,
                                             stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT,
                                             start_new_session=True, pass_fds=guard.fds)
                record.update(pid=child.pid, birth=budget.identity(child.pid))
                (run/'pid').write_text(f'{child.pid}\n')
                budget.atomic(run/'state', f'queued pid={child.pid} started={started}\n')
                (run/'admission.json').write_text(json.dumps(record, indent=2))
                budget.event(key,'registered',budget=budget.LIMIT,jobs=jobs,release=args.release)
                budget.atomic(rt/'last',key+'\n')
                active = sum(l['kind']=='worker' for l in state['leases'].values())
                position = len(budget.active_runs(state))
            head = (run/'go-head.txt').read_text().strip()[:9]
            word = 'started' if immediate else 'queued'
            print(f'census {word}: {run} @{head} position={position} active={active}/72 requested={jobs} eta=unknown')
            return 0
        except Exception:
            if child is not None:
                budget.signal_group(child.pid, 15)
            budget.atomic(run/'state','admission failed\n')
            raise


def resolve(requested):
    if requested:
        return Path(requested).resolve()
    with budget.transaction() as state:
        active = list(budget.active_runs(state))
        if len(active)>1:
            raise RuntimeError('multiple unfinished censuses; use --run <directory>')
        if active:
            return Path(active[0])
    pointer = budget.runtime()/'last'
    return Path(pointer.read_text().strip()) if pointer.exists() else None


def report(run):
    if not run or not run.is_dir():
        return 'no census', 4
    head = (run/'go-head.txt').read_text().strip()[:9] if (run/'go-head.txt').exists() else 'unknown'
    label = f'{run.name} @{head}: '
    with budget.transaction() as state:
        record = state['runs'].get(str(run))
        completed = (run/'exit-code').exists() and (run/'MANIFEST.md').exists() and (run/'summary.txt').exists()
        if completed and (record is None or record['status']=='done'):
            lines = (run/'summary.txt').read_text().splitlines()
            return label+' | '.join(lines), 0 if int((run/'exit-code').read_text()) == 0 else 1
        if not record or record['status'] in budget.TERMINAL:
            return label+f'died {run}', 4
        active = sum(l['kind']=='worker' and l['run']==str(run) for l in state['leases'].values())
        reason = 'memory' if state['memory_paused'] else 'release' if budget.release_owner(state) not in {None,str(run)} else 'capacity'
        return label+f"{record['status']} active={active}/{record['jobs']} budget=72 waiting={reason} eta=unknown", 3


def main():
    parser = argparse.ArgumentParser(prog='census.sh')
    commands = parser.add_subparsers(dest='action', required=True)
    p = commands.add_parser('start')
    p.add_argument('--name',required=True)
    p.add_argument('--scenarios')
    p.add_argument('--jobs',type=int,default=36)
    for flag in ['claims','combined','draw-parity','release','no-queue','exclusive']:
        p.add_argument('--'+flag,action='store_true')
    for action in ['wait','status','cancel']:
        p = commands.add_parser(action)
        p.add_argument('--run')
        p.add_argument('--name')
        p.add_argument('--max-seconds',type=float,default=540 if action=='wait' else 0)
    args = parser.parse_args()
    if args.action == 'start':
        return start(args)
    run = resolve(args.run)
    if args.name and (not run or run.name != args.name):
        raise RuntimeError(f'refusing: selected census is not {args.name}')
    if args.action == 'cancel':
        if not args.run:
            raise RuntimeError('cancel requires an explicit --run')
        budget.cancel(str(run))
        print(f'census cancelled: {run}')
        return 0
    if args.max_seconds < 0:
        raise RuntimeError('--max-seconds must be nonnegative')
    deadline = time.monotonic()+args.max_seconds
    while True:
        line, status = report(run)
        if status != 3 or time.monotonic() >= deadline:
            print(line)
            return status
        time.sleep(min(float(os.environ.get('CENSUS_POLL_SECONDS','.5')),max(0,deadline-time.monotonic())))


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (RuntimeError,OSError,subprocess.CalledProcessError) as exc:
        print(f'census: {exc}',file=sys.stderr)
        sys.exit(4 if sys.argv[1:2] in [['wait'],['status'],['cancel']] else 2)
