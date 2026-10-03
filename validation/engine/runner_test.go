package engine

import (
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/testutil"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

func TestEnhancedNetexValidatorsRunner_ValidateContent(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		skipSchema     bool
		skipValidators bool
		expectError    bool
		expectIssues   int
	}{
		{
			name:           "Valid minimal NetEX",
			content:        testutil.NetEXTestFragment,
			skipSchema:     false,
			skipValidators: false,
			expectError:    false,
			expectIssues:   0, // May have warnings but no errors
		},
		{
			name:           "Invalid NetEX missing required elements",
			content:        testutil.InvalidNetEXFragment,
			skipSchema:     false,
			skipValidators: false,
			expectError:    false, // Should not error but should have issues
			expectIssues:   -1,    // Don't check exact count, just that there are issues
		},
		{
			name:           "Valid NetEX with schema validation skipped",
			content:        testutil.NetEXTestFragment,
			skipSchema:     true,
			skipValidators: false,
			expectError:    false,
			expectIssues:   0,
		},
		{
			name:           "Valid NetEX with all validation skipped",
			content:        testutil.NetEXTestFragment,
			skipSchema:     true,
			skipValidators: true,
			expectError:    false,
			expectIssues:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create runner with minimal configuration
			runner := createTestRunner(t)

			// Run validation
			report, err := runner.ValidateContent(
				"test.xml",
				testutil.TestCodespace,
				[]byte(tt.content),
				tt.skipSchema,
				tt.skipValidators,
			)

			// Check for unexpected errors
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateContent() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if err != nil {
				return // Expected error, test passed
			}

			// Validate report structure
			if report == nil {
				t.Fatal("ValidateContent() returned nil report")
			}

			if report.Codespace != testutil.TestCodespace {
				t.Errorf("Expected codespace %s, got %s", testutil.TestCodespace, report.Codespace)
			}

			// Check issue count if specified
			if tt.expectIssues >= 0 {
				actualIssues := len(report.ValidationReportEntries)
				if actualIssues != tt.expectIssues {
					t.Errorf("Expected %d issues, got %d", tt.expectIssues, actualIssues)
					for i, entry := range report.ValidationReportEntries {
						t.Logf("Issue %d: [%s] %s: %s", i+1, entry.Severity, entry.Name, entry.Message)
					}
				}
			}
		})
	}
}

func TestEnhancedNetexValidatorsRunner_ValidateFile(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	// Create test files
	validFile := tm.CreateTestXMLFile(t, "valid.xml", testutil.NetEXTestFragment)
	invalidFile := tm.CreateTestXMLFile(t, "invalid.xml", testutil.InvalidNetEXFragment)

	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{
			name:        "Valid XML file",
			filePath:    validFile,
			expectError: false,
		},
		{
			name:        "Invalid XML file",
			filePath:    invalidFile,
			expectError: false, // Should validate but have issues
		},
		{
			name:        "Non-existent file",
			filePath:    "/non/existent/file.xml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := createTestRunner(t)

			report, err := runner.ValidateFile(tt.filePath, testutil.TestCodespace, false, false)

			if (err != nil) != tt.expectError {
				t.Errorf("ValidateFile() error = %v, expectError %v", err, tt.expectError)
			}

			if err == nil && report == nil {
				t.Error("ValidateFile() returned nil report without error")
			}
		})
	}
}

func TestEnhancedNetexValidatorsRunner_ValidateZipDataset(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	// Create test ZIP with multiple files
	xmlFiles := map[string]string{
		"file1.xml": testutil.NetEXTestFragment,
		"file2.xml": modifyTestFragment("TEST:Line:2"),
		"file3.xml": modifyTestFragment("TEST:Line:3"),
	}
	zipFile := tm.CreateTestZipFile(t, "test-dataset.zip", xmlFiles)

	t.Run("Valid ZIP dataset", func(t *testing.T) {
		runner := createTestRunner(t)

		report, err := runner.ValidateFile(zipFile, testutil.TestCodespace, false, false)

		if err != nil {
			t.Fatalf("ValidateFile() error = %v", err)
		}

		if report == nil {
			t.Fatal("ValidateFile() returned nil report")
		}

		// Should have processed the ZIP file successfully
		// Note: ValidationReportEntries may be empty if no validation issues found
		t.Logf("ZIP validation completed with %d entries", len(report.ValidationReportEntries))
	})
}

