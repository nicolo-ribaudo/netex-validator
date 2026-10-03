package config

import (
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

func boolPtr(b bool) *bool                         { return &b }
func severityPtr(s types.Severity) *types.Severity { return &s }

func TestNamespaceAncestors(t *testing.T) {
	got := strings.Join(namespaceAncestors("eu/line/sub"), ",")
	if got != "eu/line/sub,eu/line,eu" {
		t.Errorf("namespaceAncestors = %q", got)
	}
	if len(namespaceAncestors("")) != 0 {
		t.Error("expected no ancestors for the empty namespace")
	}
}

func TestIsRuleEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rules.Categories = map[string]RuleCategoryConfig{
		"eu":         {Enabled: boolPtr(false)},
		"other":      {Rules: map[string]RuleConfig{"1": {Enabled: boolPtr(false)}, "2": {Severity: severityPtr(types.INFO)}}},
		"other/deep": {},
	}

	cases := []struct {
		ns, key string
		want    bool
	}{
		{"eu", "1", false},          // namespace disabled
		{"eu/line", "2", false},     // ancestor disabled
		{"other", "1", false},       // rule disabled
		{"other", "2", true},        // severity-only override doesn't disable
		{"other/deep", "1", true},   // per-rule overrides are not inherited
		{"unconfigured", "1", true}, // default
		{"", "flat", true},          // flat rules
	}
	for _, tc := range cases {
		if got := cfg.IsRuleEnabled(tc.ns, tc.key); got != tc.want {
			t.Errorf("IsRuleEnabled(%q, %q) = %v, want %v", tc.ns, tc.key, got, tc.want)
		}
	}
}

func TestGetRuleSeverity(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rules.Categories = map[string]RuleCategoryConfig{
		"eu":      {DefaultSeverity: severityPtr(types.WARNING)},
		"eu/line": {Rules: map[string]RuleConfig{"2": {Severity: severityPtr(types.CRITICAL)}}},
	}

	cases := []struct {
		ns, key string
		want    types.Severity
	}{
		{"eu/line", "2", types.CRITICAL}, // per-rule override wins
		{"eu/line", "3", types.WARNING},  // nearest ancestor default
		{"eu", "1", types.WARNING},       // own default
		{"other", "1", types.ERROR},      // rule's own severity
	}
	for _, tc := range cases {
		if got := cfg.GetRuleSeverity(tc.ns, tc.key, types.ERROR); got != tc.want {
			t.Errorf("GetRuleSeverity(%q, %q) = %v, want %v", tc.ns, tc.key, got, tc.want)
		}
	}
}

func TestUnmatchedRuleConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rules.Categories = map[string]RuleCategoryConfig{
		"eu":      {Enabled: boolPtr(true)},                                       // ancestor of eu/line: ok
		"eu/line": {Rules: map[string]RuleConfig{"2": {}, "LINE_2": {}}},          // LINE_2 is stale
		"line":    {Rules: map[string]RuleConfig{"2": {Enabled: boolPtr(false)}}}, // namespace not loaded
	}
	got := cfg.UnmatchedRuleConfig(map[string][]string{"eu/line": {"2", "3"}})
	want := []string{
		`rules.categories."eu/line".rules."LINE_2" matches no loaded rule`,
		`rules.categories."line" matches no loaded rule namespace`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("UnmatchedRuleConfig =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
