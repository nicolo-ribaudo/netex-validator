package context

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/testutil"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

func TestJAXBValidationContext_Creation(t *testing.T) {
	tests := []struct {
		name        string
		reportID    string
		codespace   string
		fileName    string
		localIDMap  map[string]types.IdVersion
		expectValid bool
	}{
		{
			name:      "Valid JAXB context",
			reportID:  testutil.TestReportID,
			codespace: testutil.TestCodespace,
			fileName:  testutil.TestFileName,
			localIDMap: map[string]types.IdVersion{
				"TEST:Line:1": {
					ID:      "TEST:Line:1",
					Version: "1",
				},
			},
			expectValid: true,
		},
		{
			name:        "Empty report ID",
			reportID:    "",
			codespace:   testutil.TestCodespace,
			fileName:    testutil.TestFileName,
			localIDMap:  make(map[string]types.IdVersion),
			expectValid: true, // Empty report ID should be allowed
		},
		{
			name:        "Empty codespace",
			reportID:    testutil.TestReportID,
			codespace:   "",
			fileName:    testutil.TestFileName,
			localIDMap:  make(map[string]types.IdVersion),
			expectValid: true, // Empty codespace should be allowed
		},
		{
			name:        "Empty filename",
			reportID:    testutil.TestReportID,
			codespace:   testutil.TestCodespace,
			fileName:    "",
			localIDMap:  make(map[string]types.IdVersion),
			expectValid: true, // Empty filename should be allowed
		},
		{
			name:        "Nil local ID map",
			reportID:    testutil.TestReportID,
			codespace:   testutil.TestCodespace,
			fileName:    testutil.TestFileName,
			localIDMap:  nil,
			expectValid: true, // Nil map should be handled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewJAXBValidationContext(tt.reportID, tt.codespace, tt.fileName, tt.localIDMap)

			if !tt.expectValid {
				// If we expect invalid context, we might want to check for specific validation
				// For now, we assume all contexts are valid as created
				t.Skip("Invalid context creation not implemented yet")
			}

			// Verify basic properties
			if ctx.ValidationReportID != tt.reportID {
				t.Errorf("Expected report ID %s, got %s", tt.reportID, ctx.ValidationReportID)
			}

			if ctx.Codespace != tt.codespace {
				t.Errorf("Expected codespace %s, got %s", tt.codespace, ctx.Codespace)
			}

			if ctx.FileName != tt.fileName {
				t.Errorf("Expected filename %s, got %s", tt.fileName, ctx.FileName)
			}

			// Check local ID map handling
			if tt.localIDMap == nil {
				if ctx.LocalIDMap != nil {
					t.Error("Expected nil local ID map to be preserved")
				}
			} else {
				if len(ctx.LocalIDMap) != len(tt.localIDMap) {
					t.Errorf("Expected %d local IDs, got %d", len(tt.localIDMap), len(ctx.LocalIDMap))
				}

				for expectedID, expectedVersion := range tt.localIDMap {
					actualVersion, exists := ctx.LocalIDMap[expectedID]
					if !exists {
						t.Errorf("Expected ID %s not found in context", expectedID)
						continue
					}

					if actualVersion.ID != expectedVersion.ID {
						t.Errorf("Expected ID %s, got %s for key %s", expectedVersion.ID, actualVersion.ID, expectedID)
					}

					if actualVersion.Version != expectedVersion.Version {
						t.Errorf("Expected version %s, got %s for ID %s", expectedVersion.Version, actualVersion.Version, expectedID)
					}
				}
			}
		})
	}
}

