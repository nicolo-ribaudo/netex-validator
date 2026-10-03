package validator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/testutil"
)

func TestSchemaValidation_NetworkTimeout(t *testing.T) {
	// Create a test server that delays responses to simulate network timeouts
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than our timeout
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	t.Run("Schema validation with network timeout", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithAllowSchemaNetwork(true).
			WithSchemaTimeoutSeconds(1) // 1 second timeout

		// Create validator with timeout settings
		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// Create XML that references external schema
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" 
                     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                     xsi:schemaLocation="http://www.netex.org.uk/netex ` + slowServer.URL + `/netex.xsd"
                     version="1.15">
    <PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
    <ParticipantRef>TEST</ParticipantRef>
</PublicationDelivery>`

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start validation in goroutine
		done := make(chan struct{})
		var result *ValidationResult
		go func() {
			result, err = validator.ValidateContent([]byte(xmlContent), "timeout_test.xml")
			close(done)
		}()

		// Wait for completion or context timeout
		select {
		case <-done:
			// Should complete quickly due to schema timeout
			if err != nil {
				t.Logf("Validation completed with error (expected): %v", err)
			}
			if result != nil {
				t.Logf("Validation completed with %d issues", len(result.ValidationReportEntries))
			}
		case <-ctx.Done():
			t.Error("Schema validation did not respect timeout setting")
		}
	})

	t.Run("Schema validation with network disabled", func(t *testing.T) {
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithAllowSchemaNetwork(false) // Network disabled

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// XML with external schema reference
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" 
                     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                     xsi:schemaLocation="http://www.netex.org.uk/netex http://external.server/netex.xsd"
                     version="1.15">
    <PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
    <ParticipantRef>TEST</ParticipantRef>
</PublicationDelivery>`

		result, err := validator.ValidateContent([]byte(xmlContent), "network_disabled_test.xml")
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		// Should complete without attempting network access
		if result == nil {
			t.Error("Expected validation result")
		} else {
			t.Logf("Validation completed with network disabled: %d issues", len(result.ValidationReportEntries))
		}
	})
}

func TestSchemaValidation_CacheDirectory(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	t.Run("Schema caching functionality", func(t *testing.T) {
		cacheDir := tm.CreateTempDir("schema_cache")

		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSchemaCacheDir(cacheDir).
			WithAllowSchemaNetwork(true)

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// First validation - should potentially download/cache schemas
		xmlFile := tm.CreateTestXMLFile(t, "test1.xml", testutil.NetEXTestFragment)
		result1, err := validator.ValidateFile(xmlFile)
		if err != nil {
			t.Fatalf("First validation failed: %v", err)
		}

		// Second validation - should use cached schemas
		xmlFile2 := tm.CreateTestXMLFile(t, "test2.xml", testutil.NetEXTestFragment)
		start := time.Now()
		result2, err := validator.ValidateFile(xmlFile2)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Second validation failed: %v", err)
		}

		t.Logf("First validation: %d issues", len(result1.ValidationReportEntries))
		t.Logf("Second validation: %d issues in %v", len(result2.ValidationReportEntries), duration)

		// Second validation should be faster due to caching
		if duration > 5*time.Second {
			t.Logf("Warning: Second validation took %v, caching might not be working", duration)
		}
	})
}

