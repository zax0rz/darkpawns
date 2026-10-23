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
        def fixture_archive(path):
            # dp-release post-processes every archive in the directory, so the
            # unselected-platform fixtures must be valid bundles, not stubs.
            name = path.stem
            windows = path.suffix == '.zip'
            with tempfile.TemporaryDirectory() as temporary:
                payload = pathlib.Path(temporary) / name
                (payload / 'release-assets').mkdir(parents=True)
                (payload / 'release-assets' / 'manifest.json').write_text(json.dumps({'format': 1, 'update_format': 1, 'data_format': 1, 'version': '0.1.0', 'commit': manifest['commit'], 'files': {'web/index.html': '00'}}))
                for tool in ('darkpawns', 'darkpawns-server'):
                    executable = payload / (tool + ('.exe' if windows else ''))
                    executable.write_bytes(b'MZ' if windows else b'#!/bin/sh\nexit 0\n')
                    if not windows:
                        executable.chmod(0o755)
                if windows:
                    with zipfile.ZipFile(path, 'w', zipfile.ZIP_DEFLATED) as output:
                        for entry in payload.rglob('*'):
                            output.write(entry, entry.relative_to(pathlib.Path(temporary)))
                else:
                    with tarfile.open(path, 'w:gz') as output:
                        output.add(payload, arcname=name)
        for target in ('linux', 'darwin', 'windows'):
            for architecture in ('amd64', 'arm64'):
                path = signed / (f'darkpawns_0.1.0_{target}_{architecture}' + ('.zip' if target == 'windows' else '.tar.gz'))
                if path != archive:
                    fixture_archive(path)
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
        # L1-003: the public entry stub verifies install.sh.sig before the
        # installer runs. Drive the real stub hermetically: swap its pinned key
        # for this run's disposable publisher key, shim curl to serve the local
        # signed directory, and forward --source-dir so the installer stays
        # offline. A tampered installer must be refused without executing.
        if os.name != 'nt':
            import re
            stub_text = pathlib.Path('website/static/install.sh').read_text()
            disposable_key = '\n'.join(public.read_text().strip().splitlines())
            substituted, count = re.subn(
                r"cat > \"\$work/publisher\.pem\" <<'PUBLIC_KEY'\n.*?\nPUBLIC_KEY",
                'cat > "$work/publisher.pem" <<\'PUBLIC_KEY\'\n' + disposable_key + '\nPUBLIC_KEY',
                stub_text, flags=re.S)
            assert count == 1, 'stub publisher-key substitution failed'
            stub = root / 'entry-stub.sh'
            stub.write_text(substituted)
            curl_shim = root / 'curl-shim'
            curl_shim.mkdir()
            (curl_shim / 'curl').write_text(
                '#!/bin/sh\n'
                'url=\ndest=\nprev=\n'
                'for a in "$@"; do\n'
                '  case "$a" in http*) url=$a;; esac\n'
                '  if [ "$prev" = "-o" ]; then dest=$a; fi\n'
                '  prev=$a\n'
                'done\n'
                'case "$url" in\n'
                f'  http*install.sh.sig) cp "{signed}/install.sh.sig" "$dest"; exit 0;;\n'
                f'  http*install.sh) cp "{signed}/install.sh" "$dest"; exit 0;;\n'
                'esac\n'
                'echo "no fetchable url" >&2; exit 1\n')
            (curl_shim / 'curl').chmod(0o755)
            stub_env = dict(os.environ, PATH=f'{curl_shim}{os.pathsep}{os.environ["PATH"]}')
            stub_args = ['sh', str(stub), '--source-dir', str(signed),
                         '--install-dir', str(root / 'stubbed-install'),
                         '--bin-dir', str(root / 'stubbed-bin'), '--no-setup']
            stubbed = subprocess.run(stub_args, env=stub_env, text=True, capture_output=True, timeout=180)
            assert stubbed.returncode == 0, (stubbed.stdout, stubbed.stderr)
            marker = root / 'tampered.marker'
            installer_path = signed / 'install.sh'
            genuine = installer_path.read_bytes()
            installer_path.write_text(f'echo executed-unverified > "{marker}"\n')
            tampered = subprocess.run(stub_args, env=stub_env, text=True, capture_output=True, timeout=180)
            assert tampered.returncode != 0, 'stub ran an installer whose signature did not verify'
            assert 'verification failed' in (tampered.stdout + tampered.stderr).lower(), (tampered.stdout, tampered.stderr)
            assert not marker.exists(), 'stub executed an unverified installer'
            installer_path.write_bytes(genuine)
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
        print('PASS: signed bootstrap, tampered metadata/archive refusal before installation, entry-stub installer-signature verification before execution, native platform selection, existing release preservation, Unicode/percent command path and installed game startup')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', required=True)
    main(parser.parse_args().bundle)
