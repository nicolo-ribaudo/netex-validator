package validator

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/theoremus-urban-solutions/netex-validator/types"
)

// Test data - minimal valid NetEX XML
const validNetexXML = `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.0">
	<PublicationTimestamp>2023-01-01T12:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="TEST:CompositeFrame:1" version="1">
			<Name>Test Frame</Name>
			<TypeOfFrameRef ref="TEST:TypeOfFrame:1"/>
			<ServiceFrame id="TEST:ServiceFrame:1" version="1">
				<Name>Test Service Frame</Name>
				<lines>
					<Line id="TEST:Line:1" version="1">
						<Name>Test Line</Name>
						<PublicCode>1</PublicCode>
						<TransportMode>bus</TransportMode>
						<TransportSubmode>localBus</TransportSubmode>
						<OperatorRef ref="TEST:Operator:1"/>
					</Line>
				</lines>
			</ServiceFrame>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`

// Test data - invalid NetEX XML (missing required elements)
const invalidNetexXML = `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.0">
	<PublicationTimestamp>2023-01-01T12:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="TEST:CompositeFrame:1" version="1">
			<ServiceFrame id="TEST:ServiceFrame:1" version="1">
				<lines>
					<Line id="TEST:Line:1" version="1">
						<!-- Missing Name, PublicCode, TransportMode -->
					</Line>
				</lines>
			</ServiceFrame>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`

func TestNew(t *testing.T) {
	validator, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if validator == nil {
		t.Fatal("New() returned nil validator")
	}

	if validator != nil && validator.config == nil {
		t.Error("validator config is nil")
	}

	if validator != nil && validator.runner == nil {
		t.Error("validator runner is nil")
	}
}

func TestNewWithOptions(t *testing.T) {
	options := testOptions(t).
		WithCodespace("TEST").
		WithVerbose(true).
		WithSkipSchema(true) // Skip schema for tests to avoid network timeouts

	validator, err := NewWithOptions(options)
	if err != nil {
		t.Fatalf("NewWithOptions() failed: %v", err)
	}

	if validator == nil {
		t.Fatal("NewWithOptions() returned nil validator")
	}
}

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		filename     string
		expectError  bool
		expectIssues bool
		options      *ValidationOptions
	}{
		{
			name:         "valid NetEX content",
			content:      validNetexXML,
			filename:     "test.xml",
			expectError:  false,
			expectIssues: false, // May have some issues, but should not error
			options:      testOptions(t).WithCodespace("TEST").WithSkipSchema(true),
		},
		{
			name:         "invalid NetEX content",
			content:      invalidNetexXML,
			filename:     "invalid.xml",
			expectError:  false,
			expectIssues: true, // Should find validation issues
			options:      testOptions(t).WithCodespace("TEST").WithSkipSchema(true),
		},
		{
			name:         "malformed XML",
			content:      `<?xml version="1.0"?><invalid><unclosed>`,
			filename:     "malformed.xml",
			expectError:  false, // Should handle gracefully
			expectIssues: true,  // Should report XML issues
			options:      testOptions(t).WithCodespace("TEST").WithSkipSchema(false).WithAllowSchemaNetwork(false).WithSchemaTimeoutSeconds(5),
		},
		{
			name:         "empty content",
			content:      "",
			filename:     "empty.xml",
			expectError:  false,
			expectIssues: true, // Should report as invalid
			options:      testOptions(t).WithCodespace("TEST").WithSkipSchema(false).WithAllowSchemaNetwork(false).WithSchemaTimeoutSeconds(5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateContent([]byte(tt.content), tt.filename, tt.options)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("result is nil")
			}

			// Check if issues were found as expected
			var hasIssues bool
			if result != nil {
				hasIssues = len(result.ValidationReportEntries) > 0
			}
			if tt.expectIssues && !hasIssues {
				t.Error("expected validation issues but found none")
			}

			// Verify result structure
			if result != nil && result.CreationDate.IsZero() {
				t.Error("creation date is zero")
			}

			if result != nil && result.ProcessingTime < 0 {
				t.Error("negative processing time")
			}
		})
	}
}

