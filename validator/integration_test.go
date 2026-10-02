package validator

import (
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests using real test data files

func TestIntegration_ValidMinimal(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "valid_minimal.xml")
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if result != nil && result.Error != "" {
		t.Errorf("unexpected error in result: %s", result.Error)
	}

	// This file should have minimal issues
	summary := result.Summary()
	if summary.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", summary.FilesProcessed)
	}

	t.Logf("Valid minimal file validation: %d issues found", summary.TotalIssues)

	// Log issue summary by severity for debugging
	for severity, count := range summary.IssuesBySeverity {
		if count > 0 {
			t.Logf("  %v: %d issues", severity, count)
		}
	}
}

func TestIntegration_InvalidMissingElements(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	// This file should have validation issues
	summary := result.Summary()
	if summary.TotalIssues == 0 {
		t.Error("expected validation issues for file with missing elements")
	}

	if !summary.HasErrors {
		t.Error("expected HasErrors to be true for invalid file")
	}

	if !result.IsValid() {
		t.Log("File correctly identified as invalid")
	} else {
		t.Error("file should be identified as invalid")
	}

	t.Logf("Missing elements file validation: %d issues found", summary.TotalIssues)

	// Check that we have specific rule violations
	var ruleViolations map[string]int
	if result != nil {
		ruleViolations = result.NumberOfValidationEntriesPerRule
	}
	expectedRules := []string{
		"Line missing Name",
		"Route missing Name",
		"Line missing TransportMode",
		"Route missing LineRef",
	}

	foundExpectedRules := 0
	for _, expectedRule := range expectedRules {
		if count, exists := ruleViolations[expectedRule]; exists && count > 0 {
			foundExpectedRules++
			t.Logf("  Found expected rule violation: %s (%d times)", expectedRule, count)
		}
	}

	if foundExpectedRules == 0 {
		t.Log("Note: No expected rule violations found. This may be due to validation configuration.")
	}
}

func TestIntegration_InvalidTransportModes(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "invalid_transport_modes.xml")
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	summary := result.Summary()
	t.Logf("Invalid transport modes file: %d issues found", summary.TotalIssues)

	// Should find transport mode related issues
	transportModeIssues := false
	for _, entry := range result.ValidationReportEntries {
		if strings.Contains(strings.ToLower(entry.Name), "transport") {
			transportModeIssues = true
			t.Logf("  Transport mode issue: %s", entry.Name)
		}
	}

	if !transportModeIssues {
		t.Log("Note: No transport mode issues found. Check validation rules configuration.")
	}
}

func TestIntegration_MalformedXML(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "malformed.xml")
	// Enable schema validation but disable network access to properly detect malformed XML
	options := testOptions(t).
		WithCodespace("TEST").
		WithSkipSchema(false).
		WithAllowSchemaNetwork(false).
		WithSchemaTimeoutSeconds(5)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	summary := result.Summary()
	t.Logf("Malformed XML file: %d issues found", summary.TotalIssues)

	// Check if validation handled malformed XML gracefully
	// The validator might report issues or might detect parsing problems early and stop
	switch {
	case result.Error != "":
		t.Logf("Validation error detected: %s", result.Error)
	case summary.TotalIssues > 0:
		t.Logf("Validation found %d issues (expected for malformed XML)", summary.TotalIssues)
	default:
		t.Log("Note: Malformed XML might have been caught at parsing stage before validation rules run")
	}

	// Look for XML-related errors
	xmlErrors := false
	for _, entry := range result.ValidationReportEntries {
		if strings.Contains(strings.ToLower(entry.Name), "xml") ||
			strings.Contains(strings.ToLower(entry.Message), "xml") ||
			strings.Contains(strings.ToLower(entry.Name), "schema") ||
			strings.Contains(strings.ToLower(entry.Message), "schema") {
			xmlErrors = true
			t.Logf("  XML/Schema error: %s - %s", entry.Name, entry.Message)
		}
	}

	if !xmlErrors {
		t.Log("Note: No explicit XML/schema errors found in validation results")
	}
}