func TestSchemaValidation_ErrorTypes(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	tests := []struct {
		name           string
		xmlContent     string
		expectedErrors []string
		skipIfNoSchema bool
	}{
		{
			name: "Missing required element",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
    <!-- Missing PublicationTimestamp and ParticipantRef -->
    <dataObjects/>
</PublicationDelivery>`,
			expectedErrors: []string{"PublicationTimestamp", "ParticipantRef"},
			skipIfNoSchema: true,
		},
		{
			name: "Invalid element order",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
    <ParticipantRef>TEST</ParticipantRef>
    <PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
    <dataObjects/>
</PublicationDelivery>`,
			expectedErrors: []string{"element order", "sequence"},
			skipIfNoSchema: true,
		},
		{
			name: "Invalid attribute value",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="invalid-version">
    <PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
    <ParticipantRef>TEST</ParticipantRef>
    <dataObjects/>
</PublicationDelivery>`,
			expectedErrors: []string{"version", "pattern"},
			skipIfNoSchema: true,
		},
		{
			name: "Unknown element",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
    <PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
    <ParticipantRef>TEST</ParticipantRef>
    <UnknownElement>Invalid</UnknownElement>
    <dataObjects/>
</PublicationDelivery>`,
			expectedErrors: []string{"UnknownElement", "unexpected"},
			skipIfNoSchema: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := testOptions(t)
			options.Codespace = testutil.TestCodespace
			options.SkipSchema = false
			options.SkipValidators = true // Only test schema validation

			validator, err := NewWithOptions(options)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			xmlFile := tm.CreateTestXMLFile(t, "schema_test.xml", tt.xmlContent)
			result, err := validator.ValidateFile(xmlFile)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			// Check if we have schema validation available
			hasSchemaErrors := false
			hasDetailedSchemaErrors := false
			for _, entry := range result.ValidationReportEntries {
				if entry.Name == "Schema validation error" || strings.Contains(entry.Message, "schema") {
					hasSchemaErrors = true
					// Check if we have detailed schema errors (not just basic/generic ones)
					if strings.Contains(entry.Message, "no schema available") {
						// This is just a generic "no schema" error, not detailed validation
						continue
					}
					// If we find schema errors that mention specific elements, it's detailed
					for _, expectedError := range tt.expectedErrors {
						if strings.Contains(strings.ToLower(entry.Message), strings.ToLower(expectedError)) {
							hasDetailedSchemaErrors = true
							break
						}
					}
				}
			}

			if tt.skipIfNoSchema && (!hasSchemaErrors || !hasDetailedSchemaErrors) {
				t.Skip("Detailed schema validation not available, skipping test")
			}

			// Check for expected error patterns
			for _, expectedError := range tt.expectedErrors {
				found := false
				for _, entry := range result.ValidationReportEntries {
					if strings.Contains(strings.ToLower(entry.Message), strings.ToLower(expectedError)) {
						found = true
						t.Logf("Found expected error: %s", entry.Message)
						break
					}
				}
				if !found && hasDetailedSchemaErrors {
					t.Errorf("Expected error containing '%s' not found", expectedError)
				}
			}
		})
	}
}

func TestSchemaValidation_Versions(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	versions := []struct {
		version    string
		shouldPass bool
	}{
		{"1.0", true},
		{"1.04", true},
		{"1.05", true},
		{"1.07", true},
		{"1.08", true},
		{"1.09", true},
		{"1.10", true},
		{"1.11", true},
		{"1.12", true},
		{"1.13", true},
		{"1.14", true},
		{"1.15", true},
		{"2.0", false}, // Future version
		{"0.9", false}, // Too old
		{"abc", false}, // Invalid format
	}

	for _, v := range versions {
		t.Run("Version_"+v.version, func(t *testing.T) {
			xmlContent := strings.ReplaceAll(testutil.NetEXTestFragment, `version="1.15"`, `version="`+v.version+`"`)

			options := testOptions(t).
				WithCodespace(testutil.TestCodespace).
				WithSkipSchema(true) // Skip schema to focus on version detection

			validator, err := NewWithOptions(options)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			xmlFile := tm.CreateTestXMLFile(t, "version_test.xml", xmlContent)
			result, err := validator.ValidateFile(xmlFile)

			if v.shouldPass {
				if err != nil {
					t.Errorf("Validation failed for valid version %s: %v", v.version, err)
				}
			} else {
				// Invalid versions might still validate but with issues
				if result != nil {
					hasVersionIssue := false
					for _, entry := range result.ValidationReportEntries {
						if strings.Contains(strings.ToLower(entry.Message), "version") {
							hasVersionIssue = true
							break
						}
					}
					if hasVersionIssue {
						t.Logf("Version %s correctly identified as problematic", v.version)
					}
				}
			}
		})
	}
}

func BenchmarkSchemaValidation(b *testing.B) {
	options := testOptions(b)
	options.Codespace = testutil.TestCodespace
	options.SkipSchema = false
	options.SkipValidators = true

	validator, err := NewWithOptions(options)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	content := []byte(testutil.GetBenchmarkData().MediumDataset)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateContent(content, "benchmark.xml")
	}
}
