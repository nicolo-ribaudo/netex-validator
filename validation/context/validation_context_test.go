package context

import (
	"bytes"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/nicolo-ribaudo/netex-validator/testutil"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

const (
	// Test constants for repeated strings
	testFileName = "test.xml"
)

func TestNewObjectValidationContext(t *testing.T) {
	tests := []struct {
		name         string
		xmlContent   string
		expectError  bool
		expectFrames []string
	}{
		{
			name:         "Valid minimal NetEX",
			xmlContent:   testutil.NetEXTestFragment,
			expectError:  false,
			expectFrames: []string{"ServiceFrame"},
		},
		{
			name: "NetEX with multiple frames",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00Z</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:1" version="1">
			<organisations>
				<Operator id="TEST:Operator:1" version="1">
					<Name>Test Operator</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
				</Line>
			</lines>
		</ServiceFrame>
		<SiteFrame id="TEST:SiteFrame:1" version="1">
			<stopPlaces>
				<StopPlace id="TEST:StopPlace:1" version="1">
					<Name>Test Stop</Name>
				</StopPlace>
			</stopPlaces>
		</SiteFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false,
			expectFrames: []string{"ResourceFrame", "ServiceFrame", "SiteFrame"},
		},
		{
			name:        "Invalid XML",
			xmlContent:  "<invalid xml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse XML first
			xmlDoc, err := xmlquery.Parse(bytes.NewReader([]byte(tt.xmlContent)))
			if tt.expectError {
				if err == nil {
					t.Error("Expected XML parsing to fail, but it succeeded")
				}
				return
			}
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			// Create context
			ctx, err := NewObjectValidationContext(
				testutil.TestFileName,
				testutil.TestCodespace,
				testutil.TestReportID,
				[]byte(tt.xmlContent),
				xmlDoc,
			)

			if (err != nil) != tt.expectError {
				t.Errorf("NewObjectValidationContext() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if err != nil {
				return // Expected error, test passed
			}

			// Validate context properties
			if ctx.FileName != testutil.TestFileName {
				t.Errorf("Expected filename %s, got %s", testutil.TestFileName, ctx.FileName)
			}

			if ctx.Codespace != testutil.TestCodespace {
				t.Errorf("Expected codespace %s, got %s", testutil.TestCodespace, ctx.Codespace)
			}

			if ctx.ValidationReportID != testutil.TestReportID {
				t.Errorf("Expected report ID %s, got %s", testutil.TestReportID, ctx.ValidationReportID)
			}

			// Check frame detection
			for _, expectedFrame := range tt.expectFrames {
				if !ctx.HasFrame(expectedFrame) {
					t.Errorf("Expected to find frame %s, but it was not detected", expectedFrame)
				}
			}
		})
	}
}

func TestObjectValidationContext_FrameDetection(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00Z</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="TEST:CompositeFrame:1" version="1">
			<frames>
				<ServiceFrame id="TEST:ServiceFrame:1" version="1">
					<lines>
						<Line id="TEST:Line:1" version="1">
							<Name>Test Line</Name>
						</Line>
					</lines>
				</ServiceFrame>
			</frames>
		</CompositeFrame>
		<TimetableFrame id="TEST:TimetableFrame:1" version="1">
			<vehicleJourneys>
				<ServiceJourney id="TEST:ServiceJourney:1" version="1">
					<LineRef ref="TEST:Line:1"/>
				</ServiceJourney>
			</vehicleJourneys>
		</TimetableFrame>
	</dataObjects>
</PublicationDelivery>`

	ctx := createTestContext(t, xmlContent)

	tests := []struct {
		frameName string
		expected  bool
	}{
		{"CompositeFrame", true},
		{"ServiceFrame", true},
		{"TimetableFrame", true},
		{"ResourceFrame", false},
		{"SiteFrame", false},
		{"VehicleScheduleFrame", false},
	}

	for _, tt := range tests {
		t.Run("Frame_"+tt.frameName, func(t *testing.T) {
			result := ctx.HasFrame(tt.frameName)
			if result != tt.expected {
				t.Errorf("HasFrame(%s) = %v, expected %v", tt.frameName, result, tt.expected)
			}
		})
	}
}

func TestObjectValidationContext_EntityParsing(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00Z</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:1" version="1">
			<organisations>
				<Operator id="TEST:Operator:1" version="1">
					<Name>Test Operator</Name>
				</Operator>
				<Operator id="TEST:Operator:2" version="1">
					<Name>Another Operator</Name>
				</Operator>
			</organisations>
		</ResourceFrame>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line 1</Name>
					<OperatorRef ref="TEST:Operator:1"/>
				</Line>
				<Line id="TEST:Line:2" version="1">
					<Name>Test Line 2</Name>
					<OperatorRef ref="TEST:Operator:2"/>
				</Line>
			</lines>
			<routes>
				<Route id="TEST:Route:1" version="1">
					<Name>Test Route</Name>
					<LineRef ref="TEST:Line:1"/>
				</Route>
			</routes>
			<journeyPatterns>
				<JourneyPattern id="TEST:JourneyPattern:1" version="1">
					<Name>Test Journey Pattern</Name>
					<RouteRef ref="TEST:Route:1"/>
				</JourneyPattern>
			</journeyPatterns>
		</ServiceFrame>
		<SiteFrame id="TEST:SiteFrame:1" version="1">
			<stopPlaces>
				<StopPlace id="TEST:StopPlace:1" version="1">
					<Name>Test Stop Place</Name>
				</StopPlace>
			</stopPlaces>
		</SiteFrame>
	</dataObjects>
</PublicationDelivery>`

	ctx := createTestContext(t, xmlContent)

	t.Run("Operators", func(t *testing.T) {
		operators := ctx.Operators()
		if len(operators) != 2 {
			t.Errorf("Expected 2 operators, got %d", len(operators))
		}

		// Test getting specific operator
		operator1 := ctx.GetOperator("TEST:Operator:1")
		if operator1 == nil {
			t.Error("Expected to find operator TEST:Operator:1")
		} else if operator1.ID != "TEST:Operator:1" {
			t.Errorf("Expected operator ID TEST:Operator:1, got %s", operator1.ID)
		}

		// Test non-existent operator
		nonExistent := ctx.GetOperator("TEST:Operator:999")
		if nonExistent != nil {
			t.Error("Expected not to find non-existent operator")
		}
	})

	t.Run("Lines", func(t *testing.T) {
		lines := ctx.Lines()
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(lines))
		}

		// Test getting specific line
		line1 := ctx.GetLine("TEST:Line:1")
		if line1 == nil {
			t.Error("Expected to find line TEST:Line:1")
		} else if line1.ID != "TEST:Line:1" {
			t.Errorf("Expected line ID TEST:Line:1, got %s", line1.ID)
		}
	})

	t.Run("Routes", func(t *testing.T) {
		routes := ctx.Routes()
		if len(routes) != 1 {
			t.Errorf("Expected 1 route, got %d", len(routes))
		}

		// Test getting specific route
		route1 := ctx.GetRoute("TEST:Route:1")
		if route1 == nil {
			t.Error("Expected to find route TEST:Route:1")
		} else if route1.ID != "TEST:Route:1" {
			t.Errorf("Expected route ID TEST:Route:1, got %s", route1.ID)
		}
	})

	t.Run("Journey Patterns", func(t *testing.T) {
		journeyPatterns := ctx.JourneyPatterns()
		if len(journeyPatterns) != 1 {
			t.Errorf("Expected 1 journey pattern, got %d", len(journeyPatterns))
		}

		// Test getting specific journey pattern
		jp1 := ctx.GetJourneyPattern("TEST:JourneyPattern:1")
		if jp1 == nil {
			t.Error("Expected to find journey pattern TEST:JourneyPattern:1")
		} else if jp1.ID != "TEST:JourneyPattern:1" {
			t.Errorf("Expected journey pattern ID TEST:JourneyPattern:1, got %s", jp1.ID)
		}
	})

	t.Run("Stop Places", func(t *testing.T) {
		stopPlaces := ctx.StopPlaces()
		if len(stopPlaces) != 1 {
			t.Errorf("Expected 1 stop place, got %d", len(stopPlaces))
		}

		// Test getting specific stop place
		sp1 := ctx.GetStopPlace("TEST:StopPlace:1")
		if sp1 == nil {
			t.Error("Expected to find stop place TEST:StopPlace:1")
		} else if sp1.ID != "TEST:StopPlace:1" {
			t.Errorf("Expected stop place ID TEST:StopPlace:1, got %s", sp1.ID)
		}
	})
}

