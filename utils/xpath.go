package utils

import (
	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// XPathValidationRule represents a single XPath validation rule
type XPathValidationRule interface {
	Validate(context context.XPathValidationContext) ([]types.ValidationIssue, error)
	GetRule() types.ValidationRule
	GetXPath() string
}

// XPathRuleValidator implements XPath-based validation
type XPathRuleValidator struct {
	rules []XPathValidationRule

	// OnRuleDone, if set, is called after each rule with the file name and the
	// number of rules done so far. It may be called concurrently for different files.
	OnRuleDone func(fileName string, done, total int)
}

// NewXPathRuleValidator creates a new XPath rule validator
func NewXPathRuleValidator(rules []XPathValidationRule) *XPathRuleValidator {
	return &XPathRuleValidator{
		rules: rules,
	}
}

// NewXPathRuleValidatorFromConfig creates a new XPath rule validator from configuration
func NewXPathRuleValidatorFromConfig(cfg interface{}) *XPathRuleValidator {
	// For backward compatibility, accept any config and return empty validator
	// The actual rule creation will be handled by the library
	return &XPathRuleValidator{
		rules: make([]XPathValidationRule, 0),
	}
}

// Validate performs XPath validation on the given context
func (v *XPathRuleValidator) Validate(ctx context.XPathValidationContext) ([]types.ValidationIssue, error) {
	var issues []types.ValidationIssue

	// Execute all XPath rules
	for i, rule := range v.rules {
		ruleIssues, err := rule.Validate(ctx)
		if err != nil {
			return nil, err
		}
		issues = append(issues, ruleIssues...)
		if v.OnRuleDone != nil {
			v.OnRuleDone(ctx.GetFileName(), i+1, len(v.rules))
		}
	}

	return issues, nil
}

// GetRules returns the validation rules used by this validator
func (v *XPathRuleValidator) GetRules() []types.ValidationRule {
	var rules []types.ValidationRule
	for _, rule := range v.rules {
		rules = append(rules, rule.GetRule())
	}
	return rules
}
