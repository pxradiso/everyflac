#!/usr/bin/env bash
# Build the Windows setup.exe for everyFLAC.
#
# Requirements:
#   - makensis (NSIS 3.x). On Arch you can use the Ubuntu .deb extracted
#     locally and point MAKENSIS/NSISDIR at it, for example:
#       MAKENSIS=/tmp/nsis/usr/bin/makensis \
#       NSISDIR=/tmp/nsis/usr/share/nsis ./installer/build-windows-installer.sh
#   - curl
#
# The Windows binaries must already exist in bin/ (built with GOOS=windows).

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALLER_DIR="$ROOT/installer"
PAYLOAD="$INSTALLER_DIR/.payload"

MAKENSIS="${MAKENSIS:-makensis}"
if ! command -v "$MAKENSIS" >/dev/null 2>&1 && [ ! -x "$MAKENSIS" ]; then
    echo "error: makensis not found. Set MAKENSIS=/path/to/makensis" >&2
    exit 1
fi

mkdir -p "$PAYLOAD"

fetch() {
    local url="$1" dest="$2"
    if [ -s "$dest" ]; then
        echo "  cached: $(basename "$dest")"
        return
    fi
    echo "  downloading: $(basename "$dest")"
    curl -fL --retry 3 --progress-bar "$url" -o "$dest"
}

fetch "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe" "$PAYLOAD/yt-dlp.exe"
fetch "https://github.com/eugeneware/ffmpeg-static/releases/latest/download/ffmpeg-win32-x64" "$PAYLOAD/ffmpeg.exe"

for arch in amd64 arm64; do
    src="$ROOT/bin/everyflac-tui-windows-$arch.exe"
    if [ ! -f "$src" ]; then
        echo "error: missing $src (build it first)" >&2
        exit 1
    fi
    cp "$src" "$PAYLOAD/everyflac-$arch.exe"
done

echo "  compiling installer..."
VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"
if [ -z "$VERSION" ]; then
    echo "error: VERSION is empty" >&2
    exit 1
fi
(
    cd "$INSTALLER_DIR"
    if [ -n "${NSISDIR:-}" ]; then
        export NSISDIR
    fi
    "$MAKENSIS" -V2 -DAPP_VERSION="$VERSION" everyflac.nsi
)

cp "$INSTALLER_DIR/everyflac-setup-windows-amd64.exe" "$ROOT/setup.exe"
echo "  built: $ROOT/setup.exe"
