#!/usr/bin/env bash
set -e

C_RESET="\033[0m"
C_BOLD="\033[1m"
C_JADE="\033[38;2;115;218;202m"
C_GREEN="\033[1;32m"
C_CYAN="\033[1;36m"
C_YELLOW="\033[1;33m"
C_RED="\033[1;31m"
C_MUTED="\033[38;2;86;95;137m"

APP_NAME="everyflac"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${C_JADE}${C_BOLD}"
cat << 'ASCII'
  ▄███▄ █   █ ▄███▄ ████▄ █   █ ████ █    ▄███▄ ▄███▄ 
  █▄▄▄▄ █   █ █▄▄▄▄ █▄▄▄█  █▄█  █▄▄  █    █▄▄▄█ █     
  █▀▀▀▀ ▀█▄█▀ █▀▀▀▀ █▀▀▄    █   █    █    █   █ █     
  ▀███▀   ▀   ▀███▀ █  ▀█   █   █    ████ █   █ ▀███▀ 
ASCII
echo -e "${C_RESET}"
echo -e "  ${C_BOLD}⚡ everyFLAC by pxradise — Universal Setup & Installer${C_RESET}"
echo -e "  ${C_MUTED}───────────────────────────────────────────────────────────${C_RESET}\n"

OS_TYPE="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_RAW="$(uname -m)"

case "$ARCH_RAW" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7*|armhf)
        ARCH="arm"
        ;;
    i386|i686)
        ARCH="386"
        ;;
    *)
        ARCH="amd64"
        ;;
esac

IS_WINDOWS=false
IS_MACOS=false
IS_LINUX=false
IS_BSD=false

case "$OS_TYPE" in
    darwin*)
        OS="darwin"
        IS_MACOS=true
        PLATFORM_NAME="macOS ($ARCH)"
        ;;
    linux*)
        OS="linux"
        IS_LINUX=true
        PLATFORM_NAME="Linux ($ARCH)"
        ;;
    cygwin*|mingw*|msys*|windows*)
        OS="windows"
        IS_WINDOWS=true
        PLATFORM_NAME="Windows ($ARCH)"
        ;;
    *bsd*)
        OS="freebsd"
        IS_BSD=true
        PLATFORM_NAME="BSD ($ARCH)"
        ;;
    *)
        OS="linux"
        IS_LINUX=true
        PLATFORM_NAME="Generic UNIX ($ARCH)"
        ;;
esac

echo -e "${C_CYAN}➤ Detected Platform:${C_RESET} ${C_BOLD}${PLATFORM_NAME}${C_RESET}"
echo -e "\n${C_CYAN}➤ Checking required dependencies...${C_RESET}"

has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

if has_cmd yt-dlp; then
    echo -e "  ${C_GREEN}✔${C_RESET} yt-dlp found: $(which yt-dlp)"
elif [ -f "$HOME/.local/bin/yt-dlp" ] || [ -f "$HOME/.local/bin/yt-dlp.exe" ]; then
    echo -e "  ${C_GREEN}✔${C_RESET} yt-dlp found in ~/.local/bin"
    export PATH="$HOME/.local/bin:$PATH"
else
    echo -e "  ${C_YELLOW}⚠ yt-dlp not found.${C_RESET} Attempting automatic installation..."
    if $IS_MACOS && has_cmd brew; then
        brew install yt-dlp || true
    elif $IS_LINUX; then
        if has_cmd pacman; then
            sudo pacman -Sy --noconfirm yt-dlp 2>/dev/null || true
        elif has_cmd apt-get; then
            sudo apt-get update -qq && sudo apt-get install -y yt-dlp 2>/dev/null || true
        elif has_cmd dnf; then
            sudo dnf install -y yt-dlp 2>/dev/null || true
        elif has_cmd zypper; then
            sudo zypper install -y yt-dlp 2>/dev/null || true
        elif has_cmd apk; then
            sudo apk add yt-dlp 2>/dev/null || true
        elif has_cmd xbps-install; then
            sudo xbps-install -Sy yt-dlp 2>/dev/null || true
        fi
    elif $IS_WINDOWS && has_cmd winget; then
        winget install --id yt-dlp.yt-dlp -e --accept-source-agreements --accept-package-agreements || true
    fi

    if ! has_cmd yt-dlp; then
        echo -e "  ${C_CYAN}Downloading official yt-dlp binary...${C_RESET}"
        mkdir -p "$HOME/.local/bin"
        if [ "$OS" = "windows" ]; then
            curl -sL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe -o "$HOME/.local/bin/yt-dlp.exe"
            chmod +x "$HOME/.local/bin/yt-dlp.exe" 2>/dev/null || true
        else
            curl -sL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o "$HOME/.local/bin/yt-dlp"
            chmod +x "$HOME/.local/bin/yt-dlp"
        fi
        export PATH="$HOME/.local/bin:$PATH"
        echo -e "  ${C_GREEN}✔${C_RESET} Installed standalone yt-dlp to ~/.local/bin/yt-dlp"
    fi
fi

if has_cmd ffmpeg; then
    echo -e "  ${C_GREEN}✔${C_RESET} ffmpeg found: $(which ffmpeg)"
