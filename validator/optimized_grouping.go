package validator

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

const (
	unknownID = "unknown"
)

// OptimizedGroupedResult represents a highly optimized grouped validation result.
// This format reduces repetitive notices by intelligently grouping similar issues,
// showing counts and affected files instead of individual duplicates.
type OptimizedGroupedResult struct {
	// Metadata
	Codespace          string    `json:"codespace"`
	ValidationReportID string    `json:"validationReportId"`
	CreationDate       time.Time `json:"creationDate"`
	GeneratedAt        time.Time `json:"generatedAt"`

	// High-level summary
	Summary OptimizedSummary `json:"summary"`

	// Optimized grouped notices
	Notices OptimizedNotices `json:"notices"`

	// Processing info
	FilesProcessed int           `json:"filesProcessed"`
	ProcessingTime time.Duration `json:"processingTimeMs"`
	CacheHit       bool          `json:"cacheHit,omitempty"`
	FileHash       string        `json:"fileHash,omitempty"`

	// Incomplete validation
	FailedFiles       []string `json:"failedFiles,omitempty"`
	RuleChecksSkipped []string `json:"ruleChecksSkipped,omitempty"`
	Truncated         bool     `json:"truncated,omitempty"`
	Error             string   `json:"error,omitempty"`

	// NetEX element statistics
	Statistics NetEXStatistics `json:"statistics,omitempty"`
}

// OptimizedSummary provides enhanced summary with grouping insights
type OptimizedSummary struct {
	TotalIssues      int  `json:"totalIssues"`
	UniqueIssueTypes int  `json:"uniqueIssueTypes"`
	ErrorCount       int  `json:"errorCount"`
	WarningCount     int  `json:"warningCount"`
	InfoCount        int  `json:"infoCount"`
	FilesProcessed   int  `json:"filesProcessed"`
	FilesWithIssues  int  `json:"filesWithIssues"`
	IsValid          bool `json:"isValid"`
}

// OptimizedNotices contains optimized grouped validation notices
type OptimizedNotices struct {
	Errors   []OptimizedNoticeGroup `json:"errors,omitempty"`
	Warnings []OptimizedNoticeGroup `json:"warnings,omitempty"`
	Info     []OptimizedNoticeGroup `json:"info,omitempty"`
}

// OptimizedNoticeGroup represents a group of similar validation notices with smart aggregation.
// For ID-related issues, groups by problematic NetEX ID with context.
// For other issues, groups by file with detailed occurrence information.
type OptimizedNoticeGroup struct {
	Type string `json:"type"`
	// RuleIDs lists the IDs of the rules reported under this type (several
	// rules can share a name, e.g. when loaded from different rule sets).
	RuleIDs     []string       `json:"ruleIds,omitempty"`
	Description string         `json:"description,omitempty"`
	Count       int            `json:"count"`
	Severity    types.Severity `json:"severity"`

	// File-level aggregation
	AffectedFiles []string                   `json:"affectedFiles"`
	FileDetails   map[string]FileIssueDetail `json:"fileDetails,omitempty"`

	// For ID-related issues, group by the problematic ID
	IDGroups map[string]IDIssueGroup `json:"idGroups,omitempty"`

	// Sample occurrences (for very large groups, show just a few examples)
	SampleOccurrences []OptimizedOccurrence `json:"sampleOccurrences,omitempty"`

	// Aggregation metadata
	ShowingDetails bool `json:"showingDetails"` // false if truncated for large groups
}

// FileIssueDetail contains issue details for a specific file
type FileIssueDetail struct {
	Count       int      `json:"count"`
	ElementIDs  []string `json:"elementIds,omitempty"`
	LineNumbers []int    `json:"lineNumbers,omitempty"`
}

// IDIssueGroup contains issues grouped by problematic NetEX ID
type IDIssueGroup struct {
	ID            string   `json:"id"`
	Count         int      `json:"count"`
	AffectedFiles []string `json:"affectedFiles"`
	Context       string   `json:"context,omitempty"` // e.g., "missing version '1'"
}

