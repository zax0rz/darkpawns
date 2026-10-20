#!/usr/bin/env python3
"""Linux hosting proof using disposable ports, TLS identity and instance state."""
import argparse
import hashlib
import json
import os
import pathlib
import signal
import socket
import ssl
import subprocess
import tempfile
import time


def main(bundle):
    bundle = pathlib.Path(bundle).resolve()
    with tempfile.TemporaryDirectory(prefix='darkpawns hosting ') as temporary:
        root = pathlib.Path(temporary)
        instance = root / 'private world'
        public = root / 'public mudlet'
        public.mkdir()
        (public / 'keep.txt').write_text('unrelated public file')
        ports = []
        probes = []
        for _ in range(4):
            probe = socket.socket()
            probe.bind(('127.0.0.1', 0))
            ports.append(probe.getsockname()[1])
            probes.append(probe)
        for probe in probes:
            probe.close()
        launcher = bundle / 'darkpawns'
        command = [str(launcher), '--instance', str(instance)]
        def run(*args, good=True):
            deadline = time.monotonic() + 5
            while True:
                result = subprocess.run(command + list(args), capture_output=True, text=True, timeout=100)
                if 'another lifecycle operation is in progress' not in result.stderr or time.monotonic() >= deadline:
                    break
                time.sleep(.1)  # initial readiness can precede completion of Start's poll
            assert (result.returncode == 0) == good, (args, result.stdout, result.stderr)
            return result
        def status():
            return json.loads(run('status', '--json').stdout)
        def wait(predicate, message, seconds=40):
            deadline = time.monotonic() + seconds
            while time.monotonic() < deadline:
                try:
                    if predicate():
                        return
                except AssertionError:
                    pass  # a crash/start can briefly own the lock before control is ready
                time.sleep(.2)
            raise AssertionError(message)
        cert, key = root / 'localhost.crt', root / 'localhost.key'
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-keyout', str(key), '-out', str(cert), '-subj', '/CN=localhost', '-days', '1'], check=True, capture_output=True)
        run('init', '--port', str(ports[0]))
        cfg_path = instance / 'instance.json'
        cfg = json.loads(cfg_path.read_text())
        original_secret = cfg['jwt_secret']
        cfg.update(telnet_port=ports[1], telnet_tls_port=ports[2], whod_port=ports[3], mudlet_directory=str(public), server_env={
            'TELNET_TLS_CERT_FILE': str(cert), 'TELNET_TLS_KEY_FILE': str(key),
            'DP_MUDLET_PACKAGE_URL': 'https://example.invalid/darkpawns.xml',
            'DP_MUDLET_MAP_URL': 'https://example.invalid/darkpawns-map.xml',
        })
        cfg_path.write_text(json.dumps(cfg))
        log = (root / 'supervision.log').open('w+')
        monitor = subprocess.Popen(command + ['supervise'], stdout=log, stderr=log)
        try:
            wait(lambda: status()['state'] == 'running', 'supervised world did not start')
            assert monitor.poll() is None
            # A second manager cannot race this monitor for the same instance.
            duplicate = run('supervise', good=False)
            assert 'another system supervisor' in duplicate.stderr
            run('service', 'install', good=False)
            for port in ports[1:]:
                raw = socket.create_connection(('127.0.0.1', port), timeout=10)
                if port == ports[2]:
                    raw = ssl.create_default_context(cafile=str(cert)).wrap_socket(raw, server_hostname='localhost')
                with raw:
                    raw.settimeout(10)
                    data = b''
                    while b'By what name' not in data and b'A list of active players' not in data:
                        chunk = raw.recv(8192)
                        if not chunk:
                            break
                        data += chunk
                    assert b'By what name' in data or b'A list of active players' in data, (port, data)
            assert hashlib.sha256((public / 'darkpawns.xml').read_bytes()).digest() == hashlib.sha256((bundle / 'release-assets/mudlet/darkpawns.xml').read_bytes()).digest()
            assert (public / 'keep.txt').read_text() == 'unrelated public file'
            run('stop')
            time.sleep(2.2)
            assert status()['state'] == 'stopped', 'monitor undid an intentional stop'
            run('start')
            owner = status()
            # Kill only the disposable game child, after checking its exact world
            # argument. The monitor must recover with a different run identity.
            children = set()
            for task in pathlib.Path(f"/proc/{owner['pid']}/task").glob('*/children'):
                children.update(task.read_text().split())
            assert len(children) == 1, children
            child = int(next(iter(children)))
            assert str(instance / 'lib/world').encode() in pathlib.Path(f'/proc/{child}/cmdline').read_bytes().split(b'\0')
            os.kill(child, signal.SIGKILL)
            def restarted():
                current = status()
                return current['state'] == 'running' and bool(current.get('run_id')) and current['run_id'] != owner['run_id']
            wait(restarted, 'monitor did not restart a crashed game')
            assert monitor.poll() is None
            assert json.loads(cfg_path.read_text())['jwt_secret'] == original_secret
            run('doctor', '--json')
            print('PASS: explicit telnet/TLS/WHOD, verified TLS identity, matching public Mudlet package, singleton system monitor, intentional stop, crash restart and stable JWT')
        except BaseException:
            log.flush()
            print((root / 'supervision.log').read_text()[-12000:])
            launcher_log = instance / 'lib/logs/launcher.log'
            if launcher_log.exists():
                print(launcher_log.read_text()[-12000:])
            raise
        finally:
            monitor.send_signal(signal.SIGINT)
            try:
                assert monitor.wait(timeout=85) == 0
                assert status()['state'] == 'stopped'
            finally:
                if monitor.poll() is None:
                    monitor.kill()
                    monitor.wait()
                run('stop')
                log.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    main(parser.parse_args().bundle)
