#!/usr/bin/env bash
set -e

# Assicura che i percorsi standard (Homebrew, /usr/local, ~/.local/bin) siano nel PATH
export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:$HOME/.local/bin:$PATH"

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

if [ "$1" = "--remove" ] || [ "$1" = "-remove" ] || [ "$1" = "--uninstall" ] || [ "$1" = "-uninstall" ] || [ "$1" = "remove" ] || [ "$1" = "uninstall" ]; then
    echo -e "${C_CYAN}➤ Uninstalling everyFLAC by pxradise...${C_RESET}"
    INSTALL_DIR="$HOME/.local/bin"
    rm -f "$INSTALL_DIR/everyflac" "$INSTALL_DIR/everyflac.exe" "$INSTALL_DIR/everyflac-tui" "$INSTALL_DIR/everyflac-tui.exe" "$INSTALL_DIR/pxradise-everyflac" "$INSTALL_DIR/pxradise-everyflac.exe"
    echo -e "  ${C_GREEN}✔${C_RESET} Removed executable from $INSTALL_DIR"

    rm -f "$HOME/.config/fish/functions/everyflac.fish"
    if [ -f "$HOME/.config/fish/config.fish" ]; then
        sed -i '/alias everyflac=/d' "$HOME/.config/fish/config.fish" 2>/dev/null || sed -i '' '/alias everyflac=/d' "$HOME/.config/fish/config.fish" 2>/dev/null || true
    fi
    echo -e "  ${C_GREEN}✔${C_RESET} Removed Fish shell function & alias"

    rm -rf "$HOME/.config/everyflac"
    echo -e "  ${C_GREEN}✔${C_RESET} Removed configuration directory"

    echo -e "\n${C_GREEN}${C_BOLD}✨ everyFLAC has been successfully removed from your system.${C_RESET}"
    exit 0
fi

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

YTDLP_VALID=false
if has_cmd yt-dlp || [ -f "$HOME/.local/bin/yt-dlp" ] || [ -f "$HOME/.local/bin/yt-dlp.exe" ]; then
    YTDLP_EXEC="$(which yt-dlp 2>/dev/null || true)"
    if [ -z "$YTDLP_EXEC" ] && [ -f "$HOME/.local/bin/yt-dlp" ]; then
        YTDLP_EXEC="$HOME/.local/bin/yt-dlp"
    fi
    if [ -n "$YTDLP_EXEC" ]; then
        YTDLP_VER="$("$YTDLP_EXEC" --version 2>/dev/null || echo "0")"
        YTDLP_YEAR="$(echo "$YTDLP_VER" | cut -d. -f1)"
        if [ "$YTDLP_YEAR" -ge 2025 ] 2>/dev/null; then
            YTDLP_VALID=true
            echo -e "  ${C_GREEN}✔${C_RESET} yt-dlp up to date (v$YTDLP_VER): $YTDLP_EXEC"
        else
            echo -e "  ${C_YELLOW}⚠ yt-dlp is outdated (v$YTDLP_VER at $YTDLP_EXEC).${C_RESET}"
            echo -e "    Debian/Ubuntu apt packages are obsolete and break on YouTube. Upgrading to latest official binary..."
        fi
    fi
fi

if [ "$YTDLP_VALID" = false ]; then
    echo -e "  ${C_CYAN}➤ Installing latest official standalone yt-dlp...${C_RESET}"
    mkdir -p "$HOME/.local/bin"
    if [ "$OS" = "windows" ]; then
        curl -fSL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe -o "$HOME/.local/bin/yt-dlp.exe"
        chmod +x "$HOME/.local/bin/yt-dlp.exe" 2>/dev/null || true
    else
        curl -fSL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o "$HOME/.local/bin/yt-dlp"
        chmod +x "$HOME/.local/bin/yt-dlp"
        if $IS_MACOS; then
            xattr -c "$HOME/.local/bin/yt-dlp" 2>/dev/null || true
        fi
    fi
    export PATH="$HOME/.local/bin:$PATH"
    echo -e "  ${C_GREEN}✔${C_RESET} Installed latest yt-dlp to ~/.local/bin/yt-dlp"
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

BINARY_EXT=""
if $IS_WINDOWS; then
    BINARY_EXT=".exe"
fi

TARGET_BIN="$INSTALL_DIR/${APP_NAME}${BINARY_EXT}"
rm -f "$TARGET_BIN" "$INSTALL_DIR/everyflac-tui${BINARY_EXT}" "$INSTALL_DIR/pxradise-everyflac${BINARY_EXT}"

