package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MigrationCandidate represents an environment variable that can be migrated
type MigrationCandidate struct {
	EnvVar      string
	ConfigPath  string
	Value       string
	Description string
}

// DetectMigrationCandidates finds environment variables that can be migrated to config
func DetectMigrationCandidates() ([]MigrationCandidate, error) {
	var candidates []MigrationCandidate

	// Define known environment variables and their config mappings
	envMappings := map[string]struct {
		configPath  string
		description string
	}{
		// Provider settings
		"AI_PROVIDER":  {"defaults.provider", "Default provider for chat"},
		"IMG_PROVIDER": {"image.default_provider", "Default provider for image generation"},

		// Model parameters
		"TGPT_TEMPERATURE": {"defaults.temperature", "Default temperature setting"},
		"TGPT_TOP_P":       {"defaults.top_p", "Default top_p setting"},

		// Provider-specific API keys
		"AI_API_KEY":       {"providers.generic.api_key", "Generic API key"},
		"OPENAI_API_KEY":   {"providers.openai.api_key", "OpenAI API key"},
		"CEREBRAS_API_KEY": {"providers.cerebras.api_key", "Cerebras API key"},
		"DEEPSEEK_API_KEY": {"providers.deepseek.api_key", "DeepSeek API key"},
		"GEMINI_API_KEY":   {"providers.gemini.api_key", "Google Gemini API key"},
		"GROQ_API_KEY":     {"providers.groq.api_key", "Groq API key"},
		"KIMI_API_KEY":     {"providers.kimi.api_key", "Kimi API key"},

		// Provider-specific models
		"OPENAI_MODEL":   {"providers.openai.model", "OpenAI model name"},
		"CEREBRAS_MODEL": {"providers.cerebras.model", "Cerebras model name"},
		"DEEPSEEK_MODEL": {"providers.deepseek.model", "DeepSeek model name"},
		"GEMINI_MODEL":   {"providers.gemini.model", "Google Gemini model name"},
		"GROQ_MODEL":     {"providers.groq.model", "Groq model name"},

		// Provider-specific URLs
		"OPENAI_URL":        {"providers.openai.url", "OpenAI base URL"},
		"CEREBRAS_BASE_URL": {"providers.cerebras.url", "Cerebras base URL"},
		"OLLAMA_URL":        {"providers.ollama.url", "Ollama base URL"},

		// Search configuration
		"TGPT_GOOGLE_API_KEY":          {"search.google_api_key", "Google Custom Search API key"},
		"TGPT_GOOGLE_SEARCH_ENGINE_ID": {"search.google_search_engine_id", "Google Custom Search Engine ID"},
	}

	// Check each environment variable
	for envVar, mapping := range envMappings {
		value := os.Getenv(envVar)
		if value != "" {
			candidates = append(candidates, MigrationCandidate{
				EnvVar:      envVar,
				ConfigPath:  mapping.configPath,
				Value:       value,
				Description: mapping.description,
			})
		}
	}

	return candidates, nil
}

// MigrateFromEnv creates a configuration from current environment variables
func MigrateFromEnv() (*Config, error) {
	config := GetDefaultConfig()
	candidates, err := DetectMigrationCandidates()
	if err != nil {
		return nil, err
	}

	// Apply migrations
	for _, candidate := range candidates {
		if err := applyMigration(config, candidate); err != nil {
			return nil, fmt.Errorf("failed to migrate %s: %w", candidate.EnvVar, err)
		}
	}

	// Set up provider configurations based on available API keys
	setupProviderConfigs(config, candidates)

	return config, nil
}