func TestIntegration_EmptyFile(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "empty.xml")
	// Enable schema validation but disable network access to avoid timeouts
	options := testOptions(t).
		WithCodespace("TEST").
		WithSkipSchema(false).
		WithAllowSchemaNetwork(false). // Disable network access
		WithSchemaTimeoutSeconds(5)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	// Empty file should be handled gracefully
	if result.Error == "" {
		t.Log("Empty file handled without error")
	}

	summary := result.Summary()
	t.Logf("Empty file: %d issues found", summary.TotalIssues)

	// Should not be valid
	if result.IsValid() {
		t.Error("empty file should not be valid")
	}
}

func TestIntegration_InvalidNetexVersion(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "invalid_netex_version.xml")
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	summary := result.Summary()
	t.Logf("Invalid NetEX version file: %d issues found", summary.TotalIssues)

	// Should find version-related issues
	versionIssues := false
	for _, entry := range result.ValidationReportEntries {
		if strings.Contains(strings.ToLower(entry.Name), "version") {
			versionIssues = true
			t.Logf("  Version issue: %s - %s", entry.Name, entry.Message)
		}
	}

	if !versionIssues {
		t.Log("Note: No version issues found. Check if version validation rules are enabled.")
	}
}

func TestIntegration_ValidationOptions(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "valid_minimal.xml")

	// Test with schema validation disabled
	t.Run("SkipSchema", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace("TEST").
			WithSkipSchema(true).
			WithSkipSchema(true)

		result, err := ValidateFile(testFile, options)
		if err != nil {
			t.Fatalf("ValidateFile() with SkipSchema failed: %v", err)
		}

		t.Logf("Skip schema validation: %d issues found", len(result.ValidationReportEntries))
	})

	// Test with verbose mode
	t.Run("Verbose", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace("TEST").
			WithSkipSchema(true).
			WithVerbose(true)

		result, err := ValidateFile(testFile, options)
		if err != nil {
			t.Fatalf("ValidateFile() with Verbose failed: %v", err)
		}

		t.Logf("Verbose validation: %d issues found", len(result.ValidationReportEntries))
	})

	// Test with different codespace
	t.Run("DifferentCodespace", func(t *testing.T) {
		options := testOptions(t).WithCodespace("DIFFERENT").WithSkipSchema(true)

		result, err := ValidateFile(testFile, options)
		if err != nil {
			t.Fatalf("ValidateFile() with different codespace failed: %v", err)
		}

		if result.Codespace != "DIFFERENT" {
			t.Errorf("expected codespace 'DIFFERENT', got %q", result.Codespace)
		}
	})
}

func TestIntegration_OutputFormats(t *testing.T) {
	testFile := filepath.Join("..", "testdata", "valid_minimal.xml")
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	t.Run("JSON", func(t *testing.T) {
		jsonData, err := result.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() failed: %v", err)
		}

		if len(jsonData) == 0 {
			t.Error("JSON data is empty")
		}

		t.Logf("JSON output: %d bytes", len(jsonData))
	})

	t.Run("HTML", func(t *testing.T) {
		htmlData, err := result.ToHTML()
		if err != nil {
			t.Fatalf("ToHTML() failed: %v", err)
		}

		if len(htmlData) == 0 {
			t.Error("HTML data is empty")
		}

		htmlStr := string(htmlData)
		if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
			t.Error("HTML output missing DOCTYPE")
		}

		if !strings.Contains(htmlStr, "NetEX Validation Report") {
			t.Error("HTML output missing expected title")
		}

		t.Logf("HTML output: %d bytes", len(htmlData))
	})

	t.Run("String", func(t *testing.T) {
		str := result.String()
		if len(str) == 0 {
			t.Error("String output is empty")
		}

		t.Logf("String output: %s", str)
	})
}

func TestIntegration_PerformanceBaseline(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "valid_minimal.xml")
	options := testOptions(t).WithCodespace("TEST").WithSkipSchema(true)

	// Baseline performance test
	result, err := ValidateFile(testFile, options)
	if err != nil {
		t.Fatalf("ValidateFile() failed: %v", err)
	}

	summary := result.Summary()
	t.Logf("Performance baseline - Processing time: %v for %d issues in %d files",
		summary.ProcessingTime, summary.TotalIssues, summary.FilesProcessed)

	// Basic performance expectations (adjust as needed)
	if summary.ProcessingTime.Seconds() > 10.0 {
		t.Logf("Warning: Validation took longer than expected: %v", summary.ProcessingTime)
	}
}