LOCAL_SOURCE=""

# 1. Se siamo in locale e c'è main.go con il compilatore go disponibile
if [ -f "$SCRIPT_DIR/main.go" ] && has_cmd go; then
    echo -e "  ${C_GREEN}✔ Go compiler found!${C_RESET} Building native binary for ${PLATFORM_NAME}..."
    if (cd "$SCRIPT_DIR" && go build -ldflags="-s -w" -o "${APP_NAME}${BINARY_EXT}" main.go 2>/dev/null); then
        LOCAL_SOURCE="$SCRIPT_DIR/${APP_NAME}${BINARY_EXT}"
    else
        echo -e "  ${C_YELLOW}⚠ Native build failed, falling back to precompiled binary...${C_RESET}"
    fi
fi

# 2. Se siamo in locale e c'è un binario precompilato nella cartella bin/
if [ -z "$LOCAL_SOURCE" ]; then
    if [ -f "$SCRIPT_DIR/bin/everyflac-tui-${OS}-${ARCH}${BINARY_EXT}" ]; then
        LOCAL_SOURCE="$SCRIPT_DIR/bin/everyflac-tui-${OS}-${ARCH}${BINARY_EXT}"
    elif [ -f "$SCRIPT_DIR/bin/${APP_NAME}-${OS}-${ARCH}${BINARY_EXT}" ]; then
        LOCAL_SOURCE="$SCRIPT_DIR/bin/${APP_NAME}-${OS}-${ARCH}${BINARY_EXT}"
    fi
fi

# 3. Applica il binario locale oppure scaricalo se eseguito via curl
if [ -n "$LOCAL_SOURCE" ] && [ -f "$LOCAL_SOURCE" ]; then
    echo -e "  ${C_GREEN}✔${C_RESET} Using local binary: $(basename "$LOCAL_SOURCE")"
    cp "$LOCAL_SOURCE" "$TARGET_BIN"
else
    echo -e "  ${C_CYAN}➤ Downloading everyFLAC binary for ${PLATFORM_NAME}...${C_RESET}"
    GITHUB_REPO="${GITHUB_REPO:-pxradiso/everyflac}"
    BIN_NAME="everyflac-tui-${OS}-${ARCH}${BINARY_EXT}"
    RAW_URL="https://raw.githubusercontent.com/${GITHUB_REPO}/main/bin/${BIN_NAME}"
    RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BIN_NAME}"

    DOWNLOAD_SUCCESS=false

    if curl -fSL --progress-bar "$RAW_URL" -o "$TARGET_BIN"; then
        DOWNLOAD_SUCCESS=true
    elif curl -fSL --progress-bar "$RELEASE_URL" -o "$TARGET_BIN"; then
        DOWNLOAD_SUCCESS=true
    fi

    if [ "$DOWNLOAD_SUCCESS" = false ]; then
        echo -e "  ${C_RED}✖ Error:${C_RESET} Failed to download precompiled binary for ${PLATFORM_NAME}."
        echo -e "    URL tried: $RAW_URL"
        echo -e "    URL tried: $RELEASE_URL"
        exit 1
    fi
fi

chmod +x "$TARGET_BIN" 2>/dev/null || true

# macOS: rimuove attributi di quarantena di Gatekeeper per consentire l'esecuzione immediata
if $IS_MACOS; then
    xattr -c "$TARGET_BIN" 2>/dev/null || true
    xattr -d com.apple.quarantine "$TARGET_BIN" 2>/dev/null || true
fi

echo -e "  ${C_GREEN}✔ Installed executable:${C_RESET} $TARGET_BIN"

echo -e "\n${C_CYAN}➤ Configuring shell PATH & aliases...${C_RESET}"

SHELL_CONFIGURED=false

# 1. Configurazione Bash
for bash_prof in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile"; do
    if [ -f "$bash_prof" ]; then
        if ! grep -q ".local/bin" "$bash_prof"; then
            echo "" >> "$bash_prof"
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$bash_prof"
            echo -e "  ${C_GREEN}✔${C_RESET} Added $INSTALL_DIR to PATH in ${C_BOLD}$bash_prof${C_RESET}"
            SHELL_CONFIGURED=true
        fi
        break
    fi
done

# 2. Configurazione Zsh
if [ -f "$HOME/.zshrc" ] || has_cmd zsh; then
    ZSH_RC="$HOME/.zshrc"
    [ ! -f "$ZSH_RC" ] && touch "$ZSH_RC"
    if ! grep -q ".local/bin" "$ZSH_RC"; then
        echo "" >> "$ZSH_RC"
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$ZSH_RC"
        echo -e "  ${C_GREEN}✔${C_RESET} Added $INSTALL_DIR to PATH in ${C_BOLD}$ZSH_RC${C_RESET}"
        SHELL_CONFIGURED=true
    fi
