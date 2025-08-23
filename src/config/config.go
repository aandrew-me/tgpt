package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the complete configuration for tgpt
type Config struct {
	Defaults  DefaultConfig                `toml:"defaults"`
	Providers map[string]ProviderConfig    `toml:"providers"`
	Image     ImageConfig                  `toml:"image"`
	Search    SearchConfig                 `toml:"search"`
	Modes     map[string]ModeConfig        `toml:"modes"`
	Profiles  map[string]ProfileConfig     `toml:"profiles"`
	
	// Internal fields (not serialized to TOML)
	ConfigPath   string `toml:"-"`
	ProfileName  string `toml:"-"`
	CliOverrides map[string]interface{} `toml:"-"`
}

// DefaultConfig contains the default settings for tgpt
type DefaultConfig struct {
	Provider        string  `toml:"provider"`
	Temperature     float64 `toml:"temperature"`
	TopP           float64 `toml:"top_p"`
	Quiet          bool    `toml:"quiet"`
	Verbose        bool    `toml:"verbose"`
	MarkdownOutput bool    `toml:"markdown_output"`
	SearchProvider string  `toml:"search_provider"`
}

// ProviderConfig contains provider-specific configuration
type ProviderConfig struct {
	Type      string `toml:"type"`       // Internal provider type (e.g., "openai")
	APIKey    string `toml:"api_key"`    // Supports env var expansion like ${CEREBRAS_API_KEY}
	Model     string `toml:"model"`
	URL       string `toml:"url"`
	IsDefault bool   `toml:"is_default"`
}

// ImageConfig contains image generation settings
type ImageConfig struct {
	DefaultProvider string `toml:"default_provider"`
	Width          int    `toml:"width"`
	Height         int    `toml:"height"`
	Ratio          string `toml:"ratio"`
	Count          string `toml:"count"`
	Negative       string `toml:"negative_prompt"`
}

// SearchConfig contains search-related configuration
type SearchConfig struct {
	GoogleAPIKey         string `toml:"google_api_key"`
	GoogleSearchEngineID string `toml:"google_search_engine_id"`
	DefaultProvider      string `toml:"default_provider"`
}

// ModeConfig contains mode-specific settings
type ModeConfig struct {
	AutoExecute bool   `toml:"auto_execute"`
	Preprompt   string `toml:"preprompt"`
	HistorySize int    `toml:"history_size"`
	SaveConv    bool   `toml:"save_conversation"`
}

// ProfileConfig contains profile-specific overrides
type ProfileConfig struct {
	Provider        string                 `toml:"provider"`
	Temperature     *float64              `toml:"temperature,omitempty"`
	TopP           *float64              `toml:"top_p,omitempty"`
	Quiet          bool                  `toml:"quiet,omitempty"`
	Verbose        bool                  `toml:"verbose,omitempty"`
	MarkdownOutput bool                  `toml:"markdown_output,omitempty"`
	Modes          map[string]ModeConfig `toml:"modes,omitempty"`
}

// Manager handles configuration loading and resolution
type Manager struct {
	config     *Config
	configPath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{}
}

// Load loads the configuration from the default location or specified path
func (m *Manager) Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = GetDefaultConfigPath()
	}
	
	m.configPath = configPath
	
	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := GetDefaultConfig()
		config.ConfigPath = configPath
		m.config = config
		return config, nil
	}
	
	// Load from file
	config, err := loadConfigFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}
	
	config.ConfigPath = configPath
	m.config = config
	return config, nil
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "tgpt-config.toml"
	}
	
	configDir := filepath.Join(homeDir, ".config", "tgpt")
	return filepath.Join(configDir, "config.toml")
}

// LoadConfig is a convenience function to load configuration
func LoadConfig(configPath string) (*Config, error) {
	manager := NewManager()
	return manager.Load(configPath)
}

