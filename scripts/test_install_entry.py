"""Exercise the stable bootstrap entry without a network or real installation.

The entry stub pins a publisher key, downloads the installer and its
signature, and executes the installer only after the signature verifies
(L1-003). These tests drive that contract hermetically: a disposable
publisher key is substituted into a copy of the stub, a curl shim serves
local files, and both the refusal and the success path are asserted.
"""
import os
import pathlib
import re
import shutil
import subprocess
import tempfile
import unittest

STUB = pathlib.Path(__file__).resolve().parent.parent / 'website/static/install.sh'


def disposable_publisher(root):
    private, public = root / 'private.pem', root / 'public.pem'
    subprocess.run(['openssl', 'genpkey', '-algorithm', 'RSA', '-pkeyopt', 'rsa_keygen_bits:3072', '-out', str(private)], check=True, capture_output=True)
    subprocess.run(['openssl', 'pkey', '-in', str(private), '-pubout', '-out', str(public)], check=True, capture_output=True)
    return private, public


def key_substituted_stub(root, public):
    disposable = '\n'.join(public.read_text().strip().splitlines())
    substituted, count = re.subn(
        r"cat > \"\$work/publisher\.pem\" <<'PUBLIC_KEY'\n.*?\nPUBLIC_KEY",
        'cat > "$work/publisher.pem" <<\'PUBLIC_KEY\'\n' + disposable + '\nPUBLIC_KEY',
        STUB.read_text(), flags=re.S)
    assert count == 1, 'publisher-key substitution failed'
    path = root / 'entry.sh'
    path.write_text(substituted)
    return path


def curl_shim(root, serve, status=0):
    bins = root / 'bin'
    bins.mkdir(exist_ok=True)
    curl = bins / 'curl'
    curl.write_text(
        '#!/bin/sh\n'
        'url=\ndestination=\nprev=\n'
        'for a in "$@"; do\n'
        '  case "$a" in http*) url=$a;; esac\n'
        '  if [ "$prev" = "-o" ]; then destination=$a; fi\n'
        '  prev=$a\n'
        'done\n'
        'printf \'%s\\n\' "$url" >> "$ENTRY_ROOT/urls"\n'
        f'if [ "$ENTRY_CURL_STATUS" != 0 ]; then exit "$ENTRY_CURL_STATUS"; fi\n'
        'case "$url" in\n'
        f'  *install.sh.sig) cat "{serve}/install.sh.sig" > "$destination"; exit 0;;\n'
        f'  *install.sh) cat "{serve}/install.sh" > "$destination"; exit 0;;\n'
        'esac\n'
        'exit 22\n')
    curl.chmod(0o755)
    return bins


@unittest.skipUnless(os.name != 'nt' and shutil.which('sh') and shutil.which('openssl'), 'POSIX bootstrap entry')
class InstallEntryTest(unittest.TestCase):
    def test_unverified_installer_never_runs(self):
        with tempfile.TemporaryDirectory(prefix='darkpawns entry ') as temporary:
            root = pathlib.Path(temporary)
            private, public = disposable_publisher(root)
            entry = key_substituted_stub(root, public)
            serve = root / 'serve'
            serve.mkdir()
            # A plausible-looking installer with no valid signature.
            (serve / 'install.sh').write_text('printf "%s" "$1" > "$ENTRY_ROOT/executed"\nexit 23\n')
            (serve / 'install.sh.sig').write_bytes(b'not a signature')
            environment = dict(os.environ, PATH=f'{curl_shim(root, serve)}{os.pathsep}{os.environ["PATH"]}', ENTRY_ROOT=str(root), ENTRY_CURL_STATUS='0')
            result = subprocess.run(['sh', str(entry), 'argument with spaces'], env=environment, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse((root / 'executed').exists())
            urls = (root / 'urls').read_text().splitlines()
            self.assertIn('https://github.com/zax0rz/darkpawns/releases/latest/download/install.sh', urls)
            self.assertTrue(any(url.endswith('install.sh.sig') for url in urls), urls)

    def test_verified_installer_runs_with_arguments_and_status(self):
        with tempfile.TemporaryDirectory(prefix='darkpawns entry ') as temporary:
            root = pathlib.Path(temporary)
            private, public = disposable_publisher(root)
            entry = key_substituted_stub(root, public)
            serve = root / 'serve'
            serve.mkdir()
            installer = serve / 'install.sh'
            installer.write_text('printf "%s" "$1" > "$ENTRY_ROOT/executed"\nexit 23\n')
            subprocess.run(['openssl', 'dgst', '-sha256', '-sign', str(private), '-out', str(serve / 'install.sh.sig'), str(installer)], check=True, capture_output=True)
            environment = dict(os.environ, PATH=f'{curl_shim(root, serve)}{os.pathsep}{os.environ["PATH"]}', ENTRY_ROOT=str(root), ENTRY_CURL_STATUS='0')
            result = subprocess.run(['sh', str(entry), 'argument with spaces'], env=environment, capture_output=True)
            self.assertEqual(result.returncode, 23)
            self.assertEqual((root / 'executed').read_text(), 'argument with spaces')

    def test_download_failure_propagates_without_running(self):
        with tempfile.TemporaryDirectory(prefix='darkpawns entry ') as temporary:
            root = pathlib.Path(temporary)
            private, public = disposable_publisher(root)
            entry = key_substituted_stub(root, public)
            serve = root / 'serve'
            serve.mkdir()
            (serve / 'install.sh').write_text('printf "%s" "$1" > "$ENTRY_ROOT/executed"\nexit 23\n')
            (serve / 'install.sh.sig').write_bytes(b'x')
            environment = dict(os.environ, PATH=f'{curl_shim(root, serve)}{os.pathsep}{os.environ["PATH"]}', ENTRY_ROOT=str(root), ENTRY_CURL_STATUS='22')
            result = subprocess.run(['sh', str(entry), 'argument with spaces'], env=environment, capture_output=True)
            self.assertEqual(result.returncode, 22)
            self.assertFalse((root / 'executed').exists())


if __name__ == '__main__':
    unittest.main()
