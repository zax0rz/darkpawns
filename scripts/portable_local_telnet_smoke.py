#!/usr/bin/env python3
"""Native local listener journey against a disposable packaged release."""
import argparse
import json
import os
import pathlib
import socket
import subprocess
import tempfile
import time
import urllib.request


def main(bundle):
    bundle = pathlib.Path(bundle).resolve()
    launcher = bundle / ('darkpawns.exe' if os.name == 'nt' else 'darkpawns')
    with tempfile.TemporaryDirectory(prefix='darkpawns local telnet ') as temporary:
        instance = pathlib.Path(temporary) / 'world'
        probes = [socket.socket(), socket.socket()]
        for probe in probes:
            probe.bind(('127.0.0.1', 0))
        web_port, telnet_port = [probe.getsockname()[1] for probe in probes]
        for probe in probes:
            probe.close()
        command = [str(launcher), '--instance', str(instance)]

        def run(*args):
            result = subprocess.run(command + list(args), text=True, capture_output=True, timeout=70)
            assert result.returncode == 0, (args, result.stdout, result.stderr)
            return result

        def connections():
            deadline = time.monotonic() + 15
            while True:
                with urllib.request.urlopen(f'http://127.0.0.1:{web_port}/connection-info', timeout=3) as response:
                    info = json.load(response)
                if info.get('telnet') or time.monotonic() >= deadline:
                    return info
                time.sleep(.1)

        run('init', '--port', str(web_port), '--telnet-port', str(telnet_port))
        config_path = instance / 'instance.json'
        config = json.loads(config_path.read_text())
        assert config['telnet_bind'] == '127.0.0.1'
        assert config['telnet_port'] == telnet_port
        assert config['greeting_overlay'] is True
        try:
            run('start')
            assert connections() == {'telnet': {'bind': '127.0.0.1', 'port': telnet_port}}
            with socket.create_connection(('127.0.0.1', telnet_port), timeout=5) as client:
                client.settimeout(10)
                greeting = b''
                while b'By what name' not in greeting:
                    chunk = client.recv(8192)
                    assert chunk, greeting
                    greeting += chunk
                assert b'CircleMUD' in greeting and b'DikuMUD' in greeting
                version = json.loads((bundle / 'release-assets/manifest.json').read_text())['version']
                assert f'Dark Pawns MUD v{version} - https://darkpawns.org'.encode() in greeting
                assert b'As of 10-17-2008' not in greeting
            with urllib.request.urlopen(f'http://127.0.0.1:{web_port}/login', timeout=3) as response:
                assert response.status == 200
                assert b'<html' in response.read().lower()
            # Disabled means no splash hint, even after a previously enabled run.
            run('stop')
            config['telnet_port'] = 0
            config_path.write_text(json.dumps(config))
            run('start')
            with urllib.request.urlopen(f'http://127.0.0.1:{web_port}/connection-info', timeout=3) as response:
                assert json.load(response) == {}
            try:
                client = socket.create_connection(('127.0.0.1', telnet_port), timeout=1)
            except OSError:
                pass
            else:
                client.close()
                raise AssertionError('disabled listener still accepts connections')
            assert json.loads(config_path.read_text()) == config
            print('PASS: selected loopback telnet port, real greeting and credits, observed connection metadata, direct login route, disabled listener and unchanged configuration')
        finally:
            run('stop')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    main(parser.parse_args().bundle)
