package validator

import (
	"strings"
	"testing"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/testutil"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

const (
	errorSeverityText   = "Error"
	warningSeverityText = "Warning"
)

func TestHTMLReporter_GenerateHTML(t *testing.T) {
	reporter := NewHTMLReporter()

	// Create a test validation result
	result := &ValidationResult{
		Codespace:                        testutil.TestCodespace,
		ValidationReportID:               testutil.TestReportID,
		CreationDate:                     time.Now(),
		ValidationReportEntries:          createTestReportEntries(),
		ProcessingTime:                   2 * time.Second,
		FilesProcessed:                   3,
		NumberOfValidationEntriesPerRule: make(map[string]int),
	}

	t.Run("Generate HTML from validation result", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Basic HTML structure validation
		testutil.AssertXMLWellFormed(t, html)

		// Check for required HTML elements
		requiredElements := []string{
			"<!DOCTYPE html>",
			"<html",
			"<head>",
			"<body>",
			"NetEX Validation Report",
			testutil.TestReportID,
			testutil.TestCodespace,
		}

		for _, element := range requiredElements {
			if !strings.Contains(html, element) {
				t.Errorf("Expected HTML to contain '%s'", element)
			}
		}
	})

	t.Run("HTML contains validation statistics", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Check for statistics
		expectedStats := []string{
			"Total Issues",
			"Files Processed",
			"Processing Time",
			"3",     // Files processed count
			"2.00s", // Processing time
		}

		for _, stat := range expectedStats {
			if !strings.Contains(html, stat) {
				t.Errorf("Expected HTML to contain statistic '%s'", stat)
			}
		}
	})

	t.Run("HTML contains issue details", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Check for specific issues from test data
		expectedIssueContent := []string{
			"Test Rule 1",
			"Test Rule 2",
			"Test message 1",
			"Test message 2",
			"test1.xml",
			"test2.xml",
		}

		for _, content := range expectedIssueContent {
			if !strings.Contains(html, content) {
				t.Errorf("Expected HTML to contain issue content '%s'", content)
			}
		}
	})

	t.Run("HTML contains severity badges", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Check for severity-related content
		expectedSeverityContent := []string{
			"severity-badge",
			"Error",   // Capitalized, not all caps
			"Warning", // Capitalized, not all caps
			"❌",       // Error icon
			"⚠️",      // Warning icon
		}

		for _, content := range expectedSeverityContent {
			if !strings.Contains(html, content) {
				t.Errorf("Expected HTML to contain severity content '%s'", content)
			}
		}
	})

	t.Run("HTML contains interactive tabs", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Check for tab functionality
		expectedTabContent := []string{
			"tab-button",
			"tab-content",
			"All Issues",
			"By File",
			"By Severity",
			"By Rule",
			"showTab(",
		}

		for _, content := range expectedTabContent {
			if !strings.Contains(html, content) {
				t.Errorf("Expected HTML to contain tab content '%s'", content)
			}
		}
	})
}

func TestHTMLReporter_EmptyResult(t *testing.T) {
	reporter := NewHTMLReporter()

	// Create empty validation result
	result := &ValidationResult{
		Codespace:                        testutil.TestCodespace,
		ValidationReportID:               testutil.TestReportID,
		CreationDate:                     time.Now(),
		ValidationReportEntries:          []ValidationReportEntry{},
		ProcessingTime:                   500 * time.Millisecond,
		FilesProcessed:                   1,
		NumberOfValidationEntriesPerRule: make(map[string]int),
	}

	t.Run("Generate HTML for empty result", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Should still generate valid HTML
		testutil.AssertXMLWellFormed(t, html)

		// Should show 0 issues
		if !strings.Contains(html, "<h3>0</h3>") {
			t.Error("Expected HTML to show 0 total issues")
		}
	})
}

