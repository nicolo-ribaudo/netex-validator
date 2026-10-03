package engine

import (
	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// ObjectValidator defines the interface for object model-based validation
type ObjectValidator interface {
	// Validate performs validation using the object model context
	Validate(ctx *context.ObjectValidationContext) []types.ValidationIssue

	// GetRules returns the validation rules implemented by this validator
	GetRules() []types.ValidationRule

	// GetName returns the name of this validator
	GetName() string
}

// BaseObjectValidator provides common functionality for object validators
type BaseObjectValidator struct {
	name  string
	rules []types.ValidationRule
}

// NewBaseObjectValidator creates a new base object validator
func NewBaseObjectValidator(name string, rules []types.ValidationRule) *BaseObjectValidator {
	return &BaseObjectValidator{
		name:  name,
		rules: rules,
	}
}

// GetName returns the validator name
func (v *BaseObjectValidator) GetName() string {
	return v.name
}

// GetRules returns the validation rules
func (v *BaseObjectValidator) GetRules() []types.ValidationRule {
	return v.rules
}

// ObjectValidatorRegistry manages a collection of object validators
type ObjectValidatorRegistry struct {
	validators []ObjectValidator
}

// NewObjectValidatorRegistry creates a new object validator registry
func NewObjectValidatorRegistry() *ObjectValidatorRegistry {
	return &ObjectValidatorRegistry{
		validators: make([]ObjectValidator, 0),
	}
}

// RegisterValidator adds a validator to the registry
func (r *ObjectValidatorRegistry) RegisterValidator(validator ObjectValidator) {
	r.validators = append(r.validators, validator)
}

// ValidateAll runs all registered validators
func (r *ObjectValidatorRegistry) ValidateAll(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var allIssues []types.ValidationIssue

	for _, validator := range r.validators {
		issues := validator.Validate(ctx)
		allIssues = append(allIssues, issues...)
	}

	return allIssues
}

// GetAllRules returns all validation rules from all registered validators
func (r *ObjectValidatorRegistry) GetAllRules() []types.ValidationRule {
	var allRules []types.ValidationRule

	for _, validator := range r.validators {
		rules := validator.GetRules()
		allRules = append(allRules, rules...)
	}

	return allRules
}
