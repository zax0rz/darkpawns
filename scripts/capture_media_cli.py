#!/usr/bin/env python3
"""Record deterministic local portable-CLI walkthroughs with VHS.

All release and world state is created under /tmp/darkpawns-docs-demo. The update
capture uses a local, digest-pinned archive with --check; it never applies it or
contacts the published release feed.
"""
from __future__ import annotations

import hashlib
import json
import shutil
import socket
import subprocess
import sys
import tarfile
import time
import gzip
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WORK = Path("/tmp/darkpawns-docs-demo")
MARKER = WORK / ".owned-by-capture_media_cli"
CAPTURE_PORT = 4371
CAPTURE_TAPES = ("install-and-setup.tape", "lifecycle-commands.tape", "update-check.tape")
OUTPUTS = [
    ROOT / "docs/images/generated/install-and-setup.gif",
    ROOT / "docs/images/generated/lifecycle-commands.gif",
    ROOT / "docs/images/generated/update-check.gif",
]

SEED_DIRS = [
    "lib/world/wld", "lib/world/mob", "lib/world/obj", "lib/world/zon",
    "lib/world/shp", "lib/world/movesel", "lib/world/scripts", "lib/text",
    "lib/misc/help", "lib/misc/messages", "lib/misc/socials", "web/public",
]


def run(args: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None) -> None:
    subprocess.run(args, cwd=cwd, env=env, check=True)


def ensure_capture_port_free() -> None:
    with socket.socket() as sock:
        try:
            sock.bind(("127.0.0.1", CAPTURE_PORT))
        except OSError as exc:
            raise RuntimeError(f"capture port {CAPTURE_PORT} is already in use") from exc


def render_tape(name: str, digest: str | None = None) -> Path:
    source = ROOT / "scripts/vhs" / name
    rendered = WORK / name
    text = source.read_text()
    if "{{SHA256}}" in text:
        if digest is None:
            raise RuntimeError(f"{name} requires an archive digest")
        text = text.replace("{{SHA256}}", digest)
    rendered.write_text(text)
    return rendered


def copy_tree(source: Path, destination: Path) -> None:
    if not source.exists():
        raise RuntimeError(f"missing capture input: {source}")
    if source.is_dir():
        shutil.copytree(source, destination, dirs_exist_ok=True)
        return
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, destination)


def write_deterministic_archive(source: Path, archive: Path, arcname: str) -> None:
    """Write stable tar/gzip metadata and path ordering for visible hashes."""
    with archive.open("wb") as raw:
        with gzip.GzipFile(fileobj=raw, filename="", mode="wb", mtime=0, compresslevel=9) as compressed:
            with tarfile.open(fileobj=compressed, mode="w") as output:
                paths = [source, *sorted(source.rglob("*"), key=lambda path: path.relative_to(source).as_posix())]
                for path in paths:
                    relative = Path(arcname) / path.relative_to(source)
                    output.add(path, arcname=relative.as_posix(), recursive=False, filter=normalize_tarinfo)