fi

# 3. Configurazione Fish Shell
FISH_BIN=""
if has_cmd fish; then
    FISH_BIN="$(command -v fish)"
elif [ -x "/opt/homebrew/bin/fish" ]; then
    FISH_BIN="/opt/homebrew/bin/fish"
elif [ -x "/usr/local/bin/fish" ]; then
    FISH_BIN="/usr/local/bin/fish"
elif [ -x "/usr/bin/fish" ]; then
    FISH_BIN="/usr/bin/fish"
elif [ -x "/bin/fish" ]; then
    FISH_BIN="/bin/fish"
fi

FISH_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/fish"

if [ -n "$FISH_BIN" ] || [ -d "$FISH_CONFIG_DIR" ] || [ -f "$FISH_CONFIG_DIR/config.fish" ]; then
    FISH_CONFIG="$FISH_CONFIG_DIR/config.fish"
    mkdir -p "$FISH_CONFIG_DIR/functions"
    [ ! -f "$FISH_CONFIG" ] && touch "$FISH_CONFIG"

    # Funzione autoload nativa di Fish (funziona all'istante anche nei terminali fish già aperti!)
    cat << 'FISH_FUNC' > "$FISH_CONFIG_DIR/functions/everyflac.fish"
function everyflac --wraps="$HOME/.local/bin/everyflac" --description 'Lossless FLAC music downloader'
    $HOME/.local/bin/everyflac $argv
end
FISH_FUNC
    echo -e "  ${C_GREEN}✔${C_RESET} Added fish autoload function: ${C_BOLD}$FISH_CONFIG_DIR/functions/everyflac.fish${C_RESET}"
    SHELL_CONFIGURED=true

    # Aggiorna PATH con sintassi nativa Fish in config.fish
    if ! grep -q ".local/bin" "$FISH_CONFIG"; then
        echo "" >> "$FISH_CONFIG"
        echo '# everyFLAC PATH' >> "$FISH_CONFIG"
        echo 'if test -d "$HOME/.local/bin"' >> "$FISH_CONFIG"
        echo '    set -gx PATH "$HOME/.local/bin" $PATH' >> "$FISH_CONFIG"
        echo 'end' >> "$FISH_CONFIG"
        echo -e "  ${C_GREEN}✔${C_RESET} Added $INSTALL_DIR to PATH in ${C_BOLD}$FISH_CONFIG${C_RESET}"
        SHELL_CONFIGURED=true
    fi

    # Registra l'alias per fish in config.fish
    if ! grep -q "alias everyflac=" "$FISH_CONFIG"; then
        echo 'alias everyflac="$HOME/.local/bin/everyflac"' >> "$FISH_CONFIG"
        echo -e "  ${C_GREEN}✔${C_RESET} Added alias 'everyflac' in ${C_BOLD}$FISH_CONFIG${C_RESET}"
        SHELL_CONFIGURED=true
    fi

    # Se fish è installato, applica anche come Universal Variable
    if [ -n "$FISH_BIN" ]; then
        "$FISH_BIN" -c 'fish_add_path -U "$HOME/.local/bin"' 2>/dev/null || "$FISH_BIN" -c 'fish_add_path "$HOME/.local/bin"' 2>/dev/null || true
    fi
fi

echo -e "\n${C_GREEN}${C_BOLD}✨ Installation Complete!${C_RESET}"
echo -e "${C_MUTED}──────────────────────────────────────────────────────────${C_RESET}"
echo -e "You can now run:"
echo -e "  ${C_JADE}${C_BOLD}everyflac${C_RESET}                      (Interactive TUI)"
echo -e "  ${C_JADE}${C_BOLD}everyflac <spotify_or_yt_url>${C_RESET}   (Download directly)"
echo -e "  ${C_JADE}${C_BOLD}everyflac --search <song_name>${C_RESET}  (Search & download)"
echo -e "${C_MUTED}──────────────────────────────────────────────────────────${C_RESET}"

if [ "$SHELL_CONFIGURED" = true ]; then
    echo -e "${C_YELLOW}Note:${C_RESET} Restart your terminal or run ${C_BOLD}source ~/.config/fish/config.fish${C_RESET} (or ${C_BOLD}source ~/.bashrc${C_RESET}) to apply changes."
fi
