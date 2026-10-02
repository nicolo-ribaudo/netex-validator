package rules

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/theoremus-urban-solutions/netex-validator/testutil"
	"github.com/theoremus-urban-solutions/netex-validator/types"
)

// writeRuleFile writes content to dir/name and returns its path.
func writeRuleFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func ruleJSON(key string) string {
	return `"` + key + `": {"name": "Rule ` + key + `", "severity": "WARNING", "xpath": "//Rule"}`
}

func codes(rules []Rule) []string {
	result := make([]string, len(rules))
	for i, r := range rules {
		result[i] = r.Code
	}
	return result
}

func assertCodes(t *testing.T, got []Rule, want ...string) {
	t.Helper()
	if strings.Join(codes(got), ",") != strings.Join(want, ",") {
		t.Errorf("codes = %v, want %v", codes(got), want)
	}
}

func mustLoad(t *testing.T, sets ...RuleSet) []Rule {
	t.Helper()
	loaded, err := LoadRuleSets(sets)
	if err != nil {
		t.Fatalf("LoadRuleSets: %v", err)
	}
	return loaded
}

func assertLoadError(t *testing.T, wantSubstr string, sets ...RuleSet) {
	t.Helper()
	_, err := LoadRuleSets(sets)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantSubstr)
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("error %q does not contain %q", err, wantSubstr)
	}
}

func TestShippedRuleSets(t *testing.T) {
	eu := mustLoad(t, RuleSet{Path: testutil.RuleSetPath(t, "eu")})
	if len(eu) != 73 {
		t.Errorf("eu: got %d active rules, want 73", len(eu))
	}

	all := mustLoad(t,
		RuleSet{Path: testutil.RuleSetPath(t, "eu")},
		RuleSet{Path: testutil.RuleSetPath(t, "extended")},
	)
	if len(all) != 237 {
		t.Errorf("eu+extended: got %d active rules, want 237", len(all))
	}
	for _, r := range all {
		if !strings.HasPrefix(r.Code, "eu/") && !strings.HasPrefix(r.Code, "extended/") {
			t.Errorf("rule %q is not namespaced", r.Code)
		}
	}

	// Each aggregate must give the same IDs as passing every file of its
	// directory, so no topic file is left out of the aggregate's "extends".
	for _, set := range []string{"eu", "extended"} {
		files, err := filepath.Glob(filepath.Join(filepath.Dir(testutil.RuleSetPath(t, set)), set, "*.json"))
		if err != nil || len(files) == 0 {
			t.Fatalf("glob %s/*.json: %v (%d files)", set, err, len(files))
		}
		var sets []RuleSet
		for _, f := range files {
			sets = append(sets, RuleSet{Path: f})
		}
		individual := codes(mustLoad(t, sets...))
		aggregate := codes(mustLoad(t, RuleSet{Path: testutil.RuleSetPath(t, set)}))
		sort.Strings(individual)
		sort.Strings(aggregate)
		if strings.Join(individual, ",") != strings.Join(aggregate, ",") {
			t.Errorf("%s.json IDs differ from individual %s/*.json IDs", set, set)
		}
	}
}

func TestLoadRuleSetsNamespacing(t *testing.T) {
	dir := t.TempDir()
	a := writeRuleFile(t, dir, "a.json", `{"rules": {`+ruleJSON("foo")+`}}`)
	b := writeRuleFile(t, dir, "b.json", `{"rules": {`+ruleJSON("bar")+`}}`)

	t.Run("single unnamed set is flat", func(t *testing.T) {
		assertCodes(t, mustLoad(t, RuleSet{Path: a}), "foo")
	})
	t.Run("single named set", func(t *testing.T) {
		assertCodes(t, mustLoad(t, RuleSet{Name: "x", Path: a}), "x/foo")
	})
	t.Run("multiple unnamed sets use file names", func(t *testing.T) {
		assertCodes(t, mustLoad(t, RuleSet{Path: a}, RuleSet{Path: b}), "a/foo", "b/bar")
	})
	t.Run("explicit name overrides default", func(t *testing.T) {
		assertCodes(t, mustLoad(t, RuleSet{Name: "x", Path: a}, RuleSet{Path: b}), "x/foo", "b/bar")
	})
	t.Run("same file under two names", func(t *testing.T) {
		assertCodes(t, mustLoad(t, RuleSet{Name: "x", Path: a}, RuleSet{Name: "y", Path: a}), "x/foo", "y/foo")
	})
	t.Run("duplicate namespace", func(t *testing.T) {
		assertLoadError(t, `rule set name "a" used by both`, RuleSet{Path: a}, RuleSet{Name: "a", Path: b})
	})
	t.Run("invalid namespace", func(t *testing.T) {
		assertLoadError(t, "invalid rule set name", RuleSet{Name: "a/b", Path: a})
	})
	t.Run("invalid default name suggests an explicit one", func(t *testing.T) {
		c := writeRuleFile(t, dir, "my.rules.json", `{"rules": {`+ruleJSON("baz")+`}}`)
		assertLoadError(t, "use --rules:<name>=", RuleSet{Path: a}, RuleSet{Path: c})
	})
}

