package validator

import (
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

func TestDefaultValidationOptions(t *testing.T) {
	options := DefaultValidationOptions()

	if options == nil {
		t.Fatal("DefaultValidationOptions() returned nil")
	}

	// Check default values
	if options != nil && options.Codespace != "Default" {
		t.Errorf("expected default codespace 'Default', got %q", options.Codespace)
	}

	if options.ConfigFile != "" {
		t.Errorf("expected empty config file, got %q", options.ConfigFile)
	}

	if options.SkipSchema {
		t.Error("expected SkipSchema to be false by default")
	}

	if options.SkipValidators {
		t.Error("expected SkipValidators to be false by default")
	}

	if options.MaxSchemaErrors != 100 {
		t.Errorf("expected MaxSchemaErrors 100, got %d", options.MaxSchemaErrors)
	}

	if options.Verbose {
		t.Error("expected Verbose to be false by default")
	}

	if options.OutputFormat != "json" {
		t.Errorf("expected OutputFormat 'json', got %q", options.OutputFormat)
	}

	if options.RuleOverrides == nil {
		t.Error("RuleOverrides should be initialized")
	}

	if options.SeverityOverrides == nil {
		t.Error("SeverityOverrides should be initialized")
	}
}

func TestValidationOptions_WithCodespace(t *testing.T) {
	options := DefaultValidationOptions()

	result := options.WithCodespace("NO")

	// Should return the same instance for chaining
	if result != options {
		t.Error("WithCodespace should return the same instance for chaining")
	}

	if options.Codespace != "NO" {
		t.Errorf("expected codespace 'NO', got %q", options.Codespace)
	}
}

func TestValidationOptions_WithConfigFile(t *testing.T) {
	options := DefaultValidationOptions()

	result := options.WithConfigFile("config.yaml")

	if result != options {
		t.Error("WithConfigFile should return the same instance for chaining")
	}

	if options.ConfigFile != "config.yaml" {
		t.Errorf("expected config file 'config.yaml', got %q", options.ConfigFile)
	}
}

func TestValidationOptions_WithSkipSchema(t *testing.T) {
	options := DefaultValidationOptions()

	result := options.WithSkipSchema(true)

	if result != options {
		t.Error("WithSkipSchema should return the same instance for chaining")
	}

	if !options.SkipSchema {
		t.Error("expected SkipSchema to be true")
	}

	// Test setting back to false
	options.WithSkipSchema(false)
	if options.SkipSchema {
		t.Error("expected SkipSchema to be false")
	}
}

func TestValidationOptions_WithVerbose(t *testing.T) {
	options := DefaultValidationOptions()

	result := options.WithVerbose(true)

	if result != options {
		t.Error("WithVerbose should return the same instance for chaining")
	}

	if !options.Verbose {
		t.Error("expected Verbose to be true")
	}
}

func TestValidationOptions_WithRuleOverride(t *testing.T) {
	options := DefaultValidationOptions()

	result := options.WithRuleOverride("line/2", false)

	if result != options {
		t.Error("WithRuleOverride should return the same instance for chaining")
	}

	if options.RuleOverrides["line/2"] != false {
		t.Error("expected rule line/2 to be disabled")
	}

	// Test enabling a rule
	options.WithRuleOverride("route/3", true)
	if options.RuleOverrides["route/3"] != true {
		t.Error("expected rule route/3 to be enabled")
	}

	// Test with nil map (should initialize)
	options.RuleOverrides = nil
	options.WithRuleOverride("TEST_RULE", true)
	if options.RuleOverrides == nil {
		t.Error("RuleOverrides should be initialized")
	}
	if options.RuleOverrides["TEST_RULE"] != true {
		t.Error("expected TEST_RULE to be enabled")
	}
}

func TestValidationOptions_WithSeverityOverride(t *testing.T) {
	options := DefaultValidationOptions()

	result := options.WithSeverityOverride("line/2", types.ERROR)

	if result != options {
		t.Error("WithSeverityOverride should return the same instance for chaining")
	}

	if options.SeverityOverrides["line/2"] != types.ERROR {
		t.Errorf("expected rule line/2 severity to be ERROR, got %v", options.SeverityOverrides["line/2"])
	}

	// Test with different severity
	options.WithSeverityOverride("route/3", types.WARNING)
	if options.SeverityOverrides["route/3"] != types.WARNING {
		t.Errorf("expected rule route/3 severity to be WARNING, got %v", options.SeverityOverrides["route/3"])
	}

	// Test with nil map (should initialize)
	options.SeverityOverrides = nil
	options.WithSeverityOverride("TEST_RULE", types.CRITICAL)
	if options.SeverityOverrides == nil {
		t.Error("SeverityOverrides should be initialized")
	}
	if options.SeverityOverrides["TEST_RULE"] != types.CRITICAL {
		t.Errorf("expected TEST_RULE severity to be CRITICAL, got %v", options.SeverityOverrides["TEST_RULE"])
	}
}

func TestValidationOptions_Chaining(t *testing.T) {
	// Test method chaining works correctly
	options := DefaultValidationOptions().
		WithCodespace("SE").
		WithVerbose(true).
		WithSkipSchema(true).
		WithRuleOverride("line/2", false).
		WithSeverityOverride("route/3", types.ERROR).
		WithConfigFile("custom.yaml")

	// Verify all settings were applied
	if options.Codespace != "SE" {
		t.Errorf("expected codespace 'SE', got %q", options.Codespace)
	}

	if !options.Verbose {
		t.Error("expected Verbose to be true")
	}

	if !options.SkipSchema {
		t.Error("expected SkipSchema to be true")
	}

	if options.RuleOverrides["line/2"] != false {
		t.Error("expected rule line/2 to be disabled")
	}

	if options.SeverityOverrides["route/3"] != types.ERROR {
		t.Error("expected rule route/3 severity to be ERROR")
	}

	if options.ConfigFile != "custom.yaml" {
		t.Errorf("expected config file 'custom.yaml', got %q", options.ConfigFile)
	}
}

func TestValidationOptions_Independence(t *testing.T) {
	// Test that multiple options instances are independent
	options1 := DefaultValidationOptions().WithCodespace("NO")
	options2 := DefaultValidationOptions().WithCodespace("SE")

	if options1.Codespace != "NO" {
		t.Errorf("options1 codespace should be 'NO', got %q", options1.Codespace)
	}

	if options2.Codespace != "SE" {
		t.Errorf("options2 codespace should be 'SE', got %q", options2.Codespace)
	}

	// Modify one and ensure the other is not affected
	options1.WithVerbose(true)
	if options2.Verbose {
		t.Error("options2 should not be affected by options1 modifications")
	}
}

func TestValidationOptions_RuleOverrideMap(t *testing.T) {
	options := DefaultValidationOptions()

	// Test multiple rule overrides
	options.
		WithRuleOverride("RULE_1", true).
		WithRuleOverride("RULE_2", false).
		WithRuleOverride("RULE_3", true)

	expected := map[string]bool{
		"RULE_1": true,
		"RULE_2": false,
		"RULE_3": true,
	}

	for rule, expectedEnabled := range expected {
		if actualEnabled := options.RuleOverrides[rule]; actualEnabled != expectedEnabled {
			t.Errorf("rule %s: expected %v, got %v", rule, expectedEnabled, actualEnabled)
		}
	}

	// Test updating existing rule
	options.WithRuleOverride("RULE_1", false)
	if options.RuleOverrides["RULE_1"] != false {
		t.Error("RULE_1 should be updated to false")
	}
}

func TestValidationOptions_SeverityOverrideMap(t *testing.T) {
	options := DefaultValidationOptions()

	// Test multiple severity overrides
	options.
		WithSeverityOverride("RULE_1", types.ERROR).
		WithSeverityOverride("RULE_2", types.WARNING).
		WithSeverityOverride("RULE_3", types.CRITICAL)

	expected := map[string]types.Severity{
		"RULE_1": types.ERROR,
		"RULE_2": types.WARNING,
		"RULE_3": types.CRITICAL,
	}

	for rule, expectedSeverity := range expected {
		if actualSeverity := options.SeverityOverrides[rule]; actualSeverity != expectedSeverity {
			t.Errorf("rule %s: expected %v, got %v", rule, expectedSeverity, actualSeverity)
		}
	}

	// Test updating existing rule
	options.WithSeverityOverride("RULE_1", types.INFO)
	if options.SeverityOverrides["RULE_1"] != types.INFO {
		t.Error("RULE_1 severity should be updated to INFO")
	}
}
