#!/usr/bin/env python3
"""Host-wide census admission. One transactional registry, process-held leases.

No daemon: every registration/acquisition reconciles crashed clients. The registry
records process start ticks and child groups; a heartbeat never frees live work.
"""
from contextlib import contextmanager
from pathlib import Path
import ctypes
import fcntl
import json
import os
import signal
import subprocess
import sys
import time
import uuid

VERSION = 1
LIMIT = 72
FLOOR_KIB = 4 * 1024 * 1024
RESUME_KIB = FLOOR_KIB + 512 * 1024
BUILD_RESERVE_KIB = 1024 * 1024
TERMINAL = {"done", "died", "cancelled"}


def testing():
    if os.environ.get('CENSUS_TESTING') != '1':
        return False
    if not os.environ.get('CENSUS_TEST_RUNTIME'):
        raise RuntimeError('test mode requires an explicit isolated CENSUS_TEST_RUNTIME')
    return True


def runtime():
    # Explicit test fixtures cannot silently follow XDG_RUNTIME_DIR or TMPDIR.
    if testing():
        path = Path(os.environ["CENSUS_TEST_RUNTIME"])
    else:
        parent = Path(f"/run/user/{os.getuid()}")
        path = parent / "dp-census" if parent.is_dir() else Path(f"/tmp/dp-census-{os.getuid()}")
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    if path.stat().st_uid != os.getuid() or path.stat().st_mode & 0o077:
        raise RuntimeError(f"unsafe census runtime directory: {path}")
    return path


def atomic(path, value):
    path = Path(path)
    tmp = path.with_name(path.name + f".{os.getpid()}.tmp")
    tmp.write_text(value)
    tmp.replace(path)


def identity(pid):
    try:
        fields = Path(f"/proc/{pid}/stat").read_text().rsplit(")", 1)[1].split()
        return fields[19] if fields[0] != "Z" else None
    except (OSError, IndexError):
        return None


def alive(pid, birth):
    return bool(birth) and identity(pid) == birth


def group_alive(pgid, birth=None):
    if birth and identity(pgid) not in {None,birth}:
        return False  # A reused group leader cannot belong to the old lease.
    if not pgid:
        return False
    for path in Path('/proc').glob('[0-9]*/stat'):
        try:
            f = path.read_text().rsplit(')', 1)[1].split()
            if int(f[2]) == pgid and f[0] != 'Z':
                return True
        except (OSError, ValueError, IndexError):
            continue
    return False


def signal_group(pgid, sig, birth=None):
    if birth and identity(pgid) not in {None,birth}:
        return
    if pgid:
        try:
            os.killpg(pgid, sig)
        except ProcessLookupError:
            pass


def event(run, action, **extra):
    with (Path(run) / 'admission-events.jsonl').open('a') as out:
        out.write(json.dumps(dict(time=time.time(), event=action, **extra)) + '\n')


def available_kib():
    if testing() and os.environ.get('CENSUS_TEST_MEMORY'):
        return int(Path(os.environ['CENSUS_TEST_MEMORY']).read_text())
    for line in Path('/proc/meminfo').read_text().splitlines():
        if line.startswith('MemAvailable:'):
            return int(line.split()[1])
    raise RuntimeError('MemAvailable is unavailable; refusing admission')


def reconcile(state):
    now = time.time()
    for key, run in state['runs'].items():
        if run['status'] not in TERMINAL and not alive(run['pid'], run['birth']) and (Path(key)/'exit-code').exists() and (Path(key)/'MANIFEST.md').exists():
            run['status'] = 'done'
        if run['status'] not in TERMINAL and not alive(run['pid'], run['birth']):
            run['status'] = 'died'
            atomic(Path(key) / 'state', 'died\n')
            event(key, 'runner-died', pid=run['pid'])
        if run['status'] in {'died','cancelled'} and identity(run['pid']) in {None,run['birth']}:
            first = run.setdefault('stopping', now)
            signal_group(run['pid'], signal.SIGKILL if now-first >= 1 else signal.SIGTERM)
    for token, lease in list(state['leases'].items()):
        run = state['runs'][lease['run']]
        dead = not alive(lease['pid'], lease['birth'])
        if run['status'] in TERMINAL or dead:
            pgid = lease.get('pgid', 0)
            if group_alive(pgid, lease.get('child_birth')):
                first = lease.setdefault('stopping', now)
                signal_group(pgid, signal.SIGKILL if now-first >= 1 else signal.SIGTERM, lease.get('child_birth'))
                # Never reclaim a permit while any engine descendant is alive.
                continue
            del state['leases'][token]
            event(lease['run'], 'reaped', token=token, kind=lease['kind'])
    state['waiting'] = [p for p in state['waiting']
                        if alive(p['pid'], p['birth']) and state['runs'][p['run']]['status'] not in TERMINAL]