// ToOptimizedJSON converts the validation result to optimized grouped JSON format.
// This method intelligently groups similar validation issues to reduce output size
// and improve readability, especially for large datasets with many repetitive notices.
func (r *ValidationResult) ToOptimizedJSON() ([]byte, error) {
	optimized := r.createOptimizedGrouping()
	return json.MarshalIndent(optimized, "", "  ")
}

// createOptimizedGrouping creates optimized grouped structure with smart aggregation
func (r *ValidationResult) createOptimizedGrouping() *OptimizedGroupedResult {
	// Group by rule type and severity first
	ruleGroups := make(map[string][]ValidationReportEntry)
	for _, entry := range r.ValidationReportEntries {
		key := entry.Name + "|" + entry.Severity.String()
		ruleGroups[key] = append(ruleGroups[key], entry)
	}

	var errors, warnings, info []OptimizedNoticeGroup
	uniqueTypes := make(map[string]bool)

	for _, entries := range ruleGroups {
		if len(entries) == 0 {
			continue
		}

		firstEntry := entries[0]
		uniqueTypes[firstEntry.Name] = true

		group := r.createOptimizedGroup(firstEntry.Name, entries)

		switch firstEntry.Severity {
		case types.ERROR, types.CRITICAL:
			errors = append(errors, group)
		case types.WARNING:
			warnings = append(warnings, group)
		case types.INFO:
			info = append(info, group)
		}
	}

	// Sort groups by count (most frequent first)
	sortOptimizedGroups := func(groups []OptimizedNoticeGroup) {
		sort.Slice(groups, func(i, j int) bool {
			if groups[i].Count != groups[j].Count {
				return groups[i].Count > groups[j].Count
			}
			return groups[i].Type < groups[j].Type
		})
	}

	sortOptimizedGroups(errors)
	sortOptimizedGroups(warnings)
	sortOptimizedGroups(info)

	// Calculate optimized summary
	summary := r.calculateOptimizedSummary(len(uniqueTypes))

	// Calculate NetEX element statistics
	statistics := r.calculateNetEXStatistics()

	return &OptimizedGroupedResult{
		Codespace:          r.Codespace,
		ValidationReportID: r.ValidationReportID,
		CreationDate:       r.CreationDate,
		GeneratedAt:        time.Now(),
		Summary:            summary,
		Notices: OptimizedNotices{
			Errors:   errors,
			Warnings: warnings,
			Info:     info,
		},
		FilesProcessed:    r.FilesProcessed,
		ProcessingTime:    r.ProcessingTime,
		CacheHit:          r.CacheHit,
		FileHash:          r.FileHash,
		FailedFiles:       r.FailedFiles,
		RuleChecksSkipped: r.RuleChecksSkipped,
		Truncated:         r.Truncated,
		Error:             r.Error,
		Statistics:        statistics,
	}
}

// createOptimizedGroup creates an optimized notice group with smart aggregation
func (r *ValidationResult) createOptimizedGroup(ruleName string, entries []ValidationReportEntry) OptimizedNoticeGroup {
	firstEntry := entries[0]

	// Collect affected files
	filesMap := make(map[string][]ValidationReportEntry)
	for _, entry := range entries {
		filesMap[entry.FileName] = append(filesMap[entry.FileName], entry)
	}

	affectedFiles := make([]string, 0, len(filesMap))
	for fileName := range filesMap {
		affectedFiles = append(affectedFiles, fileName)
	}
	sort.Strings(affectedFiles)

	ruleIDSet := make(map[string]bool)
	for _, entry := range entries {
		if entry.RuleID != "" {
			ruleIDSet[entry.RuleID] = true
		}
	}
	ruleIDs := make([]string, 0, len(ruleIDSet))
	for id := range ruleIDSet {
		ruleIDs = append(ruleIDs, id)
	}
	sort.Strings(ruleIDs)

	group := OptimizedNoticeGroup{
		Type:           ruleName,
		RuleIDs:        ruleIDs,
		Description:    getDescriptionForRule(ruleName),
		Count:          len(entries),
		Severity:       firstEntry.Severity,
		AffectedFiles:  affectedFiles,
		ShowingDetails: true,
	}

	// For ID-related issues, group by problematic ID
	if strings.Contains(ruleName, "NeTEx ID") || strings.Contains(ruleName, "reference") {
		group.IDGroups = r.createIDGroups(entries)

		// If we have many ID groups, show only sample occurrences
		if len(group.IDGroups) > 10 {
			group.SampleOccurrences = r.createSampleOccurrences(entries, 5)
			group.ShowingDetails = false
		}
	} else {
		// For non-ID issues, show file details
		group.FileDetails = r.createFileDetails(filesMap)

		// If we have many files, show only sample occurrences
		if len(affectedFiles) > 20 {
			group.SampleOccurrences = r.createSampleOccurrences(entries, 10)
			group.ShowingDetails = false
			group.FileDetails = nil // Don't show detailed file info if too many
		}
	}

	return group
}

