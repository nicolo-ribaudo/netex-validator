package interfaces

import (
	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// Validator represents a generic validator interface
type Validator interface {
	Validate(context context.ValidationContext) ([]types.ValidationIssue, error)
	GetRules() []types.ValidationRule
}

// SchemaValidator represents an XML schema validator
type SchemaValidator interface {
	Validate(context context.SchemaValidationContext) ([]types.ValidationIssue, error)
	GetRules() []types.ValidationRule
}

// XPathValidator represents an XPath-based validator
type XPathValidator interface {
	Validate(context context.XPathValidationContext) ([]types.ValidationIssue, error)
	GetRules() []types.ValidationRule
}

// JAXBValidator represents an object model validator (similar to JAXB in Java)
type JAXBValidator interface {
	Validate(context context.JAXBValidationContext) ([]types.ValidationIssue, error)
	GetRules() []types.ValidationRule
}

// DatasetValidator represents a validator that operates on entire datasets
type DatasetValidator interface {
	Validate(report *types.ValidationReport) error
}

// ValidationReportEntryFactory creates validation report entries from validation issues
type ValidationReportEntryFactory interface {
	CreateValidationReportEntry(issue types.ValidationIssue) types.ValidationReportEntry
	TemplateValidationReportEntry(rule types.ValidationRule) types.ValidationReportEntry
}