@contextmanager
def transaction():
    rt = runtime()
    with (rt/'registry.lock').open('a+') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        path = rt/'registry.json'
        state = json.loads(path.read_text()) if path.exists() else dict(version=VERSION, policy=dict(limit=LIMIT,floor=FLOOR_KIB), runs={}, leases={}, waiting=[], cursor='', memory_paused=False)
        if state['version'] != VERSION or state.get('policy') != dict(limit=LIMIT,floor=FLOOR_KIB):
            raise RuntimeError('incompatible census registry; drain before upgrading')
        reconcile(state)
        try:
            yield state
        finally:
            atomic(path, json.dumps(state))


def active_runs(state):
    return {k:v for k,v in state['runs'].items() if v['status'] not in TERMINAL}


def release_owner(state):
    barriers = [(0 if v['release'] else 1, v['submitted'], k) for k,v in active_runs(state).items() if v['release'] or v['exclusive']]
    return min(barriers)[2] if barriers else None


def eligible(state, request):
    run = state['runs'][request['run']]
    leases = list(state['leases'].values())
    owner = release_owner(state)
    if owner:
        if request['run'] != owner:
            return False
        if any(l['run'] != owner for l in leases):
            return False  # Non-preemptive drain, including in-flight builds.
    exclusive = [l for l in leases if state['runs'][l['run']]['exclusive']]
    if exclusive and any(l['run'] != request['run'] for l in exclusive):
        return False
    if run['exclusive'] and any(l['run'] != request['run'] for l in leases):
        return False
    mem = available_kib()
    if state['memory_paused']:
        if mem < RESUME_KIB:
            return False
        state['memory_paused'] = False
        event(request['run'], 'memory-resumed', available_kib=mem)
    if mem < FLOOR_KIB:
        state['memory_paused'] = True
        event(request['run'], 'memory-paused', available_kib=mem)
        return False
    if request['kind'] == 'build':
        return (not any(l['kind'] == 'build' for l in leases)
                and sum(l['kind']=='worker' for l in leases) < LIMIT
                and mem >= FLOOR_KIB + BUILD_RESERVE_KIB + run['jobs']*16*1024)
    workers = [l for l in leases if l['kind'] == 'worker']
    # Account for granted workers which have not materialized their boot RSS yet.
    warming = sum(time.time()-l['granted'] < 2 for l in workers)
    return (len(workers) < LIMIT and sum(l['run'] == request['run'] for l in workers) < run['jobs']
            and mem - warming*160*1024 - (BUILD_RESERVE_KIB if any(l['kind']=='build' for l in leases) else 0) >= FLOOR_KIB)


def selected(state):
    candidates = [p for p in state['waiting'] if eligible(state, p)]
    if not candidates:
        return None
    owners = list(dict.fromkeys(p['run'] for p in candidates))
    if state['cursor'] in owners:
        at = owners.index(state['cursor']) + 1
        owners = owners[at:] + owners[:at]
    return next(p for p in candidates if p['run'] == owners[0])['token']


def acquire(run, kind):
    request = dict(token=uuid.uuid4().hex, run=run, kind=kind, pid=os.getpid(), birth=identity(os.getpid()), submitted=time.time())
    with transaction() as state:
        if run not in state['runs'] or state['runs'][run]['status'] in TERMINAL:
            raise RuntimeError('census is not registered/live')
        state['waiting'].append(request)
        event(run, 'waiting', token=request['token'], kind=kind)
    while True:
        with transaction() as state:
            if state['runs'][run]['status'] in TERMINAL:
                raise RuntimeError('census cancelled or runner died')
            if selected(state) == request['token']:
                # Recheck oracle identity before any new build/worker, including retries.
                record = state['runs'][run]
                import hashlib
                if hashlib.sha256(Path(record['oracle']).read_bytes()).hexdigest() != record['oracle_sha']:
                    raise RuntimeError('reference oracle changed since registration')
                state['waiting'] = [p for p in state['waiting'] if p['token'] != request['token']]
                state['leases'][request['token']] = request | dict(pgid=0, granted=time.time())
                state['cursor'] = run
                if record['status'] == 'queued':
                    record['status'] = 'running'
                    atomic(Path(run)/'state', f"running pid={record['pid']} started={record['started']}\n")
                event(run, 'granted', token=request['token'], kind=kind,
                      active=sum(l['kind']=='worker' for l in state['leases'].values()), budget=LIMIT)
                return request['token']
        time.sleep(.05)


