#!/usr/bin/env python3
"""Update disposable native instances only; never contacts the release feed."""
import argparse
import asyncio
import hashlib
import json
import os
import pathlib
import shutil
import socket
import struct
import sys
import subprocess
import tarfile
import tempfile
import zipfile
import time

import websockets
from smoke_test_2b import PASSWORD, cmd, create_character, login_existing, send


def main(bundle, native_service=False):
    bundle = pathlib.Path(bundle).resolve()
    suffix = '.exe' if os.name == 'nt' else ''
    launcher = bundle / ('darkpawns' + suffix)
    with tempfile.TemporaryDirectory(prefix='darkpawns update ') as temporary:
        root = pathlib.Path(temporary).resolve()
        instance = root / "owner's é %world%"
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', 0))
            port = probe.getsockname()[1]
        command = [str(launcher), '--instance', str(instance)]

        def run(*args, good=True, answer=None):
            started = time.monotonic()
            print('START:', args, flush=True)
            evidence = pathlib.Path('.portable-update-diagnostics')
            evidence.mkdir(exist_ok=True)
            try:
                result = subprocess.run(command + list(args), input=answer, text=True, encoding='utf-8', capture_output=True, timeout=100)
            except subprocess.TimeoutExpired as error:
                def decoded(value):
                    return value.decode('utf-8', 'replace') if isinstance(value, bytes) else value or ''
                print('TIMEOUT STDOUT:', decoded(error.stdout), flush=True)
                print('TIMEOUT STDERR:', decoded(error.stderr), flush=True)
                (evidence / 'timeout.stdout').write_text(decoded(error.stdout), encoding='utf-8')
                (evidence / 'timeout.stderr').write_text(decoded(error.stderr), encoding='utf-8')
                if os.name == 'nt':
                    processes = subprocess.run(['powershell.exe', '-NoProfile', '-NonInteractive', '-Command', "Get-CimInstance Win32_Process | Where-Object { $_.Name -like 'darkpawns*' } | Select-Object ProcessId,ParentProcessId,CommandLine | ConvertTo-Json"], capture_output=True, text=True, encoding='utf-8', errors='replace', timeout=15)
                    (evidence / 'processes.json').write_text(processes.stdout + processes.stderr, encoding='utf-8')
                    print('PROCESSES:', processes.stdout, flush=True)
                for name in ['.updates/transaction.json', '.updates/active.json', '.last-exit.json', 'lib/logs/launcher.log']:
                    source = instance / name
                    if source.exists():
                        target = evidence / name.replace('/', '-')
                        shutil.copyfile(source, target)
                        if source.suffix == '.json':
                            print(name, source.read_text(encoding='utf-8'), flush=True)
                raise
            print('DONE:', args[0], round(time.monotonic() - started, 2), 'seconds', 'exit', result.returncode, flush=True)
            with (evidence / 'commands.log').open('a', encoding='utf-8') as log:
                log.write(str(args) + '\n' + result.stdout + result.stderr + '\n')
            assert (result.returncode == 0) == good, (args, result.stdout, result.stderr)
            return result

        def status():
            return json.loads(run('status', '--json').stdout)

        async def player(create=False):
            async with websockets.connect(f'ws://127.0.0.1:{port}/ws') as ws:
                if create:
                    await send(ws, 'login', {'player_name': 'Updateowner', 'password': PASSWORD, 'new_char': True})
                    state = await create_character(ws)
                else:
                    state = await login_existing(ws, 'Updateowner', PASSWORD)
                assert state['player']['level'] == 40
                await cmd(ws, 'save', wait=.5)

        def archive(version, incompatible=False, broken=False):
            release = root / version
            shutil.copytree(bundle, release)
            manifest_path = release / 'release-assets/manifest.json'
            manifest = json.loads(manifest_path.read_text())
            manifest['version'] = version
            manifest['update_format'] = 1
            manifest['data_format'] = 2 if incompatible else 1
            seed = release / 'release-assets/lib/text/motd'
            assert seed.is_file()
            seed.write_text('new seed must never overwrite operator files\n')
            manifest['files']['lib/text/motd'] = hashlib.sha256(seed.read_bytes()).hexdigest()
            manifest_path.write_text(json.dumps(manifest))
            if broken:
                # A native Go executable with valid build information passes the
                # preflight but fails at the real process/readiness boundary.
                source = root / 'broken.go'
                source.write_text('package main\nimport "os"\nfunc main(){if err:=os.WriteFile("data/candidate-only",[]byte("retained failed state"),0600);err!=nil{panic(err)};os.Exit(9)}\n')
                subprocess.run(['go', 'build', '-o', str(release / ('darkpawns-server' + suffix)), str(source)], check=True)
            destination = root / (version + ('.zip' if os.name == 'nt' else '.tar.gz'))
            if os.name == 'nt':
                with zipfile.ZipFile(destination, 'w', zipfile.ZIP_DEFLATED) as output:
                    for path in release.rglob('*'):
                        output.write(path, path.relative_to(root))
            else:
                with tarfile.open(destination, 'w:gz') as output:
                    output.add(release, arcname=release.name)
            return destination, hashlib.sha256(destination.read_bytes()).hexdigest()

        installed = False
        try:
            run('setup', '--yes', '--no-browser', '--port', str(port))
            asyncio.run(player(create=True))
            config = (instance / 'instance.json').read_bytes()
            heading = b'Mon Jan  2 (Updateowner) :: preserved board\x00'
            message = b'Operator board contents\r\n\x00'
            board = struct.pack('<iiiiii', 1, 0, 0, 40, len(heading), len(message)) + heading + message
            fixtures = {
                'lib/text/motd': b'operator edited motd\n',
                'lib/world/etc/board.mort': board,
                'lib/data/notifications.json': b'{"revision":1,"pushover":{"enabled":false,"events":[]},"discord":{"enabled":false,"events":[]},"heartbeat":false,"credentials":{"discord":""}}\n',
                'lib/data/notifications.json.key': bytes(range(32)),
            }
            for name, data in fixtures.items():
                (instance / name).write_bytes(data)
            good_archive, digest = archive('poc-next')
            before = status()
            run('update', '--archive', str(good_archive), '--sha256', '0' * 64, '--yes', good=False)
            assert status()['run_id'] == before['run_id']
            run('update', '--archive', str(good_archive), '--sha256', digest, '--check')
            assert status()['run_id'] == before['run_id']
            run('update', '--archive', str(good_archive), '--sha256', digest, answer='n\n')
            assert status()['run_id'] == before['run_id']
            bad_archive, bad_digest = archive('poc-incompatible', incompatible=True)
            run('update', '--archive', str(bad_archive), '--sha256', bad_digest, '--yes', good=False)
            assert status()['run_id'] == before['run_id']
            if native_service:
                installed = True
                run('service', 'install')
            run('update', '--archive', str(good_archive), '--sha256', digest, '--yes')
            active = status()
            assert active['run_id'] != before['run_id']
            assert pathlib.Path(active['bundle']).name == 'poc-next', active
            assert (instance / 'instance.json').read_bytes() == config
            for name, data in fixtures.items():
                assert (instance / name).read_bytes() == data, name
            asyncio.run(player())
            transaction = json.loads((instance / '.updates/transaction.json').read_text())
            assert transaction['phase'] == 'committed'
            backup = instance / '.updates' / transaction['id'] / 'backup'
            for name, data in fixtures.items():
                assert (backup / name).read_bytes() == data
            run('restart')
            assert status()['bundle'] == active['bundle']
            asyncio.run(player())
            run('doctor', '--json')
            if native_service and sys.platform == 'linux':
                service = json.loads((instance / '.service.json').read_text())
                subprocess.run(['systemctl', '--user', 'stop', service['name'] + '.service'], check=True)
                assert status()['state'] == 'stopped'
                assert not json.loads((instance / '.last-exit.json').read_text()).get('error')
                run('start')
                asyncio.run(player())
            broken, broken_digest = archive('poc-broken', broken=True)
            failed = run('update', '--archive', str(broken), '--sha256', broken_digest, '--yes', good=False)
            assert 'previous release restored' in failed.stderr, failed
            assert status()['bundle'] == active['bundle']
            transaction = json.loads((instance / '.updates/transaction.json').read_text())
            assert transaction['phase'] == 'rolled-back'
            history = instance / '.updates' / transaction['id']
            assert (history / 'failed-lib/data/candidate-only').read_text() == 'retained failed state'
            assert not (instance / 'lib/data/candidate-only').exists()
            assert (instance / 'instance.json').read_bytes() == config
            for name, data in fixtures.items():
                assert (instance / name).read_bytes() == data, name
            asyncio.run(player())
            if installed:
                run('service', 'remove')
                installed = False
            run('stop')
            # Updating a stopped world keeps it stopped, including after dispatch
            # through the original launcher's stable entry point.
            stopped_archive, stopped_digest = archive('poc-stopped')
            run('update', '--archive', str(stopped_archive), '--sha256', stopped_digest, '--yes')
            assert status()['state'] == 'stopped'
            noop = run('update', '--archive', str(stopped_archive), '--sha256', stopped_digest, '--yes')
            assert 'Already up to date' in noop.stdout
            assert status()['state'] == 'stopped'
            run('start')
            asyncio.run(player())
            run('stop')
            run('doctor', '--json')
            print('PASS: digest/compatibility refusal before downtime, check/cancel, backup, active-release dispatch, preserved secret/world/boards/credentials, saved login, native registration across update, failed-start snapshot recovery and stopped-instance update')
        finally:
            if (instance / 'instance.json').exists():
                run('stop')
                if installed:
                    run('service', 'remove')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    parser.add_argument('--native-service', action='store_true')
    options = parser.parse_args()
    main(options.bundle, options.native_service)