// applyMigration applies a single migration to the configuration
func applyMigration(config *Config, candidate MigrationCandidate) error {
	pathParts := strings.Split(candidate.ConfigPath, ".")
	if len(pathParts) < 2 {
		return fmt.Errorf("invalid config path: %s", candidate.ConfigPath)
	}

	section := pathParts[0]
	key := strings.Join(pathParts[1:], ".")

	switch section {
	case "defaults":
		return applyDefaultsMigration(config, key, candidate.Value)
	case "providers":
		return applyProviderMigration(config, pathParts[1], strings.Join(pathParts[2:], "."), candidate.Value, candidate.EnvVar)
	case "image":
		return applyImageMigration(config, key, candidate.Value)
	case "search":
		return applySearchMigration(config, key, candidate.Value)
	default:
		return fmt.Errorf("unsupported config section: %s", section)
	}
}

// applyDefaultsMigration migrates default settings
func applyDefaultsMigration(config *Config, key, value string) error {
	switch key {
	case "provider":
		config.Defaults.Provider = value
	case "temperature":
		temp, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid temperature value: %s", value)
		}
		config.Defaults.Temperature = temp
	case "top_p":
		topP, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid top_p value: %s", value)
		}
		config.Defaults.TopP = topP
	default:
		return fmt.Errorf("unsupported defaults key: %s", key)
	}
	return nil
}

// applyProviderMigration migrates provider-specific settings
func applyProviderMigration(config *Config, providerName, key, value, envVar string) error {
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}

	provider := config.Providers[providerName]

	// Set provider type if not already set
	if provider.Type == "" {
		provider.Type = getProviderType(providerName)
	}

	switch key {
	case "api_key":
		// Keep as environment variable reference for security
		provider.APIKey = fmt.Sprintf("${%s}", envVar)
	case "model":
		provider.Model = value
	case "url":
		provider.URL = value
	default:
		return fmt.Errorf("unsupported provider key: %s", key)
	}

	config.Providers[providerName] = provider
	return nil
}

// applyImageMigration migrates image generation settings
func applyImageMigration(config *Config, key, value string) error {
	switch key {
	case "default_provider":
		config.Image.DefaultProvider = value
	default:
		return fmt.Errorf("unsupported image key: %s", key)
	}
	return nil
}

// applySearchMigration migrates search settings
func applySearchMigration(config *Config, key, value string) error {
	switch key {
	case "google_api_key":
		config.Search.GoogleAPIKey = "${TGPT_GOOGLE_API_KEY}"
	case "google_search_engine_id":
		config.Search.GoogleSearchEngineID = "${TGPT_GOOGLE_SEARCH_ENGINE_ID}"
	default:
		return fmt.Errorf("unsupported search key: %s", key)
	}
	return nil
}

// setupProviderConfigs sets up provider configurations based on detected environment variables
func setupProviderConfigs(config *Config, candidates []MigrationCandidate) {
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}

	// Track which providers have API keys
	providersWithKeys := make(map[string]bool)
	for _, candidate := range candidates {
		if strings.Contains(candidate.ConfigPath, ".api_key") {
			providerName := strings.Split(candidate.ConfigPath, ".")[1]
			providersWithKeys[providerName] = true
		}
	}

	// Set up complete provider configurations
	providerDefaults := map[string]struct {
		typ   string
		model string
		url   string
	}{
		"openai": {
			typ:   "openai",
			model: "gpt-4",
			url:   "",
		},
		"cerebras": {
			typ:   "openai",
			model: "qwen-3-coder-480b",
			url:   "https://api.cerebras.ai/v1/chat/completions",
		},
		"deepseek": {
			typ:   "deepseek",
			model: "deepseek-reasoner",
			url:   "",
		},
		"gemini": {
			typ:   "gemini",
			model: "gemini-pro",
			url:   "",
		},
		"groq": {
			typ:   "groq",
			model: "llama3-8b-8192",
			url:   "",
		},
	}

	for providerName, hasKey := range providersWithKeys {
		if hasKey {
			provider := config.Providers[providerName]

			// Set defaults if not already configured
			if defaults, exists := providerDefaults[providerName]; exists {
				if provider.Type == "" {
					provider.Type = defaults.typ
				}
				if provider.Model == "" {
					provider.Model = defaults.model
				}
				if provider.URL == "" {
					provider.URL = defaults.url
				}
			}

			config.Providers[providerName] = provider
		}
	}
}

