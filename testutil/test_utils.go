package testutil

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

// TestConstants commonly used in tests
const (
	TestCodespace         = "TEST"
	TestReportID          = "test-report"
	TestFileName          = "test.xml"
	TestValidationTimeout = 30 * time.Second
)

// NetEXTestFragment is a minimal valid NetEX fragment for testing
const NetEXTestFragment = `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" xmlns:gml="http://www.opengis.net/gml/3.2" version="1.15:NO-NeTEx-networktimetable:1.5">
	<PublicationTimestamp>2023-01-01T00:00:00Z</PublicationTimestamp>
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
</PublicationDelivery>`

// InvalidNetEXFragment is an invalid NetEX fragment for testing error scenarios
const InvalidNetEXFragment = `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex">
	<!-- Missing required elements -->
</PublicationDelivery>`

// TestDataManager handles test data files and resources
type TestDataManager struct {
	testDataDir string
	tempDir     string
}

// NewTestDataManager creates a new test data manager
func NewTestDataManager(t *testing.T) *TestDataManager {
	t.Helper()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "netex-validator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Clean up temp dir when test finishes
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	})

	return &TestDataManager{
		testDataDir: "testdata",
		tempDir:     tempDir,
	}
}

// CreateTestXMLFile creates a temporary XML file with the given content
func (tm *TestDataManager) CreateTestXMLFile(t *testing.T, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(tm.tempDir, filename)
	// Use restrictive permissions for test files
	err := os.WriteFile(filePath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", filename, err)
	}

	return filePath
}

// CreateTestZipFile creates a temporary ZIP file with the given XML contents
func (tm *TestDataManager) CreateTestZipFile(t *testing.T, zipName string, xmlFiles map[string]string) string {
	t.Helper()

	// Ensure zipName is a simple file name to avoid path traversal
	if filepath.Base(zipName) != zipName {
		t.Fatalf("invalid zip name: %s", zipName)
	}

	zipPath := filepath.Join(tm.tempDir, zipName)
	// Ensure the path stays within the temp directory
	absTemp, _ := filepath.Abs(tm.tempDir)
	absZip, _ := filepath.Abs(zipPath)
	if rel, err := filepath.Rel(absTemp, absZip); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("zip path escapes temp directory: %s", zipPath)
	}
	zipFile, err := os.Create(zipPath) //nolint:gosec // Path is validated above
	if err != nil {
		t.Fatalf("Failed to create zip file %s: %v", zipName, err)
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			t.Logf("Failed to close zip file %s: %v", zipPath, err)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			t.Logf("Failed to close zip writer for %s: %v", zipPath, err)
		}
	}()

	for filename, content := range xmlFiles {
		xmlWriter, err := zipWriter.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create XML file %s in zip: %v", filename, err)
		}

		_, err = xmlWriter.Write([]byte(content))
		if err != nil {
			t.Fatalf("Failed to write content to %s in zip: %v", filename, err)
		}
	}

	return zipPath
}

// CreateTempDir creates a temporary directory for testing
func (tm *TestDataManager) CreateTempDir(name string) string {
	// Ensure name is a simple directory name to avoid path traversal
	if filepath.Base(name) != name {
		return tm.tempDir
	}

	dirPath := filepath.Join(tm.tempDir, name)
	err := os.MkdirAll(dirPath, 0o750)
	if err != nil {
		// Return temp dir as fallback
		return tm.tempDir
	}
	return dirPath
}

// LoadTestFile loads a test file from testdata directory
func (tm *TestDataManager) LoadTestFile(t *testing.T, filename string) []byte {
	t.Helper()

	// Ensure filename is a simple file name to avoid path traversal
	if filepath.Base(filename) != filename {
		t.Fatalf("invalid test filename: %s", filename)
	}

	filePath := filepath.Join(tm.testDataDir, filename)
	// Ensure the resolved path is within the test data directory
	absBase, _ := filepath.Abs(tm.testDataDir)
	absPath, _ := filepath.Abs(filePath)
	if rel, err := filepath.Rel(absBase, absPath); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("test file path escapes base directory: %s", filePath)
	}
	content, err := os.ReadFile(filePath) //nolint:gosec // Path is validated above
	if err != nil {
		t.Fatalf("Failed to load test file %s: %v", filename, err)
	}

	return content
}

