#!/bin/sh
set -eu

VERSION="${1:-latest}"
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
REPO_SLUG="${AUTOCOMPLETE_REPO_SLUG:-kmal808/autocomplete-sh}"

detect_shell() {
    if [ -n "${ZSH_VERSION:-}" ]; then
        printf '%s\n' "zsh"
        return
    fi
    if [ -n "${BASH_VERSION:-}" ]; then
        printf '%s\n' "bash"
        return
    fi

    case "$(ps -p "$PPID" -o comm= 2>/dev/null || true)" in
        *zsh*) printf '%s\n' "zsh" ;;
        *) printf '%s\n' "bash" ;;
    esac
}

detect_os() {
    case "$(uname -s)" in
        Darwin) printf '%s\n' "darwin" ;;
        Linux) printf '%s\n' "linux" ;;
        *) echo "unsupported OS" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        arm64|aarch64) printf '%s\n' "arm64" ;;
        x86_64|amd64) printf '%s\n' "amd64" ;;
        *) echo "unsupported architecture" >&2; exit 1 ;;
    esac
}

SHELL_TYPE=$(detect_shell)
OS_NAME=$(detect_os)
ARCH_NAME=$(detect_arch)
BIN_DIR="${AUTOCOMPLETE_BIN_DIR:-$HOME/.local/bin}"
STATE_DIR="${AUTOCOMPLETE_HOME:-$HOME/.autocomplete}"
BIN_PATH="$BIN_DIR/autocomplete"
ALIAS_PATH="$BIN_DIR/acsh"
RC_FILE="$HOME/.${SHELL_TYPE}rc"

case "$SHELL_TYPE" in
    zsh)
        ADAPTER_SOURCE="$REPO_ROOT/autocomplete.zsh"
        ADAPTER_TARGET="$STATE_DIR/shell/autocomplete.zsh"
        ;;
    *)
        ADAPTER_SOURCE="$REPO_ROOT/autocomplete.sh"
        ADAPTER_TARGET="$STATE_DIR/shell/autocomplete.sh"
        ;;
esac

mkdir -p "$BIN_DIR" "$STATE_DIR/shell"

if [ "$VERSION" = "dev" ]; then
    GO_BIN="${GO_BIN:-go}"
    (
        cd "$REPO_ROOT"
        "$GO_BIN" build -o "$BIN_PATH" ./cmd/autocomplete
    )
else
    if [ "$VERSION" = "latest" ]; then
        ARCHIVE_URL="https://github.com/$REPO_SLUG/releases/latest/download/autocomplete_${OS_NAME}_${ARCH_NAME}.tar.gz"
    else
        ARCHIVE_URL="https://github.com/$REPO_SLUG/releases/download/$VERSION/autocomplete_${OS_NAME}_${ARCH_NAME}.tar.gz"
    fi
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT INT TERM
    curl -fsSL "$ARCHIVE_URL" | tar -xz -C "$TMP_DIR"
    cp "$TMP_DIR/autocomplete" "$BIN_PATH"
fi

cp "$ADAPTER_SOURCE" "$ADAPTER_TARGET"
chmod +x "$BIN_PATH"
ln -sf "$BIN_PATH" "$ALIAS_PATH"

MARKER_BEGIN="# >>> autocomplete-sh >>>"
MARKER_END="# <<< autocomplete-sh <<<"
if ! grep -q "$MARKER_BEGIN" "$RC_FILE" 2>/dev/null; then
    {
        printf '\n%s\n' "$MARKER_BEGIN"
        printf 'export PATH="%s:$PATH"\n' "$BIN_DIR"
        printf '. "%s"\n' "$ADAPTER_TARGET"
        printf '%s\n' "$MARKER_END"
    } >> "$RC_FILE"
fi

printf 'Installed binary: %s\n' "$BIN_PATH"
printf 'Installed alias: %s -> %s\n' "$ALIAS_PATH" "$BIN_PATH"
printf 'Installed %s adapter: %s\n' "$SHELL_TYPE" "$ADAPTER_TARGET"
printf 'Next step: source %s\n' "$RC_FILE"