def drain_group(pgid, child=None, birth=None):
    signal_group(pgid, signal.SIGTERM, birth)
    deadline = time.monotonic()+1
    while group_alive(pgid, birth) and time.monotonic() < deadline:
        time.sleep(.02)
    if group_alive(pgid, birth):
        signal_group(pgid, signal.SIGKILL, birth)
    if child:
        child.wait()
    # This wrapper is a subreaper: consume reparented descendants too.
    while True:
        try:
            pid, _ = os.waitpid(-1, os.WNOHANG)
            if pid == 0:
                if group_alive(pgid, birth):
                    time.sleep(.02)
                    continue
                break
        except ChildProcessError:
            break


@contextmanager
def legacy_guard(wait=False, allow_busy=False):
    # Older checkouts use XDG_RUNTIME_DIR or /tmp. Guard both standard paths
    # so an unset XDG variable cannot evade transition compatibility.
    rt = runtime()
    paths = [rt/'legacy.lock'] if testing() else [Path('/tmp/dp-census.lock')]
    parent = Path(f'/run/user/{os.getuid()}')
    if not testing() and parent.is_dir():
        paths.append(parent/'dp-census.lock')
    files = []
    try:
        for path in paths:
            guard = path.open('a+')
            files.append(guard)
            try:
                fcntl.flock(guard, fcntl.LOCK_SH | (0 if wait else fcntl.LOCK_NB))
            except BlockingIOError:
                if allow_busy:
                    for held in reversed(files): held.close()
                    files = []
                    from types import SimpleNamespace
                    yield SimpleNamespace(fds=(),busy=True)
                    return
                raise RuntimeError('legacy exclusive census is still running; wait for it to finish')
        from types import SimpleNamespace
        yield SimpleNamespace(fds=tuple(f.fileno() for f in files),busy=False)
    finally:
        for guard in reversed(files):
            guard.close()


def execute(kind, command):
    # Queue behind an older exclusive census before consuming any admission.
    with legacy_guard(wait=True):
        return execute_admitted(kind, command)


def execute_admitted(kind, command):
    run = os.environ['CENSUS_RUN_DIR']
    token = acquire(run, kind)
    child = None
    child_birth = None
    # Reparent engine grandchildren here if a worker shell is killed.
    if ctypes.CDLL(None).prctl(36, 1, 0, 0, 0) != 0:
        raise RuntimeError('could not enable worker subreaper')
    def interrupted(signum, frame):
        raise InterruptedError(f'signal {signum}')
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    try:
        with legacy_guard(wait=True):
            read_fd, write_fd = os.pipe()
            # Child cannot exec until its group has been persisted. Parent death
            # before publication closes the pipe; no engine can be orphaned there.
            child = subprocess.Popen([sys.executable, __file__, 'child', str(read_fd), *command],
                                     start_new_session=True, pass_fds=(read_fd,), env=dict(os.environ, CENSUS_LEASED='1' if kind=='worker' else '0'))
            os.close(read_fd)
            try:
                with transaction() as state:
                    child_birth = identity(child.pid)
                    state['leases'][token].update(pgid=child.pid,child_birth=child_birth)
                os.write(write_fd, b'1')
            finally:
                os.close(write_fd)
            status = child.wait()
            drain_group(child.pid, child, child_birth)
            return status
    finally:
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        signal.signal(signal.SIGINT, signal.SIG_IGN)
        if child:
            drain_group(child.pid, child, child_birth)
        with transaction() as state:
            state['leases'].pop(token, None)
            state['waiting'] = [p for p in state['waiting'] if p['token'] != token]
            event(run, 'released', token=token, kind=kind)


def cancel(run):
    with transaction() as state:
        record = state['runs'].get(run)
        if not record or record['status'] in TERMINAL:
            return
        record['status'] = 'cancelled'
        for lease in state['leases'].values():
            if lease['run'] == run:
                signal_group(lease.get('pgid', 0), signal.SIGTERM, lease.get('child_birth'))
        if alive(record['pid'], record['birth']):
            try:
                os.kill(record['pid'], signal.SIGTERM)
            except ProcessLookupError:
                pass
        event(run, 'cancelled')


def main():
    action = sys.argv[1]
    if action == 'child':
        fd = int(sys.argv[2])
        ready = os.read(fd, 1)
        os.close(fd)
        if ready != b'1':
            return 125
        os.execvpe(sys.argv[3], sys.argv[3:], os.environ)
    if action == 'exec':
        return execute(sys.argv[2], sys.argv[4:])
    run = os.environ.get('CENSUS_RUN_DIR', '')
    if action == 'finish':
        with transaction() as state:
            if run in state['runs']:
                state['runs'][run]['status'] = 'done'
                event(run, 'finished')
    elif action == 'cancel':
        cancel(sys.argv[2] if len(sys.argv)>2 else run)
    else:
        raise ValueError(action)
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (RuntimeError, InterruptedError, OSError) as exc:
        print(f'census-budget: {exc}', file=sys.stderr)
        sys.exit(125)