// createIDGroups groups issues by the problematic NetEX ID
func (r *ValidationResult) createIDGroups(entries []ValidationReportEntry) map[string]IDIssueGroup {
	idGroups := make(map[string]IDIssueGroup)

	for _, entry := range entries {
		// Extract ID from message (look for patterns like 'ID' or "ID")
		id := extractIDFromMessage(entry.Message)
		if id == "" {
			id = entry.Location.ElementID
		}
		if id == "" {
			id = unknownID
		}

		group, exists := idGroups[id]
		if !exists {
			group = IDIssueGroup{
				ID:            id,
				Count:         0,
				AffectedFiles: []string{},
				Context:       extractContextFromMessage(entry.Message),
			}
		}

		group.Count++

		// Add file if not already present
		fileExists := false
		for _, f := range group.AffectedFiles {
			if f == entry.FileName {
				fileExists = true
				break
			}
		}
		if !fileExists {
			group.AffectedFiles = append(group.AffectedFiles, entry.FileName)
		}

		idGroups[id] = group
	}

	// Sort affected files for each group
	for id, group := range idGroups {
		sort.Strings(group.AffectedFiles)
		idGroups[id] = group
	}

	return idGroups
}

// createFileDetails creates detailed file information for non-ID issues
func (r *ValidationResult) createFileDetails(filesMap map[string][]ValidationReportEntry) map[string]FileIssueDetail {
	fileDetails := make(map[string]FileIssueDetail)

	for fileName, fileEntries := range filesMap {
		detail := FileIssueDetail{
			Count:       len(fileEntries),
			ElementIDs:  []string{},
			LineNumbers: []int{},
		}

		// Collect unique element IDs and line numbers
		elementIDSet := make(map[string]bool)
		lineNumberSet := make(map[int]bool)

		for _, entry := range fileEntries {
			if entry.Location.ElementID != "" {
				elementIDSet[entry.Location.ElementID] = true
			}
			if entry.Location.LineNumber > 0 {
				lineNumberSet[entry.Location.LineNumber] = true
			}
		}

		// Convert to sorted slices
		for id := range elementIDSet {
			detail.ElementIDs = append(detail.ElementIDs, id)
		}
		sort.Strings(detail.ElementIDs)

		for line := range lineNumberSet {
			detail.LineNumbers = append(detail.LineNumbers, line)
		}
		sort.Ints(detail.LineNumbers)

		fileDetails[fileName] = detail
	}

	return fileDetails
}

// OptimizedOccurrence represents a single occurrence in optimized format
type OptimizedOccurrence struct {
	FileName   string `json:"fileName"`
	LineNumber int    `json:"lineNumber,omitempty"`
	XPath      string `json:"xpath,omitempty"`
	ElementID  string `json:"elementId,omitempty"`
	Message    string `json:"message,omitempty"`
}

// createSampleOccurrences creates sample occurrences for large groups
func (r *ValidationResult) createSampleOccurrences(entries []ValidationReportEntry, maxSamples int) []OptimizedOccurrence {
	samples := make([]OptimizedOccurrence, 0, maxSamples)

	// Take samples from different files if possible
	filesSeen := make(map[string]bool)

	for _, entry := range entries {
		if len(samples) >= maxSamples {
			break
		}

		// Prefer samples from different files
		if !filesSeen[entry.FileName] || len(samples) < maxSamples/2 {
			samples = append(samples, OptimizedOccurrence{
				FileName:   entry.FileName,
				LineNumber: entry.Location.LineNumber,
				XPath:      entry.Location.XPath,
				ElementID:  entry.Location.ElementID,
				Message:    entry.Message,
			})
			filesSeen[entry.FileName] = true
		}
	}

	return samples
}

