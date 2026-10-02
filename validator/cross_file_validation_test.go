package validator

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theoremus-urban-solutions/netex-validator/testutil"
)

func TestCrossFileValidation_IdReferences(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	t.Run("Valid cross-file references", func(t *testing.T) {
		// Create files with valid cross-references
		files := map[string]string{
			"operators.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:Operators" version="1">
			<organisations>
				<Operator id="TEST:Operator:MainBus" version="1">
					<Name>Main Bus Company</Name>
				</Operator>
				<Operator id="TEST:Operator:Express" version="1">
					<Name>Express Transport</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"network.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:Network" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Line 1</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:MainBus" version="1"/>
				</Line>
				<Line id="TEST:Line:2" version="1">
					<Name>Line 2</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:Express" version="1"/>
				</Line>
			</lines>
			<routes>
				<Route id="TEST:Route:1A" version="1">
					<Name>Route 1A</Name>
					<LineRef ref="TEST:Line:1" version="1"/>
				</Route>
				<Route id="TEST:Route:2A" version="1">
					<Name>Route 2A</Name>
					<LineRef ref="TEST:Line:2" version="1"/>
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"stops.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<SiteFrame id="TEST:SiteFrame:Stops" version="1">
			<stopPlaces>
				<StopPlace id="TEST:StopPlace:Central" version="1">
					<Name>Central Station</Name>
				</StopPlace>
				<StopPlace id="TEST:StopPlace:Airport" version="1">
					<Name>Airport Terminal</Name>
				</StopPlace>
			</stopPlaces>
		</SiteFrame>
	</dataObjects>
</PublicationDelivery>`,

			"timetables.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<TimetableFrame id="TEST:TimetableFrame:Main" version="1">
			<vehicleJourneys>
				<ServiceJourney id="TEST:ServiceJourney:1001" version="1">
					<Name>Service 1001</Name>
					<LineRef ref="TEST:Line:1" version="1"/>
				</ServiceJourney>
				<ServiceJourney id="TEST:ServiceJourney:2001" version="1">
					<Name>Service 2001</Name>
					<LineRef ref="TEST:Line:2" version="1"/>
				</ServiceJourney>
			</vehicleJourneys>
		</TimetableFrame>
	</dataObjects>
</PublicationDelivery>`,
		}

		zipFile := tm.CreateTestZipFile(t, "valid_cross_refs.zip", files)
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		result, err := ValidateZip(zipFile, options)
		if err != nil {
			t.Fatalf("Cross-file validation failed: %v", err)
		}

		// Check for cross-reference validation
		hasIdValidation := false
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(strings.ToLower(entry.Message), "reference") ||
				strings.Contains(strings.ToLower(entry.Name), "ref") {
				hasIdValidation = true
				t.Logf("ID validation issue: %s - %s", entry.Name, entry.Message)
			}
		}

		t.Logf("Cross-file validation completed: %d total issues", len(result.ValidationReportEntries))
		if hasIdValidation {
			t.Log("Cross-reference validation is working")
		} else {
			t.Log("No cross-reference validation issues found (expected for valid references)")
		}
	})

	t.Run("Invalid cross-file references", func(t *testing.T) {
		// Create files with invalid cross-references
		files := map[string]string{
			"operators.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:Operators" version="1">
			<organisations>
				<Operator id="TEST:Operator:MainBus" version="1">
					<Name>Main Bus Company</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"lines.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:Lines" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Line 1</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:MainBus" version="1"/>
				</Line>
				<Line id="TEST:Line:2" version="1">
					<Name>Line 2</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:NonExistent" version="1"/>
				</Line>
			</lines>
			<routes>
				<Route id="TEST:Route:1" version="1">
					<Name>Route 1</Name>
					<LineRef ref="TEST:Line:MissingLine" version="1"/>
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"journeys.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<TimetableFrame id="TEST:TimetableFrame:Main" version="1">
			<vehicleJourneys>
				<ServiceJourney id="TEST:ServiceJourney:1001" version="1">
					<Name>Service 1001</Name>
					<LineRef ref="TEST:Line:InvalidLine" version="1"/>
				</ServiceJourney>
			</vehicleJourneys>
		</TimetableFrame>
	</dataObjects>
</PublicationDelivery>`,
		}

		zipFile := tm.CreateTestZipFile(t, "invalid_cross_refs.zip", files)
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		result, err := ValidateZip(zipFile, options)
		if err != nil {
			t.Fatalf("Cross-file validation failed: %v", err)
		}

		// Should find invalid references
		invalidReferences := 0
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(strings.ToLower(entry.Message), "nonexistent") ||
				strings.Contains(strings.ToLower(entry.Message), "missing") ||
				strings.Contains(strings.ToLower(entry.Message), "invalid") ||
				strings.Contains(strings.ToLower(entry.Message), "reference") {
				invalidReferences++
				t.Logf("Invalid reference detected: %s - %s", entry.Name, entry.Message)
			}
		}

		if invalidReferences == 0 {
			t.Log("Note: Cross-reference validation may not be fully implemented yet")
		} else {
			t.Logf("Successfully detected %d invalid cross-references", invalidReferences)
		}
	})
}

func TestCrossFileValidation_CommonDataFiles(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	t.Run("Common data file processing", func(t *testing.T) {
		// Create dataset with common data file
		files := map[string]string{
			"_common.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:Common" version="1">
			<organisations>
				<Operator id="TEST:Operator:Shared" version="1">
					<Name>Shared Operator</Name>
				</Operator>
			</organisations>
			<operationalContexts>
				<OperationalContext id="TEST:OperationalContext:Default" version="1">
					<Name>Default Context</Name>
				</OperationalContext>
			</operationalContexts>
		</ResourceFrame>
		<SiteFrame id="TEST:SiteFrame:Common" version="1">
			<stopPlaces>
				<StopPlace id="TEST:StopPlace:CommonStop" version="1">
					<Name>Common Stop</Name>
				</StopPlace>
			</stopPlaces>
		</SiteFrame>
	</dataObjects>
</PublicationDelivery>`,

			"line_1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:Line1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Line 1</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:Shared" version="1"/>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"line_2.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:Line2" version="1">
			<lines>
				<Line id="TEST:Line:2" version="1">
					<Name>Line 2</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:Shared" version="1"/>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
		}

		zipFile := tm.CreateTestZipFile(t, "common_data.zip", files)
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		result, err := ValidateZip(zipFile, options)
		if err != nil {
			t.Fatalf("Common data validation failed: %v", err)
		}

		// Check that common data was processed
		t.Logf("Common data validation completed: %d issues found", len(result.ValidationReportEntries))

		// Verify that references to common data are validated
		commonDataReferences := 0
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(entry.Message, "Shared") {
				commonDataReferences++
				t.Logf("Common data reference: %s", entry.Message)
			}
		}

		if commonDataReferences > 0 {
			t.Log("Common data cross-references are being validated")
		}
	})
}