// getDefaultConfig returns a Config with sensible defaults
// GetDefaultConfig returns a configuration with default values
func GetDefaultConfig() *Config {
	return &Config{
		Defaults: DefaultConfig{
			Provider:        "phind", // Keep existing default for backward compatibility
			Temperature:     0.7,
			TopP:           0.9,
			Quiet:          false,
			Verbose:        false,
			MarkdownOutput: false,
			SearchProvider: "is-fast",
		},
		Providers: make(map[string]ProviderConfig),
		Image: ImageConfig{
			DefaultProvider: "pollinations",
			Width:          1024,
			Height:         1024,
			Ratio:          "1:1",
			Count:          "1",
			Negative:       "",
		},
		Search: SearchConfig{
			GoogleAPIKey:         "${TGPT_GOOGLE_API_KEY}",
			GoogleSearchEngineID: "${TGPT_GOOGLE_SEARCH_ENGINE_ID}",
			DefaultProvider:      "is-fast",
		},
		Modes: map[string]ModeConfig{
			"shell": {
				AutoExecute: false,
				Preprompt:   "You are a helpful shell assistant. Provide concise, accurate commands.",
			},
			"code": {
				Preprompt: "Generate clean, well-commented, production-ready code with proper error handling.",
			},
			"interactive": {
				HistorySize: 1000,
				SaveConv:   true,
			},
		},
		Profiles: make(map[string]ProfileConfig),
	}
}

// SetValue sets a configuration value using dot notation path
func (c *Config) SetValue(key, value string) error {
	pathParts := strings.Split(key, ".")
	if len(pathParts) < 2 {
		return fmt.Errorf("invalid config path: %s", key)
	}

	section := pathParts[0]
	fieldPath := strings.Join(pathParts[1:], ".")

	switch section {
	case "defaults":
		return c.setDefaultsValue(fieldPath, value)
	case "providers":
		if len(pathParts) < 3 {
			return fmt.Errorf("invalid provider config path: %s", key)
		}
		return c.setProviderValue(pathParts[1], strings.Join(pathParts[2:], "."), value)
	case "image":
		return c.setImageValue(fieldPath, value)
	case "search":
		return c.setSearchValue(fieldPath, value)
	default:
		return fmt.Errorf("unsupported config section: %s", section)
	}
}

// GetValue gets a configuration value using dot notation path
func (c *Config) GetValue(key string) (string, error) {
	pathParts := strings.Split(key, ".")
	if len(pathParts) < 2 {
		return "", fmt.Errorf("invalid config path: %s", key)
	}

	section := pathParts[0]
	fieldPath := strings.Join(pathParts[1:], ".")

	switch section {
	case "defaults":
		return c.getDefaultsValue(fieldPath)
	case "providers":
		if len(pathParts) < 3 {
			return "", fmt.Errorf("invalid provider config path: %s", key)
		}
		return c.getProviderValue(pathParts[1], strings.Join(pathParts[2:], "."))
	case "image":
		return c.getImageValue(fieldPath)
	case "search":
		return c.getSearchValue(fieldPath)
	default:
		return "", fmt.Errorf("unsupported config section: %s", section)
	}
}

func (c *Config) setDefaultsValue(field, value string) error {
	switch field {
	case "provider":
		c.Defaults.Provider = value
	case "temperature":
		temp, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid temperature value: %s", value)
		}
		c.Defaults.Temperature = temp
	case "top_p":
		topP, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid top_p value: %s", value)
		}
		c.Defaults.TopP = topP
	case "quiet":
		quiet, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid quiet value: %s", value)
		}
		c.Defaults.Quiet = quiet
	case "verbose":
		verbose, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid verbose value: %s", value)
		}
		c.Defaults.Verbose = verbose
	default:
		return fmt.Errorf("unsupported defaults field: %s", field)
	}
	return nil
}

func (c *Config) getDefaultsValue(field string) (string, error) {
	switch field {
	case "provider":
		return c.Defaults.Provider, nil
	case "temperature":
		return fmt.Sprintf("%.1f", c.Defaults.Temperature), nil
	case "top_p":
		return fmt.Sprintf("%.1f", c.Defaults.TopP), nil
	case "quiet":
		return fmt.Sprintf("%t", c.Defaults.Quiet), nil
	case "verbose":
		return fmt.Sprintf("%t", c.Defaults.Verbose), nil
	default:
		return "", fmt.Errorf("unsupported defaults field: %s", field)
	}
}

func (c *Config) setProviderValue(providerName, field, value string) error {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
	
	provider := c.Providers[providerName]
	
	switch field {
	case "api_key":
		provider.APIKey = value
	case "model":
		provider.Model = value
	case "url":
		provider.URL = value
	case "type":
		provider.Type = value
	default:
		return fmt.Errorf("unsupported provider field: %s", field)
	}
	
	c.Providers[providerName] = provider
	return nil
}