func TestValidateFile(t *testing.T) {
	// Create temporary test file
	tmpFile, err := os.CreateTemp("", "netex_test_*.xml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			t.Logf("Failed to remove temp file: %v", removeErr)
		}
	}()

	if _, err := tmpFile.WriteString(validNetexXML); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	_ = tmpFile.Close()

	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile(tmpFile.Name(), options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if result != nil && result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.FilesProcessed)
	}
}

func TestValidateFile_NonExistent(t *testing.T) {
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile("nonexistent.xml", options)
	if err != nil {
		t.Fatalf("ValidateFile() with non-existent file should not error: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	// Should have an error in the result
	if result != nil && result.Error == "" {
		t.Error("expected error in result for non-existent file")
	}
}

func TestValidationResult_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		result   *ValidationResult
		expected bool
	}{
		{
			name: "no issues",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{},
				Error:                   "",
			},
			expected: true,
		},
		{
			name: "info issues only",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{
					{Severity: types.INFO},
				},
				Error: "",
			},
			expected: true,
		},
		{
			name: "warning issues only",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{
					{Severity: types.WARNING},
				},
				Error: "",
			},
			expected: true,
		},
		{
			name: "error issues",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{
					{Severity: types.ERROR},
				},
				Error: "",
			},
			expected: false,
		},
		{
			name: "critical issues",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{
					{Severity: types.CRITICAL},
				},
				Error: "",
			},
			expected: false,
		},
		{
			name: "has error string",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{},
				Error:                   "validation failed",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.result.IsValid()
			if actual != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestValidationResult_Summary(t *testing.T) {
	result := &ValidationResult{
		ValidationReportEntries: []ValidationReportEntry{
			{Severity: types.INFO},
			{Severity: types.WARNING},
			{Severity: types.ERROR},
			{Severity: types.ERROR},
			{Severity: types.CRITICAL},
		},
		FilesProcessed: 2,
		ProcessingTime: time.Second,
	}

	summary := result.Summary()

	if summary.TotalIssues != 5 {
		t.Errorf("expected 5 total issues, got %d", summary.TotalIssues)
	}

	if summary.FilesProcessed != 2 {
		t.Errorf("expected 2 files processed, got %d", summary.FilesProcessed)
	}

	if summary.ProcessingTime != time.Second {
		t.Errorf("expected 1s processing time, got %v", summary.ProcessingTime)
	}

	if !summary.HasErrors {
		t.Error("expected HasErrors to be true")
	}

	expectedCounts := map[types.Severity]int{
		types.INFO:     1,
		types.WARNING:  1,
		types.ERROR:    2,
		types.CRITICAL: 1,
	}

	for severity, expectedCount := range expectedCounts {
		if count := summary.IssuesBySeverity[severity]; count != expectedCount {
			t.Errorf("expected %d %v issues, got %d", expectedCount, severity, count)
		}
	}
}

func TestValidationResult_GetIssuesByFile(t *testing.T) {
	result := &ValidationResult{
		ValidationReportEntries: []ValidationReportEntry{
			{FileName: "file1.xml", Name: "Issue1"},
			{FileName: "file1.xml", Name: "Issue2"},
			{FileName: "file2.xml", Name: "Issue3"},
			{FileName: "", Name: "Issue4"}, // Empty filename
		},
	}

	issuesByFile := result.GetIssuesByFile()

	if len(issuesByFile) != 3 {
		t.Errorf("expected 3 file groups, got %d", len(issuesByFile))
	}

	if len(issuesByFile["file1.xml"]) != 2 {
		t.Errorf("expected 2 issues for file1.xml, got %d", len(issuesByFile["file1.xml"]))
	}

	if len(issuesByFile["file2.xml"]) != 1 {
		t.Errorf("expected 1 issue for file2.xml, got %d", len(issuesByFile["file2.xml"]))
	}

	if len(issuesByFile["unknown"]) != 1 {
		t.Errorf("expected 1 issue for unknown file, got %d", len(issuesByFile["unknown"]))
	}
}

func TestValidationResult_GetIssuesBySeverity(t *testing.T) {
	result := &ValidationResult{
		ValidationReportEntries: []ValidationReportEntry{
			{Severity: types.INFO, Name: "Info1"},
			{Severity: types.WARNING, Name: "Warning1"},
			{Severity: types.WARNING, Name: "Warning2"},
			{Severity: types.ERROR, Name: "Error1"},
		},
	}

	issuesBySeverity := result.GetIssuesBySeverity()

	if len(issuesBySeverity[types.INFO]) != 1 {
		t.Errorf("expected 1 INFO issue, got %d", len(issuesBySeverity[types.INFO]))
	}

	if len(issuesBySeverity[types.WARNING]) != 2 {
		t.Errorf("expected 2 WARNING issues, got %d", len(issuesBySeverity[types.WARNING]))
	}

	if len(issuesBySeverity[types.ERROR]) != 1 {
		t.Errorf("expected 1 ERROR issue, got %d", len(issuesBySeverity[types.ERROR]))
	}

	if len(issuesBySeverity[types.CRITICAL]) != 0 {
		t.Errorf("expected 0 CRITICAL issues, got %d", len(issuesBySeverity[types.CRITICAL]))
	}
}

func TestValidationResult_ToJSON(t *testing.T) {
	result := &ValidationResult{
		Codespace:          "TEST",
		ValidationReportID: "test-report",
		ValidationReportEntries: []ValidationReportEntry{
			{Name: "Test Issue", Message: "Test message", Severity: types.WARNING},
		},
		FilesProcessed: 1,
	}

	jsonData, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data is empty")
	}

	// Basic validation that it contains expected fields for optimized format
	jsonStr := string(jsonData)
	expectedFields := []string{
		`"codespace"`,
		`"validationReportId"`,
		`"summary"`,
		`"notices"`,
		`"statistics"`,
		`"filesProcessed"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing expected field: %s", field)
		}
	}
}

func TestValidationResult_ToHTML(t *testing.T) {
	result := &ValidationResult{
		Codespace:          "TEST",
		ValidationReportID: "test-report",
		ValidationReportEntries: []ValidationReportEntry{
			{Name: "Test Issue", Message: "Test message", Severity: types.WARNING},
		},
		FilesProcessed: 1,
	}

	htmlData, err := result.ToHTML()
	if err != nil {
		t.Fatalf("ToHTML() failed: %v", err)
	}

	if len(htmlData) == 0 {
		t.Error("HTML data is empty")
	}

	htmlStr := string(htmlData)

	// Basic validation that it's HTML
	if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
		t.Error("HTML missing DOCTYPE declaration")
	}

	if !strings.Contains(htmlStr, "<html") {
		t.Error("HTML missing html tag")
	}

	if !strings.Contains(htmlStr, "NetEX Validation Report") {
		t.Error("HTML missing expected title")
	}
}

func TestValidationResult_String(t *testing.T) {
	tests := []struct {
		name     string
		result   *ValidationResult
		contains string
	}{
		{
			name: "validation error",
			result: &ValidationResult{
				Error: "test error",
			},
			contains: "Validation failed: test error",
		},
		{
			name: "no issues",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{},
				FilesProcessed:          1,
			},
			contains: "Validation passed: No issues found",
		},
		{
			name: "with issues",
			result: &ValidationResult{
				ValidationReportEntries: []ValidationReportEntry{
					{Name: "Issue1"}, {Name: "Issue2"},
				},
				FilesProcessed: 2,
			},
			contains: "Validation completed: 2 issues found (2 files processed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.result.String()
			if !strings.Contains(str, tt.contains) {
				t.Errorf("String() = %q, expected to contain %q", str, tt.contains)
			}
		})
	}
}
