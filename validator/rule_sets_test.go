package validator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/logging"
	"github.com/nicolo-ribaudo/netex-validator/testutil"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

const ruleSetTestXML = `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<!-- Missing Name: triggers EU line/2 -->
					<TransportMode>bus</TransportMode>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

func validateWithRuleSets(t *testing.T, sets ...RuleSet) map[string]bool {
	t.Helper()
	options := DefaultValidationOptions().WithCodespace(testutil.TestCodespace).WithSkipSchema(true)
	options.RuleSets = sets
	v, err := NewWithOptions(options)
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	result, err := v.ValidateContent([]byte(ruleSetTestXML), "rule_sets.xml")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	found := make(map[string]bool)
	for _, entry := range result.ValidationReportEntries {
		found[entry.Name] = true
	}
	return found
}

func TestRuleSets_CustomFileReplacesShippedRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.json")
	content := `{"rules": {"bus-line": {"name": "Bus line found", "severity": "INFO", "xpath": "//lines/Line[TransportMode='bus']"}}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	found := validateWithRuleSets(t, RuleSet{Name: "custom", Path: path})
	if !found["Bus line found"] {
		t.Errorf("custom rule did not fire; got %v", found)
	}
	if found["Line missing Name"] {
		t.Error("EU rule fired although only the custom rule set was loaded")
	}
}

func TestRuleSets_ShippedEU(t *testing.T) {
	found := validateWithRuleSets(t, RuleSet{Path: testutil.RuleSetPath(t, "eu")})
	if !found["Line missing Name"] {
		t.Errorf("expected EU rule line/2 to fire; got %v", found)
	}
}

func TestRuleSets_NoneMeansNoXPathRules(t *testing.T) {
	if found := validateWithRuleSets(t); len(found) != 0 {
		t.Errorf("expected no findings without rule sets, got %v", found)
	}
}

func TestRuleSets_LoadErrorFailsInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte(`{"rules": {"x": {"name": "X", "severity": "BAD", "xpath": "//x"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	options := DefaultValidationOptions().WithRuleSet("", path)
	_, err := NewWithOptions(options)
	if err == nil || !strings.Contains(err.Error(), "broken.json") {
		t.Errorf("expected load error naming broken.json, got %v", err)
	}
}

func TestRuleSets_WarnsAboutOverridesMatchingNoRule(t *testing.T) {
	var logs bytes.Buffer
	options := testOptions(t).
		WithLogger(logging.NewLogger(logging.LoggerConfig{Output: &logs})).
		WithRuleOverride("LINE_2", false).          // old-style code: no longer matches
		WithSeverityOverride("line/3", types.ERROR) // matches
	if _, err := NewWithOptions(options); err != nil {
		t.Fatal(err)
	}
	out := logs.String()
	if !strings.Contains(out, "Rule override has no effect") || !strings.Contains(out, "LINE_2") {
		t.Errorf("expected a warning for the stale override, got:\n%s", out)
	}
	if strings.Contains(out, "line/3") {
		t.Errorf("unexpected warning for a valid override:\n%s", out)
	}
}

func TestWithRuleSetIgnoresDuplicates(t *testing.T) {
	options := DefaultValidationOptions().WithRuleSet("eu", "a.json").WithRuleSet("eu", "a.json").WithRuleSet("", "b.json")
	if len(options.RuleSets) != 2 {
		t.Errorf("RuleSets = %v, want the duplicate to be ignored", options.RuleSets)
	}
}
