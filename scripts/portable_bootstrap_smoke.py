#!/usr/bin/env python3
"""Use a disposable publisher key and local signed release; no public publishing."""
import argparse
import hashlib
import json
import os
import pathlib
import platform
import shutil
import subprocess
import tarfile
import tempfile
import urllib.request
import zipfile


def main(bundle):
    bundle = pathlib.Path(bundle).resolve()
    system = {'Linux': 'linux', 'Darwin': 'darwin', 'Windows': 'windows'}[platform.system()]
    arch = 'arm64' if platform.machine().lower() in ('arm64', 'aarch64') else 'amd64'
    suffix = '.exe' if os.name == 'nt' else ''
    with tempfile.TemporaryDirectory(prefix='darkpawns bootstrap ') as temporary:
        root = pathlib.Path(temporary).resolve()
        public, private = root / 'public.pem', root / 'private.pem'
        subprocess.run(['go', 'run', './cmd/dp-release', '--generate-key', str(private), '--public', str(public)], check=True)
        signed = root / 'signed'
        signed.mkdir()
        name = f'darkpawns_0.1.0_{system}_{arch}'
        payload = root / name
        shutil.copytree(bundle, payload)
        manifest_path = payload / 'release-assets/manifest.json'
        manifest = json.loads(manifest_path.read_text())
        manifest['version'] = '0.1.0'
        manifest_path.write_text(json.dumps(manifest))
        extension = '.zip' if os.name == 'nt' else '.tar.gz'
        archive = signed / (name + extension)
        if os.name == 'nt':
            with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED) as output:
                for path in payload.rglob('*'):
                    output.write(path, path.relative_to(root))
        else:
            with tarfile.open(archive, 'w:gz') as output:
                output.add(payload, arcname=name)
        for target in ('linux', 'darwin', 'windows'):
            for architecture in ('amd64', 'arm64'):
                path = signed / (f'darkpawns_0.1.0_{target}_{architecture}' + ('.zip' if target == 'windows' else '.tar.gz'))
                if path != archive:
                    path.write_bytes(b'disposable unselected platform fixture')
        foreign = subprocess.run(['go', 'run', './cmd/dp-release', '--dir', str(signed), '--tag', 'v0.1.0', '--commit', manifest['commit'], '--key', str(private)], text=True, capture_output=True)
        assert foreign.returncode != 0 and 'does not match the pinned public key' in foreign.stderr
        subprocess.run(['go', 'run', './cmd/dp-release', '--dir', str(signed), '--tag', 'v0.1.0', '--commit', manifest['commit'], '--key', str(private), '--public', str(public)], check=True)
        install = root / "installation's é %release%"
        bins = root / "command's é %bin%"

        def invoke(good=True, extra=''):
            if os.name == 'nt':
                command = ['powershell.exe', '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', str(signed / 'install.ps1'), '-SourceDir', str(signed), '-InstallDir', str(install), '-BinDir', str(bins), '-NoSetup', '-NoPath']
            else:
                command = ['sh', str(signed / 'install.sh'), '--source-dir', str(signed), '--install-dir', str(install), '--bin-dir', str(bins), '--no-setup']
            result = subprocess.run(command, text=True, encoding='utf-8', errors='replace', capture_output=True, timeout=180)
            assert (result.returncode == 0) == good, (result.stdout, result.stderr, extra)
            return result

        metadata = signed / 'release-manifest.txt'
        original = metadata.read_bytes()
        metadata.write_bytes(original + b'tampered\n')
        invoke(good=False, extra='changed signed metadata')
        assert not install.exists()
        metadata.write_bytes(original)
        original_archive = archive.read_bytes()
        archive.write_bytes(original_archive + b'tampered')
        invoke(good=False, extra='changed archive')
        assert not install.exists()
        archive.write_bytes(original_archive)
        invoke()
        installed = install / name
        launcher = installed / ('darkpawns' + suffix)
        before = hashlib.sha256(launcher.read_bytes()).hexdigest()
        invoke(good=False, extra='existing release must not be merged or overwritten')
        assert hashlib.sha256(launcher.read_bytes()).hexdigest() == before
        if os.name == 'nt':
            shim = bins / 'darkpawns.cmd'
            environment = os.environ.copy()
            environment['PATH'] = str(bins) + os.pathsep + environment['PATH']
            result = subprocess.run([os.environ.get('COMSPEC', 'cmd.exe'), '/d', '/c', 'darkpawns', '--version'], env=environment, capture_output=True, timeout=20)
            assert result.returncode == 0, (result.stdout, result.stderr)
        else:
            subprocess.run([str(bins / 'darkpawns'), '--version'], check=True, capture_output=True)
        # Exercise the installed binary, not the source checkout or staging copy.
        instance = root / 'world'
        import socket
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', 0))
            port = probe.getsockname()[1]
        command = [str(launcher), '--instance', str(instance)]
        subprocess.run(command + ['init', '--port', str(port), '--telnet-port', '0'], check=True, capture_output=True)
        try:
            subprocess.run(command + ['start'], check=True, capture_output=True, timeout=60)
            assert urllib.request.urlopen(f'http://127.0.0.1:{port}/health', timeout=3).status == 200
        finally:
            subprocess.run(command + ['stop'], check=True, capture_output=True, timeout=60)
        print('PASS: signed bootstrap, tampered metadata/archive refusal before installation, native platform selection, existing release preservation, Unicode/percent command path and installed game startup')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    main(parser.parse_args().bundle)
