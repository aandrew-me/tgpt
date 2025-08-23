package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// expandEnvVars expands environment variables in the configuration
func expandEnvVars(config *Config) error {
	// Expand environment variables in provider configurations
	for name, provider := range config.Providers {
		expandedProvider := provider

		var err error
		expandedProvider.APIKey, err = expandEnvVar(provider.APIKey)
		if err != nil {
			return fmt.Errorf("error expanding API key for provider '%s': %w", name, err)
		}

		expandedProvider.Model, err = expandEnvVar(provider.Model)
		if err != nil {
			return fmt.Errorf("error expanding model for provider '%s': %w", name, err)
		}

		expandedProvider.URL, err = expandEnvVar(provider.URL)
		if err != nil {
			return fmt.Errorf("error expanding URL for provider '%s': %w", name, err)
		}

		config.Providers[name] = expandedProvider
	}

	// Expand environment variables in search configuration
	var err error
	config.Search.GoogleAPIKey, err = expandEnvVar(config.Search.GoogleAPIKey)
	if err != nil {
		return fmt.Errorf("error expanding Google API key: %w", err)
	}

	config.Search.GoogleSearchEngineID, err = expandEnvVar(config.Search.GoogleSearchEngineID)
	if err != nil {
		return fmt.Errorf("error expanding Google Search Engine ID: %w", err)
	}

	// Expand environment variables in mode preprompts
	for modeName, mode := range config.Modes {
		expandedMode := mode
		expandedMode.Preprompt, err = expandEnvVar(mode.Preprompt)
		if err != nil {
			return fmt.Errorf("error expanding preprompt for mode '%s': %w", modeName, err)
		}
		config.Modes[modeName] = expandedMode
	}

	return nil
}

// expandEnvVar expands environment variables in a string
// Supports syntax: ${VAR_NAME} and $VAR_NAME
func expandEnvVar(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	// Regular expression to match ${VAR_NAME} and $VAR_NAME patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	result := re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name
		var varName string
		if strings.HasPrefix(match, "${") {
			// ${VAR_NAME} format
			varName = match[2 : len(match)-1]
		} else {
			// $VAR_NAME format
			varName = match[1:]
		}

		// Get environment variable value
		value := os.Getenv(varName)
		return value
	})

	return result, nil
}

// expandEnvVarWithDefault expands environment variables with a default value
// Supports syntax: ${VAR_NAME:default_value}
func expandEnvVarWithDefault(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	// Regular expression to match ${VAR_NAME:default} patterns
	re := regexp.MustCompile(`\$\{([^:}]+):([^}]*)\}`)

	result := re.ReplaceAllStringFunc(input, func(match string) string {
		// Parse variable name and default value
		parts := strings.SplitN(match[2:len(match)-1], ":", 2)
		if len(parts) != 2 {
			return match // Return original if parsing fails
		}

		varName := parts[0]
		defaultValue := parts[1]

		// Get environment variable value, use default if empty
		value := os.Getenv(varName)
		if value == "" {
			value = defaultValue
		}

		return value
	})

	// Handle simple ${VAR_NAME} patterns without defaults
	result, err := expandEnvVar(result)
	return result, err
}

// validateEnvExpansion checks if all required environment variables are available
// ValidateEnvExpansion checks if all required environment variables are available
func ValidateEnvExpansion(config *Config) []string {
	var missingVars []string

	// Check provider configurations
	for name, provider := range config.Providers {
		missing := findMissingEnvVars(provider.APIKey)
		for _, mv := range missing {
			missingVars = append(missingVars, fmt.Sprintf("provider '%s' requires env var: %s", name, mv))
		}

		missing = findMissingEnvVars(provider.Model)
		for _, mv := range missing {
			missingVars = append(missingVars, fmt.Sprintf("provider '%s' model requires env var: %s", name, mv))
		}

		missing = findMissingEnvVars(provider.URL)
		for _, mv := range missing {
			missingVars = append(missingVars, fmt.Sprintf("provider '%s' URL requires env var: %s", name, mv))
		}
	}

	// Check search configuration
	missing := findMissingEnvVars(config.Search.GoogleAPIKey)
	for _, mv := range missing {
		missingVars = append(missingVars, fmt.Sprintf("search configuration requires env var: %s", mv))
	}

	missing = findMissingEnvVars(config.Search.GoogleSearchEngineID)
	for _, mv := range missing {
		missingVars = append(missingVars, fmt.Sprintf("search configuration requires env var: %s", mv))
	}

	return missingVars
}