func TestHTMLReporter_LargeResult(t *testing.T) {
	reporter := NewHTMLReporter()

	// Create result with many issues
	entries := make([]ValidationReportEntry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = ValidationReportEntry{
			Name:     "Large Test Rule " + string(rune(i)),
			Message:  "Large test message " + string(rune(i)),
			Severity: types.WARNING,
			FileName: "large_test_" + string(rune(i)) + ".xml",
			Location: ValidationReportLocation{
				FileName: "large_test_" + string(rune(i)) + ".xml",
				XPath:    "/test/path[" + string(rune(i)) + "]",
			},
		}
	}

	result := &ValidationResult{
		Codespace:                        testutil.TestCodespace,
		ValidationReportID:               testutil.TestReportID,
		CreationDate:                     time.Now(),
		ValidationReportEntries:          entries,
		ProcessingTime:                   10 * time.Second,
		FilesProcessed:                   100,
		NumberOfValidationEntriesPerRule: make(map[string]int),
	}

	t.Run("Generate HTML for large result", func(t *testing.T) {
		html, err := reporter.GenerateHTML(result)
		if err != nil {
			t.Fatalf("GenerateHTML() error = %v", err)
		}

		// Should handle large datasets
		if !strings.Contains(html, "<h3>1000</h3>") {
			t.Error("Expected HTML to show 1000 total issues")
		}

		// Should contain percentage calculations
		if !strings.Contains(html, "100.0%") {
			t.Error("Expected HTML to contain percentage calculations")
		}
	})
}

func TestHTMLTemplateHelpers(t *testing.T) {
	t.Run("severityClass", func(t *testing.T) {
		tests := []struct {
			severity types.Severity
			expected string
		}{
			{types.CRITICAL, "critical"},
			{types.ERROR, "error"},
			{types.WARNING, "warning"},
			{types.INFO, "info"},
		}

		for _, tt := range tests {
			result := severityClass(tt.severity)
			if result != tt.expected {
				t.Errorf("severityClass(%v) = %s, expected %s", tt.severity, result, tt.expected)
			}
		}
	})

	t.Run("severityIcon", func(t *testing.T) {
		tests := []struct {
			severity types.Severity
			expected string
		}{
			{types.CRITICAL, "⛔"},
			{types.ERROR, "❌"},
			{types.WARNING, "⚠️"},
			{types.INFO, "ℹ️"},
		}

		for _, tt := range tests {
			result := severityIcon(tt.severity)
			if result != tt.expected {
				t.Errorf("severityIcon(%v) = %s, expected %s", tt.severity, result, tt.expected)
			}
		}
	})

	t.Run("severityText", func(t *testing.T) {
		tests := []struct {
			severity types.Severity
			expected string
		}{
			{types.CRITICAL, "Critical"},
			{types.ERROR, "Error"},
			{types.WARNING, "Warning"},
			{types.INFO, "Info"},
		}

		for _, tt := range tests {
			result := severityText(tt.severity)
			if result != tt.expected {
				t.Errorf("severityText(%v) = %s, expected %s", tt.severity, result, tt.expected)
			}
		}
	})

	t.Run("formatTime", func(t *testing.T) {
		testTime := time.Date(2023, 12, 25, 14, 30, 45, 0, time.UTC)
		expected := "2023-12-25 14:30:45"

		result := formatTime(testTime)
		if result != expected {
			t.Errorf("formatTime(%v) = %s, expected %s", testTime, result, expected)
		}
	})

	t.Run("percentage", func(t *testing.T) {
		tests := []struct {
			part     int
			total    int
			expected float64
		}{
			{25, 100, 25.0},
			{1, 3, 33.333333333333336},
			{0, 100, 0.0},
			{100, 100, 100.0},
			{10, 0, 0.0}, // Division by zero case
		}

		for _, tt := range tests {
			result := percentage(tt.part, tt.total)
			// Use tolerance for floating-point comparison
			tolerance := 1e-10
			if diff := result - tt.expected; diff < -tolerance || diff > tolerance {
				t.Errorf("percentage(%d, %d) = %f, expected %f", tt.part, tt.total, result, tt.expected)
			}
		}
	})
}