func (c *Config) getProviderValue(providerName, field string) (string, error) {
	provider, exists := c.Providers[providerName]
	if !exists {
		return "", fmt.Errorf("provider not found: %s", providerName)
	}
	
	switch field {
	case "api_key":
		return provider.APIKey, nil
	case "model":
		return provider.Model, nil
	case "url":
		return provider.URL, nil
	case "type":
		return provider.Type, nil
	default:
		return "", fmt.Errorf("unsupported provider field: %s", field)
	}
}

func (c *Config) setImageValue(field, value string) error {
	switch field {
	case "default_provider":
		c.Image.DefaultProvider = value
	case "width":
		width, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid width value: %s", value)
		}
		c.Image.Width = width
	case "height":
		height, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid height value: %s", value)
		}
		c.Image.Height = height
	default:
		return fmt.Errorf("unsupported image field: %s", field)
	}
	return nil
}

func (c *Config) getImageValue(field string) (string, error) {
	switch field {
	case "default_provider":
		return c.Image.DefaultProvider, nil
	case "width":
		return fmt.Sprintf("%d", c.Image.Width), nil
	case "height":
		return fmt.Sprintf("%d", c.Image.Height), nil
	default:
		return "", fmt.Errorf("unsupported image field: %s", field)
	}
}

func (c *Config) setSearchValue(field, value string) error {
	switch field {
	case "google_api_key":
		c.Search.GoogleAPIKey = value
	case "google_search_engine_id":
		c.Search.GoogleSearchEngineID = value
	case "default_provider":
		c.Search.DefaultProvider = value
	default:
		return fmt.Errorf("unsupported search field: %s", field)
	}
	return nil
}

func (c *Config) getSearchValue(field string) (string, error) {
	switch field {
	case "google_api_key":
		return c.Search.GoogleAPIKey, nil
	case "google_search_engine_id":
		return c.Search.GoogleSearchEngineID, nil
	case "default_provider":
		return c.Search.DefaultProvider, nil
	default:
		return "", fmt.Errorf("unsupported search field: %s", field)
	}
}

// GetEffectiveProvider returns the provider to use based on configuration and overrides
func (c *Config) GetEffectiveProvider(cliProvider string, envProvider string, isImage bool) string {
	// 1. CLI flag has highest priority
	if cliProvider != "" {
		return cliProvider
	}
	
	// 2. Environment variables
	if envProvider != "" {
		return envProvider
	}
	
	// 3. Profile override (if profile is set)
	if c.ProfileName != "" {
		if profile, exists := c.Profiles[c.ProfileName]; exists && profile.Provider != "" {
			return profile.Provider
		}
	}
	
	// 4. Check if any provider is marked as default
	for name, provider := range c.Providers {
		if provider.IsDefault {
			return name
		}
	}
	
	// 5. Config file default
	if c.Defaults.Provider != "" {
		return c.Defaults.Provider
	}
	
	// 6. Built-in default
	return "phind"
}

// GetEffectiveValue returns the effective value for a configuration field
func (c *Config) GetEffectiveValue(fieldName string, cliValue interface{}, envValue string) interface{} {
	// 1. CLI flag has highest priority
	if cliValue != nil && !isEmptyValue(cliValue, fieldName) {
		return cliValue
	}
	
	// 2. Environment variable
	if envValue != "" {
		return parseEnvValue(envValue, fieldName)
	}
	
	// 3. Profile override
	if c.ProfileName != "" {
		if profileValue := c.getProfileValue(fieldName); profileValue != nil {
			return profileValue
		}
	}
	
	// 4. Config file value
	if configValue := c.getConfigValue(fieldName); configValue != nil {
		return configValue
	}
	
	// 5. Built-in default
	return getBuiltinDefault(fieldName)
}

// Helper functions
func isEmptyValue(value interface{}, fieldName string) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case bool:
		return false // bools are never "empty"
	case float64:
		// For temperature and top_p, zero is a valid value and should not be considered empty
		if fieldName == "temperature" || fieldName == "top_p" {
			return false
		}
		return v == 0
	case int:
		return v == 0
	default:
		return value == nil
	}
}

func parseEnvValue(envValue string, fieldName string) interface{} {
	switch fieldName {
	case "temperature", "top_p":
		if val, err := parseFloat64(envValue); err == nil {
			return val
		}
	case "quiet", "verbose":
		return envValue == "true" || envValue == "1"
	}
	return envValue
}

