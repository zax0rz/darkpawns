#!/usr/bin/env python3
"""Capture the packaged Mudlet dock against a disposable local server."""
from __future__ import annotations

import os
import shutil
import signal
import socket
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LOCAL = ROOT / ".local/media-capture"
OUT = ROOT / "docs/images/generated/mudlet-dock.png"


def write_qstring(path: Path, value: str | None) -> None:
    """Write the QDataStream QString sidecars Mudlet uses for profile hosts."""
    if value is None:
        path.write_bytes(b"\xff\xff\xff\xff")
        return
    encoded = value.encode("utf-16-be")
    path.write_bytes(len(encoded).to_bytes(4, "big") + encoded)


def wait_port(port: int, proc: subprocess.Popen, label: str) -> None:
    deadline = time.monotonic() + 90
    while time.monotonic() < deadline:
        if proc.poll() is not None:
            raise RuntimeError(f"{label} exited early ({proc.returncode})")
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.2):
                return
        except OSError:
            time.sleep(0.2)
    raise RuntimeError(f"timed out waiting for {label} on port {port}")


def preflight() -> None:
    for port in (4360, 7780, 8088):
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            try:
                listener.bind(("127.0.0.1", port))
            except OSError as error:
                raise RuntimeError(f"refusing capture: fixed port {port} is already occupied") from error
    if Path("/tmp/.X11-unix/X88").exists():
        raise RuntimeError("refusing capture: X display :88 is already occupied")