func TestHTMLReporter_TemplateDataPreparation(t *testing.T) {
	reporter := NewHTMLReporter()

	// Create test result with diverse issues
	result := &ValidationResult{
		Codespace:          testutil.TestCodespace,
		ValidationReportID: testutil.TestReportID,
		CreationDate:       time.Now(),
		ValidationReportEntries: []ValidationReportEntry{
			{
				Name:     "Rule A",
				Message:  "Message A1",
				Severity: types.ERROR,
				FileName: "file1.xml",
				Location: ValidationReportLocation{FileName: "file1.xml", XPath: "/a/b"},
			},
			{
				Name:     "Rule A",
				Message:  "Message A2",
				Severity: types.ERROR,
				FileName: "file2.xml",
				Location: ValidationReportLocation{FileName: "file2.xml", XPath: "/a/c"},
			},
			{
				Name:     "Rule B",
				Message:  "Message B1",
				Severity: types.WARNING,
				FileName: "file1.xml",
				Location: ValidationReportLocation{FileName: "file1.xml", XPath: "/b/c"},
			},
		},
		ProcessingTime:                   1 * time.Second,
		FilesProcessed:                   2,
		NumberOfValidationEntriesPerRule: make(map[string]int),
	}

	templateData := reporter.prepareTemplateData(result)

	t.Run("Template data structure", func(t *testing.T) {
		if templateData.Result != result {
			t.Error("Template data should reference original result")
		}

		if templateData.Statistics.TotalIssues != 3 {
			t.Errorf("Expected 3 total issues, got %d", templateData.Statistics.TotalIssues)
		}

		if templateData.Statistics.FilesProcessed != 2 {
			t.Errorf("Expected 2 files processed, got %d", templateData.Statistics.FilesProcessed)
		}
	})

	t.Run("Issues grouped by file", func(t *testing.T) {
		if len(templateData.IssuesByFile) != 2 {
			t.Errorf("Expected 2 files, got %d", len(templateData.IssuesByFile))
		}

		file1Issues := templateData.IssuesByFile["file1.xml"]
		if len(file1Issues) != 2 {
			t.Errorf("Expected 2 issues in file1.xml, got %d", len(file1Issues))
		}

		file2Issues := templateData.IssuesByFile["file2.xml"]
		if len(file2Issues) != 1 {
			t.Errorf("Expected 1 issue in file2.xml, got %d", len(file2Issues))
		}
	})

	t.Run("Issues grouped by severity", func(t *testing.T) {
		errorIssues := templateData.IssuesBySeverity[errorSeverityText]
		if len(errorIssues) != 2 {
			t.Errorf("Expected 2 error issues, got %d", len(errorIssues))
		}

		warningIssues := templateData.IssuesBySeverity[warningSeverityText]
		if len(warningIssues) != 1 {
			t.Errorf("Expected 1 warning issue, got %d", len(warningIssues))
		}
	})

	t.Run("Issues grouped by rule", func(t *testing.T) {
		ruleAIssues := templateData.IssuesByRule["Rule A"]
		if len(ruleAIssues) != 2 {
			t.Errorf("Expected 2 issues for Rule A, got %d", len(ruleAIssues))
		}

		ruleBIssues := templateData.IssuesByRule["Rule B"]
		if len(ruleBIssues) != 1 {
			t.Errorf("Expected 1 issue for Rule B, got %d", len(ruleBIssues))
		}
	})

	t.Run("Severity statistics", func(t *testing.T) {
		if templateData.Statistics.SeverityCounts[errorSeverityText] != 2 {
			t.Errorf("Expected 2 error issues in stats, got %d", templateData.Statistics.SeverityCounts[errorSeverityText])
		}

		if templateData.Statistics.SeverityCounts[warningSeverityText] != 1 {
			t.Errorf("Expected 1 warning issue in stats, got %d", templateData.Statistics.SeverityCounts[warningSeverityText])
		}

		// Check percentages (with tolerance for floating point precision)
		expectedErrorPercent := 2.0 / 3.0 * 100 // ~66.67%
		actualErrorPercent := templateData.Statistics.SeverityPercents[errorSeverityText]
		tolerance := 0.01
		if actualErrorPercent < expectedErrorPercent-tolerance || actualErrorPercent > expectedErrorPercent+tolerance {
			t.Errorf("Expected error percentage %.2f, got %.2f", expectedErrorPercent, actualErrorPercent)
		}
	})

	t.Run("Severity keys filtering", func(t *testing.T) {
		// Should only contain severities that have issues
		expectedKeys := []string{errorSeverityText, warningSeverityText}
		if len(templateData.SeverityKeys) != len(expectedKeys) {
			t.Errorf("Expected %d severity keys, got %d", len(expectedKeys), len(templateData.SeverityKeys))
		}

		// Should not contain INFO or CRITICAL since there are no such issues
		for _, key := range templateData.SeverityKeys {
			if key != errorSeverityText && key != warningSeverityText {
				t.Errorf("Unexpected severity key: %s", key)
			}
		}
	})
}

