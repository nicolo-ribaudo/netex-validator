package validator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	errors "github.com/nicolo-ribaudo/netex-validator/reporting"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// ValidationResult represents the outcome of a NetEX validation
type ValidationResult struct {
	// Validation metadata
	Codespace          string    `json:"codespace"`
	ValidationReportID string    `json:"validationReportId"`
	CreationDate       time.Time `json:"creationDate"`

	// Validation entries
	ValidationReportEntries []ValidationReportEntry `json:"validationReportEntries"`

	// Summary statistics
	NumberOfValidationEntriesPerRule map[string]int `json:"numberOfValidationEntriesPerRule"`

	// Processing statistics
	FilesProcessed int           `json:"filesProcessed"`
	ProcessingTime time.Duration `json:"processingTimeMs"`

	// Error information (if validation failed)
	Error string `json:"error,omitempty"`

	// Cache information
	CacheHit bool   `json:"cacheHit,omitempty"`
	FileHash string `json:"fileHash,omitempty"`

	// Rules lists the XPath business rules that were configured for this validation.
	Rules []RuleInfo `json:"-"`

	// RuleChecksSkipped lists files whose XPath rule checks did not run because
	// schema validation found errors (or the findings cap was reached) first.
	RuleChecksSkipped []string `json:"ruleChecksSkipped,omitempty"`

	// FailedFiles lists files of a dataset that could not be validated, as "file: error".
	FailedFiles []string `json:"failedFiles,omitempty"`

	// Truncated is set when the findings cap (MaxFindings) was reached, so
	// validation stopped early and the results may be incomplete.
	Truncated bool `json:"truncated,omitempty"`

	// Raw content for statistics (not serialized to JSON)
	rawContent map[string][]byte `json:"-"`
}

// RuleInfo describes a configured validation rule.
type RuleInfo struct {
	ID       string
	Name     string
	Severity types.Severity
}

// ValidationReportEntry represents a single validation issue
type ValidationReportEntry struct {
	// RuleID is the full rule ID, e.g. "line/2" or "eu/line/2" for rules from JSON rule sets
	RuleID   string                   `json:"ruleId,omitempty"`
	Name     string                   `json:"name"`
	Message  string                   `json:"message"`
	Severity types.Severity           `json:"severity"`
	FileName string                   `json:"fileName"`
	Location ValidationReportLocation `json:"location"`
}

// ValidationReportLocation provides location information for a validation issue
type ValidationReportLocation struct {
	FileName   string `json:"FileName"`
	LineNumber int    `json:"LineNumber"`
	XPath      string `json:"XPath"`
	ElementID  string `json:"ElementID"`
}

// Summary returns a summary of validation results
func (r *ValidationResult) Summary() ValidationSummary {
	summary := ValidationSummary{
		TotalIssues:      len(r.ValidationReportEntries),
		FilesProcessed:   r.FilesProcessed,
		ProcessingTime:   r.ProcessingTime,
		HasErrors:        false,
		IssuesBySeverity: make(map[types.Severity]int),
	}

	for _, entry := range r.ValidationReportEntries {
		summary.IssuesBySeverity[entry.Severity]++
		if entry.Severity >= types.ERROR {
			summary.HasErrors = true
		}
	}

	return summary
}

// ValidationSummary provides a high-level summary of validation results
type ValidationSummary struct {
	TotalIssues      int                    `json:"totalIssues"`
	FilesProcessed   int                    `json:"filesProcessed"`
	ProcessingTime   time.Duration          `json:"processingTimeMs"`
	HasErrors        bool                   `json:"hasErrors"`
	IssuesBySeverity map[types.Severity]int `json:"issuesBySeverity"`
}

// IsValid returns true if validation passed (no errors or critical issues)
func (r *ValidationResult) IsValid() bool {
	if len(r.FailedFiles) > 0 {
		return false
	}
	for _, entry := range r.ValidationReportEntries {
		if entry.Severity >= types.ERROR {
			return false
		}
	}
	return r.Error == ""
}

// GetIssuesByFile returns validation issues grouped by filename
func (r *ValidationResult) GetIssuesByFile() map[string][]ValidationReportEntry {
	result := make(map[string][]ValidationReportEntry)

	for _, entry := range r.ValidationReportEntries {
		fileName := entry.FileName
		if fileName == "" {
			fileName = "unknown"
		}
		result[fileName] = append(result[fileName], entry)
	}

	return result
}

// GetIssuesBySeverity returns validation issues grouped by severity
func (r *ValidationResult) GetIssuesBySeverity() map[types.Severity][]ValidationReportEntry {
	result := make(map[types.Severity][]ValidationReportEntry)

	for _, entry := range r.ValidationReportEntries {
		result[entry.Severity] = append(result[entry.Severity], entry)
	}

	return result
}

// ToJSON converts the validation result to optimized grouped JSON format by default
func (r *ValidationResult) ToJSON() ([]byte, error) {
	// Use optimized grouped format by default for better user display
	return r.ToOptimizedJSON()
}

