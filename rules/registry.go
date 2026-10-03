package rules

import (
	"github.com/nicolo-ribaudo/netex-validator/config"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// Rule represents a single validation rule
type Rule struct {
	// Code is the full rule ID: the namespace path and Key joined with "/".
	Code     string
	Key      string
	Name     string
	Message  string
	Severity types.Severity
	XPath    string
	// Category is the namespace path the rule was loaded under (may be empty).
	Category    string
	Description string
}

// RuleRegistry manages all validation rules
type RuleRegistry struct {
	rules  []Rule
	config *config.ValidatorConfig
}

// NewRuleRegistry creates a new rule registry holding the given rules,
// typically loaded with LoadRuleSets.
func NewRuleRegistry(cfg *config.ValidatorConfig, rules []Rule) *RuleRegistry {
	return &RuleRegistry{
		rules:  rules,
		config: cfg,
	}
}

// GetEnabledRules returns all enabled rules based on configuration
func (r *RuleRegistry) GetEnabledRules() []Rule {
	var enabled []Rule

	for _, rule := range r.rules {
		if r.config.IsRuleEnabled(rule.Category, rule.Key) {
			// Apply severity overrides from config
			rule.Severity = r.config.GetRuleSeverity(rule.Category, rule.Key, rule.Severity)
			enabled = append(enabled, rule)
		}
	}

	// Add custom rules from config
	for _, customRule := range r.config.GetCustomRules() {
		enabled = append(enabled, Rule{
			Code:     customRule.Code,
			Key:      customRule.Code,
			Name:     customRule.Name,
			Message:  customRule.Message,
			Severity: customRule.Severity,
			XPath:    customRule.XPath,
			Category: "custom",
		})
	}

	return enabled
}

// GetRuleByCode returns a specific rule by its code
func (r *RuleRegistry) GetRuleByCode(code string) (Rule, bool) {
	for _, rule := range r.rules {
		if rule.Code == code {
			return rule, true
		}
	}
	return Rule{}, false
}

// GetRulesByCategory returns all rules in a specific category
func (r *RuleRegistry) GetRulesByCategory(category string) []Rule {
	var categoryRules []Rule

	for _, rule := range r.rules {
		if rule.Category == category {
			categoryRules = append(categoryRules, rule)
		}
	}

	return categoryRules
}

// GetAllCategories returns all available rule categories
func (r *RuleRegistry) GetAllCategories() []string {
	categories := make(map[string]bool)

	for _, rule := range r.rules {
		categories[rule.Category] = true
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}

	return result
}
