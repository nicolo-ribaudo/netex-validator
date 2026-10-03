package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nicolo-ribaudo/netex-validator/types"
	"gopkg.in/yaml.v3"
)

// ValidatorConfig represents the complete validator configuration
type ValidatorConfig struct {
	Validator ValidatorSettings `yaml:"validator"`
	Rules     RulesConfig       `yaml:"rules"`
	Output    OutputConfig      `yaml:"output"`
}

// ValidatorSettings contains general validator settings
type ValidatorSettings struct {
	Profile         string `yaml:"profile"`         // e.g., "eu", "custom"
	MaxFileSize     int64  `yaml:"maxFileSize"`     // Maximum file size in bytes
	MaxSchemaErrors int    `yaml:"maxSchemaErrors"` // Maximum schema errors to report
	ConcurrentFiles int    `yaml:"concurrentFiles"` // Number of files to process concurrently
	EnableCache     bool   `yaml:"enableCache"`     // Enable validation result caching
	CacheTimeout    int    `yaml:"cacheTimeout"`    // Cache timeout in minutes
}

// RulesConfig contains rule-specific configuration
type RulesConfig struct {
	// Categories configures rule namespaces, keyed by namespace path (e.g. "line"
	// or "eu/line", matching the rule IDs "line/2" or "eu/line/2").
	Categories map[string]RuleCategoryConfig `yaml:"categories"`
	Custom     []CustomRuleConfig            `yaml:"custom"`
}

// RuleCategoryConfig configures a rule namespace and everything nested in it
type RuleCategoryConfig struct {
	Enabled         *bool                 `yaml:"enabled,omitempty"` // nil = enabled
	DefaultSeverity *types.Severity       `yaml:"defaultSeverity,omitempty"`
	Rules           map[string]RuleConfig `yaml:"rules,omitempty"` // keyed by rule key within the namespace
}

// RuleConfig configures a specific validation rule
type RuleConfig struct {
	Enabled  *bool           `yaml:"enabled,omitempty"` // nil = enabled
	Severity *types.Severity `yaml:"severity,omitempty"`
	Message  string          `yaml:"message,omitempty"`
	XPath    string          `yaml:"xpath,omitempty"`
}

// CustomRuleConfig defines a custom validation rule
type CustomRuleConfig struct {
	Code     string         `yaml:"code"`
	Name     string         `yaml:"name"`
	Message  string         `yaml:"message"`
	Severity types.Severity `yaml:"severity"`
	XPath    string         `yaml:"xpath"`
	Enabled  bool           `yaml:"enabled"`
}

// OutputConfig configures output settings
type OutputConfig struct {
	Format          string `yaml:"format"`          // json, text, html
	IncludeDetails  bool   `yaml:"includeDetails"`  // Include detailed location info
	GroupBySeverity bool   `yaml:"groupBySeverity"` // Group output by severity
	MaxEntries      int    `yaml:"maxEntries"`      // Maximum entries to output (0 = unlimited)
}