// findMissingEnvVars finds environment variables that are referenced but not set
func findMissingEnvVars(input string) []string {
	if input == "" {
		return nil
	}

	var missing []string

	// Find all environment variable references
	re := regexp.MustCompile(`\$\{([^}:]+)(?::[^}]*)?\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
	matches := re.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		var varName string
		if match[1] != "" {
			// ${VAR_NAME} or ${VAR_NAME:default} format
			varName = match[1]
		} else if match[2] != "" {
			// $VAR_NAME format
			varName = match[2]
		}

		if varName != "" && os.Getenv(varName) == "" {
			// Check if it has a default value
			if !hasDefaultValue(input, varName) {
				missing = append(missing, varName)
			}
		}
	}

	return missing
}

// hasDefaultValue checks if a variable has a default value in ${VAR:default} syntax
func hasDefaultValue(input, varName string) bool {
	pattern := fmt.Sprintf(`\$\{%s:[^}]+\}`, regexp.QuoteMeta(varName))
	matched, _ := regexp.MatchString(pattern, input)
	return matched
}

// GetEnvVarUsage returns a map of environment variables used in the configuration
func GetEnvVarUsage(config *Config) map[string][]string {
	usage := make(map[string][]string)

	// Check providers
	for name, provider := range config.Providers {
		addEnvVarUsage(usage, provider.APIKey, fmt.Sprintf("provider '%s' API key", name))
		addEnvVarUsage(usage, provider.Model, fmt.Sprintf("provider '%s' model", name))
		addEnvVarUsage(usage, provider.URL, fmt.Sprintf("provider '%s' URL", name))
	}

	// Check search configuration
	addEnvVarUsage(usage, config.Search.GoogleAPIKey, "Google Search API key")
	addEnvVarUsage(usage, config.Search.GoogleSearchEngineID, "Google Search Engine ID")

	// Check modes
	for modeName, mode := range config.Modes {
		addEnvVarUsage(usage, mode.Preprompt, fmt.Sprintf("mode '%s' preprompt", modeName))
	}

	return usage
}

// addEnvVarUsage extracts environment variables from a string and adds them to the usage map
func addEnvVarUsage(usage map[string][]string, input, context string) {
	if input == "" {
		return
	}

	re := regexp.MustCompile(`\$\{([^}:]+)(?::[^}]*)?\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
	matches := re.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		var varName string
		if match[1] != "" {
			varName = match[1]
		} else if match[2] != "" {
			varName = match[2]
		}

		if varName != "" {
			if _, exists := usage[varName]; !exists {
				usage[varName] = []string{}
			}
			usage[varName] = append(usage[varName], context)
		}
	}
}

// PreviewExpansion shows what the configuration would look like after expansion
// without actually expanding it (useful for debugging)
func PreviewExpansion(config *Config) (*Config, error) {
	// Create a deep copy of the configuration
	preview := *config

	// Deep copy maps
	preview.Providers = make(map[string]ProviderConfig)
	for k, v := range config.Providers {
		preview.Providers[k] = v
	}

	preview.Modes = make(map[string]ModeConfig)
	for k, v := range config.Modes {
		preview.Modes[k] = v
	}

	preview.Profiles = make(map[string]ProfileConfig)
	for k, v := range config.Profiles {
		preview.Profiles[k] = v
	}

	// Expand environment variables
	if err := expandEnvVars(&preview); err != nil {
		return nil, err
	}

	return &preview, nil
}
