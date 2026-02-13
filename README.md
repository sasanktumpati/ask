# ask

`ask` is a Go CLI to query LLMs from your terminal with SDK-free HTTP clients.

## Build

```bash
go build -o ask .
```

## Quick Start

1. Run `ask --help` once. If missing, `config.json` and `config.template.json` are created automatically.
2. Set a provider key:

```bash
ask key set openai --value sk-...
```

3. Set default provider and model:

```bash
ask provider set openai
ask models select --provider openai --search mini
```

4. Ask:

```bash
ask "command to remove a commit from git"
```

## Command Prefill UX

If the model returns `command`, `ask` opens an editable terminal prompt with the command prefilled.

- `Enter`: run command
- Edit text first: run edited command
- `Ctrl+C`: copy suggested command to clipboard and exit prompt
- `Ctrl+D`: exit prompt without running

## Core Commands

```bash
ask "question" [options]
ask models list|select|set|current
ask provider list|current|set|show|add|remove
ask key set|show|clear
ask config show|path|template
ask markdown on|off|status
ask help ask|models|provider|key|config|markdown
```

## Ask Options

- `-p, --provider <name>`
- `-m, --model <id>`
- `--timeout <dur|sec>` (default: `90s`)
- `--no-markdown`
- `--no-run`
- `--json`

If your question starts with `-`, use:

```bash
ask -- "-question that starts with dash"
```

## Providers

Built-in providers:

- `openai`
- `anthropic`
- `gemini`
- `ollama`
- `openrouter`

Add a custom OpenAI-compatible provider:

```bash
ask provider add myproxy \
  --base-url https://llm.example.com/v1 \
  --api-key-env MYPROXY_API_KEY
```

## Config

Default config path:

- macOS/Linux: `~/.ask/config.json`
- Windows: `%USERPROFILE%\.ask\config.json`

Overrides:

- `ASK_CONFIG=/path/to/config.json`
- `ASK_CONFIG_DIR=/path/to/config/dir`
- `ask --config /path/to/config.json ...`

Security defaults:

- config directory mode: `0700`
- config file mode: `0600`

API key resolution order:

1. Environment variable from `api_key_env` (or built-in default env var)
2. `api_key` in `config.json`

Show active paths:

```bash
ask config path
ask config template
```

Show raw config content:

```bash
ask config show
```

## Config Template (Example)

```json
{
  "version": 1,
  "current_provider": "",
  "providers": {
    "openai": {
      "api_key": "",
      "model": "gpt-5-nano",
      "api_key_env": "OPENAI_API_KEY"
    },
    "anthropic": {
      "api_key": "",
      "model": "",
      "api_key_env": "ANTHROPIC_API_KEY"
    },
    "gemini": {
      "api_key": "",
      "model": "",
      "api_key_env": "GEMINI_API_KEY"
    },
    "ollama": {
      "api_key": "",
      "model": "",
      "base_url": "http://127.0.0.1:11434"
    },
    "openrouter": {
      "api_key": "",
      "model": "",
      "api_key_env": "OPENROUTER_API_KEY"
    }
  },
  "custom_providers": {
    "myproxy": {
      "base_url": "https://llm.example.com/v1",
      "api_key": "",
      "model": "",
      "api_key_env": "MYPROXY_API_KEY",
      "headers": {
        "X-Client-Name": "ask"
      }
    }
  },
  "render_markdown": true
}
```

## Notes

- Model lists are fetched from provider APIs
- Responses are requested in structured JSON (`answer`, `command`) with fallback parsing.
- Markdown rendering uses `charmbracelet/glamour`.

## Development

```bash
go test ./...
go vet ./...
go build ./...
```