// DefaultConfig returns a default configuration
func DefaultConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Validator: ValidatorSettings{
			Profile:         "eu",
			MaxFileSize:     100 * 1024 * 1024, // 100MB
			MaxSchemaErrors: 100,
			ConcurrentFiles: 4,
			EnableCache:     false,
			CacheTimeout:    30,
		},
		Rules: RulesConfig{
			Categories: map[string]RuleCategoryConfig{},
			Custom:     []CustomRuleConfig{},
		},
		Output: OutputConfig{
			Format:          "json",
			IncludeDetails:  true,
			GroupBySeverity: true,
			MaxEntries:      0, // Unlimited
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*ValidatorConfig, error) {
	// Start with default config
	config := DefaultConfig()

	// If no config file specified, return default
	if configPath == "" {
		return config, nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Validate file path to prevent path traversal
	if !filepath.IsAbs(configPath) && strings.Contains(configPath, "..") {
		return nil, fmt.Errorf("invalid config file path: %s", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath) //nolint:gosec // Path is validated above
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a YAML file
func (c *ValidatorConfig) SaveConfig(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *ValidatorConfig) Validate() error {
	// Validate validator settings
	if c.Validator.MaxFileSize <= 0 {
		return fmt.Errorf("maxFileSize must be positive")
	}

	if c.Validator.MaxSchemaErrors < 0 {
		return fmt.Errorf("maxSchemaErrors cannot be negative")
	}

	if c.Validator.ConcurrentFiles <= 0 {
		return fmt.Errorf("concurrentFiles must be positive")
	}

	// Validate output format
	validFormats := map[string]bool{"json": true, "text": true, "html": true}
	if !validFormats[c.Output.Format] {
		return fmt.Errorf("invalid output format: %s (valid: json, text, html)", c.Output.Format)
	}

	// Validate custom rules
	for i, rule := range c.Rules.Custom {
		if rule.Code == "" {
			return fmt.Errorf("custom rule %d: code cannot be empty", i)
		}
		if rule.Name == "" {
			return fmt.Errorf("custom rule %d: name cannot be empty", i)
		}
		if rule.XPath == "" {
			return fmt.Errorf("custom rule %d: xpath cannot be empty", i)
		}
	}

	return nil
}

// IsRuleEnabled checks if the rule with the given key in the given namespace
// path is enabled. A rule is disabled if its namespace or any ancestor
// namespace is disabled, or if it is disabled by a per-rule override.
func (c *ValidatorConfig) IsRuleEnabled(namespace, key string) bool {
	for _, ns := range namespaceAncestors(namespace) {
		if catConfig, exists := c.Rules.Categories[ns]; exists && catConfig.Enabled != nil && !*catConfig.Enabled {
			return false
		}
	}

	if catConfig, exists := c.Rules.Categories[namespace]; exists {
		if ruleConfig, exists := catConfig.Rules[key]; exists && ruleConfig.Enabled != nil {
			return *ruleConfig.Enabled
		}
	}

	return true
}

// GetRuleSeverity gets the effective severity for the rule with the given key
// in the given namespace path: a per-rule override, else the defaultSeverity
// of the nearest namespace that sets one, else defaultSeverity.
func (c *ValidatorConfig) GetRuleSeverity(namespace, key string, defaultSeverity types.Severity) types.Severity {
	if catConfig, exists := c.Rules.Categories[namespace]; exists {
		if ruleConfig, exists := catConfig.Rules[key]; exists && ruleConfig.Severity != nil {
			return *ruleConfig.Severity
		}
	}

	for _, ns := range namespaceAncestors(namespace) {
		if catConfig, exists := c.Rules.Categories[ns]; exists && catConfig.DefaultSeverity != nil {
			return *catConfig.DefaultSeverity
		}
	}

	return defaultSeverity
}

// GetCustomRules returns all enabled custom rules
func (c *ValidatorConfig) GetCustomRules() []CustomRuleConfig {
	var enabled []CustomRuleConfig
	for _, rule := range c.Rules.Custom {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}

// namespaceAncestors returns the namespace path and its ancestors, nearest
// first: "eu/line" -> ["eu/line", "eu"].
func namespaceAncestors(namespace string) []string {
	var result []string
	for namespace != "" {
		result = append(result, namespace)
		i := strings.LastIndex(namespace, "/")
		if i < 0 {
			break
		}
		namespace = namespace[:i]
	}
	return result
}

// GenerateDefaultConfigFile creates a default configuration file
func GenerateDefaultConfigFile(configPath string) error {
	config := DefaultConfig()
	return config.SaveConfig(configPath)
}

// UnmatchedRuleConfig describes the entries of rules.categories that match no
// loaded rule, so that stale or mistyped configuration can be reported instead
// of being silently ignored. ruleIDs maps each loaded rule's namespace path to
// its keys.
func (c *ValidatorConfig) UnmatchedRuleConfig(ruleIDs map[string][]string) []string {
	namespaces := make(map[string]bool)
	ids := make(map[string]bool)
	for ns, keys := range ruleIDs {
		namespaces[ns] = true
		for _, ancestor := range namespaceAncestors(ns) {
			namespaces[ancestor] = true
		}
		for _, key := range keys {
			ids[ns+"/"+key] = true
		}
	}

	var problems []string
	for ns, catConfig := range c.Rules.Categories {
		if !namespaces[ns] {
			problems = append(problems, fmt.Sprintf("rules.categories.%q matches no loaded rule namespace", ns))
			continue
		}
		for key := range catConfig.Rules {
			if !ids[ns+"/"+key] {
				problems = append(problems, fmt.Sprintf("rules.categories.%q.rules.%q matches no loaded rule", ns, key))
			}
		}
	}
	sort.Strings(problems)
	return problems
}
