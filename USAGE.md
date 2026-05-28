# Usage

`autocomplete` is the primary command. `acsh` is installed as a shorthand alias.

## Install

Release install:

```bash
curl -fsSL https://raw.githubusercontent.com/kmal808/autocomplete-sh/main/docs/install.sh | sh
```

Local development install:

```bash
./docs/install.sh dev
```

## Initialize Config

```bash
autocomplete config init
autocomplete config show
```

## Add Your API Key

`OPENAI_API_KEY` is read from the environment, not stored in `config.json`.

```bash
export OPENAI_API_KEY='your-key-here'
autocomplete doctor
```

## Ask For Suggestions

```bash
autocomplete complete --shell bash --line "git sta" --cwd "$PWD" --plain
acsh complete --shell zsh --line "docker bu" --cwd "$PWD" --json
```

## Inspect The Prompt

```bash
autocomplete prompt --shell bash --line "ffmpeg # shrink for email" --cwd "$PWD"
```

## Tune Config

```bash
autocomplete config set provider openai
autocomplete config set model gpt-4o-mini
autocomplete config set timeout 10s
autocomplete config get model
```

## Verify Environment

```bash
autocomplete doctor
autocomplete version
```

## Shell Adapters

Source the adapter for your shell:

```bash
source ~/.autocomplete/shell/autocomplete.sh
source ~/.autocomplete/shell/autocomplete.zsh
```