func TestLoadRuleSetsRuleFields(t *testing.T) {
	dir := t.TempDir()
	path := writeRuleFile(t, dir, "line.json", `{
  "rules": {
    "2": {
      "name": "Line missing Name",
      "message": "Line is missing Name",
      "severity": "ERROR",
      "xpath": "//Line[not(Name)]",
      "description": "Lines need names"
    },
    "1": {"name": "Second", "severity": "INFO", "xpath": "//Other"}
  }
}`)

	loaded := mustLoad(t, RuleSet{Name: "eu", Path: path})
	assertCodes(t, loaded, "eu/2", "eu/1") // document order, not sorted

	want := Rule{
		Code:        "eu/2",
		Key:         "2",
		Name:        "Line missing Name",
		Message:     "Line is missing Name",
		Severity:    types.ERROR,
		XPath:       "//Line[not(Name)]",
		Category:    "eu",
		Description: "Lines need names",
	}
	if loaded[0] != want {
		t.Errorf("rule = %+v, want %+v", loaded[0], want)
	}
}

func TestLoadRuleSetsExtends(t *testing.T) {
	dir := t.TempDir()
	writeRuleFile(t, dir, "sub/stop_points.json", `{"rules": {`+ruleJSON("1")+`}}`)
	writeRuleFile(t, dir, "sub/deeper.json", `{"extends": {"stops": "./stop_points.json"}, "rules": {`+ruleJSON("own")+`}}`)
	top := writeRuleFile(t, dir, "top.json", `{
  "extends": {
    "deep": "./sub/deeper.json",
    "stop_points": "sub/stop_points.json",
    "again": "./sub/stop_points.json"
  },
  "rules": {`+ruleJSON("top")+`}
}`)

	t.Run("nested IDs, extends first, relative paths", func(t *testing.T) {
		assertCodes(t, mustLoad(t, RuleSet{Path: top}),
			"deep/stops/1", "deep/own", "stop_points/1", "again/1", "top")
	})
	t.Run("namespace applies above extends", func(t *testing.T) {
		loaded := mustLoad(t, RuleSet{Name: "foo", Path: top})
		if loaded[0].Code != "foo/deep/stops/1" || loaded[0].Category != "foo/deep/stops" || loaded[0].Key != "1" {
			t.Errorf("got code=%q category=%q key=%q", loaded[0].Code, loaded[0].Category, loaded[0].Key)
		}
	})
	t.Run("extends-only file", func(t *testing.T) {
		agg := writeRuleFile(t, dir, "agg.json", `{"extends": {"s": "./sub/stop_points.json"}}`)
		assertCodes(t, mustLoad(t, RuleSet{Path: agg}), "s/1")
	})
	t.Run("cycle", func(t *testing.T) {
		writeRuleFile(t, dir, "cycle_a.json", `{"extends": {"b": "./cycle_b.json"}}`)
		writeRuleFile(t, dir, "cycle_b.json", `{"extends": {"a": "./cycle_a.json"}}`)
		assertLoadError(t, "extends cycle", RuleSet{Path: filepath.Join(dir, "cycle_a.json")})
	})
	t.Run("missing target", func(t *testing.T) {
		p := writeRuleFile(t, dir, "missing.json", `{"extends": {"x": "./nope.json"}}`)
		assertLoadError(t, `missing.json: extends "x"`, RuleSet{Path: p})
	})
	t.Run("duplicate extends name", func(t *testing.T) {
		p := writeRuleFile(t, dir, "dupext.json", `{"extends": {"x": "./sub/stop_points.json", "x": "./sub/deeper.json"}}`)
		assertLoadError(t, `duplicate key "x"`, RuleSet{Path: p})
	})
	t.Run("invalid extends name", func(t *testing.T) {
		p := writeRuleFile(t, dir, "badext.json", `{"extends": {"a/b": "./sub/stop_points.json"}}`)
		assertLoadError(t, "invalid rule set name", RuleSet{Path: p})
	})
}

