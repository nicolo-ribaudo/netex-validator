package validator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/testutil"
)

func TestMalformedXML_Recovery(t *testing.T) {
	tests := []struct {
		name         string
		xmlContent   string
		expectError  bool
		expectIssues bool
		description  string
	}{
		{
			name: "Unclosed XML tag",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
					<!-- Missing closing Line tag -->
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false, // Validator handles gracefully, returns error via result
			expectIssues: true,  // Issues reported via error field
			description:  "XML with unclosed tag should be handled gracefully",
		},
		{
			name: "Mismatched XML tags",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
				</Route> <!-- Wrong closing tag -->
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false, // Validator handles gracefully, returns error via result
			expectIssues: true,  // Issues reported via error field
			description:  "XML with mismatched tags should be handled gracefully",
		},
		{
			name: "Invalid XML characters",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line ` + string(rune(0x00)) + `</Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false, // Validator handles gracefully
			expectIssues: true,  // Issues reported via error field
			description:  "XML with invalid characters should be handled gracefully",
		},
		{
			name: "Missing XML declaration",
			xmlContent: `<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false,
			expectIssues: true,
			description:  "XML without declaration should still be parseable",
		},
		{
			name:         "Completely empty content",
			xmlContent:   "",
			expectError:  false,
			expectIssues: true,
			description:  "Empty content should be handled gracefully",
		},
		{
			name:         "Only whitespace",
			xmlContent:   "   \n\t  \n   ",
			expectError:  false,
			expectIssues: true,
			description:  "Whitespace-only content should be handled gracefully",
		},
		{
			name:         "Non-XML content",
			xmlContent:   "This is not XML at all!",
			expectError:  false,
			expectIssues: true,
			description:  "Non-XML content should be handled gracefully",
		},
		{
			name: "XML with BOM (Byte Order Mark)",
			xmlContent: "\uFEFF<?xml version=\"1.0\" encoding=\"UTF-8\"?>" + `
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false,
			expectIssues: true,
			description:  "XML with BOM should be handled correctly",
		},
		{
			name: "XML with invalid encoding",
			xmlContent: `<?xml version="1.0" encoding="INVALID-ENCODING"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
</PublicationDelivery>`,
			expectError:  false,
			expectIssues: true,
			description:  "XML with invalid encoding should be handled gracefully",
		},
		{
			name: "XML with unescaped characters",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST & OTHER</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test & Line < > "</Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false, // Validator handles gracefully
			expectIssues: true,  // Issues reported via error field
			description:  "XML with unescaped characters should be detected",
		},
		{
			name: "XML with CDATA section",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name><![CDATA[Test & Line < > " with special chars]]></Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false,
			expectIssues: true,
			description:  "XML with CDATA should be processed correctly",
		},
		{
			name: "Deeply nested malformed XML",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
					<routes>
						<Route id="TEST:Route:1" version="1">
							<Name>Route Name</Name>
							<journeyPatterns>
								<JourneyPattern id="TEST:JP:1" version="1">
									<Name>Pattern
									<!-- Unclosed element deep in hierarchy -->
								</JourneyPattern>
							</journeyPatterns>
						</Route>
					</routes>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`,
			expectError:  false, // Validator handles gracefully
			expectIssues: true,  // Issues reported via error field
			description:  "Deeply nested malformed XML should be handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := testOptions(t)
			options.Codespace = testutil.TestCodespace
			options.SkipSchema = false // Test schema validation with malformed XML

			validator, err := NewWithOptions(options)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			result, err := validator.ValidateContent([]byte(tt.xmlContent), "malformed_test.xml")

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("%s: Expected validation error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: Unexpected validation error: %v", tt.description, err)
			}

			// If we got a result (no error), check for issues
			if result != nil {
				hasIssues := len(result.ValidationReportEntries) > 0 || result.Error != ""
				// For malformed XML, errors during validation count as issues too
				hasIssuesOrError := hasIssues || err != nil
				if tt.expectIssues && !hasIssuesOrError {
					t.Errorf("%s: Expected validation issues but found none", tt.description)
				}
				if hasIssues {
					t.Logf("%s: Found %d validation issues", tt.description, len(result.ValidationReportEntries))
					if result.Error != "" {
						t.Logf("%s: Validation error: %s", tt.description, result.Error)
					}
				}
			}
		})
	}
}

func TestMalformedXML_LargeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	t.Run("Large malformed XML handling", func(t *testing.T) {
		// Create a large malformed XML content
		var content strings.Builder
		content.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		content.WriteString(`<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">`)
		content.WriteString(`<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>`)
		content.WriteString(`<ParticipantRef>TEST</ParticipantRef>`)
		content.WriteString(`<dataObjects><ServiceFrame id="TEST:ServiceFrame:1" version="1"><lines>`)

		// Add many line elements
		for i := 0; i < 1000; i++ {
			content.WriteString(`<Line id="TEST:Line:`)
			content.WriteString(fmt.Sprintf("%d", i))
			content.WriteString(`" version="1"><Name>Line `)
			content.WriteString(fmt.Sprintf("%d", i))
			content.WriteString(`</Name>`)
			// Intentionally miss closing tag for every 10th element
			if i%10 != 0 {
				content.WriteString(`</Line>`)
			}
		}

		content.WriteString(`</lines></ServiceFrame></dataObjects>`)
		// Missing final closing tag to make it malformed
		// content.WriteString(`</PublicationDelivery>`)

		options := testOptions(t)
		options.Codespace = testutil.TestCodespace

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		result, err := validator.ValidateContent([]byte(content.String()), "large_malformed.xml")

		// Should handle large malformed files gracefully
		if err != nil {
			t.Logf("Large malformed XML handling result: validation error (expected)")
		}

		if result != nil && result.Error != "" {
			t.Logf("Large malformed XML handled with validation error: %s", result.Error)
		}

		t.Log("Large malformed XML test completed - system remained stable")
	})
}

func TestMalformedXML_EncodingIssues(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		description string
	}{
		{
			name: "UTF-8 with invalid bytes",
			content: []byte{0xEF, 0xBF, 0xBD, // UTF-8 BOM
				'<', '?', 'x', 'm', 'l', ' ', 'v', 'e', 'r', 's', 'i', 'o', 'n', '=', '"', '1', '.', '0', '"', '?', '>',
				'<', 'r', 'o', 'o', 't', '>',
				0xFF, 0xFE, // Invalid UTF-8 bytes
				'<', '/', 'r', 'o', 'o', 't', '>'},
			description: "UTF-8 with invalid byte sequences",
		},
		{
			name: "Mixed encoding indicators",
			content: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<Name>Tëst Nåme with spëcial chars</Name>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`),
			description: "UTF-8 content with special characters",
		},
		{
			name: "Large content with encoding issues",
			content: append([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<dataObjects>`), append(make([]byte, 100000), []byte(`</dataObjects></PublicationDelivery>`)...)...),
			description: "Large content buffer with potential encoding edge cases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := testOptions(t)
			options.Codespace = testutil.TestCodespace

			validator, err := NewWithOptions(options)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			result, err := validator.ValidateContent(tt.content, "encoding_test.xml")

			// Should handle encoding issues gracefully without crashing
			if err != nil {
				t.Logf("%s: Validation error (expected): %v", tt.description, err)
			}

			if result != nil {
				if result.Error != "" {
					t.Logf("%s: Validation result error: %s", tt.description, result.Error)
				}
				t.Logf("%s: Found %d validation issues", tt.description, len(result.ValidationReportEntries))
			}

			// Most important: no panic or crash
			t.Logf("%s: Handled gracefully", tt.description)
		})
	}
}

