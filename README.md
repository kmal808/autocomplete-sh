# autocomplete-sh

`autocomplete-sh`, or `acsh` for short, is an AI-assisted terminal autocomplete
library and CLI with thin shell adapters for `bash` and `zsh`.

Canonical repo: `https://github.com/kmal808/autocomplete-sh`

Forked from `closedloop-technologies/autocomplete-sh`; rewritten and maintained
here as an independent project. See [NOTICE](./NOTICE) for provenance.

Suggested GitHub repo settings for description, homepage, and topics live in
[REPO_METADATA.md](./REPO_METADATA.md). Release steps live in
[RELEASING.md](./RELEASING.md).

## Architecture

The project has three explicit layers:

- a Go library for prompt construction, provider calls, and caching
- a stable `autocomplete` CLI, with `acsh` installed as a shorthand alias
- sourceable shell adapters in `autocomplete.sh` and `autocomplete.zsh`

The initial supported providers are `OpenAI` and `Ollama`.

## Install

Published install path:

```bash
curl -fsSL https://raw.githubusercontent.com/kmal808/autocomplete-sh/main/docs/install.sh | sh
```

Development install from a local checkout:

```bash
./docs/install.sh dev
```

That builds `autocomplete`, creates the `acsh` alias, installs the matching
shell adapter, and appends a marked block to your shell rc file.

## Build

```bash
go build ./cmd/autocomplete
```

## CLI

```bash
autocomplete complete --shell bash --line "git sta" --cwd "$PWD" --json
acsh prompt --shell zsh --line "ffmpeg # shrink for email" --cwd "$PWD"
autocomplete config init
autocomplete config show
autocomplete config set model gpt-4o-mini
autocomplete doctor
autocomplete cache clear
autocomplete version
```

The shell adapters pass richer context through environment variables:

- `AUTOCOMPLETE_HISTORY`
- `AUTOCOMPLETE_RECENT_FILES`
- `AUTOCOMPLETE_HELP`
- `AUTOCOMPLETE_ENV`

## Configuration

Config lives at:

- `$AUTOCOMPLETE_CONFIG` if set
- otherwise `$XDG_CONFIG_HOME/autocomplete-sh/config.json`
- otherwise your platform config dir, for example
  `~/Library/Application Support/autocomplete-sh/config.json` on macOS

Environment variables override file values.

Common values:

- `OPENAI_API_KEY`
- `OLLAMA_HOST`
- `AUTOCOMPLETE_PROVIDER`
- `AUTOCOMPLETE_MODEL`
- `AUTOCOMPLETE_TIMEOUT`
- `AUTOCOMPLETE_CACHE_DIR`
- `AUTOCOMPLETE_CACHE_TTL`
- `AUTOCOMPLETE_CACHE_SIZE`

## Release Artifacts

Tagged releases are expected to publish tarballs named:

- `autocomplete_darwin_arm64.tar.gz`
- `autocomplete_darwin_amd64.tar.gz`
- `autocomplete_linux_arm64.tar.gz`
- `autocomplete_linux_amd64.tar.gz`

Those are the artifacts consumed by `docs/install.sh`.

## CI

GitHub Actions runs:

- `.github/workflows/ci.yml` on pushes and pull requests
- `.github/workflows/release.yml` on version tags

## Tests

```bash
go test ./...
```