func TestLoadRuleSetsSkip(t *testing.T) {
	dir := t.TempDir()
	path := writeRuleFile(t, dir, "r.json", `{"rules": {
  `+ruleJSON("kept")+`,
  "skipped": {"name": "Uses current()", "severity": "ERROR", "xpath": "//a[@id=current()/b]", "skip": true}
}}`)
	assertCodes(t, mustLoad(t, RuleSet{Path: path}), "kept")
}

func TestLoadRuleSetsErrors(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"duplicate key", `{"rules": {` + ruleJSON("a") + `,` + ruleJSON("a") + `}}`, `duplicate key "a"`},
		{"key with slash", `{"rules": {` + ruleJSON("a/b") + `}}`, "invalid rule key"},
		{"key with space", `{"rules": {` + ruleJSON("a b") + `}}`, "invalid rule key"},
		{"empty", "  \n", "file is empty"},
		{"unknown rule field", `{"rules": {"a": {"name": "A", "severity": "ERROR", "xpath": "//a", "xPath": "//b"}}}`, `unknown field "xPath"`},
		{"duplicate rule field", `{"rules": {"a": {"name": "A", "name": "B", "severity": "ERROR", "xpath": "//a"}}}`, `duplicate key "name"`},
		{"unknown top-level field", `{"rule": {}}`, `unknown field "rule"`},
		{"missing name", `{"rules": {"a": {"severity": "ERROR", "xpath": "//a"}}}`, `missing "name"`},
		{"missing severity", `{"rules": {"a": {"name": "A", "xpath": "//a"}}}`, `missing "severity"`},
		{"missing xpath", `{"rules": {"a": {"name": "A", "severity": "ERROR"}}}`, `missing "xpath"`},
		{"invalid severity", `{"rules": {"a": {"name": "A", "severity": "FATAL", "xpath": "//a"}}}`, "invalid severity"},
		{"invalid xpath", `{"rules": {"a": {"name": "A", "severity": "ERROR", "xpath": "//a[@id=current()/b]"}}}`, "invalid xpath"},
		{"empty file", `{}`, `must have "rules" or "extends"`},
		{"rules not an object", `{"rules": []}`, "expected a JSON object"},
		{"trailing data", `{"rules": {}} {}`, "unexpected data"},
		{"malformed json", `{"rules": {`, "r.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeRuleFile(t, t.TempDir(), "r.json", tc.content)
			assertLoadError(t, tc.want, RuleSet{Path: path})
		})
	}

	t.Run("error names file and rule", func(t *testing.T) {
		path := writeRuleFile(t, t.TempDir(), "named.json", `{"rules": {"bad": {"name": "A", "severity": "ERROR", "xpath": "//a["}}}`)
		_, err := LoadRuleSets([]RuleSet{{Path: path}})
		if err == nil || !strings.Contains(err.Error(), "named.json") || !strings.Contains(err.Error(), `"bad"`) {
			t.Errorf("error %v should name the file and the rule", err)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		assertLoadError(t, "failed to read rule file", RuleSet{Path: filepath.Join(t.TempDir(), "nope.json")})
	})
}

func TestDefaultRuleSetName(t *testing.T) {
	for path, want := range map[string]string{
		"rulesets/eu.json": "eu",
		"./a.b.json":       "a.b",
		"noext":            "noext",
	} {
		if got := DefaultRuleSetName(path); got != want {
			t.Errorf("DefaultRuleSetName(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestLoadRuleSetsSymlinkedExtends(t *testing.T) {
	dir := t.TempDir()
	writeRuleFile(t, dir, "real/sub/x.json", `{"rules": {`+ruleJSON("1")+`}}`)
	writeRuleFile(t, dir, "real/agg.json", `{"extends": {"x": "./sub/x.json"}}`)
	link := filepath.Join(dir, "other", "link.json")
	if err := os.MkdirAll(filepath.Dir(link), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "real", "agg.json"), link); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}
	// "./sub/x.json" is resolved relative to the real file, not the link
	assertCodes(t, mustLoad(t, RuleSet{Path: link}), "x/1")
}