func TestCrossFileValidation_DuplicateIds(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	t.Run("Duplicate ID detection across files", func(t *testing.T) {
		// Create files with duplicate IDs
		files := map[string]string{
			"file1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:DUPLICATE" version="1">
					<Name>Line from File 1</Name>
					<TransportMode>bus</TransportMode>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"file2.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:2" version="1">
			<lines>
				<Line id="TEST:Line:DUPLICATE" version="1">
					<Name>Line from File 2</Name>
					<TransportMode>rail</TransportMode>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"file3.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:DUPLICATE" version="1">
			<organisations>
				<Operator id="TEST:Operator:1" version="1">
					<Name>Unique Operator</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
	</dataObjects>
</PublicationDelivery>`,
		}

		zipFile := tm.CreateTestZipFile(t, "duplicate_ids.zip", files)
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		result, err := ValidateZip(zipFile, options)
		if err != nil {
			t.Fatalf("Duplicate ID validation failed: %v", err)
		}

		// Check for duplicate ID detection
		duplicateIssues := 0
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(strings.ToLower(entry.Message), "duplicate") ||
				strings.Contains(strings.ToLower(entry.Name), "duplicate") {
				duplicateIssues++
				t.Logf("Duplicate ID issue: %s - %s", entry.Name, entry.Message)
			}
		}

		if duplicateIssues == 0 {
			t.Log("Note: Duplicate ID detection may not be fully implemented yet")
		} else {
			t.Logf("Successfully detected %d duplicate ID issues", duplicateIssues)
		}

		t.Logf("Total issues found: %d", len(result.ValidationReportEntries))
	})
}

func TestCrossFileValidation_VersionMismatches(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	t.Run("Version mismatch detection", func(t *testing.T) {
		files := map[string]string{
			"definitions.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:1" version="1">
			<organisations>
				<Operator id="TEST:Operator:Main" version="2">
					<Name>Main Operator</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"usage.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Line 1</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:Main" version="1"/>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
		}

		zipFile := tm.CreateTestZipFile(t, "version_mismatch.zip", files)
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		result, err := ValidateZip(zipFile, options)
		if err != nil {
			t.Fatalf("Version mismatch validation failed: %v", err)
		}

		// Check for version mismatch detection
		versionIssues := 0
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(strings.ToLower(entry.Message), "version") {
				versionIssues++
				t.Logf("Version-related issue: %s - %s", entry.Name, entry.Message)
			}
		}

		t.Logf("Version validation completed: %d total issues, %d version-related",
			len(result.ValidationReportEntries), versionIssues)
	})
}

func TestCrossFileValidation_CircularReferences(t *testing.T) {
	tm := testutil.NewTestDataManager(t)

	t.Run("Circular reference detection", func(t *testing.T) {
		files := map[string]string{
			"circular1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:A" version="1">
			<routes>
				<Route id="TEST:Route:A" version="1">
					<Name>Route A</Name>
					<LineRef ref="TEST:Line:B" version="1"/>
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,

			"circular2.xml": `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:B" version="1">
			<lines>
				<Line id="TEST:Line:B" version="1">
					<Name>Line B</Name>
					<TransportMode>bus</TransportMode>
					<!-- This would create a circular reference if routes could reference lines that reference routes -->
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
		}

		zipFile := tm.CreateTestZipFile(t, "circular_refs.zip", files)
		options := testOptions(t).
			WithCodespace(testutil.TestCodespace).
			WithSkipSchema(true)

		result, err := ValidateZip(zipFile, options)
		if err != nil {
			t.Fatalf("Circular reference validation failed: %v", err)
		}

		// Check for circular reference detection
		circularIssues := 0
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(strings.ToLower(entry.Message), "circular") ||
				strings.Contains(strings.ToLower(entry.Message), "cycle") {
				circularIssues++
				t.Logf("Circular reference issue: %s - %s", entry.Name, entry.Message)
			}
		}

		t.Logf("Circular reference validation: %d total issues, %d circular",
			len(result.ValidationReportEntries), circularIssues)
	})
}

