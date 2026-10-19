#!/usr/bin/env python3
"""Curate a portable release from committed starter assets and freshly built UI.

Build tooling only; operators do not need Python, Git, Go, or Node.
"""
import argparse
import hashlib
import io
import json
import pathlib
import shutil
import subprocess
import tarfile
import tempfile

WORLD_PARTS = ('wld', 'mob', 'obj', 'zon', 'shp', 'movesel', 'scripts')
SEEDS = [f'lib/world/{part}' for part in WORLD_PARTS] + [
    'lib/text', 'lib/misc/help', 'lib/misc/messages', 'lib/misc/socials', 'web/public',
]


def stage(root, version):
    root = pathlib.Path(root)
    destination = root / 'release-assets'
    if destination.exists():
        if destination.is_symlink() or not (destination / 'manifest.json').is_file():
            raise RuntimeError('Refusing to replace an unrecognized release-assets directory')
        old = json.loads((destination / 'manifest.json').read_text())
        if old.get('format') != 1:
            raise RuntimeError('Refusing to replace an unrecognized release-assets directory')
    commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
    source = subprocess.check_output(['git', 'archive', '--format=tar', 'HEAD', *SEEDS], cwd=root)
    with tempfile.TemporaryDirectory(prefix='.release-stage-', dir=root) as temporary:
        staged = pathlib.Path(temporary) / 'release-assets'
        staged.mkdir()
        with tarfile.open(fileobj=io.BytesIO(source)) as archive:
            for member in archive:
                if member.isdir():
                    continue
                if not member.isfile():
                    raise RuntimeError(f'Non-regular starter asset: {member.name}')
                path = pathlib.PurePosixPath(member.name)
                if path.is_absolute() or '..' in path.parts:
                    raise RuntimeError('Invalid starter asset path')
                name = member.name
                if name.startswith('web/public/'):
                    name = 'web/' + name.removeprefix('web/public/')
                target = staged / name
                target.parent.mkdir(parents=True, exist_ok=True)
                with archive.extractfile(member) as content:
                    target.write_bytes(content.read())
        ui = root / 'admin-ui/dist'
        if not (ui / 'index.html').is_file():
            raise RuntimeError('Build admin-ui before staging the portable release')
        for asset in ui.rglob('*'):
            if asset.is_symlink():
                raise RuntimeError('Symlink in admin build output')
            if asset.is_file():
                target = staged / 'admin-ui' / asset.relative_to(ui)
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(asset, target)
        files = {
            p.relative_to(staged).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in sorted(staged.rglob('*')) if p.is_file()
        }
        manifest = {'format': 1, 'version': version, 'commit': commit, 'files': files}
        (staged / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
        if destination.exists():
            shutil.rmtree(destination)
        shutil.move(staged, destination)
    return manifest


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', required=True)
    arguments = parser.parse_args()
    result = stage(pathlib.Path(__file__).resolve().parent.parent, arguments.version)
    print(f"Staged {len(result['files'])} committed starter/browser assets; no player or runtime data.")