func parseFloat64(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	return strconv.ParseFloat(s, 64)
}

func (c *Config) getProfileValue(fieldName string) interface{} {
	if c.ProfileName == "" {
		return nil
	}
	
	profile, exists := c.Profiles[c.ProfileName]
	if !exists {
		return nil
	}
	
	switch fieldName {
	case "provider":
		if profile.Provider != "" {
			return profile.Provider
		}
	case "temperature":
		if profile.Temperature != nil {
			return *profile.Temperature
		}
	case "top_p":
		if profile.TopP != nil {
			return *profile.TopP
		}
	case "quiet":
		return profile.Quiet
	case "verbose":
		return profile.Verbose
	case "markdown_output":
		return profile.MarkdownOutput
	}
	
	return nil
}

func (c *Config) getConfigValue(fieldName string) interface{} {
	switch fieldName {
	case "provider":
		return c.Defaults.Provider
	case "temperature":
		return c.Defaults.Temperature
	case "top_p":
		return c.Defaults.TopP
	case "quiet":
		return c.Defaults.Quiet
	case "verbose":
		return c.Defaults.Verbose
	case "markdown_output":
		return c.Defaults.MarkdownOutput
	case "search_provider":
		return c.Defaults.SearchProvider
	}
	
	return nil
}

func getBuiltinDefault(fieldName string) interface{} {
	defaults := map[string]interface{}{
		"provider":         "phind",
		"temperature":      0.7,
		"top_p":           0.9,
		"quiet":           false,
		"verbose":         false,
		"markdown_output": false,
		"search_provider": "is-fast",
	}
	
	return defaults[fieldName]
}

// ApplyProfile applies a profile configuration to the main configuration
func (c *Config) ApplyProfile(profile ProfileConfig) {
	if profile.Provider != "" {
		c.Defaults.Provider = profile.Provider
	}
	if profile.Temperature != nil {
		c.Defaults.Temperature = *profile.Temperature
	}
	if profile.TopP != nil {
		c.Defaults.TopP = *profile.TopP
	}
	c.Defaults.Quiet = profile.Quiet
	c.Defaults.Verbose = profile.Verbose
	c.Defaults.MarkdownOutput = profile.MarkdownOutput
	
	// Apply mode-specific settings from profile
	for modeName, modeConfig := range profile.Modes {
		if c.Modes == nil {
			c.Modes = make(map[string]ModeConfig)
		}
		c.Modes[modeName] = modeConfig
	}
}

// ResolvedConfig contains all resolved configuration values with proper precedence
type ResolvedConfig struct {
	Provider    string
	APIKey      string
	Model       string
	Temperature string
	TopP        string
	URL         string
	Quiet       bool
	Verbose     bool
	Markdown    bool
}

// ResolveConfig resolves all configuration values using proper precedence logic
// Returns a ResolvedConfig with all values resolved according to precedence rules
func (c *Config) ResolveConfig(cliFlags map[string]string, isImage bool) *ResolvedConfig {
	resolved := &ResolvedConfig{}
	
	// 1. Resolve provider first as it affects other fields
	resolved.Provider = c.resolveProvider(cliFlags["provider"], isImage)
	
	// 2. Get provider-specific config if available
	var providerConfig *ProviderConfig
	if pc, exists := c.Providers[resolved.Provider]; exists {
		providerConfig = &pc
	}
	
	// 3. Resolve other configuration values
	resolved.APIKey = c.resolveString("api_key", cliFlags["key"], providerConfig)
	resolved.Model = c.resolveString("model", cliFlags["model"], providerConfig)
	resolved.Temperature = c.resolveFloat("temperature", cliFlags["temperature"])
	resolved.TopP = c.resolveFloat("top_p", cliFlags["top_p"])
	resolved.URL = c.resolveString("url", cliFlags["url"], providerConfig)
	
	// Boolean flags (these are handled differently as they can have profile/config defaults)
	resolved.Quiet = c.resolveBool("quiet", cliFlags["quiet"])
	resolved.Verbose = c.resolveBool("verbose", cliFlags["verbose"])
	resolved.Markdown = c.resolveBool("markdown_output", cliFlags["markdown_output"])
	
	return resolved
}

