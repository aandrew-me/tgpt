package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	ModelAlias map[string]string `json:"model_alias"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: rotate <prompt>\n")
		fmt.Fprintf(os.Stderr, "Env: ROTATE_PROVIDERS (comma-separated, e.g. anyapi,opencode,deepseek)\n")
		fmt.Fprintf(os.Stderr, "Env: ROTATE_MODEL (logical model name, e.g. deepseek-v4-flash)\n")
		fmt.Fprintf(os.Stderr, "Env: ROTATE_ALIAS_FILE (path to alias JSON, e.g. md/deepseek.json)\n")
		os.Exit(1)
	}

	prompt := os.Args[1]
	providers := strings.Split(os.Getenv("ROTATE_PROVIDERS"), ",")
	if len(providers) == 0 || (len(providers) == 1 && providers[0] == "") {
		fmt.Fprintln(os.Stderr, "ROTATE_PROVIDERS env var is required (comma-separated)")
		os.Exit(1)
	}

	model := os.Getenv("ROTATE_MODEL")
	aliasFile := os.Getenv("ROTATE_ALIAS_FILE")
	aliasJSON := os.Getenv("ROTATE_ALIAS_JSON")

	aliases := loadAliases(aliasFile)
	if aliases == nil && aliasJSON != "" {
		var cfg Config
		if err := json.Unmarshal([]byte(aliasJSON), &cfg); err == nil {
			aliases = cfg.ModelAlias
		}
	}

	for i, provider := range providers {
		provider = strings.TrimSpace(provider)
		if provider == "" {
			continue
		}

		args := []string{"--provider", provider}

		providerModel := providerModelName(provider, model, aliases)
		if providerModel != "" {
			args = append(args, "--model", providerModel)
		}

		args = append(args, prompt)

		tgptPath := findTgpt()
		cmd := exec.Command(tgptPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if i > 0 {
			fmt.Fprintf(os.Stderr, "\rFalling back to %s...\n", provider)
		}

		err := cmd.Run()
		if err == nil {
			return
		}

		fmt.Fprintf(os.Stderr, "\rProvider %s failed, trying next...\n", provider)
	}

	fmt.Fprintln(os.Stderr, "All providers failed")
	os.Exit(1)
}

func findTgpt() string {
	if _, err := os.Stat("./tgpt"); err == nil {
		return "./tgpt"
	}
	if p, err := exec.LookPath("tgpt"); err == nil {
		return p
	}
	return "tgpt"
}

func loadAliases(path string) map[string]string {
	if path == "" {
		return nil
	}

	absPath := path
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		absPath = filepath.Join(wd, path)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	return cfg.ModelAlias
}

func providerModelName(provider, model string, aliases map[string]string) string {
	if model == "" {
		return ""
	}

	key := strings.ToUpper(provider)
	if env := os.Getenv("MODEL_ALIAS_" + key); env != "" {
		return env
	}

	if aliases != nil {
		if alias, ok := aliases[provider]; ok {
			return alias
		}
	}

	return model
}
