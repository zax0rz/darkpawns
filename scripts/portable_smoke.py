#!/usr/bin/env python3
"""Exercise an extracted native development bundle, using only disposable state."""
import argparse
from contextlib import closing
import asyncio
import hashlib
import json
import os
import pathlib
import re
import shutil
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.request

import websockets
from smoke_test_2b import PASSWORD, cmd, create_character, login_existing, send


def main(bundle):
    suffix = '.exe' if os.name == 'nt' else ''
    bundle = pathlib.Path(bundle).resolve()
    manifest = json.loads((bundle / 'release-assets/manifest.json').read_text())
    assert not any(name.startswith(('lib/etc/', 'lib/plrobjs/', 'lib/data/', 'lib/world/etc/')) for name in manifest['files'])
    with tempfile.TemporaryDirectory(prefix='darkpawns portable ') as temporary:
        root = pathlib.Path(temporary)
        instance = root / 'my edited world'
        unrelated = root / 'unrelated cwd'
        unrelated.mkdir()
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', 0))
            port = probe.getsockname()[1]
        command = [str(bundle / ('darkpawns' + suffix))]
        subprocess.run(command + ['init', '--instance', str(instance), '--port', str(port)], cwd=unrelated, check=True)
        original_config = (instance / 'instance.json').read_bytes()
        duplicate = subprocess.run(command + ['init', '--instance', str(instance)], cwd=unrelated, capture_output=True)
        assert duplicate.returncode != 0
        assert (instance / 'instance.json').read_bytes() == original_config
        log = open(root / 'server.log', 'w+', encoding='utf-8')

        def start(executable):
            flags = subprocess.CREATE_NEW_PROCESS_GROUP if os.name == 'nt' else 0
            process = subprocess.Popen([str(executable), 'run', '--instance', str(instance)], cwd=unrelated, stdout=log, stderr=log, creationflags=flags)
            try:
                deadline = time.monotonic() + 30
                while time.monotonic() < deadline:
                    if process.poll() is not None:
                        raise RuntimeError('Launcher exited before readiness')
                    try:
                        with urllib.request.urlopen(f'http://127.0.0.1:{port}/health', timeout=1) as response:
                            if response.status == 200:
                                return process
                    except OSError:
                        time.sleep(.1)
                raise RuntimeError('Server did not become ready')
            except BaseException:
                if process.poll() is None:
                    stop(process)
                raise

        def stop(process):
            if process.poll() is not None:
                return
            process.send_signal(signal.CTRL_BREAK_EVENT if os.name == 'nt' else signal.SIGINT)
            try:
                assert process.wait(timeout=40) == 0, 'Graceful launcher shutdown failed'
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise

        async def create_players():
            for name, level in [('Portowner', 40), ('Portplayer', 1)]:
                async with websockets.connect(f'ws://127.0.0.1:{port}/ws') as ws:
                    await send(ws, 'login', {'player_name': name, 'password': PASSWORD, 'new_char': True})
                    state = await create_character(ws)
                    assert state['player']['level'] == level, state['player']
                    await cmd(ws, 'save', wait=.5)
            await asyncio.sleep(.3)

        async def saved_login():
            async with websockets.connect(f'ws://127.0.0.1:{port}/ws') as ws:
                state = await login_existing(ws, 'Portplayer', PASSWORD)
                assert state['player']['level'] == 1

        process = None
        try:
            process = start(bundle / ('darkpawns' + suffix))
            duplicate = subprocess.run(command + ['run', '--instance', str(instance)], cwd=unrelated, capture_output=True, timeout=20)
            assert duplicate.returncode != 0 and b'already running' in duplicate.stderr
            for path in ['/', '/admin/']:
                with urllib.request.urlopen(f'http://127.0.0.1:{port}{path}') as response:
                    html = response.read().decode()
                assert '<html' in html.lower()
                if path == '/admin/':
                    for asset in re.findall(r'(?:src|href)="(/assets/[^\"]+)"', html):
                        with urllib.request.urlopen(f'http://127.0.0.1:{port}{asset}') as response:
                            assert response.status == 200 and response.read()
            asyncio.run(create_players())
            stop(process)
            process = None
            with closing(sqlite3.connect(instance / 'lib/data/darkpawns.db')) as db:
                assert db.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
                assert dict(db.execute('SELECT name,level FROM players')) == {'Portowner': 40, 'Portplayer': 1}
            # An edited world stays with the instance, even when the replacement
            # release carries a different (valid) starter seed for that same file.
            world_index = instance / 'lib/world/wld/index'
            edited = world_index.read_bytes() + b'\n'
            world_index.write_bytes(edited)
            second = root / 'replacement release'
            shutil.copytree(bundle, second)
            seed = second / 'release-assets/lib/world/wld/index'
            seed.write_bytes(seed.read_bytes() + b'\n\n')
            manifest['files']['lib/world/wld/index'] = hashlib.sha256(seed.read_bytes()).hexdigest()
            (second / 'release-assets/manifest.json').write_text(json.dumps(manifest))
            process = start(second / ('darkpawns' + suffix))
            asyncio.run(saved_login())
            stop(process)
            process = None
            assert world_index.read_bytes() == edited
            assert (instance / 'instance.json').read_bytes() == original_config
            assert not (unrelated / 'data').exists()
            assert not (bundle / 'lib').exists()
            print('PASS: isolated init, served assets, first immortal, second mortal, lock, graceful stop, edited world and saved login across release relocation')
        except BaseException:
            log.flush()
            print((root / 'server.log').read_text(encoding='utf-8')[-12000:])
            raise
        finally:
            if process is not None:
                stop(process)
            log.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    main(parser.parse_args().bundle)