// resolveProvider resolves the provider using precedence logic
func (c *Config) resolveProvider(cliProvider string, isImage bool) string {
	// 1. CLI flag has highest priority
	if cliProvider != "" {
		return cliProvider
	}
	
	// 2. Image-specific default if in image mode
	if isImage && c.Image.DefaultProvider != "" {
		return c.Image.DefaultProvider
	}
	
	// 3. Environment variables for backward compatibility
	envProvider := ""
	if isImage {
		envProvider = expandStringVars(os.Getenv("IMG_PROVIDER"))
	} else {
		envProvider = expandStringVars(os.Getenv("AI_PROVIDER"))
	}
	if envProvider != "" {
		return envProvider
	}
	
	// 4. Profile override (if profile is set)
	if c.ProfileName != "" {
		if profile, exists := c.Profiles[c.ProfileName]; exists && profile.Provider != "" {
			return profile.Provider
		}
	}
	
	// 5. Check if any provider is marked as default
	for name, provider := range c.Providers {
		if provider.IsDefault {
			return name
		}
	}
	
	// 6. Config file default
	if c.Defaults.Provider != "" {
		return c.Defaults.Provider
	}
	
	// 7. Built-in default
	return "phind"
}

// resolveString resolves string configuration values with precedence
func (c *Config) resolveString(fieldName string, cliValue string, providerConfig *ProviderConfig) string {
	// 1. CLI flag
	if cliValue != "" {
		return cliValue
	}
	
	// 2. Environment variable
	envKey := "TGPT_" + strings.ToUpper(strings.Replace(fieldName, "_", "_", -1))
	if envValue := os.Getenv(envKey); envValue != "" {
		return expandStringVars(envValue)
	}
	
	// 3. Provider-specific configuration
	if providerConfig != nil {
		switch fieldName {
		case "api_key":
			if providerConfig.APIKey != "" {
				return expandStringVars(providerConfig.APIKey)
			}
		case "model":
			if providerConfig.Model != "" {
				return providerConfig.Model
			}
		case "url":
			if providerConfig.URL != "" {
				return providerConfig.URL
			}
		}
	}
	
	// 4. Profile override
	if c.ProfileName != "" {
		if profileValue := c.getProfileValue(fieldName); profileValue != nil {
			if str, ok := profileValue.(string); ok && str != "" {
				return str
			}
		}
	}
	
	// 5. Config defaults and built-in defaults are not applicable for these string fields
	return ""
}

// resolveFloat resolves float configuration values (temperature, top_p)
func (c *Config) resolveFloat(fieldName string, cliValue string) string {
	// 1. CLI flag
	if cliValue != "" {
		return cliValue
	}
	
	// 2. Environment variable
	envKey := "TGPT_" + strings.ToUpper(fieldName)
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	
	// 3. Profile override
	if c.ProfileName != "" {
		if profileValue := c.getProfileValue(fieldName); profileValue != nil {
			if f, ok := profileValue.(float64); ok {
				return fmt.Sprintf("%.1f", f)
			}
		}
	}
	
	// 4. Config file value (zero is a valid value for temperature and top_p)
	switch fieldName {
	case "temperature":
		return fmt.Sprintf("%.1f", c.Defaults.Temperature)
	case "top_p":
		return fmt.Sprintf("%.1f", c.Defaults.TopP)
	}
	
	// 5. Built-in defaults
	switch fieldName {
	case "temperature":
		return "0.7"
	case "top_p":
		return "0.9"
	}
	
	return ""
}

// resolveBool resolves boolean configuration values
func (c *Config) resolveBool(fieldName string, cliValue string) bool {
	// 1. CLI flag (if provided, assumes true for boolean flags)
	if cliValue == "true" || cliValue == "1" {
		return true
	}
	
	// 2. Environment variable
	envKey := "TGPT_" + strings.ToUpper(fieldName)
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue == "true" || envValue == "1"
	}
	
	// 3. Profile override
	if c.ProfileName != "" {
		if profileValue := c.getProfileValue(fieldName); profileValue != nil {
			if b, ok := profileValue.(bool); ok {
				return b
			}
		}
	}
	
	// 4. Config file value
	switch fieldName {
	case "quiet":
		return c.Defaults.Quiet
	case "verbose":
		return c.Defaults.Verbose
	case "markdown_output":
		return c.Defaults.MarkdownOutput
	}
	
	// 5. Built-in default is false for all boolean flags
	return false
}

// expandStringVars expands environment variables in config values like ${VAR_NAME}
func expandStringVars(value string) string {
	return os.ExpandEnv(value)
}