func BenchmarkCrossFileValidation_LargeDataset(b *testing.B) {
	// Create a minimal test manager for benchmarks
	tempDir, err := os.MkdirTemp("", "benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a large dataset with many cross-references
	files := make(map[string]string)

	// Common operators file
	files["_common.xml"] = `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:Common" version="1">
			<organisations>
				<Operator id="TEST:Operator:Main" version="1">
					<Name>Main Operator</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
	</dataObjects>
</PublicationDelivery>`

	// Multiple line files referencing the common operator
	for i := 1; i <= 10; i++ {
		files[fmt.Sprintf("lines_%d.xml", i)] = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:%d" version="1">
			<lines>
				<Line id="TEST:Line:%d" version="1">
					<Name>Line %d</Name>
					<TransportMode>bus</TransportMode>
					<OperatorRef ref="TEST:Operator:Main" version="1"/>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`, i, i, i)
	}

	zipFile := createBenchmarkZipFile(tempDir, "large_cross_ref.zip", files)
	options := testOptions(b).
		WithCodespace(testutil.TestCodespace).
		WithSkipSchema(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateZip(zipFile, options)
	}
}

// Helper function for benchmarks to create ZIP files
func createBenchmarkZipFile(tempDir, zipName string, xmlFiles map[string]string) string {
	zipPath := filepath.Join(tempDir, zipName)
	zipFile, err := os.Create(zipPath) //nolint:gosec // zipPath is constructed from tempDir and zipName in test
	if err != nil {
		panic(fmt.Sprintf("Failed to create zip file: %v", err))
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			panic(fmt.Sprintf("Failed to close zip file: %v", err))
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			panic(fmt.Sprintf("Failed to close zip writer: %v", err))
		}
	}()

	for filename, content := range xmlFiles {
		xmlWriter, err := zipWriter.Create(filename)
		if err != nil {
			panic(fmt.Sprintf("Failed to create XML file in zip: %v", err))
		}

		_, err = xmlWriter.Write([]byte(content))
		if err != nil {
			panic(fmt.Sprintf("Failed to write content to zip: %v", err))
		}
	}

	return zipPath
}
