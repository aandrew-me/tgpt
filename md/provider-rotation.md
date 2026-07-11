### Provider Rotation

Standalone tool that tries multiple providers in order — if one fails, the next is tried automatically. Does **not** modify tgpt itself; runs it as a subprocess.

#### Build

```bash
go build -o rotate/rotate ./rotate/
```

#### Env Vars

| Var | Required | Description |
|-----|----------|-------------|
| `ROTATE_PROVIDERS` | ✅ | Comma-separated provider list, tried in order |
| `ROTATE_MODEL` | ❌ | Logical model name (e.g. `deepseek-v4-flash`) |
| `ROTATE_ALIAS_FILE` | ❌ | Path to JSON alias file |
| `ROTATE_ALIAS_JSON` | ❌ | Inline JSON alias string |
| `MODEL_ALIAS_<PROVIDER>` | ❌ | Per-provider model override (highest priority) |

#### Usage

```bash
# Basic: try opencode, fallback to anyapi
ROTATE_PROVIDERS="opencode,anyapi" ./rotate/rotate "your prompt"

# With logical model name
ROTATE_PROVIDERS="opencode,anyapi" ROTATE_MODEL="deepseek-v4-flash" ./rotate/rotate "your prompt"

# With alias JSON file
ROTATE_ALIAS_FILE="md/deepseek.json" ROTATE_PROVIDERS="opencode,anyapi" ./rotate/rotate "your prompt"

# With inline alias JSON
ROTATE_ALIAS_JSON='{"model_alias":{"opencode":"deepseek-v4-flash-free","anyapi":"deepseek/deepseek-v4-flash"}}' \
  ROTATE_PROVIDERS="opencode,anyapi" ./rotate/rotate "your prompt"

# With per-provider env alias (highest priority)
MODEL_ALIAS_OPENCODE="deepseek-v4-flash-free" \
  ROTATE_PROVIDERS="opencode,anyapi" \
  ROTATE_MODEL="deepseek-v4-flash" \
  ./rotate/rotate "your prompt"
```

#### Alias Precedence

`MODEL_ALIAS_<PROVIDER>` env > `ROTATE_ALIAS_JSON` > `ROTATE_ALIAS_FILE` > `ROTATE_MODEL`

#### Alias JSON Format

```json
{
  "model_alias": {
    "anyapi": "deepseek/deepseek-v4-flash",
    "opencode": "deepseek-v4-flash-free"
  }
}
```

#### How It Works

1. `ROTATE_PROVIDERS` is split by comma into a provider list
2. For each provider, `tgpt --provider <name> --model <model> <prompt>` is spawned
3. If `tgpt` exits with non-zero, the error is logged and the next provider is tried
4. If `tgpt` exits zero, output is printed and the tool exits successfully
5. If all providers fail, the tool exits with code 1

#### Reference

- [`deepseek.json`](./deepseek.json) — alias mapping for deepseek models across providers
- [`anyapi` provider](../src/providers/anyapi/) — multi-model API with 100k free tokens/day
- [`opencode` provider](../src/providers/opencode/) — free OpenAI-compatible API
