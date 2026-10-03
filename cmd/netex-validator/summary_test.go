package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validator"
)

func entry(id, name string, sev types.Severity) validator.ValidationReportEntry {
	return validator.ValidationReportEntry{RuleID: id, Name: name, Severity: sev}
}

func TestCountByRule(t *testing.T) {
	counts := countByRule([]validator.ValidationReportEntry{
		entry("line/3", "Line missing PublicCode", types.WARNING),
		entry("line/2", "Line missing Name", types.ERROR),
		entry("line/3", "Line missing PublicCode", types.WARNING),
		entry("route/2", "Route missing Name", types.ERROR),
		entry("route/2", "Route missing Name", types.ERROR),
		entry("x/1", "Critical thing", types.CRITICAL),
	})

	var got []string
	for _, c := range counts {
		got = append(got, c.id)
	}
	// Most severe first, then most frequent, then by ID
	want := "x/1,route/2,line/2,line/3"
	if strings.Join(got, ",") != want {
		t.Errorf("order = %v, want %s", got, want)
	}
	if counts[1].count != 2 || counts[3].count != 2 {
		t.Errorf("unexpected counts: %+v", counts)
	}
}

func renderSummary(result *validator.ValidationResult) string {
	var buf bytes.Buffer
	printSummary(&buf, result, "data.xml", false)
	return buf.String()
}

func TestPrintSummary(t *testing.T) {
	result := &validator.ValidationResult{
		FilesProcessed: 1,
		ValidationReportEntries: []validator.ValidationReportEntry{
			entry("line/2", "Line missing Name", types.ERROR),
			entry("line/3", "Line missing PublicCode", types.WARNING),
			entry("line/3", "Line missing PublicCode", types.WARNING),
		},
		Rules: []validator.RuleInfo{{ID: "line/2"}, {ID: "line/3"}, {ID: "line/4"}, {ID: "route/2"}},
	}
	out := renderSummary(result)

	for _, want := range []string{
		"Validation summary for data.xml",
		"ERROR    line/2      1  Line missing Name",
		"WARNING  line/3      2  Line missing PublicCode",
		"3 findings (1 error, 2 warnings) in 1 file",
		"2 of 4 rules found no issues",
		"✘ Invalid: 1 error",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\033[") {
		t.Error("summary contains color codes although color is disabled")
	}
}

func TestPrintSummaryValid(t *testing.T) {
	out := renderSummary(&validator.ValidationResult{FilesProcessed: 1, Rules: []validator.RuleInfo{{ID: "a"}}})
	for _, want := range []string{"0 findings in 1 file", "1 of 1 rule found no issues", "✔ Valid"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestPrintSummaryRulesSkipped(t *testing.T) {
	out := renderSummary(&validator.ValidationResult{
		FilesProcessed:          1,
		ValidationReportEntries: []validator.ValidationReportEntry{entry("SCHEMA_ERROR", "Schema validation error", types.ERROR)},
		Rules:                   []validator.RuleInfo{{ID: "a"}, {ID: "b"}},
		RuleChecksSkipped:       []string{"data.xml"},
	})
	if !strings.Contains(out, "2 rules did not run because of schema errors") {
		t.Errorf("expected rules-did-not-run line:\n%s", out)
	}
	if strings.Contains(out, "found no issues") {
		t.Errorf("should not claim rules found no issues when they did not run:\n%s", out)
	}
}

func TestPrintSummaryPartialSkipAndFailedFiles(t *testing.T) {
	out := renderSummary(&validator.ValidationResult{
		FilesProcessed:    3,
		Rules:             []validator.RuleInfo{{ID: "a"}},
		RuleChecksSkipped: []string{"b.xml"},
		FailedFiles:       []string{"c.xml: failed to parse XML"},
	})
	for _, want := range []string{
		"1 of 1 rule found no issues",
		"rule checks were skipped in 1 file because of schema errors",
		"could not validate 1 file:",
		"c.xml: failed to parse XML",
		"✘ Invalid: 1 file not validated",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestPrintSummaryTruncated(t *testing.T) {
	out := renderSummary(&validator.ValidationResult{
		FilesProcessed:          1,
		ValidationReportEntries: []validator.ValidationReportEntry{entry("line/2", "Line missing Name", types.ERROR)},
		Rules:                   []validator.RuleInfo{{ID: "line/2"}, {ID: "line/3"}},
		Truncated:               true,
	})
	if !strings.Contains(out, "stopped after 1 finding (--max-findings)") {
		t.Errorf("expected truncation notice:\n%s", out)
	}
	if strings.Contains(out, "found no issues") {
		t.Errorf("must not claim rules found no issues when results are truncated:\n%s", out)
	}
}

func TestPrintSummaryError(t *testing.T) {
	out := renderSummary(&validator.ValidationResult{FilesProcessed: 1, Error: "failed to parse XML"})
	for _, want := range []string{"✘ failed to parse XML", "✘ Invalid: validation failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}
