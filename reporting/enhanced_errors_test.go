package reporting

import (
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

const (
	// Test constants for repeated strings
	testOperatorRef = "OperatorRef"
	testLine2       = "line/2"
)

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError("TEST_001", "Test error message").
		WithFile("test.xml").
		WithLocation(42, 10).
		WithRule("TEST_RULE").
		WithDetails("Additional details")

	errorStr := err.Error()

	// Check that all components are present
	if !strings.Contains(errorStr, "test.xml:42:10") {
		t.Errorf("Expected location in error string, got: %s", errorStr)
	}

	if !strings.Contains(errorStr, "[TEST_001]") {
		t.Errorf("Expected error code in error string, got: %s", errorStr)
	}

	if !strings.Contains(errorStr, "Rule: TEST_RULE") {
		t.Errorf("Expected rule in error string, got: %s", errorStr)
	}

	if !strings.Contains(errorStr, "Test error message") {
		t.Errorf("Expected message in error string, got: %s", errorStr)
	}

	if !strings.Contains(errorStr, "Details: Additional details") {
		t.Errorf("Expected details in error string, got: %s", errorStr)
	}
}

func TestValidationError_GetFormattedMessage(t *testing.T) {
	err := NewValidationError("TEST_001", "Test error message").
		WithFile("test.xml").
		WithLocation(42, 10).
		WithRule("TEST_RULE").
		WithSeverity(types.ERROR).
		WithDetails("Additional details").
		WithSuggestion("First suggestion").
		WithSuggestion("Second suggestion").
		WithContext("context_key", "context_value").
		WithRelatedIssue("RELATED_001")

	formatted := err.GetFormattedMessage()

	// Check for expected emoji markers and content
	if !strings.Contains(formatted, "❌") {
		t.Error("Expected error emoji in formatted message")
	}

	if !strings.Contains(formatted, "📁 File: test.xml (line 42, column 10)") {
		t.Error("Expected file location in formatted message")
	}

	if !strings.Contains(formatted, "📋 Rule: TEST_RULE") {
		t.Error("Expected rule in formatted message")
	}

	if !strings.Contains(formatted, "🔍 Code: TEST_001") {
		t.Error("Expected error code in formatted message")
	}

	if !strings.Contains(formatted, "💡 Suggestions:") {
		t.Error("Expected suggestions section in formatted message")
	}

	if !strings.Contains(formatted, "1. First suggestion") {
		t.Error("Expected first suggestion in formatted message")
	}

	if !strings.Contains(formatted, "2. Second suggestion") {
		t.Error("Expected second suggestion in formatted message")
	}

	if !strings.Contains(formatted, "🔧 Context:") {
		t.Error("Expected context section in formatted message")
	}

	if !strings.Contains(formatted, "context_key: context_value") {
		t.Error("Expected context information in formatted message")
	}

	if !strings.Contains(formatted, "🔗 Related issues: RELATED_001") {
		t.Error("Expected related issues in formatted message")
	}
}

func TestValidationError_Chaining(t *testing.T) {
	err := NewValidationError("CHAIN_001", "Chaining test").
		WithFile("chain.xml").
		WithLocation(1, 1).
		WithRule("CHAIN_RULE").
		WithSeverity(types.WARNING).
		WithDetails("Chain details").
		WithSuggestion("Chain suggestion").
		WithContext("chain", "value").
		WithRelatedIssue("CHAIN_002")

	if err.Code != "CHAIN_001" {
		t.Errorf("Expected code 'CHAIN_001', got: %s", err.Code)
	}

	if err.Message != "Chaining test" {
		t.Errorf("Expected message 'Chaining test', got: %s", err.Message)
	}

	if err.File != "chain.xml" {
		t.Errorf("Expected file 'chain.xml', got: %s", err.File)
	}

	if err.Line != 1 {
		t.Errorf("Expected line 1, got: %d", err.Line)
	}

	if err.Column != 1 {
		t.Errorf("Expected column 1, got: %d", err.Column)
	}

	if err.Rule != "CHAIN_RULE" {
		t.Errorf("Expected rule 'CHAIN_RULE', got: %s", err.Rule)
	}

	if err.Severity != types.WARNING {
		t.Errorf("Expected severity WARNING, got: %v", err.Severity)
	}

	if err.Details != "Chain details" {
		t.Errorf("Expected details 'Chain details', got: %s", err.Details)
	}

	if len(err.Suggestions) != 1 || err.Suggestions[0] != "Chain suggestion" {
		t.Errorf("Expected one suggestion 'Chain suggestion', got: %v", err.Suggestions)
	}

	if err.Context["chain"] != "value" {
		t.Errorf("Expected context 'chain' = 'value', got: %v", err.Context["chain"])
	}

	if len(err.RelatedIssues) != 1 || err.RelatedIssues[0] != "CHAIN_002" {
		t.Errorf("Expected one related issue 'CHAIN_002', got: %v", err.RelatedIssues)
	}
}