// ToFlatJSON converts the validation result to flat JSON format (original format)
func (r *ValidationResult) ToFlatJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// String returns a human-readable string representation
func (r *ValidationResult) String() string {
	if r.Error != "" {
		return "Validation failed: " + r.Error
	}

	summary := r.Summary()
	if summary.TotalIssues == 0 {
		return "Validation passed: No issues found"
	}

	return fmt.Sprintf("Validation completed: %d issues found (%d files processed)",
		summary.TotalIssues, summary.FilesProcessed)
}

// ToHTML converts the validation result to HTML format
func (r *ValidationResult) ToHTML() ([]byte, error) {
	if r.Error != "" {
		return []byte(fmt.Sprintf("<html><body><h1>Validation Error</h1><p>%s</p></body></html>", r.Error)), nil
	}

	reporter := NewHTMLReporter()
	html, err := reporter.GenerateHTML(r)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTML report: %w", err)
	}

	return []byte(html), nil
}

// GetEnhancedErrors returns validation issues as enhanced errors with suggestions and context.
func (r *ValidationResult) GetEnhancedErrors() []*errors.ValidationError {
	enhancedErrors := make([]*errors.ValidationError, 0, len(r.ValidationReportEntries))

	for _, entry := range r.ValidationReportEntries {
		enhancedError := r.convertToEnhancedError(entry)
		enhancedErrors = append(enhancedErrors, enhancedError)
	}

	return enhancedErrors
}

// convertToEnhancedError converts a ValidationReportEntry to an enhanced ValidationError.
func (r *ValidationResult) convertToEnhancedError(entry ValidationReportEntry) *errors.ValidationError {
	// Create base error
	enhancedError := errors.NewValidationError("VALIDATION_ISSUE", entry.Message).
		WithFile(entry.FileName).
		WithLocation(entry.Location.LineNumber, 0).
		WithSeverity(entry.Severity).
		WithDetails(entry.Message)

	// Add rule-specific enhancements based on the entry name/message
	r.enhanceErrorBasedOnRule(enhancedError, entry)

	return enhancedError
}

// enhanceErrorBasedOnRule adds rule-specific suggestions and context to the error.
func (r *ValidationResult) enhanceErrorBasedOnRule(err *errors.ValidationError, entry ValidationReportEntry) {
	ruleName := entry.Name
	message := entry.Message

	// Set rule information
	_ = err.WithRule(ruleName)

	// Add rule-specific enhancements
	switch {
	case contains(ruleName, "missing Name") || contains(message, "missing Name"):
		r.enhanceNameMissingError(err, entry)
	case contains(ruleName, "missing TransportMode") || contains(message, "TransportMode"):
		r.enhanceTransportModeError(err, entry)
	case contains(ruleName, "missing OperatorRef") || contains(message, "OperatorRef"):
		r.enhanceOperatorRefError(err, entry)
	case contains(ruleName, "missing LineRef") || contains(message, "LineRef"):
		r.enhanceLineRefError(err, entry)
	case contains(ruleName, "departure") && contains(message, "time"):
		r.enhanceDepartureTimeError(err, entry)
	case contains(ruleName, "arrival") && contains(message, "time"):
		r.enhanceArrivalTimeError(err, entry)
	case contains(ruleName, "schema") || contains(message, "schema"):
		r.enhanceSchemaError(err, entry)
	case contains(ruleName, "reference") || contains(message, "reference"):
		r.enhanceReferenceError(err, entry)
	default:
		r.enhanceGenericError(err, entry)
	}
}

// enhanceNameMissingError adds suggestions for missing Name elements.
func (r *ValidationResult) enhanceNameMissingError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Add a descriptive <Name> element to provide user-friendly identification",
		"Names should be clear and meaningful to passengers",
		"Use the local language appropriate for your region",
		"Avoid technical codes or abbreviations in passenger-facing names",
	).WithContext("element_type", "Name").
		WithContext("requirement", "mandatory")
}

// enhanceTransportModeError adds suggestions for transport mode issues.
func (r *ValidationResult) enhanceTransportModeError(err *errors.ValidationError, entry ValidationReportEntry) {
	validModes := []string{"bus", "rail", "tram", "metro", "coach", "water", "air", "taxi", "cableway", "funicular"}

	_ = err.WithSuggestions(
		"Add a <TransportMode> element specifying the type of transport service",
		fmt.Sprintf("Valid transport modes: %v", validModes),
		"Choose 'bus' for most road-based public transport services",
		"Use 'rail' for train services, 'tram' for light rail",
		"Select 'metro' for subway/underground services",
	).WithContext("valid_modes", validModes).
		WithContext("element_type", "TransportMode")
}

