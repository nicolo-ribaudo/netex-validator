package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validator"
)

// ANSI styles used by the summary.
const (
	styleReset  = "\033[0m"
	styleBold   = "\033[1m"
	styleDim    = "\033[2m"
	styleRed    = "\033[31m"
	styleGreen  = "\033[32m"
	styleYellow = "\033[33m"
	styleCyan   = "\033[36m"
	styleBoldRd = "\033[1;31m"
)

type painter struct{ enabled bool }

func (p painter) paint(style, s string) string {
	if !p.enabled {
		return s
	}
	return style + s + styleReset
}

// useColor reports whether output to f should be colored: f must be a
// terminal and NO_COLOR (https://no-color.org) must not be set.
func useColor(f *os.File) bool {
	return os.Getenv("NO_COLOR") == "" && isTerminal(f)
}

func severityStyle(s types.Severity) string {
	switch s {
	case types.CRITICAL:
		return styleBoldRd
	case types.ERROR:
		return styleRed
	case types.WARNING:
		return styleYellow
	default:
		return styleCyan
	}
}

// ruleCount is the number of findings reported for one rule.
type ruleCount struct {
	id       string
	name     string
	severity types.Severity
	count    int
}

// countByRule groups findings by rule, most severe first, then most frequent.
func countByRule(entries []validator.ValidationReportEntry) []ruleCount {
	index := make(map[string]int)
	var counts []ruleCount
	for _, e := range entries {
		key := e.RuleID + "\x00" + e.Name + "\x00" + e.Severity.String()
		i, ok := index[key]
		if !ok {
			i = len(counts)
			index[key] = i
			counts = append(counts, ruleCount{id: e.RuleID, name: e.Name, severity: e.Severity})
		}
		counts[i].count++
	}
	sort.SliceStable(counts, func(i, j int) bool {
		a, b := counts[i], counts[j]
		if a.severity != b.severity {
			return a.severity > b.severity
		}
		if a.count != b.count {
			return a.count > b.count
		}
		return a.id < b.id
	})
	return counts
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func padRight(s string, width int) string {
	if n := utf8.RuneCountInString(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

// printSummary writes a human-readable overview of result to w.
func printSummary(w io.Writer, result *validator.ValidationResult, input string, color bool) {
	// Write errors are ignored: there is nothing useful to do if stderr is gone
	printf := func(format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
	p := painter{enabled: color}
	counts := countByRule(result.ValidationReportEntries)
	summary := result.Summary()

	printf("%s %s\n", p.paint(styleBold, "Validation summary for"), input)
	printf("\n")

	if len(counts) > 0 {
		idWidth, levelWidth, countWidth := len("Rule"), len("Level"), len("Count")
		for _, c := range counts {
			idWidth = max(idWidth, utf8.RuneCountInString(c.id))
			levelWidth = max(levelWidth, len(c.severity.String()))
			countWidth = max(countWidth, len(fmt.Sprint(c.count)))
		}

		header := fmt.Sprintf("  %s  %s  %*s  %s", padRight("Level", levelWidth), padRight("Rule", idWidth), countWidth, "Count", "Description")
		printf("%s\n", p.paint(styleDim, header))
		for _, c := range counts {
			level := p.paint(severityStyle(c.severity), padRight(c.severity.String(), levelWidth))
			id := p.paint(styleDim, padRight(c.id, idWidth))
			printf("  %s  %s  %*d  %s\n", level, id, countWidth, c.count, c.name)
		}
		printf("\n")
	}

	// Totals per severity, most severe first
	var parts []string
	for _, s := range []types.Severity{types.CRITICAL, types.ERROR, types.WARNING, types.INFO} {
		if n := summary.IssuesBySeverity[s]; n > 0 {
			parts = append(parts, p.paint(severityStyle(s), plural(n, strings.ToLower(s.String()))))
		}
	}
	findings := plural(summary.TotalIssues, "finding")
	if len(parts) > 0 {
		findings += " (" + strings.Join(parts, ", ") + ")"
	}
	printf("  %s in %s\n", findings, plural(max(summary.FilesProcessed, 1), "file"))

	// Rules that ran without findings (only meaningful if they ran on at least one file)
	files := max(summary.FilesProcessed, 1)
	skipped := len(result.RuleChecksSkipped)
	switch {
	case result.Truncated:
		// Validation stopped early, so we can't tell which rules found nothing
		printf("  %s stopped after %s (--max-findings): there may be more issues\n",
			p.paint(styleYellow, "!"), plural(summary.TotalIssues, "finding"))
	case len(result.Rules) > 0 && skipped >= files:
		printf("  %s %s did not run because of schema errors\n",
			p.paint(styleYellow, "!"), plural(len(result.Rules), "rule"))
	case len(result.Rules) > 0:
		triggered := make(map[string]bool)
		for _, c := range counts {
			triggered[c.id] = true
		}
		clean := 0
		for _, r := range result.Rules {
			if !triggered[r.ID] {
				clean++
			}
		}
		printf("  %s of %s found no issues\n",
			p.paint(styleGreen, fmt.Sprint(clean)), plural(len(result.Rules), "rule"))
		if skipped > 0 {
			printf("  %s rule checks were skipped in %s because of schema errors\n",
				p.paint(styleYellow, "!"), plural(skipped, "file"))
		}
	}
	if result.Error != "" {
		printf("  %s %s\n", p.paint(styleBoldRd, "✘"), result.Error)
	}
	if n := len(result.FailedFiles); n > 0 {
		printf("  %s could not validate %s:\n", p.paint(styleBoldRd, "✘"), plural(n, "file"))
		for _, f := range result.FailedFiles {
			printf("      %s\n", f)
		}
	}

	printf("\n")
	if result.IsValid() {
		printf("%s\n", p.paint(styleGreen+styleBold, "✔ Valid"))
	} else {
		errorCount := summary.IssuesBySeverity[types.ERROR] + summary.IssuesBySeverity[types.CRITICAL]
		var reasons []string
		if errorCount > 0 || (result.Error == "" && len(result.FailedFiles) == 0) {
			reasons = append(reasons, plural(errorCount, "error"))
		}
		if n := len(result.FailedFiles); n > 0 {
			reasons = append(reasons, plural(n, "file")+" not validated")
		}
		if result.Error != "" {
			reasons = append(reasons, "validation failed")
		}
		reason := strings.Join(reasons, ", ")
		printf("%s\n", p.paint(styleBoldRd, "✘ Invalid: "+reason))
	}
}