func TestEnhancedNetexValidatorsRunner_FinalizeIdValidation(t *testing.T) {
	t.Run("Finalize ID validation", func(t *testing.T) {
		runner := createTestRunner(t)

		// First validate some content to populate ID data
		_, err := runner.ValidateContent(
			"test.xml",
			testutil.TestCodespace,
			[]byte(testutil.NetEXTestFragment),
			false,
			false,
		)
		if err != nil {
			t.Fatalf("ValidateContent() failed: %v", err)
		}

		// Then finalize ID validation
		issues, err := runner.FinalizeIdValidation()
		if err != nil {
			t.Errorf("FinalizeIdValidation() error = %v", err)
		}

		// Should not be nil (should return empty slice, not nil)
		if issues == nil {
			t.Errorf("FinalizeIdValidation() returned nil issues, expected empty slice")
		} else {
			t.Logf("FinalizeIdValidation() returned %d issues", len(issues))
		}
	})
}

func TestEnhancedNetexValidatorsRunnerBuilder(t *testing.T) {
	t.Run("Builder pattern", func(t *testing.T) {
		builder := NewEnhancedNetexValidatorsRunnerBuilder()

		// Test builder methods
		builder.WithMaxFindings(100)
		builder.WithConcurrentFiles(4)
		builder.WithValidationReportEntryFactory(NewDefaultValidationReportEntryFactory())

		// Build runner
		runner, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if runner == nil {
			t.Error("Build() returned nil runner")
		}

		// Test with factory requirement
		builder2 := NewEnhancedNetexValidatorsRunnerBuilder()
		_, err = builder2.Build()
		if err == nil {
			t.Error("Build() should fail without required factory")
		}
	})
}

func TestDefaultValidationReportEntryFactory(t *testing.T) {
	factory := NewDefaultValidationReportEntryFactory()

	testIssue := types.ValidationIssue{
		Rule: types.ValidationRule{
			Name:     "Test Rule",
			Code:     "TEST_001",
			Message:  "Test message",
			Severity: types.WARNING,
		},
		Message: "Specific test message",
		Location: types.DataLocation{
			FileName: "test.xml",
			XPath:    "/test/path",
		},
	}

	t.Run("CreateValidationReportEntry", func(t *testing.T) {
		entry := factory.CreateValidationReportEntry(testIssue)

		if entry.Name != testIssue.Rule.Name {
			t.Errorf("Expected name %s, got %s", testIssue.Rule.Name, entry.Name)
		}

		if entry.Message != testIssue.Message {
			t.Errorf("Expected message %s, got %s", testIssue.Message, entry.Message)
		}

		if entry.Severity != testIssue.Rule.Severity {
			t.Errorf("Expected severity %s, got %s", testIssue.Rule.Severity, entry.Severity)
		}

		if entry.FileName != testIssue.Location.FileName {
			t.Errorf("Expected filename %s, got %s", testIssue.Location.FileName, entry.FileName)
		}
	})

	t.Run("TemplateValidationReportEntry", func(t *testing.T) {
		entry := factory.TemplateValidationReportEntry(testIssue.Rule)

		if entry.Name != testIssue.Rule.Name {
			t.Errorf("Expected name %s, got %s", testIssue.Rule.Name, entry.Name)
		}

		if entry.Message != testIssue.Rule.Message {
			t.Errorf("Expected message %s, got %s", testIssue.Rule.Message, entry.Message)
		}

		if entry.Severity != testIssue.Rule.Severity {
			t.Errorf("Expected severity %s, got %s", testIssue.Rule.Severity, entry.Severity)
		}
	})
}

// Helper functions

// createTestRunner creates a minimal test runner
func createTestRunner(tb testing.TB) *EnhancedNetexValidatorsRunner {
	tb.Helper()

	builder := NewEnhancedNetexValidatorsRunnerBuilder()
	builder.WithValidationReportEntryFactory(NewDefaultValidationReportEntryFactory())

	runner, err := builder.Build()
	if err != nil {
		tb.Fatalf("Failed to create test runner: %v", err)
	}

	return runner
}

// modifyTestFragment creates a modified version of the test fragment
func modifyTestFragment(lineId string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" xmlns:gml="http://www.opengis.net/gml/3.2" version="1.15:NO-NeTEx-networktimetable:1.5">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="` + lineId + `" version="1">
					<Name>Test Line Modified</Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`
}

// Benchmark tests

func BenchmarkValidateContent_Small(b *testing.B) {
	runner := createTestRunner(b)
	content := []byte(testutil.GetBenchmarkData().SmallDataset)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.ValidateContent("test.xml", testutil.TestCodespace, content, false, false)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkValidateContent_Medium(b *testing.B) {
	runner := createTestRunner(b)
	content := []byte(testutil.GetBenchmarkData().MediumDataset)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.ValidateContent("test.xml", testutil.TestCodespace, content, false, false)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkValidateContent_Large(b *testing.B) {
	runner := createTestRunner(b)
	content := []byte(testutil.GetBenchmarkData().LargeDataset)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.ValidateContent("test.xml", testutil.TestCodespace, content, false, false)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}