def prepare_bundle() -> tuple[Path, Path, str]:
    """Build a minimal real portable bundle plus a local update-check archive."""
    release = WORK / "build-release"
    assets = release / "release-assets"
    release.mkdir(parents=True)
    assets.mkdir()

    # The current admin UI is part of the portable release. Build only when the
    # shared media workflow has not already produced it.
    ui = ROOT / "admin-ui" / "dist"
    if not (ui / "index.html").is_file():
        run(["npm", "run", "build"], cwd=ROOT / "admin-ui")
    if not (ui / "index.html").is_file():
        raise RuntimeError("admin-ui build did not create dist/index.html")

    # This key is deliberately a public test fixture. Never use it for a real
    # release; a fixed key keeps the visible archive SHA stable between runs.
    private_key = ROOT / "scripts/fixtures/docs-capture-test-private.pem"
    public_key = ROOT / "scripts/fixtures/docs-capture-test-public.pem"
    if not private_key.is_file() or not public_key.is_file():
        raise RuntimeError("missing documented docs-capture-only signing key fixture")
    overlay = WORK / "go-overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(ROOT / "internal/releaseauth/trusted-key.pem"): str(public_key)}}))
    # Build metadata reflects the worktree state; omit it so running this
    # capture twice cannot change an archive hash after generated assets dirty
    # the checkout.
    run(["go", "build", "-buildvcs=false", "-overlay", str(overlay), "-o", str(release / "darkpawns"), "./cmd/darkpawns"])
    run(["go", "build", "-buildvcs=false", "-o", str(release / "darkpawns-server"), "./cmd/server"])

    for relative in SEED_DIRS:
        source = ROOT / relative
        if relative == "web/public":
            copy_tree(source, assets / "web")
        else:
            copy_tree(source, assets / relative)
    copy_tree(ui, assets / "admin-ui")
    motd = ROOT / "bootstrap/motd"
    if not motd.is_file():
        raise RuntimeError("missing bootstrap/motd starter message")
    shutil.copyfile(motd, assets / "lib/text/motd")

    files = {
        path.relative_to(assets).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in sorted(assets.rglob("*")) if path.is_file()
    }
    for required in ("lib/world/wld/index", "web/index.html", "admin-ui/index.html"):
        if required not in files:
            raise RuntimeError(f"portable asset missing required file: {required}")
    manifest = {
        "format": 1, "update_format": 1, "data_format": 1,
        "version": "v0.1.7", "commit": "media-capture", "files": files,
    }
    (assets / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    run(["go", "run", "./cmd/dp-release", "--sign-bundle", str(release), "--key", str(private_key), "--public", str(public_key)])

    install_archive = WORK / "release.tar.gz"
    write_deterministic_archive(release, install_archive, "bundle")

    # Local archive differs only by version. update --check verifies and stages
    # it, then cleans staging; no approval path or Apply call is involved.
    candidate = WORK / "candidate"
    shutil.copytree(release, candidate)
    candidate_assets = candidate / "release-assets"
    candidate_manifest_path = candidate_assets / "manifest.json"
    candidate_manifest = json.loads(candidate_manifest_path.read_text())
    candidate_manifest["version"] = "v0.1.8"
    candidate_manifest_path.write_text(json.dumps(candidate_manifest, indent=2) + "\n")
    run(["go", "run", "./cmd/dp-release", "--sign-bundle", str(candidate), "--key", str(private_key), "--public", str(public_key)])
    archive = WORK / "candidate.tar.gz"
    write_deterministic_archive(candidate, archive, "candidate")
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    (WORK / "candidate.sha256").write_text(digest + "\n")
    return install_archive, archive, digest


def normalize_tarinfo(info: tarfile.TarInfo) -> tarfile.TarInfo:
    info.mtime = 0
    info.uid = 0
    info.gid = 0
    info.uname = ""
    info.gname = ""
    return info


def wait_for_status(launcher: Path, instance: Path, timeout: float = 45) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        result = subprocess.run(
            [str(launcher), "--instance", str(instance), "status", "--json"],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode == 0:
            try:
                status = json.loads(result.stdout)
            except json.JSONDecodeError:
                time.sleep(0.2)
                continue
            if status.get("state") == "running" and status.get("url"):
                return
        time.sleep(0.2)
    raise RuntimeError("throwaway managed world did not become ready")


def stop_world(launcher: Path, instance: Path) -> None:
    if not (instance / "instance.json").is_file():
        if port_is_open():
            raise RuntimeError(
                f"capture port {CAPTURE_PORT} is open but the owned instance record is missing; "
                f"preserving staging files at {WORK}"
            )
        return
    if not launcher.is_file():
        raise RuntimeError(f"managed instance launcher is missing; preserving staging files at {WORK}")

    # `setup` created this exact managed instance; stop it through the
    # launcher's lifecycle API before deleting its files.
    result = subprocess.run(
        [str(launcher), "--instance", str(instance), "stop", "--plain"],
        cwd=ROOT, capture_output=True, text=True, timeout=45, check=False,
    )
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip()
        raise RuntimeError(
            f"could not confirm stop for the owned capture instance ({detail or result.returncode}); "
            f"preserving staging files at {WORK}"
        )

    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        status = subprocess.run(
            [str(launcher), "--instance", str(instance), "status", "--json"],
            cwd=ROOT, capture_output=True, text=True, check=False,
        )
        if status.returncode == 0:
            try:
                if json.loads(status.stdout).get("state") == "stopped" and not port_is_open():
                    return
            except json.JSONDecodeError:
                pass
        time.sleep(0.2)
    raise RuntimeError(
        f"owned capture instance did not stop cleanly; preserving staging files at {WORK}"
    )


def port_is_open() -> bool:
    with socket.socket() as sock:
        sock.settimeout(0.2)
        return sock.connect_ex(("127.0.0.1", CAPTURE_PORT)) == 0


def verify_gif(path: Path) -> None:
    if not path.is_file() or path.stat().st_size < 1024:
        raise RuntimeError(f"VHS did not produce a usable GIF: {path}")
    probe = subprocess.run(
        ["ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries",
         "stream=width,height,nb_frames", "-show_entries", "format=duration",
         "-of", "json", str(path)], capture_output=True, text=True, check=True,
    )
    data = json.loads(probe.stdout)
    stream = data["streams"][0]
    duration = float(data["format"]["duration"])
    if int(stream["width"]) != 1200 or int(stream["height"]) != 720:
        raise RuntimeError(f"unexpected GIF dimensions for {path}: {stream}")
    if duration <= 0 or duration >= 15:
        raise RuntimeError(f"GIF must be animated and under 15 seconds: {path} ({duration:.2f}s)")
    if int(stream.get("nb_frames", "0")) < 2:
        raise RuntimeError(f"GIF has no animation: {path}")


def main() -> int:
    if shutil.which("vhs") is None or shutil.which("ffmpeg") is None or shutil.which("ffprobe") is None:
        raise RuntimeError("VHS, ffmpeg, and ffprobe must already be installed")
    # A previous failed shutdown may have deliberately retained this directory.
    # Refuse an occupied port before deleting even our own marked staging tree.
    ensure_capture_port_free()
    if WORK.exists():
        if not MARKER.is_file() or MARKER.read_text().strip() != "Dark Pawns media CLI capture work directory":
            raise RuntimeError(f"refusing to replace unowned work directory: {WORK}")
        shutil.rmtree(WORK)
    WORK.mkdir(parents=True)
    MARKER.write_text("Dark Pawns media CLI capture work directory\n")
    instance = WORK / "world"
    try:
        ensure_capture_port_free()
        _, _, digest = prepare_bundle()
        print(f"Candidate archive SHA-256: {digest}")

        # VHS executes setup itself so the reader sees the reviewed setup command
        # and the real readiness output, not a fabricated terminal transcript.
        run(["vhs", str(render_tape(CAPTURE_TAPES[0]))])
        bundle = WORK / "bundle"
        launcher = bundle / "darkpawns"
        wait_for_status(launcher, instance)

        # The lifecycle tape uses the real status/doctor. Update checking has
        # its own shorter recording; --check only stages/verifies, then removes
        # its staging tree without stopping the world.
        run(["vhs", str(render_tape(CAPTURE_TAPES[1]))])
        run(["vhs", str(render_tape(CAPTURE_TAPES[2], digest))])
        for output in OUTPUTS:
            verify_gif(output)
        print("CLI media captures verified:")
        for output in OUTPUTS:
            print(f"  {output.relative_to(ROOT)} ({output.stat().st_size:,} bytes)")
        return 0
    finally:
        stop_world(WORK / "bundle/darkpawns", instance)
        if WORK.exists() and MARKER.is_file():
            shutil.rmtree(WORK)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f"capture_media_cli: {exc}", file=sys.stderr)
        raise SystemExit(1)
