#!/usr/bin/env zsh

autoload -Uz compinit
compinit -u

_autocomplete_bin() {
    if [[ -n "${AUTOCOMPLETE_BIN:-}" ]]; then
        print -r -- "$AUTOCOMPLETE_BIN"
        return 0
    fi

    command -v autocomplete 2>/dev/null
}

_autocomplete_history() {
    local limit="${AUTOCOMPLETE_MAX_HISTORY:-20}"
    fc -ln -"$limit" 2>/dev/null || true
}

_autocomplete_recent_files() {
    local limit="${AUTOCOMPLETE_MAX_RECENT_FILES:-20}"
    command ls -1tp 2>/dev/null | sed '/\/$/d' | head -n "$limit"
}

_autocomplete_help() {
    local line="$1"
    local command="${line%% *}"

    if [[ -z "$command" ]]; then
        return 0
    fi

    "$command" --help 2>/dev/null | head -n 40 || true
}

_autocomplete_env() {
    print -r -- "USER=${USER:-}"
    print -r -- "HOME=${HOME:-}"
    print -r -- "SHELL=${SHELL:-}"
    print -r -- "TERM=${TERM:-}"
    print -r -- "HOSTNAME=${HOST:-${HOSTNAME:-}}"
}

_autocomplete_request_suggestions() {
    local line="$1"
    local cwd="${2:-$PWD}"
    local binary

    binary="$(_autocomplete_bin)" || return 1
    AUTOCOMPLETE_SHELL="zsh" \
    AUTOCOMPLETE_LINE="$line" \
    AUTOCOMPLETE_CWD="$cwd" \
    AUTOCOMPLETE_HISTORY="$(_autocomplete_history)" \
    AUTOCOMPLETE_RECENT_FILES="$(_autocomplete_recent_files)" \
    AUTOCOMPLETE_HELP="$(_autocomplete_help "$line")" \
    AUTOCOMPLETE_ENV="$(_autocomplete_env)" \
    "$binary" complete --shell zsh --line "$line" --cwd "$cwd" --plain
}

_autocomplete_complete() {
    local suggestions
    local -a items

    if [[ "${compstate[list]:-}" != *force* ]]; then
        return 1
    fi

    suggestions="$(_autocomplete_request_suggestions "${BUFFER:-${(j: :)words}}" "$PWD" 2>/dev/null)" || return 1
    [[ -n "$suggestions" ]] || return 1

    items=("${(@f)suggestions}")
    (( ${#items[@]} > 0 )) || return 1
    compadd -- "${items[@]}"
}

autocomplete_enable() {
    compdef _autocomplete_complete -default-
}

autocomplete_disable() {
    unfunction _autocomplete_complete 2>/dev/null || true
}

autocomplete_enable
