package reporting

import (
	"fmt"
	"strings"

	"github.com/theoremus-urban-solutions/netex-validator/types"
)

// ValidationError represents an enhanced validation error with context and suggestions.
type ValidationError struct {
	// Code is the error code (e.g., "VALIDATION_001")
	Code string
	// Message is the primary error message
	Message string
	// Details provides additional context about the error
	Details string
	// File is the filename where the error occurred
	File string
	// Line is the line number where the error occurred (if available)
	Line int
	// Column is the column number where the error occurred (if available)
	Column int
	// Rule is the validation rule that was violated (if applicable)
	Rule string
	// Severity indicates the severity level of the error
	Severity types.Severity
	// Suggestions provides actionable suggestions for fixing the error
	Suggestions []string
	// Context provides additional context about the validation
	Context map[string]interface{}
	// RelatedIssues contains references to related validation issues
	RelatedIssues []string
	// Cause is the underlying error that caused this validation error
	Cause error
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	var parts []string

	if e.File != "" {
		if e.Line > 0 {
			if e.Column > 0 {
				parts = append(parts, fmt.Sprintf("%s:%d:%d", e.File, e.Line, e.Column))
			} else {
				parts = append(parts, fmt.Sprintf("%s:%d", e.File, e.Line))
			}
		} else {
			parts = append(parts, e.File)
		}
	}

	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}

	if e.Rule != "" {
		parts = append(parts, fmt.Sprintf("Rule: %s", e.Rule))
	}

	parts = append(parts, e.Message)

	if e.Details != "" {
		parts = append(parts, fmt.Sprintf("Details: %s", e.Details))
	}

	return strings.Join(parts, " - ")
}