// getProviderType maps provider names to their internal types
func getProviderType(providerName string) string {
	typeMap := map[string]string{
		"openai":       "openai",
		"cerebras":     "openai", // Uses OpenAI-compatible API
		"deepseek":     "deepseek",
		"gemini":       "gemini",
		"groq":         "groq",
		"ollama":       "ollama",
		"kimi":         "kimi",
		"phind":        "phind",
		"sky":          "sky",
		"isou":         "isou",
		"duckduckgo":   "duckduckgo",
		"koboldai":     "koboldai",
		"pollinations": "pollinations",
	}

	if typ, exists := typeMap[providerName]; exists {
		return typ
	}

	return providerName // Default to provider name
}

// GenerateMigrationReport creates a human-readable migration report
func GenerateMigrationReport(candidates []MigrationCandidate) string {
	if len(candidates) == 0 {
		return "No environment variables found that can be migrated to configuration file."
	}

	var report strings.Builder
	report.WriteString("Environment Variables Available for Migration:\n")
	report.WriteString("========================================\n\n")

	// Group by category
	categories := map[string][]MigrationCandidate{
		"Provider Selection":   {},
		"API Keys":             {},
		"Model Configuration":  {},
		"URLs":                 {},
		"Model Parameters":     {},
		"Search Configuration": {},
	}

	for _, candidate := range candidates {
		switch {
		case strings.Contains(candidate.EnvVar, "PROVIDER"):
			categories["Provider Selection"] = append(categories["Provider Selection"], candidate)
		case strings.Contains(candidate.EnvVar, "API_KEY"):
			categories["API Keys"] = append(categories["API Keys"], candidate)
		case strings.Contains(candidate.EnvVar, "MODEL"):
			categories["Model Configuration"] = append(categories["Model Configuration"], candidate)
		case strings.Contains(candidate.EnvVar, "URL"):
			categories["URLs"] = append(categories["URLs"], candidate)
		case strings.Contains(candidate.EnvVar, "TEMPERATURE") || strings.Contains(candidate.EnvVar, "TOP_P"):
			categories["Model Parameters"] = append(categories["Model Parameters"], candidate)
		case strings.Contains(candidate.EnvVar, "GOOGLE"):
			categories["Search Configuration"] = append(categories["Search Configuration"], candidate)
		}
	}

	for category, items := range categories {
		if len(items) > 0 {
			report.WriteString(fmt.Sprintf("%s:\n", category))
			for _, item := range items {
				report.WriteString(fmt.Sprintf("  %s = %s\n", item.EnvVar, item.Value))
				report.WriteString(fmt.Sprintf("    → %s\n", item.Description))
				report.WriteString(fmt.Sprintf("    Config path: %s\n", item.ConfigPath))
				report.WriteString("\n")
			}
			report.WriteString("\n")
		}
	}

	report.WriteString("Migration Benefits:\n")
	report.WriteString("- Centralized configuration management\n")
	report.WriteString("- Reduced environment variable conflicts\n")
	report.WriteString("- Profile support for different use cases\n")
	report.WriteString("- Environment variable expansion in config file\n")
	report.WriteString("- Better documentation and validation\n")

	return report.String()
}

// BackupEnvironment creates a backup script to restore environment variables
func BackupEnvironment(candidates []MigrationCandidate) string {
	if len(candidates) == 0 {
		return ""
	}

	var backup strings.Builder
	backup.WriteString("#!/bin/bash\n")
	backup.WriteString("# Backup of environment variables for TGPT\n")
	backup.WriteString("# Generated by tgpt config migrate\n\n")

	for _, candidate := range candidates {
		backup.WriteString(fmt.Sprintf("export %s=\"%s\"\n", candidate.EnvVar, candidate.Value))
	}

	return backup.String()
}