func TestJAXBValidationContext_LocalIDManagement(t *testing.T) {
	t.Run("Local ID lookup operations", func(t *testing.T) {
		localIDMap := map[string]types.IdVersion{
			"TEST:Operator:1": {
				ID:      "TEST:Operator:1",
				Version: "1",
			},
			"TEST:Line:1": {
				ID:      "TEST:Line:1",
				Version: "2",
			},
			"TEST:StopPlace:Central": {
				ID:      "TEST:StopPlace:Central",
				Version: "1",
			},
		}

		ctx := NewJAXBValidationContext(testutil.TestReportID, testutil.TestCodespace, testutil.TestFileName, localIDMap)

		// Test existing ID lookup
		if version, exists := ctx.LocalIDMap["TEST:Operator:1"]; !exists {
			t.Error("Expected to find TEST:Operator:1 in local ID map")
		} else if version.Version != "1" {
			t.Errorf("Expected version 1 for TEST:Operator:1, got %s", version.Version)
		}

		// Test version differences
		if version, exists := ctx.LocalIDMap["TEST:Line:1"]; !exists {
			t.Error("Expected to find TEST:Line:1 in local ID map")
		} else if version.Version != "2" {
			t.Errorf("Expected version 2 for TEST:Line:1, got %s", version.Version)
		}

		// Test non-existent ID
		if _, exists := ctx.LocalIDMap["TEST:NonExistent:1"]; exists {
			t.Error("Did not expect to find non-existent ID in local ID map")
		}

		// Test ID with different casing (should not match)
		if _, exists := ctx.LocalIDMap["test:operator:1"]; exists {
			t.Error("Did not expect case-insensitive match for ID")
		}
	})

	t.Run("Large local ID map performance", func(t *testing.T) {
		// Create a large local ID map to test performance
		largeIDMap := make(map[string]types.IdVersion)
		for i := 0; i < 10000; i++ {
			id := fmt.Sprintf("TEST:Entity:%d", i)
			largeIDMap[id] = types.IdVersion{
				ID:      id,
				Version: "1",
			}
		}

		ctx := NewJAXBValidationContext(testutil.TestReportID, testutil.TestCodespace, testutil.TestFileName, largeIDMap)

		// Test lookup performance
		testIDs := []string{
			"TEST:Entity:0",
			"TEST:Entity:5000",
			"TEST:Entity:9999",
			"TEST:Entity:NonExistent",
		}

		for _, testID := range testIDs {
			_, exists := ctx.LocalIDMap[testID]
			if testID == "TEST:Entity:NonExistent" && exists {
				t.Error("Did not expect to find non-existent ID")
			} else if testID != "TEST:Entity:NonExistent" && !exists {
				t.Errorf("Expected to find %s in large ID map", testID)
			}
		}

		t.Logf("Large ID map test completed with %d entries", len(ctx.LocalIDMap))
	})
}

func TestJAXBValidationContext_ContextInheritance(t *testing.T) {
	t.Run("Context property inheritance", func(t *testing.T) {
		parentReportID := "PARENT_REPORT"
		parentCodespace := "PARENT_SPACE"
		childFileName := "child.xml"

		parentIDMap := map[string]types.IdVersion{
			"PARENT:Entity:1": {
				ID:      "PARENT:Entity:1",
				Version: "1",
			},
		}

		childIDMap := map[string]types.IdVersion{
			"CHILD:Entity:1": {
				ID:      "CHILD:Entity:1",
				Version: "1",
			},
			"PARENT:Entity:1": { // Override parent
				ID:      "PARENT:Entity:1",
				Version: "2", // Different version
			},
		}

		parentCtx := NewJAXBValidationContext(parentReportID, parentCodespace, "parent.xml", parentIDMap)
		childCtx := NewJAXBValidationContext(parentReportID, parentCodespace, childFileName, childIDMap)

		// Child should inherit report ID and codespace
		if childCtx.ValidationReportID != parentReportID {
			t.Errorf("Child context should inherit report ID %s, got %s", parentReportID, childCtx.ValidationReportID)
		}

		if childCtx.Codespace != parentCodespace {
			t.Errorf("Child context should inherit codespace %s, got %s", parentCodespace, childCtx.Codespace)
		}

		// Child should have its own filename
		if childCtx.FileName != childFileName {
			t.Errorf("Child context should have filename %s, got %s", childFileName, childCtx.FileName)
		}

		// Check ID resolution in child context
		if version, exists := childCtx.LocalIDMap["CHILD:Entity:1"]; !exists {
			t.Error("Child context should contain child-specific ID")
		} else if version.Version != "1" {
			t.Errorf("Expected version 1 for child entity, got %s", version.Version)
		}

		// Check override behavior
		if version, exists := childCtx.LocalIDMap["PARENT:Entity:1"]; !exists {
			t.Error("Child context should contain overridden parent ID")
		} else if version.Version != "2" {
			t.Errorf("Expected overridden version 2, got %s", version.Version)
		}

		// Parent should be unchanged
		if version, exists := parentCtx.LocalIDMap["PARENT:Entity:1"]; !exists {
			t.Error("Parent context should still contain original ID")
		} else if version.Version != "1" {
			t.Errorf("Parent context should have original version 1, got %s", version.Version)
		}
	})
}

