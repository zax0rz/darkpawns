#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
deps="$root/.local/capture-tools"
mudlet="$root/.local/mudlet"
mkdir -p "$deps/debs" "$deps/root" "$mudlet"

if [[ ! -d "$root/admin-ui/node_modules/@playwright/test" ]]; then
  npm ci --prefix "$root/admin-ui"
fi
if ! node -e 'const fs=require("fs"); const {chromium}=require(process.argv[1]); process.exit(fs.existsSync(chromium.executablePath()) ? 0 : 1)' "$root/admin-ui/node_modules/@playwright/test"; then
  npm exec --prefix "$root/admin-ui" -- playwright install chromium
fi
for command in vhs ffmpeg ffprobe; do
  command -v "$command" >/dev/null || { echo "Missing $command; install the local docs-media tools before running make media." >&2; exit 1; }
done

if [[ ! -x "$mudlet/Mudlet.AppImage" ]]; then
  curl --fail --location --retry 2 \
    https://github.com/Mudlet/Mudlet/releases/download/Mudlet-5.0.2/Mudlet-5.0.2-linux-x64.AppImage.tar \
    --output "$mudlet/Mudlet.AppImage.tar"
  tar -xf "$mudlet/Mudlet.AppImage.tar" -C "$mudlet"
  chmod +x "$mudlet/Mudlet.AppImage"
fi
expected=50f922ea2c8b4b694c20560f1d63e7fca5f6e5a2fa38cc8141fc49b7cd30fc26
actual="$(sha256sum "$mudlet/Mudlet.AppImage" | cut -d' ' -f1)"
[[ "$actual" == "$expected" ]] || { echo "Mudlet 5.0.2 AppImage SHA-256 mismatch: $actual" >&2; exit 1; }

if [[ ! -x "$deps/root/usr/bin/Xvfb" || ! -x "$deps/root/usr/bin/xdotool" ]]; then
  (cd "$deps/debs" && apt download xvfb xdotool libxdo3)
  for package in "$deps"/debs/*.deb; do dpkg-deb -x "$package" "$deps/root"; done
fi
if LD_LIBRARY_PATH="$deps/root/usr/lib/x86_64-linux-gnu${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}" \
  ldd "$deps/root/usr/bin/Xvfb" "$deps/root/usr/bin/xdotool" 2>/dev/null | grep -q 'not found'; then
  echo "An Xvfb or xdotool shared library is missing; see local toolchain packages printed by ldd." >&2
  LD_LIBRARY_PATH="$deps/root/usr/lib/x86_64-linux-gnu${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}" \
    ldd "$deps/root/usr/bin/Xvfb" "$deps/root/usr/bin/xdotool" >&2
  exit 1
fi
printf 'Ready: Mudlet 5.0.2 AppImage, Xvfb and xdotool extracted under .local/.\n'