// Note: ValidateTest* functions moved to avoid import cycles.
// Use validator package directly in integration tests.

// AssertValidationResult provides convenient assertions for validation results
type AssertValidationResult struct {
	t      *testing.T
	report *types.ValidationReport
}

// NewAssertValidationResult creates a new assertion helper
func NewAssertValidationResult(t *testing.T, report *types.ValidationReport) *AssertValidationResult {
	t.Helper()
	return &AssertValidationResult{t: t, report: report}
}

// HasNoIssues asserts that the validation report has no issues
func (a *AssertValidationResult) HasNoIssues() *AssertValidationResult {
	a.t.Helper()
	if len(a.report.ValidationReportEntries) > 0 {
		a.t.Errorf("Expected no validation issues, but found %d issues", len(a.report.ValidationReportEntries))
		a.printIssues()
	}
	return a
}

// HasIssues asserts that the validation report has the expected number of issues
func (a *AssertValidationResult) HasIssues(expectedCount int) *AssertValidationResult {
	a.t.Helper()
	actualCount := len(a.report.ValidationReportEntries)
	if actualCount != expectedCount {
		a.t.Errorf("Expected %d validation issues, but found %d", expectedCount, actualCount)
		a.printIssues()
	}
	return a
}

// HasIssueWithRule asserts that the validation report contains an issue with the specified rule name
func (a *AssertValidationResult) HasIssueWithRule(ruleName string) *AssertValidationResult {
	a.t.Helper()
	for _, entry := range a.report.ValidationReportEntries {
		if entry.Name == ruleName {
			return a
		}
	}
	a.t.Errorf("Expected to find issue with rule '%s', but it was not found", ruleName)
	a.printIssues()
	return a
}

// HasIssueWithSeverity asserts that the validation report contains at least one issue with the specified severity
func (a *AssertValidationResult) HasIssueWithSeverity(severity types.Severity) *AssertValidationResult {
	a.t.Helper()
	for _, entry := range a.report.ValidationReportEntries {
		if entry.Severity == severity {
			return a
		}
	}
	a.t.Errorf("Expected to find issue with severity '%s', but it was not found", severity)
	a.printIssues()
	return a
}

// HasIssueInFile asserts that the validation report contains at least one issue in the specified file
func (a *AssertValidationResult) HasIssueInFile(fileName string) *AssertValidationResult {
	a.t.Helper()
	for _, entry := range a.report.ValidationReportEntries {
		if entry.FileName == fileName {
			return a
		}
	}
	a.t.Errorf("Expected to find issue in file '%s', but it was not found", fileName)
	a.printIssues()
	return a
}

// HasCodespace asserts that the validation report has the expected codespace
func (a *AssertValidationResult) HasCodespace(expectedCodespace string) *AssertValidationResult {
	a.t.Helper()
	if a.report.Codespace != expectedCodespace {
		a.t.Errorf("Expected codespace '%s', but got '%s'", expectedCodespace, a.report.Codespace)
	}
	return a
}

// IsValid asserts that the validation report indicates the data is valid
func (a *AssertValidationResult) IsValid() *AssertValidationResult {
	a.t.Helper()
	if !a.isReportValid() {
		a.t.Error("Expected validation to pass (no ERROR or CRITICAL issues), but it failed")
		a.printIssues()
	}
	return a
}

// IsInvalid asserts that the validation report indicates the data is invalid
func (a *AssertValidationResult) IsInvalid() *AssertValidationResult {
	a.t.Helper()
	if a.isReportValid() {
		a.t.Error("Expected validation to fail (have ERROR or CRITICAL issues), but it passed")
		a.printIssues()
	}
	return a
}