// GetFormattedMessage returns a user-friendly formatted error message with suggestions.
func (e *ValidationError) GetFormattedMessage() string {
	var builder strings.Builder

	// Error header
	builder.WriteString(fmt.Sprintf("❌ %s Error: %s\n", e.Severity, e.Message))

	// Location information
	if e.File != "" {
		builder.WriteString(fmt.Sprintf("📁 File: %s", e.File))
		if e.Line > 0 {
			builder.WriteString(fmt.Sprintf(" (line %d", e.Line))
			if e.Column > 0 {
				builder.WriteString(fmt.Sprintf(", column %d", e.Column))
			}
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	}

	// Rule information
	if e.Rule != "" {
		builder.WriteString(fmt.Sprintf("📋 Rule: %s\n", e.Rule))
	}

	// Error code
	if e.Code != "" {
		builder.WriteString(fmt.Sprintf("🔍 Code: %s\n", e.Code))
	}

	// Details
	if e.Details != "" {
		builder.WriteString(fmt.Sprintf("📝 Details: %s\n", e.Details))
	}

	// Context information
	if len(e.Context) > 0 {
		builder.WriteString("🔧 Context:\n")
		for key, value := range e.Context {
			builder.WriteString(fmt.Sprintf("   • %s: %v\n", key, value))
		}
	}

	// Suggestions
	if len(e.Suggestions) > 0 {
		builder.WriteString("💡 Suggestions:\n")
		for i, suggestion := range e.Suggestions {
			builder.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
		}
	}

	// Related issues
	if len(e.RelatedIssues) > 0 {
		builder.WriteString("🔗 Related issues: ")
		builder.WriteString(strings.Join(e.RelatedIssues, ", "))
		builder.WriteString("\n")
	}

	// Underlying cause
	if e.Cause != nil {
		builder.WriteString(fmt.Sprintf("⚠️  Underlying cause: %s\n", e.Cause.Error()))
	}

	return builder.String()
}

// NewValidationError creates a new ValidationError with the provided parameters.
func NewValidationError(code, message string) *ValidationError {
	return &ValidationError{
		Code:        code,
		Message:     message,
		Context:     make(map[string]interface{}),
		Suggestions: make([]string, 0),
	}
}

// WithFile adds file information to the error.
func (e *ValidationError) WithFile(file string) *ValidationError {
	e.File = file
	return e
}

// WithLocation adds line and column information to the error.
func (e *ValidationError) WithLocation(line, column int) *ValidationError {
	e.Line = line
	e.Column = column
	return e
}

// WithRule adds validation rule information to the error.
func (e *ValidationError) WithRule(rule string) *ValidationError {
	e.Rule = rule
	return e
}

// WithSeverity sets the severity level of the error.
func (e *ValidationError) WithSeverity(severity types.Severity) *ValidationError {
	e.Severity = severity
	return e
}

// WithDetails adds detailed error information.
func (e *ValidationError) WithDetails(details string) *ValidationError {
	e.Details = details
	return e
}

// WithSuggestion adds a suggestion for fixing the error.
func (e *ValidationError) WithSuggestion(suggestion string) *ValidationError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithSuggestions adds multiple suggestions for fixing the error.
func (e *ValidationError) WithSuggestions(suggestions ...string) *ValidationError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// WithContext adds context information to the error.
func (e *ValidationError) WithContext(key string, value interface{}) *ValidationError {
	e.Context[key] = value
	return e
}

// WithRelatedIssue adds a reference to a related validation issue.
func (e *ValidationError) WithRelatedIssue(issue string) *ValidationError {
	e.RelatedIssues = append(e.RelatedIssues, issue)
	return e
}

// WithCause sets the underlying error that caused this validation error.
func (e *ValidationError) WithCause(cause error) *ValidationError {
	e.Cause = cause
	return e
}

// Common validation error creators with built-in suggestions

// NewSchemaValidationError creates a schema validation error with helpful suggestions.
func NewSchemaValidationError(file string, line int, message string) *ValidationError {
	return NewValidationError("SCHEMA_001", "XML Schema Validation Failed").
		WithFile(file).
		WithLocation(line, 0).
		WithSeverity(types.ERROR).
		WithDetails(message).
		WithSuggestions(
			"Check that all required elements are present",
			"Verify element names and spelling match the NetEX schema",
			"Ensure proper nesting of XML elements",
			"Validate that attribute values conform to expected formats",
			"Use an XML editor with schema validation to identify issues",
		)
}

// NewMissingElementError creates an error for missing required elements with context-specific suggestions.
func NewMissingElementError(file, element, parent string, line int) *ValidationError {
	suggestions := []string{
		fmt.Sprintf("Add the missing <%s> element inside <%s>", element, parent),
		"Check the NetEX documentation for required child elements",
		"Review similar valid files to see the correct structure",
	}

	// Add element-specific suggestions
	switch element {
	case "Name":
		suggestions = append(suggestions,
			"Names should be descriptive and user-friendly",
			"Use the local language appropriate for your region",
		)
	case "TransportMode":
		suggestions = append(suggestions,
			"Valid modes: bus, rail, tram, metro, coach, water, air, taxi, cableway, funicular",
			"Choose the mode that best describes your service type",
		)
	case "OperatorRef":
		suggestions = append(suggestions,
			"Reference must point to an existing Operator in the ResourceFrame",
			"Ensure the Operator ID follows your codespace naming convention",
		)
	}

	return NewValidationError("MISSING_ELEMENT", fmt.Sprintf("Missing required element: %s", element)).
		WithFile(file).
		WithLocation(line, 0).
		WithRule(fmt.Sprintf("%s_MISSING_%s", strings.ToUpper(parent), strings.ToUpper(element))).
		WithSeverity(types.ERROR).
		WithDetails(fmt.Sprintf("The <%s> element is required inside <%s> but was not found", element, parent)).
		WithContext("missing_element", element).
		WithContext("parent_element", parent).
		WithSuggestions(suggestions...)
}

// NewInvalidTransportModeError creates an error for invalid transport modes with valid options.
func NewInvalidTransportModeError(file, invalidMode string, line int) *ValidationError {
	validModes := []string{
		"bus", "rail", "tram", "metro", "coach", "water", "air", "taxi", "cableway", "funicular", "unknown",
	}

	suggestions := []string{
		fmt.Sprintf("Replace '%s' with one of the valid transport modes", invalidMode),
		fmt.Sprintf("Valid modes: %s", strings.Join(validModes, ", ")),
		"Choose 'bus' for most road-based public transport services",
		"Use 'rail' for train services, 'tram' for light rail/streetcars",
		"Select 'metro' for subway/underground services",
	}

	// Add specific suggestions based on the invalid mode
	switch strings.ToLower(invalidMode) {
	case "train":
		suggestions = append(suggestions, "Use 'rail' instead of 'train'")
	case "subway", "underground":
		suggestions = append(suggestions, "Use 'metro' instead of '"+invalidMode+"'")
	case "boat", "ferry", "ship":
		suggestions = append(suggestions, "Use 'water' for marine transport services")
	case "plane", "airplane":
		suggestions = append(suggestions, "Use 'air' for aviation services")
	}

	return NewValidationError("INVALID_TRANSPORT_MODE", fmt.Sprintf("Invalid transport mode: %s", invalidMode)).
		WithFile(file).
		WithLocation(line, 0).
		WithRule("TRANSPORT_MODE_VALIDATION").
		WithSeverity(types.ERROR).
		WithDetails(fmt.Sprintf("'%s' is not a valid NetEX transport mode", invalidMode)).
		WithContext("invalid_mode", invalidMode).
		WithContext("valid_modes", validModes).
		WithSuggestions(suggestions...)
}

// NewInvalidReferenceError creates an error for broken references with resolution suggestions.
func NewInvalidReferenceError(file, refType, refValue, targetType string, line int) *ValidationError {
	suggestions := []string{
		fmt.Sprintf("Ensure a %s with ID '%s' exists in the file", targetType, refValue),
		"Check for typos in the reference ID",
		"Verify the referenced element is in the correct frame (ServiceFrame, ResourceFrame, etc.)",
		"Ensure the referenced element appears before this reference in the file",
	}

	// Add reference-specific suggestions
	switch refType {
	case "OperatorRef":
		suggestions = append(suggestions,
			"Operators should be defined in the ResourceFrame",
			"Check that the Operator has the correct codespace prefix",
		)
	case "LineRef":
		suggestions = append(suggestions,
			"Lines should be defined in the ServiceFrame",
			"Ensure the Line exists and has the correct ID format",
		)
	case "JourneyPatternRef":
		suggestions = append(suggestions,
			"JourneyPatterns should be defined in the ServiceFrame",
			"Verify the JourneyPattern references a valid Route",
		)
	}

	return NewValidationError("INVALID_REFERENCE", fmt.Sprintf("Invalid %s: %s", refType, refValue)).
		WithFile(file).
		WithLocation(line, 0).
		WithRule("REFERENCE_VALIDATION").
		WithSeverity(types.ERROR).
		WithDetails(fmt.Sprintf("Reference '%s' does not point to a valid %s", refValue, targetType)).
		WithContext("reference_type", refType).
		WithContext("reference_value", refValue).
		WithContext("target_type", targetType).
		WithSuggestions(suggestions...)
}

// NewBusinessRuleViolationError creates an error for business rule violations with rule-specific guidance.
func NewBusinessRuleViolationError(file, ruleCode, ruleName, message string, line int) *ValidationError {
	suggestions := getBusinessRuleSuggestions(ruleCode, message)

	return NewValidationError("BUSINESS_RULE_VIOLATION", fmt.Sprintf("Business rule violation: %s", ruleName)).
		WithFile(file).
		WithLocation(line, 0).
		WithRule(ruleCode).
		WithSeverity(types.ERROR).
		WithDetails(message).
		WithContext("rule_code", ruleCode).
		WithContext("rule_name", ruleName).
		WithSuggestions(suggestions...)
}

// getBusinessRuleSuggestions returns rule-specific suggestions for common business rule violations.
func getBusinessRuleSuggestions(ruleCode, message string) []string {
	suggestions := []string{
		"Review the EU NeTEx Profile documentation for this rule",
		"Check similar valid files to see the correct implementation",
		"Consider if this is a data quality issue that needs fixing at the source",
	}

	// Match on the last namespace segment and the key, so that the outer
	// namespaces (e.g. "eu/line/2" vs "line/2") do not matter. This assumes the
	// shipped topic files are loaded under their usual names (via the eu.json
	// aggregate or as one of several --rules files).
	switch lastPathSegments(ruleCode, 2) {
	case "line/2":
		suggestions = append(suggestions,
			"Add a descriptive <Name> element to the Line",
			"Names should be passenger-friendly and match published information",
		)
	case "line/4":
		suggestions = append(suggestions,
			"Add a <TransportMode> element specifying the type of transport",
			"Common modes: bus, rail, tram, metro",
		)
	case "service_journey/5":
		suggestions = append(suggestions,
			"Add a <DepartureTime> to the first TimetabledPassingTime",
			"Departure times are mandatory for the first stop of each journey",
		)
	case "service_journey/6":
		suggestions = append(suggestions,
			"Add an <ArrivalTime> to the last TimetabledPassingTime",
			"Arrival times are mandatory for the last stop of each journey",
		)
	case "route/3":
		suggestions = append(suggestions,
			"Add a <LineRef> element referencing the parent Line",
			"Routes must be associated with a specific Line",
		)
	default:
		// Add generic suggestion for unknown rules
		suggestions = append(suggestions,
			"Consult the validation rule documentation for specific guidance",
		)
	}

	return suggestions
}

// NewPerformanceWarningError creates a warning for performance issues.
func NewPerformanceWarningError(operation string, duration, threshold int64) *ValidationError {
	return NewValidationError("PERFORMANCE_WARNING", fmt.Sprintf("Performance warning: %s took %dms", operation, duration)).
		WithSeverity(types.WARNING).
		WithDetails(fmt.Sprintf("Operation exceeded threshold of %dms", threshold)).
		WithContext("operation", operation).
		WithContext("duration_ms", duration).
		WithContext("threshold_ms", threshold).
		WithSuggestions(
			"Consider optimizing the validation rules or XML structure",
			"For large files, consider splitting into smaller datasets",
			"Enable schema skipping if schema validation is not required",
			"Use verbose logging to identify slow validation rules",
		)
}

// ErrorFormatter provides methods for formatting validation errors in different styles.
type ErrorFormatter struct{}

// NewErrorFormatter creates a new error formatter.
func NewErrorFormatter() *ErrorFormatter {
	return &ErrorFormatter{}
}

// FormatAsText formats the error as plain text.
func (f *ErrorFormatter) FormatAsText(err *ValidationError) string {
	return err.GetFormattedMessage()
}

// FormatAsJSON formats the error as JSON-like structure (for structured logging).
func (f *ErrorFormatter) FormatAsJSON(err *ValidationError) map[string]interface{} {
	result := map[string]interface{}{
		"code":     err.Code,
		"message":  err.Message,
		"severity": err.Severity.String(),
	}

	if err.File != "" {
		result["file"] = err.File
	}

	if err.Line > 0 {
		location := map[string]interface{}{"line": err.Line}
		if err.Column > 0 {
			location["column"] = err.Column
		}
		result["location"] = location
	}

	if err.Rule != "" {
		result["rule"] = err.Rule
	}

	if err.Details != "" {
		result["details"] = err.Details
	}

	if len(err.Suggestions) > 0 {
		result["suggestions"] = err.Suggestions
	}

	if len(err.Context) > 0 {
		result["context"] = err.Context
	}

	if len(err.RelatedIssues) > 0 {
		result["related_issues"] = err.RelatedIssues
	}

	if err.Cause != nil {
		result["cause"] = err.Cause.Error()
	}

	return result
}

// FormatAsList formats multiple errors as a numbered list.
func (f *ErrorFormatter) FormatAsList(errors []*ValidationError) string {
	if len(errors) == 0 {
		return "No validation errors found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d validation error(s):\n\n", len(errors)))

	for i, err := range errors {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.Error()))
		if len(err.Suggestions) > 0 {
			builder.WriteString("   Suggestions:\n")
			for _, suggestion := range err.Suggestions {
				builder.WriteString(fmt.Sprintf("   - %s\n", suggestion))
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// lastPathSegments returns the last n "/"-separated segments of a rule ID.
func lastPathSegments(id string, n int) string {
	parts := strings.Split(id, "/")
	if len(parts) > n {
		parts = parts[len(parts)-n:]
	}
	return strings.Join(parts, "/")
}