func TestNewSchemaValidationError(t *testing.T) {
	err := NewSchemaValidationError("schema.xml", 10, "Element not allowed")

	if err.Code != "SCHEMA_001" {
		t.Errorf("Expected code 'SCHEMA_001', got: %s", err.Code)
	}

	if err.File != "schema.xml" {
		t.Errorf("Expected file 'schema.xml', got: %s", err.File)
	}

	if err.Line != 10 {
		t.Errorf("Expected line 10, got: %d", err.Line)
	}

	if err.Severity != types.ERROR {
		t.Errorf("Expected severity ERROR, got: %v", err.Severity)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions for schema validation error")
	}

	// Check for specific suggestions
	found := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, "required elements") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected suggestion about required elements")
	}
}

func TestNewMissingElementError(t *testing.T) {
	err := NewMissingElementError("missing.xml", "Name", "Line", 15)

	if !strings.Contains(err.Message, "Missing required element: Name") {
		t.Errorf("Expected message about missing Name element, got: %s", err.Message)
	}

	if err.Context["missing_element"] != "Name" {
		t.Errorf("Expected context missing_element = 'Name', got: %v", err.Context["missing_element"])
	}

	if err.Context["parent_element"] != "Line" {
		t.Errorf("Expected context parent_element = 'Line', got: %v", err.Context["parent_element"])
	}

	// Check for specific suggestions based on element type
	foundNameSuggestion := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, "descriptive and user-friendly") {
			foundNameSuggestion = true
			break
		}
	}
	if !foundNameSuggestion {
		t.Error("Expected Name-specific suggestion for missing Name element")
	}
}

func TestNewInvalidTransportModeError(t *testing.T) {
	err := NewInvalidTransportModeError("transport.xml", "automobile", 20)

	if !strings.Contains(err.Message, "Invalid transport mode: automobile") {
		t.Errorf("Expected message about invalid transport mode, got: %s", err.Message)
	}

	if err.Context["invalid_mode"] != "automobile" {
		t.Errorf("Expected context invalid_mode = 'automobile', got: %v", err.Context["invalid_mode"])
	}

	// Check for valid modes in context
	validModes, ok := err.Context["valid_modes"].([]string)
	if !ok {
		t.Error("Expected valid_modes in context as []string")
	} else {
		found := false
		for _, mode := range validModes {
			if mode == "bus" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'bus' in valid modes list")
		}
	}

	// Check for suggestions with valid modes
	foundValidModes := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, "Valid modes:") {
			foundValidModes = true
			break
		}
	}
	if !foundValidModes {
		t.Error("Expected suggestion listing valid modes")
	}
}

func TestNewInvalidReferenceError(t *testing.T) {
	err := NewInvalidReferenceError("ref.xml", "OperatorRef", "NO:Operator:Missing", "Operator", 25)

	if !strings.Contains(err.Message, "Invalid OperatorRef: NO:Operator:Missing") {
		t.Errorf("Expected message about invalid OperatorRef, got: %s", err.Message)
	}

	if err.Context["reference_type"] != testOperatorRef {
		t.Errorf("Expected context reference_type = 'OperatorRef', got: %v", err.Context["reference_type"])
	}

	// Check for OperatorRef-specific suggestions
	foundOperatorSuggestion := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, "ResourceFrame") {
			foundOperatorSuggestion = true
			break
		}
	}
	if !foundOperatorSuggestion {
		t.Error("Expected OperatorRef-specific suggestion about ResourceFrame")
	}
}

func TestNewBusinessRuleViolationError(t *testing.T) {
	err := NewBusinessRuleViolationError("business.xml", testLine2, "Line missing Name", "Line element is missing required Name", 30)

	if err.Rule != testLine2 {
		t.Errorf("Expected rule 'line/2', got: %s", err.Rule)
	}

	if err.Context["rule_code"] != testLine2 {
		t.Errorf("Expected context rule_code = 'line/2', got: %v", err.Context["rule_code"])
	}

	// Check for line/2-specific suggestions
	foundLineSuggestion := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, "descriptive <Name> element") {
			foundLineSuggestion = true
			break
		}
	}
	if !foundLineSuggestion {
		t.Error("Expected line/2-specific suggestion about adding Name element")
	}
}

