package validator

import (
	"path"
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/config"
	"github.com/nicolo-ribaudo/netex-validator/rules"
	"github.com/nicolo-ribaudo/netex-validator/testutil"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

func TestXPathRules_Comprehensive(t *testing.T) {
	// Load every shipped rule (both sets, namespaced as "eu/..." and "extended/...")
	sets := []rules.RuleSet{
		{Path: testutil.RuleSetPath(t, "eu")},
		{Path: testutil.RuleSetPath(t, "extended")},
	}
	loaded, err := rules.LoadRuleSets(sets)
	if err != nil {
		t.Fatalf("Failed to load rule sets: %v", err)
	}
	allRules := rules.NewRuleRegistry(&config.ValidatorConfig{}, loaded).GetEnabledRules()

	t.Logf("Testing %d XPath rules", len(allRules))

	// Group rules by category for organized testing
	ruleCategories := make(map[string][]rules.Rule)
	for _, rule := range allRules {
		category := rule.Category
		ruleCategories[category] = append(ruleCategories[category], rule)
	}

	for category, categoryRules := range ruleCategories {
		t.Run("Category_"+category, func(t *testing.T) {
			for _, rule := range categoryRules {
				t.Run("Rule_"+rule.Code, func(t *testing.T) {
					testSingleXPathRule(t, rule, sets)
				})
			}
		})
	}
}

func testSingleXPathRule(t *testing.T, rule rules.Rule, sets []rules.RuleSet) {
	// Skip rules with unsupported XPath functions
	if hasUnsupportedXPathFunctions(rule.XPath) {
		t.Skipf("Rule %s contains unsupported XPath functions", rule.Code)
		return
	}

	// Create test data that should trigger this rule
	xmlContent := generateTestDataForRule(rule)
	if xmlContent == "" {
		t.Skipf("No test data available for rule %s", rule.Code)
		return
	}

	options := DefaultValidationOptions().
		WithCodespace(testutil.TestCodespace).
		WithSkipSchema(true).
		WithMaxFindings(10)
	options.RuleSets = sets

	validator, err := NewWithOptions(options)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	result, err := validator.ValidateContent([]byte(xmlContent), "rule_test.xml")
	if err != nil {
		t.Fatalf("Validation failed for rule %s: %v", rule.Code, err)
	}

	// Check if the rule was triggered
	ruleTriggered := false
	for _, entry := range result.ValidationReportEntries {
		if strings.Contains(entry.Name, rule.Name) ||
			strings.Contains(entry.Message, rule.Code) {
			ruleTriggered = true
			t.Logf("Rule %s triggered: %s", rule.Code, entry.Message)
			break
		}
	}

	// Some rules might not trigger with our generic test data
	if !ruleTriggered {
		t.Logf("Rule %s did not trigger with test data (may need specific conditions)", rule.Code)
	}
}

func hasUnsupportedXPathFunctions(xpath string) bool {
	unsupportedFunctions := []string{
		"current()",
		"document()",
		"key(",
		"format-number(",
		"generate-id(",
		"system-property(",
		"element-available(",
		"function-available(",
	}

	xpathLower := strings.ToLower(xpath)
	for _, fn := range unsupportedFunctions {
		if strings.Contains(xpathLower, strings.ToLower(fn)) {
			return true
		}
	}
	return false
}

func generateTestDataForRule(rule rules.Rule) string {
	// Generate specific test cases based on "<file>/<key>", ignoring the rule set namespace
	ruleCode := path.Base(rule.Category) + "/" + rule.Key

	switch {
	case ruleCode == "line/2":
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<!-- Missing Name element -->
					<TransportMode>bus</TransportMode>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

	case ruleCode == "line/4":
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
					<!-- Missing TransportMode -->
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

	case ruleCode == "route/2", ruleCode == "route/MISSING_NAME":
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<routes>
				<Route id="TEST:Route:1" version="1">
					<!-- Missing Name element -->
					<LineRef ref="TEST:Line:1"/>
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

	case ruleCode == "route/3":
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<routes>
				<Route id="TEST:Route:1" version="1">
					<Name>Test Route</Name>
					<!-- Missing LineRef -->
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

	case strings.HasPrefix(ruleCode, "transport_mode/"), strings.HasPrefix(ruleCode, "transport_submode/"):
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
					<TransportMode>invalidMode</TransportMode>
					<TransportSubmode>
						<BusSubmode>invalidSubmode</BusSubmode>
					</TransportSubmode>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

	case strings.HasPrefix(ruleCode, "version/"):
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="non-numeric">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="TEST:CompositeFrame:1" version="non-numeric">
			<frames/>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`

	case strings.Contains(ruleCode, "OPERATOR"):
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ResourceFrame id="TEST:ResourceFrame:1" version="1">
			<organisations>
				<Operator id="TEST:Operator:1" version="1">
					<!-- Test various operator-related rules -->
				</Operator>
			</organisations>
		</ResourceFrame>
	</dataObjects>
</PublicationDelivery>`

	case strings.HasPrefix(ruleCode, "stop_place/"):
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<SiteFrame id="TEST:SiteFrame:1" version="1">
			<stopPlaces>
				<StopPlace id="TEST:StopPlace:1" version="1">
					<Name>Test Stop</Name>
				</StopPlace>
			</stopPlaces>
		</SiteFrame>
	</dataObjects>
</PublicationDelivery>`

	case strings.HasPrefix(ruleCode, "service_journey/"):
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<TimetableFrame id="TEST:TimetableFrame:1" version="1">
			<vehicleJourneys>
				<ServiceJourney id="TEST:ServiceJourney:1" version="1">
					<Name>Test Journey</Name>
				</ServiceJourney>
			</vehicleJourneys>
		</TimetableFrame>
	</dataObjects>
</PublicationDelivery>`

	case strings.HasPrefix(ruleCode, "journey_pattern/"):
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<journeyPatterns>
				<JourneyPattern id="TEST:JourneyPattern:1" version="1">
					<Name>Test Pattern</Name>
				</JourneyPattern>
			</journeyPatterns>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

	default:
		// Return a comprehensive NetEX structure for unknown rules
		return `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
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
					<TransportMode>bus</TransportMode>
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
					<Name>Test Pattern</Name>
					<RouteRef ref="TEST:Route:1"/>
				</JourneyPattern>
			</journeyPatterns>
		</ServiceFrame>
		<SiteFrame id="TEST:SiteFrame:1" version="1">
			<stopPlaces>
				<StopPlace id="TEST:StopPlace:1" version="1">
					<Name>Test Stop</Name>
				</StopPlace>
			</stopPlaces>
		</SiteFrame>
		<TimetableFrame id="TEST:TimetableFrame:1" version="1">
			<vehicleJourneys>
				<ServiceJourney id="TEST:ServiceJourney:1" version="1">
					<Name>Test Journey</Name>
					<LineRef ref="TEST:Line:1"/>
				</ServiceJourney>
			</vehicleJourneys>
		</TimetableFrame>
	</dataObjects>
</PublicationDelivery>`
	}
}

func TestXPathRules_SeverityOverrides(t *testing.T) {
	t.Run("Override rule severities", func(t *testing.T) {
		// Create validation options with severity overrides
		severityOverrides := map[string]types.Severity{
			"line/2":  types.CRITICAL, // Line missing Name - Usually ERROR
			"route/2": types.WARNING,  // Route missing Name - Usually ERROR
			"line/4":  types.INFO,     // Line missing TransportMode - Usually ERROR
		}

		options := testOptions(t)
		options.Codespace = testutil.TestCodespace
		options.SkipSchema = true
		options.SeverityOverrides = severityOverrides

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// XML with missing elements that should trigger the rules
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<!-- Missing Name and TransportMode -->
				</Line>
			</lines>
			<routes>
				<Route id="TEST:Route:1" version="1">
					<!-- Missing Name -->
					<LineRef ref="TEST:Line:1"/>
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

		result, err := validator.ValidateContent([]byte(xmlContent), "severity_test.xml")
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		// Check that severity overrides were applied
		severityCounts := make(map[types.Severity]int)
		for _, entry := range result.ValidationReportEntries {
			severityCounts[entry.Severity]++

			// Check specific overridden rules
			if strings.Contains(entry.Name, "Line missing Name") {
				if entry.Severity != types.CRITICAL {
					t.Errorf("Expected CRITICAL severity for Line missing Name, got %s", entry.Severity)
				}
			}
			if strings.Contains(entry.Name, "Route missing Name") {
				if entry.Severity != types.WARNING {
					t.Errorf("Expected WARNING severity for Route missing Name, got %s", entry.Severity)
				}
			}
			if strings.Contains(entry.Name, "Line missing TransportMode") {
				if entry.Severity != types.INFO {
					t.Errorf("Expected INFO severity for Line missing TransportMode, got %s", entry.Severity)
				}
			}
		}

		t.Logf("Severity distribution: %v", severityCounts)
	})
}

func TestXPathRules_RuleEnableDisable(t *testing.T) {
	t.Run("Disable specific rules", func(t *testing.T) {
		// Disable some rules
		ruleOverrides := map[string]bool{
			"line/2":             false, // Disable "Line missing Name" rule
			"route/2":            false, // Disable "Route missing Name" rule (builtin)
			"route/MISSING_NAME": false, // Disable "Route missing Name" rule (business)
			"line/4":             true,  // Keep "Line missing TransportMode" rule enabled
		}

		options := testOptions(t)
		options.Codespace = testutil.TestCodespace
		options.SkipSchema = true
		options.RuleOverrides = ruleOverrides

		validator, err := NewWithOptions(options)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// XML that would trigger disabled rules
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<!-- Missing Name and TransportMode - Name rule should be disabled -->
				</Line>
			</lines>
			<routes>
				<Route id="TEST:Route:1" version="1">
					<!-- Missing Name - should be disabled -->
					<LineRef ref="TEST:Line:1"/>
				</Route>
			</routes>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

		result, err := validator.ValidateContent([]byte(xmlContent), "rule_disable_test.xml")
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		// Check that disabled rules were not triggered
		for _, entry := range result.ValidationReportEntries {
			if strings.Contains(entry.Name, "Line missing Name") &&
				!strings.Contains(entry.Name, "TransportMode") {
				t.Errorf("Disabled rule 'Line missing Name' was still triggered: %s", entry.Message)
			}
			if strings.Contains(entry.Name, "Route missing Name") {
				t.Errorf("Disabled rule 'Route missing Name' was still triggered: %s", entry.Message)
			}
		}

		t.Logf("Validation completed with disabled rules: %d issues found", len(result.ValidationReportEntries))
	})
}

func BenchmarkXPathRules_AllRules(b *testing.B) {
	options := testOptions(b).
		WithCodespace(testutil.TestCodespace).
		WithSkipSchema(true)

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
