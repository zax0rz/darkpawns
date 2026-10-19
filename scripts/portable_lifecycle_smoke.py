#!/usr/bin/env python3
"""Verify native background ownership, saved login, doctor and optional startup jobs."""
import argparse
import asyncio
from contextlib import closing
import hashlib
import json
import os
import pathlib
import socket
import signal
import sys
import sqlite3
import subprocess
import tempfile
import urllib.error
import urllib.request

import websockets
from smoke_test_2b import PASSWORD, cmd, create_character, login_existing, send


def main(bundle, native_service=False, shutdown_failure=False):
    bundle = pathlib.Path(bundle).resolve()
    launcher = bundle / ('darkpawns.exe' if os.name == 'nt' else 'darkpawns')
    with tempfile.TemporaryDirectory(prefix='darkpawns lifecycle ') as temporary:
        instance = pathlib.Path(temporary).resolve() / "my world's é %data%"
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', 0))
            port = probe.getsockname()[1]
        command = [str(launcher), '--instance', str(instance)]

        def run(*args, good=True, timeout=60):
            result = subprocess.run(command + list(args), text=True, encoding='utf-8', capture_output=True, timeout=timeout)
            if good:
                assert result.returncode == 0, (args, result.stdout, result.stderr)
            else:
                assert result.returncode != 0, (args, result.stdout, result.stderr)
            return result

        def status():
            return json.loads(run('status', '--json').stdout)

        async def player(create=False):
            async with websockets.connect(f'ws://127.0.0.1:{port}/ws') as ws:
                if create:
                    await send(ws, 'login', {'player_name': 'Lifeowner', 'password': PASSWORD, 'new_char': True})
                    state = await create_character(ws)
                else:
                    state = await login_existing(ws, 'Lifeowner', PASSWORD)
                assert state['player']['level'] == 40
                await cmd(ws, 'save', wait=.5)

        def recovery(operation, folder=instance):
            def quote(value):
                value = str(value)
                return "'" + value.replace("'", "''" if os.name == 'nt' else "'\"'\"'") + "'"
            return ('& ' if os.name == 'nt' else '') + quote(launcher) + ' ' + operation + ' --instance ' + quote(folder)

        installed = False
        try:
            # Force failure before touching the native manager, after the world
            # was created. Recovery commands must target this custom instance.
            failed_registration = instance.parent / "failed startup world's é"
            no_home = os.environ.copy()
            no_home.pop('USERPROFILE' if os.name == 'nt' else 'HOME', None)
            failed = subprocess.run([str(launcher), 'setup', '--yes', '--autostart', '--no-browser', '--instance', str(failed_registration), '--port', str(port)], env=no_home, text=True, encoding='utf-8', capture_output=True, timeout=30)
            assert failed.returncode != 0 and (failed_registration / 'instance.json').exists(), failed
            assert recovery('service remove', failed_registration) in failed.stderr, failed.stderr
            assert recovery('run', failed_registration) in failed.stderr, failed.stderr

            # Default setup returns after readiness while the server survives it.
            run('setup', '--yes', '--no-browser', '--port', str(port))
            cfg = (instance / 'instance.json').read_bytes()
            assert status()['state'] == 'running'
            first_record = (instance / '.control.json').read_bytes()
            record = json.loads(first_record)
            for token, origin in [('', ''), ('wrong', ''), (record['token'], 'https://example.com')]:
                request = urllib.request.Request('http://' + record['address'] + '/stop', method='POST', headers={'Authorization': 'Bearer ' + token})
                if origin:
                    request.add_header('Origin', origin)
                try:
                    urllib.request.urlopen(request)
                    raise AssertionError('Unauthorized control request accepted')
                except urllib.error.HTTPError as error:
                    assert error.code == 403
            assert status()['state'] == 'running'
            # Starting twice is idempotent; it must not create another owner.
            run('start')
            assert (instance / '.control.json').read_bytes() == first_record
            asyncio.run(player(create=True))
            run('doctor', '--json')
            run('restart')
            assert (instance / '.control.json').read_bytes() != first_record
            asyncio.run(player())
            run('stop')
            assert status()['state'] == 'stopped'
            assert not (instance / '.control.json').exists()
            run('stop')
            with socket.socket() as occupied:
                occupied.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
                occupied.bind(('127.0.0.1', port))
                occupied.listen()
                failed = run('start', good=False)
                expected = recovery('doctor')
                assert expected in failed.stderr, failed.stderr
                # Execute the printed shell command with the launcher absent
                # from PATH, including apostrophes/Unicode in the instance path.
                clean_environment = os.environ.copy()
                clean_environment['PATH'] = str(instance.parent / 'empty-path')
                shell = [str(pathlib.Path(os.environ['SystemRoot']) / 'System32/WindowsPowerShell/v1.0/powershell.exe'), '-NoProfile', '-NonInteractive', '-Command'] if os.name == 'nt' else ['/bin/sh', '-c']
                checked = subprocess.run(shell + [expected], env=clean_environment, text=True, encoding='utf-8', capture_output=True, timeout=30)
                assert checked.returncode != 0 and 'Web port' in checked.stdout, checked
            checks = json.loads(run('doctor', '--json').stdout)
            assert any(c['name'] == 'SQLite' and c['ok'] and 'Integrity' in c['detail'] for c in checks)
            with closing(sqlite3.connect(instance / 'lib/data/darkpawns.db')) as database:
                assert database.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
                assert database.execute('SELECT level FROM players WHERE name="Lifeowner"').fetchone()[0] == 40
            if native_service:
                installed = True  # remove the owned partial registration on failure too
                run('service', 'install')
                run('service', 'install', good=False)
                run('start')
                asyncio.run(player())
                run('restart')
                asyncio.run(player())
                run('service', 'remove')
                installed = False
                assert not (instance / '.service.json').exists()
                assert status()['state'] == 'stopped'
                run('start')  # returns to standalone background mode
                asyncio.run(player())
                run('stop')
            # A corrupt database is reported, never repaired or replaced.
            database_path = instance / 'lib/data/darkpawns.db'
            original = database_path.read_bytes()
            database_path.write_bytes(b'not a sqlite database')
            damaged = hashlib.sha256(database_path.read_bytes()).hexdigest()
            result = run('doctor', '--json', good=False)
            assert any(c['name'] == 'SQLite' and not c['ok'] for c in json.loads(result.stdout))
            assert hashlib.sha256(database_path.read_bytes()).hexdigest() == damaged
            database_path.write_bytes(original)
            assert (instance / 'instance.json').read_bytes() == cfg
            if shutdown_failure:
                assert sys.platform == 'linux', 'stalled-child proof uses Linux /proc and SIGSTOP'
                run('start')
                owner = status()
                children = set()
                for task in pathlib.Path(f"/proc/{owner['pid']}/task").glob('*/children'):
                    try:
                        children.update(task.read_text().split())
                    except FileNotFoundError:
                        pass
                assert len(children) == 1, children
                child = int(next(iter(children)))
                argv = pathlib.Path(f'/proc/{child}/cmdline').read_bytes().split(b'\0')
                assert str(instance / 'lib/world').encode() in argv, argv
                os.kill(child, signal.SIGSTOP)
                failed = run('stop', good=False)
                assert 'server did not stop within 30 seconds' in failed.stderr, failed.stderr
                receipt = json.loads((instance / '.last-exit.json').read_text())
                assert receipt['run_id'] == owner['run_id'] and receipt['error']
                assert status()['state'] == 'stopped'
                checks = json.loads(run('doctor', '--json', good=False).stdout)
                assert any(c['name'] == 'Last shutdown' and not c['ok'] for c in checks)
                print('PASS: stalled child shutdown fails explicitly and doctor reports the matching run result')
            print('PASS: setup survives parent exit, authenticated ownership, duplicate start, graceful stop/restart, saved immortal login, read-only doctor' + (', native startup install/start/restart/remove' if native_service else ''))
        finally:
            try:
                if (instance / 'instance.json').exists():
                    run('stop')
                if installed:
                    run('service', 'remove')
            except BaseException:
                log = instance / 'lib/logs/launcher.log'
                if log.exists():
                    print(log.read_text(errors='replace')[-12000:])
                raise


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    parser.add_argument('--native-service', action='store_true')
    parser.add_argument('--shutdown-failure', action='store_true')
    options = parser.parse_args()
    main(options.bundle, options.native_service, options.shutdown_failure)