func TestErrorFormatter_FormatAsJSON(t *testing.T) {
	formatter := NewErrorFormatter()

	err := NewValidationError("JSON_001", "JSON test").
		WithFile("json.xml").
		WithLocation(5, 10).
		WithRule("JSON_RULE").
		WithSeverity(types.ERROR).
		WithDetails("JSON details").
		WithSuggestion("JSON suggestion").
		WithContext("json_key", "json_value")

	result := formatter.FormatAsJSON(err)

	if result["code"] != "JSON_001" {
		t.Errorf("Expected code 'JSON_001', got: %v", result["code"])
	}

	if result["message"] != "JSON test" {
		t.Errorf("Expected message 'JSON test', got: %v", result["message"])
	}

	if result["file"] != "json.xml" {
		t.Errorf("Expected file 'json.xml', got: %v", result["file"])
	}

	if result["rule"] != "JSON_RULE" {
		t.Errorf("Expected rule 'JSON_RULE', got: %v", result["rule"])
	}

	if result["severity"] != "ERROR" {
		t.Errorf("Expected severity 'ERROR', got: %v", result["severity"])
	}

	location, ok := result["location"].(map[string]interface{})
	if !ok {
		t.Error("Expected location as map[string]interface{}")
	} else {
		if location["line"] != 5 {
			t.Errorf("Expected location line 5, got: %v", location["line"])
		}
		if location["column"] != 10 {
			t.Errorf("Expected location column 10, got: %v", location["column"])
		}
	}

	suggestions, ok := result["suggestions"].([]string)
	if !ok {
		t.Error("Expected suggestions as []string")
	} else if len(suggestions) != 1 || suggestions[0] != "JSON suggestion" {
		t.Errorf("Expected one suggestion 'JSON suggestion', got: %v", suggestions)
	}

	context, ok := result["context"].(map[string]interface{})
	if !ok {
		t.Error("Expected context as map[string]interface{}")
	} else if context["json_key"] != "json_value" {
		t.Errorf("Expected context json_key = 'json_value', got: %v", context["json_key"])
	}
}

func TestErrorFormatter_FormatAsList(t *testing.T) {
	formatter := NewErrorFormatter()

	errors := []*ValidationError{
		NewValidationError("LIST_001", "First error").WithSuggestion("First suggestion"),
		NewValidationError("LIST_002", "Second error").WithSuggestion("Second suggestion"),
	}

	result := formatter.FormatAsList(errors)

	if !strings.Contains(result, "Found 2 validation error(s)") {
		t.Error("Expected count of errors in formatted list")
	}

	if !strings.Contains(result, "1. [LIST_001]") {
		t.Error("Expected first error in formatted list")
	}

	if !strings.Contains(result, "2. [LIST_002]") {
		t.Error("Expected second error in formatted list")
	}

	if !strings.Contains(result, "- First suggestion") {
		t.Error("Expected first suggestion in formatted list")
	}

	if !strings.Contains(result, "- Second suggestion") {
		t.Error("Expected second suggestion in formatted list")
	}
}

func TestErrorFormatter_FormatAsList_Empty(t *testing.T) {
	formatter := NewErrorFormatter()

	result := formatter.FormatAsList([]*ValidationError{})

	if result != "No validation errors found." {
		t.Errorf("Expected 'No validation errors found.' for empty list, got: %s", result)
	}
}

func TestGetBusinessRuleSuggestions(t *testing.T) {
	// Test specific rule suggestions
	suggestions := getBusinessRuleSuggestions("eu/line/2", "Line missing Name")

	foundNameSuggestion := false
	for _, suggestion := range suggestions {
		if strings.Contains(suggestion, "descriptive <Name> element") {
			foundNameSuggestion = true
			break
		}
	}
	if !foundNameSuggestion {
		t.Error("Expected line/2-specific suggestion about Name element")
	}

	// Test unknown rule
	unknownSuggestions := getBusinessRuleSuggestions("UNKNOWN_RULE", "Unknown message")

	foundGeneric := false
	for _, suggestion := range unknownSuggestions {
		if strings.Contains(suggestion, "rule documentation") {
			foundGeneric = true
			break
		}
	}
	if !foundGeneric {
		t.Error("Expected generic suggestion for unknown rule")
	}
}
