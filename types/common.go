package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// IdVersion represents a NetEX ID with version information
type IdVersion struct {
	ID       string
	Version  string
	FileName string
}

// NewIdVersion creates a new IdVersion
func NewIdVersion(id, version, fileName string) IdVersion {
	return IdVersion{
		ID:       id,
		Version:  version,
		FileName: fileName,
	}
}

// DataLocation represents the location of data in an XML document
type DataLocation struct {
	FileName   string
	LineNumber int
	XPath      string
	ElementID  string
}

// Severity represents the severity level of a validation issue
type Severity int

const (
	INFO Severity = iota
	WARNING
	ERROR
	CRITICAL
)

func (s Severity) String() string {
	switch s {
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case CRITICAL:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// MarshalYAML implements the yaml.Marshaler interface
func (s Severity) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (s *Severity) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	return s.parseFromString(str)
}

// MarshalJSON encodes severity as its string label (e.g., "ERROR")
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON decodes severity from its string label
func (s *Severity) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	return s.parseFromString(str)
}

func (s *Severity) parseFromString(str string) error {
	switch str {
	case "INFO":
		*s = INFO
	case "WARNING":
		*s = WARNING
	case "ERROR":
		*s = ERROR
	case "CRITICAL":
		*s = CRITICAL
	default:
		return fmt.Errorf("invalid severity: %s", str)
	}
	return nil
}

// ValidationRule represents a validation rule configuration
type ValidationRule struct {
	Code     string   `yaml:"code"`
	Name     string   `yaml:"name"`
	Message  string   `yaml:"message"`
	Severity Severity `yaml:"severity"`
}

// ValidationIssue represents a validation finding
type ValidationIssue struct {
	Rule     ValidationRule
	Location DataLocation
	Message  string
	Data     interface{} // Optional additional data
}

// ValidationReportEntry represents a single entry in a validation report
type ValidationReportEntry struct {
	RuleID   string       `json:"ruleId,omitempty"`
	Name     string       `json:"name"`
	Message  string       `json:"message"`
	Severity Severity     `json:"severity"`
	FileName string       `json:"fileName"`
	Location DataLocation `json:"location"`
}

// ValidationReport represents the complete validation report
type ValidationReport struct {
	Codespace                        string                  `json:"codespace"`
	ValidationReportID               string                  `json:"validationReportId"`
	CreationDate                     time.Time               `json:"creationDate"`
	ValidationReportEntries          []ValidationReportEntry `json:"validationReportEntries"`
	NumberOfValidationEntriesPerRule map[string]int64        `json:"numberOfValidationEntriesPerRule"`
	// RuleChecksSkipped lists files whose XPath rule checks did not run because
	// schema validation found errors (or the findings cap was reached) first.
	RuleChecksSkipped []string `json:"-"`
	// FailedFiles lists files of a dataset that could not be validated, as "file: error".
	FailedFiles []string `json:"-"`
	// Truncated is set when the findings cap (MaxFindings) was reached, so
	// validation stopped early and the report may be incomplete.
	Truncated bool `json:"-"`
}

// NewValidationReport creates a new validation report
func NewValidationReport(codespace, reportID string) *ValidationReport {
	return &ValidationReport{
		Codespace:                        codespace,
		ValidationReportID:               reportID,
		CreationDate:                     time.Now(),
		ValidationReportEntries:          make([]ValidationReportEntry, 0),
		NumberOfValidationEntriesPerRule: make(map[string]int64),
	}
}

// AddValidationReportEntry adds a single validation report entry
func (vr *ValidationReport) AddValidationReportEntry(entry ValidationReportEntry) {
	vr.ValidationReportEntries = append(vr.ValidationReportEntries, entry)
	vr.NumberOfValidationEntriesPerRule[entry.Name]++
}

// AddAllValidationReportEntries adds multiple validation report entries
func (vr *ValidationReport) AddAllValidationReportEntries(entries []ValidationReportEntry) {
	for _, entry := range entries {
		vr.AddValidationReportEntry(entry)
	}
}

// HasError returns true if the validation report contains any errors or critical issues
func (vr *ValidationReport) HasError() bool {
	for _, entry := range vr.ValidationReportEntries {
		if entry.Severity == ERROR || entry.Severity == CRITICAL {
			return true
		}
	}
	return false
}

// MergeReport merges another validation report into this one
func (vr *ValidationReport) MergeReport(other *ValidationReport) {
	if other == nil {
		return
	}

	// Add all entries from the other report
	for _, entry := range other.ValidationReportEntries {
		vr.AddValidationReportEntry(entry)
	}
}