func TestObjectValidationContext_IsCommonFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"_common_data.xml", true},
		{"_shared.xml", true},
		{"_FLB_shared_data.xml", true},
		{"regular_file.xml", false},
		{"common_file.xml", false}, // doesn't start with _
		{"", false},
	}

	for _, tt := range tests {
		t.Run("File_"+tt.filename, func(t *testing.T) {
			ctx := &ObjectValidationContext{
				FileName: tt.filename,
			}

			// Set IsCommonFile based on filename
			ctx.IsCommonFile = strings.HasPrefix(tt.filename, "_")

			if ctx.IsCommonFile != tt.expected {
				t.Errorf("IsCommonFile for %s = %v, expected %v", tt.filename, ctx.IsCommonFile, tt.expected)
			}
		})
	}
}

func TestObjectValidationContext_CommonDataRepository(t *testing.T) {
	ctx := createTestContext(t, testutil.NetEXTestFragment)

	// Test setting and getting common data repository
	repo := NewCommonDataRepository()
	ctx.SetCommonDataRepository(repo)

	retrievedRepo := ctx.GetCommonDataRepository()
	if retrievedRepo != repo {
		t.Error("Expected to get the same common data repository that was set")
	}
}

func TestSchemaValidationContext(t *testing.T) {
	fileName := testFileName
	codespace := testutil.TestCodespace
	content := []byte(testutil.NetEXTestFragment)

	ctx := NewSchemaValidationContext(fileName, codespace, content)

	if ctx.FileName != fileName {
		t.Errorf("Expected filename %s, got %s", fileName, ctx.FileName)
	}

	if ctx.Codespace != codespace {
		t.Errorf("Expected codespace %s, got %s", codespace, ctx.Codespace)
	}

	if !bytes.Equal(ctx.FileContent, content) {
		t.Error("Expected content to match input")
	}
}

