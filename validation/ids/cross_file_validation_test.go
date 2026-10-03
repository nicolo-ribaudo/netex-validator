package ids

import (
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

const unresolvedReferenceCode = "NETEX_ID_5"

func TestCrossFileValidation(t *testing.T) {
	t.Run("Cross-file reference validation with shared IDs", func(t *testing.T) {
		repo := NewNetexIdRepository()

		// Add ID to common file
		err := repo.AddId("BISCARROSSE:Line:BIRB00024004640046:LOC", "1", "common.xml")
		if err != nil {
			t.Fatalf("Error adding ID to common file: %v", err)
		}
		repo.MarkAsCommonFile("common.xml")

		// Add reference from service file
		repo.AddReference("BISCARROSSE:Line:BIRB00024004640046:LOC", "1", "service.xml")

		// Validate references - should find no issues since reference is in common file
		issues := repo.ValidateReferences()

		// Should have no unresolved reference errors
		for _, issue := range issues {
			if issue.Rule.Code == unresolvedReferenceCode {
				t.Errorf("Expected no unresolved reference errors, got: %s", issue.Message)
			}
		}
	})

	t.Run("Shared NetEX IDs retrieval", func(t *testing.T) {
		repo := NewNetexIdRepository()

		// Add IDs to regular file
		if err := repo.AddId("TEST:Line:1", "1", "regular.xml"); err != nil {
			t.Fatalf("Error adding ID to regular file: %v", err)
		}

		// Add IDs to common file
		if err := repo.AddId("SHARED:Line:1", "1", "common.xml"); err != nil {
			t.Fatalf("Error adding ID to common file: %v", err)
		}
		if err := repo.AddId("SHARED:Network:1", "1", "common.xml"); err != nil {
			t.Fatalf("Error adding ID to common file: %v", err)
		}
		repo.MarkAsCommonFile("common.xml")

		// Get shared IDs
		sharedIds := repo.GetSharedNetexIds("test-report")

		// Should only return IDs from common files
		if !sharedIds["SHARED:Line:1"] {
			t.Error("Expected SHARED:Line:1 to be in shared IDs")
		}
		if !sharedIds["SHARED:Network:1"] {
			t.Error("Expected SHARED:Network:1 to be in shared IDs")
		}
		if sharedIds["TEST:Line:1"] {
			t.Error("Expected TEST:Line:1 NOT to be in shared IDs (not from common file)")
		}
	})

	t.Run("External reference validation", func(t *testing.T) {
		repo := NewNetexIdRepository()

		// Add external references that should be validated by external validator
		repo.AddReference("BISCARROSSE:Unknown:999", "1", "service.xml")
		repo.AddReference("FR:Unknown:123", "1", "service.xml")
		repo.AddReference("EXTERNAL:Unknown:456", "1", "service.xml")

		// Validate references
		issues := repo.ValidateReferences()

		// Count unresolved reference errors
		unresolvedCount := 0
		for _, issue := range issues {
			if issue.Rule.Code == unresolvedReferenceCode {
				unresolvedCount++
				// Should only report EXTERNAL: references as errors
				// BISCARROSSE: and FR: should be validated by FrenchExternalReferenceValidator
				if issue.Location.ElementID != "EXTERNAL:Unknown:456" {
					t.Errorf("Expected only EXTERNAL: references to be unresolved, got: %s", issue.Location.ElementID)
				}
			}
		}

		// Should have exactly 1 unresolved reference (EXTERNAL:Unknown:456)
		if unresolvedCount != 1 {
			t.Errorf("Expected 1 unresolved reference error, got: %d", unresolvedCount)
		}
	})
}

func TestEntityTypeValidation(t *testing.T) {
	t.Run("Valid entity types", func(t *testing.T) {
		repo := NewNetexIdRepository()

		testCases := []struct {
			id       string
			expected bool
		}{
			// Standard entity types
			{"BISCARROSSE:Line:1:LOC", true},
			{"BISCARROSSE:StopPlace:1:LOC", true},
			{"BISCARROSSE:Quay:1:LOC", true},

			// Calendar entity types (our fix)
			{"BISCARROSSE:DayTypeAssignment:476864163:LOC", true},
			{"BISCARROSSE:DayType:47712:LOC", true},
			{"BISCARROSSE:OperatingDay:1:LOC", true},

			// Route entity types (our fix)
			{"BISCARROSSE:RouteLink:428:LOC", true},
			{"BISCARROSSE:RoutePoint:1:LOC", true},

			// Accessibility entity types (our fix)
			{"BISCARROSSE:AccessibilityLimitation:GTFS_0346:LOC", true},

			// French frame patterns (our fix)
			{"BISCARROSSE:NETEX_LIGNE-20250617051421Z:LOC", true},
			{"BISCARROSSE:NETEX_HORAIRE-20250617051421Z:LOC", true},

			// Plain numeric IDs (our fix)
			{"3373611", true},
			{"123456", true},

			// Invalid entity types
			{"BISCARROSSE:UnknownEntity:1:LOC", false},
		}

		for _, tc := range testCases {
			t.Run(tc.id, func(t *testing.T) {
				isValid := repo.isValidNetexIdFormat(tc.id)
				if isValid != tc.expected {
					t.Errorf("ID %s: expected valid=%t, got valid=%t", tc.id, tc.expected, isValid)
				}
			})
		}
	})
}

func TestExternalReferenceValidator(t *testing.T) {
	t.Run("Default external reference validator", func(t *testing.T) {
		validator := NewDefaultExternalReferenceValidator()

		testRefs := []types.IdVersion{
			types.NewIdVersion("FR:Unknown:123", "1", "test.xml"),
			types.NewIdVersion("NSR:Unknown:456", "1", "test.xml"),
			types.NewIdVersion("UNKNOWN:Unknown:789", "1", "test.xml"),
		}

		validRefs := validator.ValidateReferenceIds(testRefs)

		// Should validate FR: and NSR: patterns
		validIds := make(map[string]bool)
		for _, ref := range validRefs {
			validIds[ref.ID] = true
		}

		if !validIds["FR:Unknown:123"] {
			t.Error("Expected FR: reference to be validated")
		}
		if !validIds["NSR:Unknown:456"] {
			t.Error("Expected NSR: reference to be validated")
		}
		if validIds["UNKNOWN:Unknown:789"] {
			t.Error("Expected UNKNOWN: reference NOT to be validated")
		}
	})

	t.Run("French external reference validator", func(t *testing.T) {
		validator := NewFrenchExternalReferenceValidator()

		testRefs := []types.IdVersion{
			types.NewIdVersion("BISCARROSSE:Unknown:123", "1", "test.xml"),
			types.NewIdVersion("MOBIITI:Unknown:456", "1", "test.xml"),
			types.NewIdVersion("GTFS:Unknown:789", "1", "test.xml"),
			types.NewIdVersion("UNKNOWN:Unknown:999", "1", "test.xml"),
		}

		validRefs := validator.ValidateReferenceIds(testRefs)

		// Should validate French patterns
		validIds := make(map[string]bool)
		for _, ref := range validRefs {
			validIds[ref.ID] = true
		}

		if !validIds["BISCARROSSE:Unknown:123"] {
			t.Error("Expected BISCARROSSE: reference to be validated")
		}
		if !validIds["MOBIITI:Unknown:456"] {
			t.Error("Expected MOBIITI: reference to be validated")
		}
		if !validIds["GTFS:Unknown:789"] {
			t.Error("Expected GTFS: reference to be validated")
		}
		if validIds["UNKNOWN:Unknown:999"] {
			t.Error("Expected UNKNOWN: reference NOT to be validated")
		}
	})
}

func TestJavaCompatibleValidation(t *testing.T) {
	t.Run("Complete Java-compatible validation workflow", func(t *testing.T) {
		repo := NewNetexIdRepository()

		// Simulate Biscarrosse dataset scenario
		// 1. Add ID to common file
		if err := repo.AddId("BISCARROSSE:Line:BIRB00024004640046:LOC", "1", "BISCARROSSE_commun.xml"); err != nil {
			t.Fatalf("Error adding ID to common file: %v", err)
		}
		repo.MarkAsCommonFile("BISCARROSSE_commun.xml")

		// 2. Add local IDs within service file
		if err := repo.AddId("BISCARROSSE:ScheduledStopPoint:BIRB_123:LOC", "any", "service.xml"); err != nil {
			t.Fatalf("Error adding ID to service file: %v", err)
		}

		// 3. Add references
		repo.AddReference("BISCARROSSE:Line:BIRB00024004640046:LOC", "1", "service.xml")    // Cross-file ref to common
		repo.AddReference("BISCARROSSE:ScheduledStopPoint:BIRB_123:LOC", "", "service.xml") // Local ref within same file
		repo.AddReference("BISCARROSSE:Unknown:999", "1", "service.xml")                    // External ref (French pattern)
		repo.AddReference("EXTERNAL:Unknown:123", "1", "service.xml")                       // True external ref (should error)

		// 4. Validate using Java-compatible algorithm
		issues := repo.ValidateReferences()

		// 5. Verify results match Java behavior
		errorCount := 0
		warningCount := 0

		for _, issue := range issues {
			switch issue.Rule.Severity {
			case types.ERROR:
				errorCount++
				// Should only have error for EXTERNAL: reference
				if issue.Rule.Code == unresolvedReferenceCode && issue.Location.ElementID != "EXTERNAL:Unknown:123" {
					t.Errorf("Unexpected ERROR for: %s", issue.Location.ElementID)
				}
			case types.WARNING:
				warningCount++
			}
		}

		// Should have exactly 1 ERROR (EXTERNAL:Unknown:123) and some WARNINGs for version mismatches
		if errorCount != 1 {
			t.Errorf("Expected exactly 1 ERROR, got: %d", errorCount)
		}

		t.Logf("Java-compatible validation: %d ERRORs, %d WARNINGs", errorCount, warningCount)
	})

	t.Run("French production dataset patterns", func(t *testing.T) {
		repo := NewNetexIdRepository()

		// Test all the ID patterns we found in production data that should be valid
		testIds := []string{
			// Calendar entities
			"BISCARROSSE:DayTypeAssignment:476864163:LOC",
			"BISCARROSSE:DayType:47712:LOC",

			// Route entities
			"BISCARROSSE:RouteLink:428:LOC",

			// Accessibility entities
			"BISCARROSSE:AccessibilityLimitation:GTFS_0346:LOC",

			// Frame patterns
			"BISCARROSSE:NETEX_LIGNE-20250617051421Z:LOC",
			"BISCARROSSE:NETEX_HORAIRE-20250617051421Z:LOC",

			// Plain numeric IDs
			"3373611",
			"123456",

			// External patterns that should be valid
			"MOBIITI:Quay:96309",
			"FR:57193:Quay:ENN.J2:RIV",
		}

		for _, id := range testIds {
			t.Run(id, func(t *testing.T) {
				if !repo.isValidNetexIdFormat(id) {
					t.Errorf("ID should be valid but was rejected: %s", id)
				}
			})
		}
	})
}