func TestJAXBValidationContext_SpecialIDFormats(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		version     string
		shouldStore bool
		description string
	}{
		{
			name:        "Standard NetEX ID",
			id:          "NO:Line:1",
			version:     "1",
			shouldStore: true,
			description: "Normal NetEX ID format",
		},
		{
			name:        "Long codespace",
			id:          "VERY_LONG_CODESPACE:Entity:12345",
			version:     "1",
			shouldStore: true,
			description: "ID with long codespace",
		},
		{
			name:        "Special characters in ID",
			id:          "TEST:Entity-With_Special.Chars:1",
			version:     "1",
			shouldStore: true,
			description: "ID containing special characters",
		},
		{
			name:        "Unicode in ID",
			id:          "TEST:Línea:1",
			version:     "1",
			shouldStore: true,
			description: "ID with Unicode characters",
		},
		{
			name:        "Empty ID",
			id:          "",
			version:     "1",
			shouldStore: true, // Should be allowed, though not recommended
			description: "Empty ID string",
		},
		{
			name:        "ID without colons",
			id:          "SimpleID",
			version:     "1",
			shouldStore: true,
			description: "ID not following NetEX colon convention",
		},
		{
			name:        "Very long version",
			id:          "TEST:Entity:1",
			version:     "1.0.0.0.0.0.BUILD.123456789",
			shouldStore: true,
			description: "Very long version string",
		},
		{
			name:        "Empty version",
			id:          "TEST:Entity:1",
			version:     "",
			shouldStore: true,
			description: "Empty version string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localIDMap := map[string]types.IdVersion{
				tt.id: {
					ID:      tt.id,
					Version: tt.version,
				},
			}

			ctx := NewJAXBValidationContext(testutil.TestReportID, testutil.TestCodespace, testutil.TestFileName, localIDMap)

			if tt.shouldStore {
				if version, exists := ctx.LocalIDMap[tt.id]; !exists {
					t.Errorf("Expected to store ID %q (%s)", tt.id, tt.description)
				} else {
					if version.ID != tt.id {
						t.Errorf("Stored ID %q does not match original %q", version.ID, tt.id)
					}
					if version.Version != tt.version {
						t.Errorf("Stored version %q does not match original %q", version.Version, tt.version)
					}
				}
			}

			t.Logf("Test case: %s - %s", tt.name, tt.description)
		})
	}
}

func TestJAXBValidationContext_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	t.Run("Concurrent access to JAXB context", func(t *testing.T) {
		// Create a context with a reasonable number of IDs
		localIDMap := make(map[string]types.IdVersion)
		for i := 0; i < 1000; i++ {
			id := fmt.Sprintf("TEST:Entity:%d", i)
			localIDMap[id] = types.IdVersion{
				ID:      id,
				Version: "1",
			}
		}

		ctx := NewJAXBValidationContext(testutil.TestReportID, testutil.TestCodespace, testutil.TestFileName, localIDMap)

		const numGoroutines = 10
		const operationsPerGoroutine = 100
		var wg sync.WaitGroup

		successCount := int64(0)
		errorCount := int64(0)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Test different types of access
					switch j % 4 {
					case 0:
						// Access existing ID
						testID := fmt.Sprintf("TEST:Entity:%d", j%1000)
						if _, exists := ctx.LocalIDMap[testID]; exists {
							atomic.AddInt64(&successCount, 1)
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					case 1:
						// Access non-existent ID
						testID := fmt.Sprintf("TEST:NonExistent:%d", j)
						if _, exists := ctx.LocalIDMap[testID]; !exists {
							atomic.AddInt64(&successCount, 1)
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					case 2:
						// Access context properties
						if ctx.ValidationReportID == testutil.TestReportID {
							atomic.AddInt64(&successCount, 1)
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					case 3:
						// Check map length
						if len(ctx.LocalIDMap) == 1000 {
							atomic.AddInt64(&successCount, 1)
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					}
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Concurrency test results:")
		t.Logf("  Successful operations: %d", successCount)
		t.Logf("  Failed operations: %d", errorCount)

		if errorCount > 0 {
			t.Errorf("Got %d errors in concurrent access test", errorCount)
		}
	})
}

func BenchmarkJAXBValidationContext_Creation(b *testing.B) {
	localIDMap := make(map[string]types.IdVersion)
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("TEST:Entity:%d", i)
		localIDMap[id] = types.IdVersion{
			ID:      id,
			Version: "1",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewJAXBValidationContext(testutil.TestReportID, testutil.TestCodespace, testutil.TestFileName, localIDMap)
	}
}

func BenchmarkJAXBValidationContext_IDLookup(b *testing.B) {
	localIDMap := make(map[string]types.IdVersion)
	for i := 0; i < 10000; i++ {
		id := fmt.Sprintf("TEST:Entity:%d", i)
		localIDMap[id] = types.IdVersion{
			ID:      id,
			Version: "1",
		}
	}

	ctx := NewJAXBValidationContext(testutil.TestReportID, testutil.TestCodespace, testutil.TestFileName, localIDMap)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testID := fmt.Sprintf("TEST:Entity:%d", i%10000)
		_ = ctx.LocalIDMap[testID]
	}
}