func TestMalformedXML_RecoveryStrategies(t *testing.T) {
	t.Run("Partial parsing recovery", func(t *testing.T) {
		// XML that's valid up to a point, then becomes malformed
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Valid Line 1</Name>
				</Line>
				<Line id="TEST:Line:2" version="1">
					<Name>Valid Line 2</Name>
				</Line>
				<!-- From here it becomes malformed -->
				<Line id="TEST:Line:3" version="1">
					<Name>Malformed Line
					<!-- No closing Name or Line tags -->
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

		options := testOptions(t)
		options.Codespace = testutil.TestCodespace

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		result, err := validator.ValidateContent([]byte(xmlContent), "partial_recovery.xml")

		// Test should not panic or crash
		if err != nil {
			t.Logf("Partial parsing handled with error: %v", err)
		}

		if result != nil && result.Error != "" {
			t.Logf("Partial parsing result: %s", result.Error)
		}

		t.Log("Partial parsing recovery test completed successfully")
	})

	t.Run("Namespace corruption handling", func(t *testing.T) {
		// XML with corrupted namespace declarations
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" 
                     xmlns:corrupted="invalid://namespace/uri"
                     xmlns:missing=
                     version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
</PublicationDelivery>`

		options := testOptions(t)
		options.Codespace = testutil.TestCodespace

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		result, err := validator.ValidateContent([]byte(xmlContent), "namespace_corruption.xml")

		// Should handle namespace issues gracefully
		t.Logf("Namespace corruption test - Error: %v, Result: %v", err != nil, result != nil)
		t.Log("Namespace corruption handling test completed")
	})
}

func BenchmarkMalformedXML_HandlingPerformance(b *testing.B) {
	// Test performance impact of malformed XML handling
	malformedContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Malformed line without closing tag
			</lines>
		</ServiceFrame>
	</dataObjects>`

	options := testOptions(b)
	options.Codespace = testutil.TestCodespace

	validator, err := NewWithOptions(options)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateContent([]byte(malformedContent), "malformed_benchmark.xml")
	}
}