// Helper function to create test report entries
func createTestReportEntries() []ValidationReportEntry {
	return []ValidationReportEntry{
		{
			Name:     "Test Rule 1",
			Message:  "Test message 1",
			Severity: types.ERROR,
			FileName: "test1.xml",
			Location: ValidationReportLocation{
				FileName: "test1.xml",
				XPath:    "/test/path[1]",
			},
		},
		{
			Name:     "Test Rule 2",
			Message:  "Test message 2",
			Severity: types.WARNING,
			FileName: "test2.xml",
			Location: ValidationReportLocation{
				FileName: "test2.xml",
				XPath:    "/test/path[2]",
			},
		},
		{
			Name:     "Test Rule 3",
			Message:  "Test message 3",
			Severity: types.WARNING,
			FileName: "test1.xml",
			Location: ValidationReportLocation{
				FileName: "test1.xml",
				XPath:    "/test/path[3]",
			},
		},
	}
}

// Benchmark tests

func BenchmarkHTMLReporter_GenerateHTML_Small(b *testing.B) {
	reporter := NewHTMLReporter()
	result := &ValidationResult{
		Codespace:                        testutil.TestCodespace,
		ValidationReportID:               testutil.TestReportID,
		CreationDate:                     time.Now(),
		ValidationReportEntries:          createTestReportEntries(),
		ProcessingTime:                   1 * time.Second,
		FilesProcessed:                   3,
		NumberOfValidationEntriesPerRule: make(map[string]int),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reporter.GenerateHTML(result)
		if err != nil {
			b.Fatalf("GenerateHTML failed: %v", err)
		}
	}
}

func BenchmarkHTMLReporter_GenerateHTML_Large(b *testing.B) {
	reporter := NewHTMLReporter()

	// Create large result set
	entries := make([]ValidationReportEntry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = ValidationReportEntry{
			Name:     "Benchmark Rule " + string(rune(i%10)),
			Message:  "Benchmark message " + string(rune(i)),
			Severity: []types.Severity{types.ERROR, types.WARNING, types.INFO}[i%3],
			FileName: "file" + string(rune(i%5)) + ".xml",
			Location: ValidationReportLocation{
				FileName: "file" + string(rune(i%5)) + ".xml",
				XPath:    "/benchmark/path[" + string(rune(i)) + "]",
			},
		}
	}

	result := &ValidationResult{
		Codespace:                        testutil.TestCodespace,
		ValidationReportID:               testutil.TestReportID,
		CreationDate:                     time.Now(),
		ValidationReportEntries:          entries,
		ProcessingTime:                   10 * time.Second,
		FilesProcessed:                   5,
		NumberOfValidationEntriesPerRule: make(map[string]int),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reporter.GenerateHTML(result)
		if err != nil {
			b.Fatalf("GenerateHTML failed: %v", err)
		}
	}
}