def main() -> int:
    preflight()
    mudlet = Path(os.environ.get("MUDLET_APP", str(ROOT / ".local/mudlet/Mudlet.AppImage")))
    xvfb = Path(os.environ.get("XVFB", str(ROOT / ".local/capture-tools/root/usr/bin/Xvfb")))
    xdotool = Path(os.environ.get("XDOTOOL", str(ROOT / ".local/capture-tools/root/usr/bin/xdotool")))
    ffmpeg = shutil.which("ffmpeg")
    if not mudlet.is_file() or not xvfb.is_file() or not xdotool.is_file() or not ffmpeg:
        print("Mudlet capture needs the official Mudlet AppImage, Xvfb, xdotool and ffmpeg. See scripts/bootstrap_media_deps.sh.", file=sys.stderr)
        return 2
    app_dir = LOCAL / "mudlet-app"
    app = app_dir / "squashfs-root/mudlet"
    LOCAL.mkdir(parents=True, exist_ok=True)
    app_dir.mkdir(parents=True, exist_ok=True)
    if not app.is_file():
        subprocess.run([str(mudlet), "--appimage-extract"], cwd=app_dir, check=True, stdout=subprocess.DEVNULL)
    if not app.is_file():
        raise RuntimeError("AppImage extraction did not produce squashfs-root/mudlet")

    binary = LOCAL / "server"
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(binary), "./cmd/server"], cwd=ROOT, check=True)
    base = {"DISPLAY": ":88", "QT_QPA_PLATFORM": "xcb", "QT_HASH_SEED": "0", "HOME": str(LOCAL / "home"),
            "XDG_CONFIG_HOME": str(LOCAL / "config"), "XDG_DATA_HOME": str(LOCAL / "data"),
            "ENVIRONMENT": "development", "DP_MUDLET_PACKAGE_URL": "http://127.0.0.1:8088/darkpawns.xml",
            "DP_MUDLET_MAP_URL": "http://127.0.0.1:4360/darkpawns-map.xml"}
    for name in ("home", "config", "data"):
        shutil.rmtree(LOCAL / name, ignore_errors=True)
        (LOCAL / name).mkdir(exist_ok=True)
    (LOCAL / "config/mudlet").mkdir(parents=True, exist_ok=True)
    (LOCAL / "config/mudlet/Mudlet.ini").write_text(
        "[General]\nfirstLaunchDate=2026-10-29T00:00:00Z\n"
        "telnetHandlerAsked=true\ntelnetHandlerEnabled=true\nuiTourShown=true\n\n"
        "[DBLSQD]\nautoDownload=true\n"
    )
    profile = LOCAL / "config/mudlet/profiles/Dark Pawns Media/current"
    profile.mkdir(parents=True, exist_ok=True)
    profile_root = profile.parent
    shutil.copyfile(ROOT / "scripts/fixtures/mudlet-host.xml", profile / "2026-10-29#00-00-00.xml")
    write_qstring(profile_root / "url", "127.0.0.1")
    write_qstring(profile_root / "port", "7780")
    write_qstring(profile_root / "login", None)
    for old in (LOCAL / "world.sqlite", LOCAL / "world.sqlite-shm", LOCAL / "world.sqlite-wal"):
        old.unlink(missing_ok=True)
    server_log = (LOCAL / "server.log").open("wb")
    server = subprocess.Popen([str(binary), "-port", "4360", "-telnet-port", "7780", "-db", str(LOCAL / "world.sqlite")], cwd=ROOT, env={**os.environ, **base}, stdout=server_log, stderr=subprocess.STDOUT, start_new_session=True)
    package = subprocess.Popen([sys.executable, "-m", "http.server", "8088", "--bind", "127.0.0.1", "--directory", str(ROOT / "mudlet")], cwd=ROOT, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
    display = subprocess.Popen([str(xvfb), ":88", "-screen", "0", "1440x900x24", "-nolisten", "tcp"], env=os.environ.copy(), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
    browser = None
    try:
        wait_port(4360, server, "local game server")
        wait_port(8088, package, "local Mudlet package server")
        web_capture = subprocess.run(
            ["node", "scripts/capture-web-client.mjs"],
            cwd=ROOT / "admin-ui",
            env={**os.environ, "MEDIA_CAPTURE_SERVER_URL": "http://127.0.0.1:4360"},
            check=False,
        )
        if web_capture.returncode:
            raise RuntimeError(f"web client capture exited with status {web_capture.returncode}")
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline and not Path("/tmp/.X11-unix/X88").exists():
            if display.poll() is not None:
                raise RuntimeError("Xvfb exited before creating display :88")
            time.sleep(0.1)
        if not Path("/tmp/.X11-unix/X88").exists():
            raise RuntimeError("timed out waiting for Xvfb display :88")
        ld = str(ROOT / ".local/capture-tools/root/usr/lib/x86_64-linux-gnu")
        env = {**os.environ, **base, "LD_LIBRARY_PATH": ld}
        browser = subprocess.Popen([str(app), "--no-sandbox", "--profile=Dark Pawns Media"], cwd=app_dir, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)

        def key(*args: str) -> None:
            subprocess.run([str(xdotool), *args], env=env, check=True)
        def click(x: int, y: int) -> None:
            key("mousemove", str(x), str(y))
            key("click", "1")
        def type_text(value: str, delay: int = 80) -> None:
            key("type", "--delay", str(delay), value)
        def press(value: str) -> None:
            key("key", value)
        def pause(seconds: float = 0.8) -> None:
            time.sleep(seconds)

        pause(20)
        windows = subprocess.run([str(xdotool), "search", "--onlyvisible", "--name", ".*"], env=env, text=True, capture_output=True, check=False)
        ids = windows.stdout.splitlines()
        names = [(value, subprocess.run([str(xdotool), "getwindowname", value], env=env, text=True, capture_output=True).stdout.strip()) for value in ids]
        wid = next((value for value, name in names if "Mudlet" in name or "Dark Pawns" in name), ids[0] if ids else "")
        if not wid:
            raise RuntimeError(f"Mudlet has no visible window after connecting: {windows.stderr}")
        key("windowsize", wid, "1440", "900")
        key("windowmove", wid, "0", "0")
        key("windowfocus", "--sync", wid)
        pause(2)
        click(300, 884)
        key("type", "--clearmodifiers", "--delay", "150", "Guest")
        key("key", "--clearmodifiers", "Return")
        pause(20)  # Actual server package/map transfer and guest entry.
        server_log.flush()
        server_output = (LOCAL / "server.log").read_text(errors="replace")
        if "Telnet connect" not in server_output:
            raise RuntimeError("Mudlet never connected to the disposable server; refusing to keep a fake or empty capture")
        if "GMCP client identified" not in server_output or not (profile_root / "darkpawns.xml").is_file() or not (profile_root / "darkpawns-map.xml").is_file():
            raise RuntimeError("Mudlet did not identify through GMCP and receive the local package and map")
        click(300, 884)
        key("key", "ctrl+a")
        key("key", "BackSpace")
        # Ask the live server for the current room so the screenshot includes
        # actual gameplay beneath the connected local client.
        type_text("look", 100)
        press("Return")
        pause(2)
        click(300, 884)
        key("key", "ctrl+a")
        key("key", "BackSpace")
        key("mousemove", "900", "100")
        pause(1)
        OUT.parent.mkdir(parents=True, exist_ok=True)
        full = LOCAL / "mudlet-full.png"
        subprocess.run([ffmpeg, "-v", "error", "-y", "-f", "x11grab", "-draw_mouse", "0", "-video_size", "1440x900", "-i", ":88", "-frames:v", "1", str(full)], check=True)
        # Keep the actual Mudlet dock, vitals and room map as the subject. The
        # console contains a server-reported map audit duration that varies by
        # machine load; the dock capture ends before Mudlet's empty Chat pane.
        subprocess.run([ffmpeg, "-v", "error", "-y", "-i", str(full), "-vf", "crop=450:588:990:22", str(OUT)], check=True)
        return 0
    finally:
        for proc in (browser, display, package, server):
            if proc and proc.poll() is None:
                os.killpg(proc.pid, signal.SIGTERM)
                try:
                    proc.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    os.killpg(proc.pid, signal.SIGKILL)
                    proc.wait()
        server_log.close()


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f"Mudlet capture failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
