#!/usr/bin/env bash

_autocomplete_bin() {
    if [[ -n "${AUTOCOMPLETE_BIN:-}" ]]; then
        printf '%s\n' "$AUTOCOMPLETE_BIN"
        return 0
    fi

    command -v autocomplete 2>/dev/null
}

_autocomplete_history() {
    local limit="${AUTOCOMPLETE_MAX_HISTORY:-20}"
    history 2>/dev/null | tail -n "$limit" | sed -E 's/^[[:space:]]*[0-9]+[[:space:]]+//'
}

_autocomplete_recent_files() {
    local limit="${AUTOCOMPLETE_MAX_RECENT_FILES:-20}"
    command ls -1tp 2>/dev/null | sed '/\/$/d' | head -n "$limit"
}

_autocomplete_help() {
    local line="$1"
    local command

    command="${line%% *}"
    if [[ -z "$command" ]]; then
        return 0
    fi

    "$command" --help 2>/dev/null | head -n 40 || true
}

_autocomplete_env() {
    printf 'USER=%s\n' "${USER:-}"
    printf 'HOME=%s\n' "${HOME:-}"
    printf 'SHELL=%s\n' "${SHELL:-}"
    printf 'TERM=%s\n' "${TERM:-}"
    printf 'HOSTNAME=%s\n' "${HOSTNAME:-}"
}

_autocomplete_request_suggestions() {
    local line="$1"
    local cwd="${2:-$PWD}"
    local binary

    binary="$(_autocomplete_bin)" || return 1
    AUTOCOMPLETE_SHELL="bash" \
    AUTOCOMPLETE_LINE="$line" \
    AUTOCOMPLETE_CWD="$cwd" \
    AUTOCOMPLETE_HISTORY="$(_autocomplete_history)" \
    AUTOCOMPLETE_RECENT_FILES="$(_autocomplete_recent_files)" \
    AUTOCOMPLETE_HELP="$(_autocomplete_help "$line")" \
    AUTOCOMPLETE_ENV="$(_autocomplete_env)" \
    "$binary" complete --shell bash --line "$line" --cwd "$cwd" --plain
}

_autocomplete_complete() {
    local suggestions suggestion

    COMPREPLY=()
    if [[ "${COMP_TYPE:-0}" -ne 63 ]]; then
        return 1
    fi

    suggestions="$(_autocomplete_request_suggestions "${COMP_LINE:-}" "$PWD" 2>/dev/null)" || return 1
    if [[ -z "$suggestions" ]]; then
        return 1
    fi

    while IFS= read -r suggestion; do
        [[ -n "$suggestion" ]] || continue
        COMPREPLY[${#COMPREPLY[@]}]="$suggestion"
    done <<< "$suggestions"

    [[ ${#COMPREPLY[@]} -gt 0 ]]
}

autocomplete_enable() {
    if help complete 2>/dev/null | grep -q -- '\-D'; then
        complete -D -o default -o bashdefault -F _autocomplete_complete
    fi
}

autocomplete_disable() {
    if help complete 2>/dev/null | grep -q -- '\-D'; then
        complete -r -D 2>/dev/null || true
    fi
}

autocomplete_enable