// calculateOptimizedSummary calculates summary statistics for optimized grouping
func (r *ValidationResult) calculateOptimizedSummary(uniqueTypes int) OptimizedSummary {
	severityCounts := r.GetIssuesBySeverity()

	errorCount := len(severityCounts[types.ERROR]) + len(severityCounts[types.CRITICAL])
	warningCount := len(severityCounts[types.WARNING])
	infoCount := len(severityCounts[types.INFO])

	// Calculate files with issues
	filesWithIssues := make(map[string]bool)
	for _, entry := range r.ValidationReportEntries {
		if entry.FileName != "" {
			filesWithIssues[entry.FileName] = true
		}
	}

	return OptimizedSummary{
		TotalIssues:      len(r.ValidationReportEntries),
		UniqueIssueTypes: uniqueTypes,
		ErrorCount:       errorCount,
		WarningCount:     warningCount,
		InfoCount:        infoCount,
		FilesProcessed:   r.FilesProcessed,
		FilesWithIssues:  len(filesWithIssues),
		IsValid:          r.IsValid(),
	}
}

// extractIDFromMessage extracts NetEX ID from validation messages
func extractIDFromMessage(message string) string {
	if strings.Contains(message, "'") || strings.Contains(message, "\"") {
		// Extract first quoted string that looks like a NetEX ID
		start := strings.Index(message, "'")
		if start == -1 {
			start = strings.Index(message, "\"")
		}
		if start != -1 {
			start++
			end := strings.IndexAny(message[start:], "'\"")
			if end != -1 {
				id := message[start : start+end]
				// Check if it looks like a NetEX ID (contains colons)
				if strings.Contains(id, ":") {
					return id
				}
			}
		}
	}

	return ""
}

// extractContextFromMessage extracts context information from validation messages
func extractContextFromMessage(message string) string {
	// Extract useful context like "missing version '1'" or "has version 'any'"
	if strings.Contains(message, "missing version") {
		if strings.Contains(message, "while target has version") {
			start := strings.Index(message, "while target has version")
			if start != -1 {
				context := strings.TrimSpace(message[start:])
				// Clean up the context
				if len(context) > 50 {
					context = context[:50] + "..."
				}
				return context
			}
		}
		return "missing version"
	}

	if strings.Contains(message, "non-numeric") {
		return "version should be numeric"
	}

	return ""
}

// calculateNetEXStatistics extracts NetEX element statistics from raw content
func (r *ValidationResult) calculateNetEXStatistics() NetEXStatistics {
	if len(r.rawContent) == 0 {
		return NetEXStatistics{
			TotalFiles: r.FilesProcessed,
		}
	}

	var allStats []NetEXStatistics
	for fileName, content := range r.rawContent {
		stats := ExtractStatistics(content, fileName)
		allStats = append(allStats, stats)
	}

	merged := MergeStatistics(allStats)
	if merged.TotalFiles == 0 {
		merged.TotalFiles = r.FilesProcessed
	}
	return merged
}

// getDescriptionForRule returns a human-readable description for a validation rule
func getDescriptionForRule(ruleName string) string {
	descriptions := map[string]string{
		"Non-numeric NeTEx version":             "NetEX version should be numeric (e.g., '1.0', '1.4')",
		"NeTEx ID missing version on reference": "References to NetEX IDs should include version numbers when the target has a version",
		"NeTEx ID missing version on elements":  "NetEX elements should include version attributes",
		"Missing required Name element":         "Name element is required for this NetEX object",
		"Missing required TransportMode":        "TransportMode must be specified for this element",
		"Invalid TransportMode value":           "TransportMode value is not valid according to NetEX specification",
		"Missing OperatorRef":                   "OperatorRef is required to identify the operator",
		"Missing LineRef":                       "LineRef is required to identify the line",
		"Duplicate ID":                          "This ID is already used elsewhere in the dataset",
		"Invalid reference":                     "Referenced element does not exist",
		"Schema validation error":               "XML does not conform to NetEX schema",
	}

	if desc, ok := descriptions[ruleName]; ok {
		return desc
	}
	return ""
}
