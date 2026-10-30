#!/usr/bin/env python3
"""Start disposable preview services, generate media, then stop only those PIDs."""

from __future__ import annotations

import os
import signal
import socket
import subprocess
import json
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def require_free_port(port: int) -> None:
    with socket.socket() as listener:
        listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            listener.bind(("127.0.0.1", port))
        except OSError as error:
            raise RuntimeError(f"refusing media capture: fixed port {port} is already occupied") from error


def preflight() -> None:
    for port in (4350, 5187, 4360, 7780, 8088):
        require_free_port(port)
    if Path("/tmp/.X11-unix/X88").exists():
        raise RuntimeError("refusing media capture: X display :88 is already occupied")


def wait_for(url: str, proc: subprocess.Popen[bytes], log, label: str) -> None:
    deadline = time.monotonic() + 120
    while time.monotonic() < deadline:
        if proc.poll() is not None:
            raise RuntimeError(f"{label} exited early ({proc.returncode}):\n{log.read_text(errors='replace')}")
        try:
            with urllib.request.urlopen(url, timeout=1):
                return
        except (urllib.error.URLError, TimeoutError):
            time.sleep(0.25)
    raise RuntimeError(f"timed out waiting for {label} at {url}")


def main() -> int:
    preflight()
    release = subprocess.run(
        ["git", "describe", "--tags", "--abbrev=0"], cwd=ROOT,
        capture_output=True, text=True, check=False,
    )
    release_name = release.stdout.strip() if release.returncode == 0 else "unreleased"
    git_head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    with tempfile.TemporaryDirectory(prefix="darkpawns-media-") as temp:
        tempdir = Path(temp)
        go_log_path, vite_log_path = tempdir / "preview.log", tempdir / "vite.log"
        go_log = go_log_path.open("wb")
        vite_log = vite_log_path.open("wb")
        procs: list[subprocess.Popen[bytes]] = []
        try:
            env = os.environ.copy()
            env.update({"ADMIN_PREVIEW": "1", "ADMIN_PREVIEW_LOGIN": "1", "ADMIN_PREVIEW_PORT": "4350", "MEDIA_RELEASE": release_name})
            go = subprocess.Popen(
                ["go", "test", "-tags", "adminpreview", "-run", "^TestConsolePreview$", "-timeout", "0", "./pkg/admin"],
                cwd=ROOT, env=env, stdout=go_log, stderr=subprocess.STDOUT,
                start_new_session=True,
            )
            procs.append(go)
            vite = subprocess.Popen(
                ["npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", "5187", "--strictPort"],
                cwd=ROOT / "admin-ui", env=env, stdout=vite_log, stderr=subprocess.STDOUT,
                start_new_session=True,
            )
            procs.append(vite)
            wait_for("http://127.0.0.1:4350/health", go, go_log_path, "admin preview")
            wait_for("http://127.0.0.1:5187/admin/@vite/client", vite, vite_log_path, "Vite")
            capture = subprocess.run(
                ["node", "scripts/media-capture.mjs"], cwd=ROOT / "admin-ui", env=env,
                check=False,
            )
            if capture.returncode:
                print(f"capture exit: {capture.returncode}", file=sys.stderr)
                return capture.returncode
            mudlet = subprocess.run([sys.executable, "scripts/capture_mudlet.py"], cwd=ROOT, env=env, check=False)
            if mudlet.returncode:
                return mudlet.returncode
            cli = subprocess.run([sys.executable, "scripts/capture_media_cli.py"], cwd=ROOT, env=env, check=False)
            if cli.returncode:
                return cli.returncode
            manifest_path = ROOT / "docs/images/generated/manifest.json"
            manifest = json.loads(manifest_path.read_text())
            manifest["release"] = release_name
            manifest["git_head"] = git_head
            manifest["media"].append({
                "path": "docs/images/generated/mudlet-dock.png",
                "alt": "Mudlet connected to a disposable local Dark Pawns world, showing the player dock, vitals, and Temple Altar map.",
                "sources": ["mudlet/src/00-core.lua", "mudlet/src/10-ui.lua", "mudlet/src/20-mapper.lua", "mudlet/darkpawns.xml", "pkg/session/gmcp.go", "scripts/fixtures/mudlet-host.xml", "scripts/bootstrap_media_deps.sh", "scripts/capture_mudlet.py"],
                "release": release_name,
            })
            manifest["media"].append({
                "path": "docs/images/generated/web-client-first-steps.png",
                "alt": "A new Guest at the Temple Altar in the web client, with the connection status and vitals below the game text.",
                "sources": ["web/public/index.html", "web/public/client.js", "web/public/mud-client.js", "web/public/style.css", "lib/world/wld/80.wld", "scripts/capture_mudlet.py", "admin-ui/scripts/capture-web-client.mjs", "admin-ui/scripts/media-capture.mjs", "website-astro/DESIGN.md"],
                "release": release_name,
            })
            for name, alt, sources in (
                ("install-and-setup.gif", "Portable bundle setup starts an isolated local world.", ["cmd/darkpawns/setup.go", "cmd/darkpawns/main.go", "internal/portable/instance.go", "scripts/capture_media_cli.py", "scripts/fixtures/docs-capture-test-private.pem", "scripts/fixtures/docs-capture-test-public.pem", "scripts/vhs/install-and-setup.tape"]),
                ("lifecycle-commands.gif", "Status reports the local world, and doctor checks its release, instance, and services.", ["cmd/darkpawns/lifecycle.go", "cmd/darkpawns/presentation.go", "internal/portable/doctor.go", "scripts/capture_media_cli.py", "scripts/fixtures/docs-capture-test-private.pem", "scripts/fixtures/docs-capture-test-public.pem", "scripts/vhs/lifecycle-commands.tape"]),
                ("update-check.gif", "An offline candidate release passes verification; --check leaves the running world untouched.", ["cmd/darkpawns/update.go", "cmd/darkpawns/update_presentation.go", "internal/portable/update.go", "internal/portable/update_archive.go", "scripts/capture_media_cli.py", "scripts/fixtures/docs-capture-test-private.pem", "scripts/fixtures/docs-capture-test-public.pem", "scripts/vhs/update-check.tape"]),
            ):
                manifest["media"].append({
                    "path": f"docs/images/generated/{name}", "alt": alt,
                    "sources": sources, "release": release_name,
                })
            for item in manifest["media"]:
                if "scripts/media_pipeline.py" not in item["sources"]:
                    item["sources"].append("scripts/media_pipeline.py")
            dirty = subprocess.run(["git", "status", "--porcelain"], cwd=ROOT, capture_output=True, text=True, check=True).stdout
            manifest["working_tree_dirty"] = bool(dirty.strip())
            manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")
            for label, path in (("admin preview", go_log_path), ("Vite", vite_log_path)):
                print(f"--- {label} log ---")
                print(path.read_text(errors="replace"), end="")
            return 0
        finally:
            for proc in reversed(procs):
                if proc.poll() is None:
                    os.killpg(proc.pid, signal.SIGTERM)
                    try:
                        proc.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        os.killpg(proc.pid, signal.SIGKILL)
                        proc.wait()
            go_log.close()
            vite_log.close()


if __name__ == "__main__":
    raise SystemExit(main())