// printIssues prints all validation issues for debugging
func (a *AssertValidationResult) printIssues() {
	a.t.Helper()
	if len(a.report.ValidationReportEntries) == 0 {
		a.t.Log("No validation issues found")
		return
	}

	a.t.Logf("Validation issues (%d total):", len(a.report.ValidationReportEntries))
	for i, entry := range a.report.ValidationReportEntries {
		a.t.Logf("  %d. [%s] %s: %s (File: %s)", i+1, entry.Severity, entry.Name, entry.Message, entry.FileName)
	}
}

// isReportValid checks if the validation report indicates valid data (no ERROR or CRITICAL issues)
func (a *AssertValidationResult) isReportValid() bool {
	for _, entry := range a.report.ValidationReportEntries {
		if entry.Severity == types.ERROR || entry.Severity == types.CRITICAL {
			return false
		}
	}
	return true
}

// Note: CreateTestObjectValidationContext removed to avoid import cycles.
// Use validation/context package directly in integration tests.

// MockValidator is a simple validator for testing
type MockValidator struct {
	Name      string
	Issues    []types.ValidationIssue
	ShouldErr bool
}

// GetName returns the validator name
func (m *MockValidator) GetName() string {
	return m.Name
}

// Validate returns the predefined issues or an error
func (m *MockValidator) Validate(ctx interface{}) ([]types.ValidationIssue, error) {
	if m.ShouldErr {
		return nil, fmt.Errorf("mock validator error")
	}
	return m.Issues, nil
}

// CreateMockIssue creates a mock validation issue for testing
func CreateMockIssue(ruleName, message string, severity types.Severity, fileName string) types.ValidationIssue {
	return types.ValidationIssue{
		Rule: types.ValidationRule{
			Name:     ruleName,
			Code:     "MOCK_" + strings.ToUpper(strings.ReplaceAll(ruleName, " ", "_")),
			Message:  message,
			Severity: severity,
		},
		Message: message,
		Location: types.DataLocation{
			FileName: fileName,
		},
	}
}

// AssertXMLWellFormed checks if the given content is well-formed XML
func AssertXMLWellFormed(t *testing.T, xmlContent string) {
	t.Helper()

	var v interface{}
	err := xml.Unmarshal([]byte(xmlContent), &v)
	if err != nil {
		t.Errorf("XML is not well-formed: %v", err)
	}
}

// BenchmarkTestData provides utilities for benchmark testing
type BenchmarkTestData struct {
	SmallDataset  string
	MediumDataset string
	LargeDataset  string
}

// GetBenchmarkData returns test datasets of different sizes for benchmarking
func GetBenchmarkData() *BenchmarkTestData {
	return &BenchmarkTestData{
		SmallDataset:  NetEXTestFragment,
		MediumDataset: generateMediumTestDataset(),
		LargeDataset:  generateLargeTestDataset(),
	}
}

// generateMediumTestDataset creates a medium-sized test dataset
func generateMediumTestDataset() string {
	var lines strings.Builder
	lines.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00Z</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>`)

	// Generate 100 lines
	for i := 1; i <= 100; i++ {
		lines.WriteString(fmt.Sprintf(`
				<Line id="TEST:Line:%d" version="1">
					<Name>Test Line %d</Name>
				</Line>`, i, i))
	}

	lines.WriteString(`
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`)

	return lines.String()
}

// generateLargeTestDataset creates a large test dataset
func generateLargeTestDataset() string {
	var lines strings.Builder
	lines.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00Z</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>`)

	// Generate 1000 lines
	for i := 1; i <= 1000; i++ {
		lines.WriteString(fmt.Sprintf(`
				<Line id="TEST:Line:%d" version="1">
					<Name>Test Line %d</Name>
				</Line>`, i, i))
	}

	lines.WriteString(`
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`)

	return lines.String()
}

// RuleSetPath returns the absolute path of the shipped rule file rulesets/<name>.json
// (e.g. "eu" or "eu/line"), failing the test if it does not exist.
func RuleSetPath(tb testing.TB, name string) string {
	tb.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("cannot determine testutil source location")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "rulesets", filepath.FromSlash(name)+".json")
	if _, err := os.Stat(path); err != nil {
		tb.Fatalf("rule set %q not found: %v", name, err)
	}
	return path
}
