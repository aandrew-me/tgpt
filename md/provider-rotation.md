### Provider Rotation

Built into tgpt. Tries multiple providers in order — if one fails, the next is tried automatically.

#### Usage

```bash
# Via --rotate flag
tgpt --rotate "anyapi,opencode,deepseek" "your prompt"

# Via env var
AI_ROTATE_PROVIDERS="anyapi,opencode,deepseek" tgpt "your prompt"
```

If all providers fail, the last error is shown and the program exits.

#### Model Aliases

Same model, different names across providers. Set `MODEL_ALIAS_<PROVIDER>` (uppercase) to map:

```bash
MODEL_ALIAS_ANYAPI="deepseek/deepseek-v4-flash" \
MODEL_ALIAS_OPENCODE="deepseek-v4-flash-free" \
tgpt --rotate "anyapi,opencode" "your prompt"
```

When a provider in the rotation is tried, tgpt checks for `MODEL_ALIAS_<PROVIDER>` and overrides `--model` with it.

#### Model Mapping Reference

See [`deepseek.json`](./deepseek.json) for an alias example.

#### How It Works

1. `--rotate` flag or `AI_ROTATE_PROVIDERS` env var lists providers
2. Invalid/unregistered providers are silently skipped
3. Each provider is tried in order
4. On connection error or HTTP 4xx/5xx, logs the error and tries the next
5. On success, prints normally (shows "Fell back to <provider>" after fallback)
6. If all fail, exits with the last error