else
    echo -e "  ${C_YELLOW}⚠ ffmpeg not found.${C_RESET} Attempting automatic installation..."
    if $IS_MACOS && has_cmd brew; then
        brew install ffmpeg || true
    elif $IS_LINUX; then
        if has_cmd pacman; then
            sudo pacman -Sy --noconfirm ffmpeg 2>/dev/null || true
        elif has_cmd apt-get; then
            sudo apt-get update -qq && sudo apt-get install -y ffmpeg 2>/dev/null || true
        elif has_cmd dnf; then
            sudo dnf install -y ffmpeg 2>/dev/null || true
        elif has_cmd zypper; then
            sudo zypper install -y ffmpeg 2>/dev/null || true
        elif has_cmd apk; then
            sudo apk add ffmpeg 2>/dev/null || true
        elif has_cmd xbps-install; then
            sudo xbps-install -Sy ffmpeg 2>/dev/null || true
        fi
    elif $IS_WINDOWS && has_cmd winget; then
        winget install --id Gyan.FFmpeg -e --accept-source-agreements --accept-package-agreements || true
    fi
fi

if ! has_cmd yt-dlp && [ ! -f "$HOME/.local/bin/yt-dlp" ] && [ ! -f "$HOME/.local/bin/yt-dlp.exe" ]; then
    echo -e "  ${C_RED}✖ Error:${C_RESET} yt-dlp is required. Please install yt-dlp manually."
fi
if ! has_cmd ffmpeg; then
    echo -e "  ${C_YELLOW}⚠ Warning:${C_RESET} ffmpeg was not detected. Audio transcoding & album art injection might fail."
fi

echo -e "\n${C_CYAN}➤ Installing everyFLAC by pxradise...${C_RESET}"

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

BINARY_SOURCE=""
BINARY_EXT=""
if $IS_WINDOWS; then
    BINARY_EXT=".exe"
fi

if has_cmd go; then
    echo -e "  ${C_GREEN}✔ Go compiler found!${C_RESET} Building native binary for ${PLATFORM_NAME}..."
    cd "$SCRIPT_DIR"
    go build -ldflags="-s -w" -o "${APP_NAME}${BINARY_EXT}" main.go
    BINARY_SOURCE="$SCRIPT_DIR/${APP_NAME}${BINARY_EXT}"
else
    PRECOMPILED="$SCRIPT_DIR/bin/${APP_NAME}-${OS}-${ARCH}${BINARY_EXT}"
    if [ -f "$PRECOMPILED" ]; then
        echo -e "  ${C_GREEN}✔${C_RESET} Using precompiled binary: $(basename "$PRECOMPILED")"
        BINARY_SOURCE="$PRECOMPILED"
    else
        echo -e "  ${C_RED}✖ Error:${C_RESET} Go compiler not found and no precompiled binary at $PRECOMPILED"
        exit 1
    fi
fi

TARGET_BIN="$INSTALL_DIR/${APP_NAME}${BINARY_EXT}"

rm -f "$TARGET_BIN" "$INSTALL_DIR/everyflac-tui${BINARY_EXT}" "$INSTALL_DIR/pxradise-everyflac${BINARY_EXT}"
cp "$BINARY_SOURCE" "$TARGET_BIN"
chmod +x "$TARGET_BIN" 2>/dev/null || true

echo -e "  ${C_GREEN}✔ Installed executable:${C_RESET} $TARGET_BIN"

SHELL_NAME="$(basename "${SHELL:-bash}")"
PROFILE_FILES=()

if [ "$SHELL_NAME" = "zsh" ]; then
    PROFILE_FILES+=("$HOME/.zshrc")
elif [ "$SHELL_NAME" = "fish" ]; then
    PROFILE_FILES+=("$HOME/.config/fish/config.fish")
else
    PROFILE_FILES+=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile")
fi

PATH_LINE='export PATH="$HOME/.local/bin:$PATH"'
PATH_CONFIGURED=false

if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
    PATH_CONFIGURED=true
fi

if [ "$PATH_CONFIGURED" = false ]; then
    for prof in "${PROFILE_FILES[@]}"; do
        if [ -f "$prof" ] && ! grep -q ".local/bin" "$prof"; then
            echo "" >> "$prof"
            echo "$PATH_LINE" >> "$prof"
            echo -e "  ${C_GREEN}✔${C_RESET} Added $INSTALL_DIR to PATH in ${C_BOLD}$prof${C_RESET}"
            PATH_CONFIGURED=true
            break
        fi
    done
fi

echo -e "\n${C_GREEN}${C_BOLD}✨ Installation Complete!${C_RESET}"
echo -e "${C_MUTED}──────────────────────────────────────────────────────────${C_RESET}"
echo -e "You can now run:"
echo -e "  ${C_JADE}${C_BOLD}everyflac${C_RESET}                      (Interactive TUI)"
echo -e "  ${C_JADE}${C_BOLD}everyflac <spotify_or_yt_url>${C_RESET}   (Download directly)"
echo -e "  ${C_JADE}${C_BOLD}everyflac --search <song_name>${C_RESET}  (Search & download)"
echo -e "${C_MUTED}──────────────────────────────────────────────────────────${C_RESET}"

if [ "$PATH_CONFIGURED" = true ] && [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${C_YELLOW}Note:${C_RESET} Run '${C_BOLD}source ~/.bashrc${C_RESET}' or restart your terminal to reload PATH."
fi