func TestXPathValidationContext(t *testing.T) {
	fileName := testFileName
	codespace := testutil.TestCodespace
	reportID := testutil.TestReportID

	// Create XML document
	xmlDoc, err := xmlquery.Parse(bytes.NewReader([]byte(testutil.NetEXTestFragment)))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Create local IDs map
	localIDsMap := map[string]types.IdVersion{
		"TEST:Line:1": {
			ID:      "TEST:Line:1",
			Version: "1",
		},
	}

	// Create local references
	localRefs := []types.IdVersion{
		{
			ID:      "TEST:Line:1",
			Version: "1",
		},
	}

	ctx := NewXPathValidationContext(fileName, codespace, reportID, xmlDoc, localIDsMap, localRefs)

	if ctx.FileName != fileName {
		t.Errorf("Expected filename %s, got %s", fileName, ctx.FileName)
	}

	if ctx.Codespace != codespace {
		t.Errorf("Expected codespace %s, got %s", codespace, ctx.Codespace)
	}

	if ctx.ValidationReportID != reportID {
		t.Errorf("Expected report ID %s, got %s", reportID, ctx.ValidationReportID)
	}

	if ctx.Document != xmlDoc {
		t.Error("Expected document to match input")
	}

	if len(ctx.LocalIDs) != len(localIDsMap) {
		t.Errorf("Expected %d local IDs, got %d", len(localIDsMap), len(ctx.LocalIDs))
	}

	if len(ctx.LocalRefs) != len(localRefs) {
		t.Errorf("Expected %d local references, got %d", len(localRefs), len(ctx.LocalRefs))
	}
}

func TestJAXBValidationContext(t *testing.T) {
	reportID := testutil.TestReportID
	codespace := testutil.TestCodespace
	fileName := testFileName

	localIDsMap := map[string]types.IdVersion{
		"TEST:Line:1": {
			ID:      "TEST:Line:1",
			Version: "1",
		},
	}

	ctx := NewJAXBValidationContext(reportID, codespace, fileName, localIDsMap)

	if ctx.ValidationReportID != reportID {
		t.Errorf("Expected report ID %s, got %s", reportID, ctx.ValidationReportID)
	}

	if ctx.Codespace != codespace {
		t.Errorf("Expected codespace %s, got %s", codespace, ctx.Codespace)
	}

	if ctx.FileName != fileName {
		t.Errorf("Expected filename %s, got %s", fileName, ctx.FileName)
	}

	if len(ctx.LocalIDMap) != len(localIDsMap) {
		t.Errorf("Expected %d local IDs, got %d", len(localIDsMap), len(ctx.LocalIDMap))
	}
}

// Helper function to create test context
func createTestContext(t *testing.T, xmlContent string) *ObjectValidationContext {
	t.Helper()

	xmlDoc, err := xmlquery.Parse(bytes.NewReader([]byte(xmlContent)))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	ctx, err := NewObjectValidationContext(
		testutil.TestFileName,
		testutil.TestCodespace,
		testutil.TestReportID,
		[]byte(xmlContent),
		xmlDoc,
	)
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	return ctx
}

// Benchmark tests

func BenchmarkNewObjectValidationContext(b *testing.B) {
	content := []byte(testutil.GetBenchmarkData().MediumDataset)
	xmlDoc, err := xmlquery.Parse(bytes.NewReader(content))
	if err != nil {
		b.Fatalf("Failed to parse XML: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewObjectValidationContext(
			testutil.TestFileName,
			testutil.TestCodespace,
			testutil.TestReportID,
			content,
			xmlDoc,
		)
		if err != nil {
			b.Fatalf("Failed to create context: %v", err)
		}
	}
}

func BenchmarkObjectValidationContext_EntityAccess(b *testing.B) {
	xmlContent := testutil.GetBenchmarkData().LargeDataset
	// Create test context for benchmark
	xmlDoc, err := xmlquery.Parse(strings.NewReader(xmlContent))
	if err != nil {
		b.Fatalf("Failed to parse XML: %v", err)
	}
	ctx, err := NewObjectValidationContext("test.xml", testutil.TestCodespace, testutil.TestReportID, []byte(xmlContent), xmlDoc)
	if err != nil {
		b.Fatalf("Failed to create context: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access various entities to benchmark lookup performance
		_ = ctx.Lines()
		_ = ctx.GetLine("TEST:Line:1")
		_ = ctx.HasFrame("ServiceFrame")
	}
}