// enhanceOperatorRefError adds suggestions for OperatorRef issues.
func (r *ValidationResult) enhanceOperatorRefError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Add an <OperatorRef> element referencing the operating company",
		"Operators must be defined in the ResourceFrame",
		"Ensure the operator ID follows your codespace naming convention",
		"Reference format should be: 'Codespace:Operator:OperatorId'",
		"Verify that the referenced Operator exists in the same file",
	).WithContext("reference_type", "OperatorRef").
		WithContext("target_frame", "ResourceFrame")
}

// enhanceLineRefError adds suggestions for LineRef issues.
func (r *ValidationResult) enhanceLineRefError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Add a <LineRef> element referencing the parent Line",
		"Lines must be defined in the ServiceFrame",
		"Ensure the Line ID follows your codespace naming convention",
		"Reference format should be: 'Codespace:Line:LineId'",
		"Routes and ServiceJourneys must reference valid Lines",
	).WithContext("reference_type", "LineRef").
		WithContext("target_frame", "ServiceFrame")
}

// enhanceDepartureTimeError adds suggestions for departure time issues.
func (r *ValidationResult) enhanceDepartureTimeError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Add a <DepartureTime> element to the TimetabledPassingTime",
		"Departure times are mandatory for the first stop of each journey",
		"Use 24-hour format: HH:MM:SS (e.g., '09:30:00')",
		"Times should be in the local timezone of the service",
		"Ensure departure time is before or equal to arrival time at the same stop",
	).WithContext("time_type", "departure").
		WithContext("format", "HH:MM:SS")
}

// enhanceArrivalTimeError adds suggestions for arrival time issues.
func (r *ValidationResult) enhanceArrivalTimeError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Add an <ArrivalTime> element to the TimetabledPassingTime",
		"Arrival times are mandatory for the last stop of each journey",
		"Use 24-hour format: HH:MM:SS (e.g., '17:45:00')",
		"Times should be in the local timezone of the service",
		"Ensure arrival time is after or equal to departure time at the same stop",
	).WithContext("time_type", "arrival").
		WithContext("format", "HH:MM:SS")
}

// enhanceSchemaError adds suggestions for XML schema validation issues.
func (r *ValidationResult) enhanceSchemaError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Check that all required elements are present and properly nested",
		"Verify element names match the NetEX schema exactly (case-sensitive)",
		"Ensure all XML tags are properly closed",
		"Validate attribute values conform to expected formats",
		"Use an XML editor with NetEX schema validation for real-time feedback",
		"Check the NetEX schema documentation for element requirements",
	).WithContext("validation_type", "schema")
	_ = err.WithContext("xpath", entry.Location.XPath)
}

// enhanceReferenceError adds suggestions for reference validation issues.
func (r *ValidationResult) enhanceReferenceError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Ensure the referenced element exists in the appropriate frame",
		"Check for typos in the reference ID",
		"Verify the reference follows the correct naming convention",
		"Make sure referenced elements are defined before they are used",
		"Cross-check that all references point to valid, existing elements",
	).WithContext("validation_type", "reference")
}

// enhanceGenericError adds general suggestions for unknown validation issues.
func (r *ValidationResult) enhanceGenericError(err *errors.ValidationError, entry ValidationReportEntry) {
	_ = err.WithSuggestions(
		"Review the EU NeTEx Profile documentation for this validation rule",
		"Check similar valid NetEX files to understand the correct structure",
		"Consider if this is a data quality issue that needs fixing at the source",
		"Consult the NetEX validation rule documentation for specific guidance",
		"If this error persists, consider filing a support request with details",
	).WithContext("validation_type", "generic")
}

// GetEnhancedErrorsAsText returns all validation errors formatted as human-readable text.
func (r *ValidationResult) GetEnhancedErrorsAsText() string {
	enhancedErrors := r.GetEnhancedErrors()
	formatter := errors.NewErrorFormatter()
	return formatter.FormatAsList(enhancedErrors)
}

// GetErrorsForRule returns all validation errors for a specific rule with enhanced information.
func (r *ValidationResult) GetErrorsForRule(ruleName string) []*errors.ValidationError {
	enhancedErrors := make([]*errors.ValidationError, 0)

	for _, entry := range r.ValidationReportEntries {
		if entry.Name == ruleName {
			enhancedError := r.convertToEnhancedError(entry)
			enhancedErrors = append(enhancedErrors, enhancedError)
		}
	}

	return enhancedErrors
}

// GetErrorsByServerity returns validation errors grouped by severity level.
func (r *ValidationResult) GetErrorsBySeverity() map[types.Severity][]*errors.ValidationError {
	result := make(map[types.Severity][]*errors.ValidationError)
	enhancedErrors := r.GetEnhancedErrors()

	for _, err := range enhancedErrors {
		result[err.Severity] = append(result[err.Severity], err)
	}

	return result
}

// SetRawContent stores raw XML content for statistics extraction
func (r *ValidationResult) SetRawContent(fileName string, content []byte) {
	if r.rawContent == nil {
		r.rawContent = make(map[string][]byte)
	}
	r.rawContent[fileName] = content
}

// contains is a helper function to check if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}
