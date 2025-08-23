package config

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/BurntSushi/toml"
)

// Helper function to create a pointer to a float64 value
func floatPtr(f float64) *float64 {
	return &f
}

// loadConfigFromFile loads configuration from a TOML file
func loadConfigFromFile(configPath string) (*Config, error) {
	// Ensure the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}
	
	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse TOML
	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Initialize maps if they're nil
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}
	if config.Modes == nil {
		config.Modes = make(map[string]ModeConfig)
	}
	if config.Profiles == nil {
		config.Profiles = make(map[string]ProfileConfig)
	}
	
	// Validate the configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Expand environment variables in the configuration
	if err := expandEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}
	
	return &config, nil
}

// SaveConfig saves the configuration to a TOML file
func SaveConfig(config *Config, configPath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create a temporary file first for atomic write
	tmpPath := configPath + ".tmp"
	
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	defer file.Close()
	
	// Encode to TOML
	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to save config file: %w", err)
	}
	
	return nil
}

// validateConfig validates the configuration for correctness
func validateConfig(config *Config) error {
	// Validate temperature range
	if config.Defaults.Temperature < 0 || config.Defaults.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2, got: %f", config.Defaults.Temperature)
	}
	
	// Validate top_p range
	if config.Defaults.TopP < 0 || config.Defaults.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1, got: %f", config.Defaults.TopP)
	}
	
	// Validate providers
	for name, provider := range config.Providers {
		if provider.Type == "" {
			return fmt.Errorf("provider '%s' must have a type", name)
		}
		
		// Validate supported provider types
		validTypes := []string{"openai", "gemini", "deepseek", "groq", "ollama", "phind", "kimi", "sky", "isou", "duckduckgo", "koboldai", "pollinations"}
		if !contains(validTypes, provider.Type) {
			return fmt.Errorf("provider '%s' has unsupported type '%s'", name, provider.Type)
		}
	}
	
	// Validate profiles reference valid providers
	for profileName, profile := range config.Profiles {
		if profile.Provider != "" {
			if _, exists := config.Providers[profile.Provider]; !exists {
				// Check if it's a built-in provider
				builtinProviders := []string{"openai", "gemini", "deepseek", "groq", "ollama", "phind", "kimi", "sky", "isou", "duckduckgo", "koboldai", "pollinations"}
				if !contains(builtinProviders, profile.Provider) {
					return fmt.Errorf("profile '%s' references undefined provider '%s'", profileName, profile.Provider)
				}
			}
		}
		
		// Validate profile temperature and top_p ranges
		if profile.Temperature != nil {
			temp := *profile.Temperature
			if temp < 0 || temp > 2 {
				return fmt.Errorf("profile '%s' temperature must be between 0 and 2, got: %f", profileName, temp)
			}
		}
		
		if profile.TopP != nil {
			topP := *profile.TopP
			if topP < 0 || topP > 1 {
				return fmt.Errorf("profile '%s' top_p must be between 0 and 1, got: %f", profileName, topP)
			}
		}
	}
	
	// Validate image settings
	if config.Image.Width <= 0 || config.Image.Height <= 0 {
		return fmt.Errorf("image dimensions must be positive")
	}
	
	// Validate search provider
	validSearchProviders := []string{"is-fast", "firecrawl", "google"}
	if config.Defaults.SearchProvider != "" && !contains(validSearchProviders, config.Defaults.SearchProvider) {
		return fmt.Errorf("invalid search provider '%s'", config.Defaults.SearchProvider)
	}
	
	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// InitConfig creates a new configuration file with default values
func InitConfig(configPath string) error {
	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configPath)
	}
	
	// Create default config
	config := GetDefaultConfig()
	
	// Add example provider configurations
	config.Providers = map[string]ProviderConfig{
		"cerebras": {
			Type:      "openai",
			APIKey:    "${CEREBRAS_API_KEY}",
			Model:     "qwen-3-coder-480b",
			URL:       "https://api.cerebras.ai/v1/chat/completions",
			IsDefault: false,
		},
		"openai": {
			Type:   "openai",
			APIKey: "${OPENAI_API_KEY}",
			Model:  "gpt-4",
			URL:    "",
		},
		"gemini": {
			Type:   "gemini",
			APIKey: "${GEMINI_API_KEY}",
			Model:  "gemini-pro",
		},
		"deepseek": {
			Type:   "deepseek",
			APIKey: "${DEEPSEEK_API_KEY}",
			Model:  "deepseek-reasoner",
		},
	}
	
	// Add example profiles
	config.Profiles = map[string]ProfileConfig{
		"quick": {
			Provider:    "cerebras",
			Quiet:       true,
			Temperature: floatPtr(0.3),
		},
		"detailed": {
			Provider:       "openai",
			Verbose:        true,
			Temperature:    floatPtr(0.7),
			MarkdownOutput: true,
		},
		"coding": {
			Provider:    "cerebras",
			Temperature: floatPtr(0.2),
		},
	}
	
	// Save the configuration
	return SaveConfig(config, configPath)
}

// GetConfigTemplate returns a template configuration as a string for the init command
func GetConfigTemplate() string {
	return `# TGPT Configuration File
# This file configures default settings for tgpt to avoid complex command-line flags

# Default provider settings
[defaults]
provider = "cerebras"          # Your preferred provider
temperature = 0.7              # Response creativity (0.0-2.0)
top_p = 0.9                   # Response diversity (0.0-1.0)
quiet = false                 # Skip loading animations
verbose = false               # Show detailed output
markdown_output = false       # Format output as markdown (future feature)
search_provider = "is-fast"   # Default search provider (is-fast/firecrawl)

# Provider configurations
# Environment variable expansion is supported with ${VAR_NAME} syntax

[providers.cerebras]
type = "openai"               # Uses OpenAI-compatible API
api_key = "${CEREBRAS_API_KEY}"
model = "qwen-3-coder-480b"
url = "https://api.cerebras.ai/v1/chat/completions"
is_default = true             # Mark as default provider

[providers.openai]
type = "openai"
api_key = "${OPENAI_API_KEY}"
model = "gpt-4"
url = ""                      # Uses default OpenAI URL

[providers.gemini]
type = "gemini"
api_key = "${GEMINI_API_KEY}"
model = "gemini-pro"

[providers.deepseek]
type = "deepseek"
api_key = "${DEEPSEEK_API_KEY}"
model = "deepseek-reasoner"

# Image generation settings
[image]
default_provider = "pollinations"
width = 1024
height = 1024
ratio = "1:1"
count = "1"
negative_prompt = ""

# Search configuration
[search]
google_api_key = "${TGPT_GOOGLE_API_KEY}"
google_search_engine_id = "${TGPT_GOOGLE_SEARCH_ENGINE_ID}"
default_provider = "is-fast"

# Mode-specific settings
[modes.shell]
auto_execute = false          # Auto-execute shell commands with -y
preprompt = "You are a helpful shell assistant. Provide concise, accurate commands."

[modes.code]
preprompt = "Generate clean, well-commented, production-ready code with proper error handling."

[modes.interactive]
history_size = 1000
save_conversation = true

# Profiles for different use cases
# Use with: tgpt --profile quick "your question"

[profiles.quick]
provider = "cerebras"
quiet = true
temperature = 0.3

[profiles.detailed]
provider = "openai"
verbose = true
temperature = 0.7
markdown_output = true

[profiles.coding]
provider = "cerebras"
temperature = 0.2
`
}