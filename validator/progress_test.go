package validator

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/theoremus-urban-solutions/netex-validator/types"
)

// eventRecorder collects progress events; it is safe for concurrent use.
type eventRecorder struct {
	mu     sync.Mutex
	events []types.ProgressEvent
}

func (r *eventRecorder) record(ev types.ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *eventRecorder) count(stage types.ProgressStage) int {
	n := 0
	for _, ev := range r.events {
		if ev.Stage == stage {
			n++
		}
	}
	return n
}

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dataset.zip")
	f, err := os.Create(path) //nolint:gosec // path is in t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProgressEventsSingleFile(t *testing.T) {
	rec := &eventRecorder{}
	options := testOptions(t).WithSkipSchema(true).WithProgress(rec.record)
	v, err := NewWithOptions(options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.ValidateContent([]byte(ruleSetTestXML), "progress.xml")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Rules) == 0 {
		t.Fatal("expected result.Rules to list the configured rules")
	}
	if got := rec.count(types.StageRules); got != len(result.Rules) {
		t.Errorf("got %d rule events, want one per rule (%d)", got, len(result.Rules))
	}
	last := rec.events[len(rec.events)-1]
	if last.Stage != types.StageFileDone || last.File != "progress.xml" {
		t.Errorf("last event = %+v, want file-done for progress.xml", last)
	}
	if rec.count(types.StageParsing) != 1 {
		t.Errorf("expected one parsing event")
	}
}

func TestRuleChecksSkippedOnSchemaErrors(t *testing.T) {
	v, err := NewWithOptions(testOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	// Not NetEX at all: the basic schema checks fail, so rule checks are skipped
	result, err := v.ValidateContent([]byte(`<?xml version="1.0"?><root/>`), "bad.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RuleChecksSkipped) != 1 || result.RuleChecksSkipped[0] != "bad.xml" {
		t.Errorf("RuleChecksSkipped = %v, want [bad.xml]", result.RuleChecksSkipped)
	}
}

func TestZipProgressFailedFilesAndCounts(t *testing.T) {
	zipPath := writeZip(t, map[string]string{
		"good.xml":    ruleSetTestXML,
		"broken.xml":  `<?xml version="1.0"?><PublicationDelivery xmlns="http://www.netex.org.uk/netex"><dataObjects><Line></dataObjects>`,
		"ignored.txt": "not xml",
	})

	rec := &eventRecorder{}
	options := testOptions(t).WithSkipSchema(true).WithProgress(rec.record)
	v, err := NewWithOptions(options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.ValidateZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesProcessed != 2 {
		t.Errorf("FilesProcessed = %d, want 2", result.FilesProcessed)
	}
	if len(result.FailedFiles) != 1 {
		t.Errorf("FailedFiles = %v, want one entry for broken.xml", result.FailedFiles)
	}
	if result.IsValid() {
		t.Error("a dataset with a file that could not be validated must not be valid")
	}

	var dataset *types.ProgressEvent
	for i := range rec.events {
		if rec.events[i].Stage == types.StageDataset {
			dataset = &rec.events[i]
		}
	}
	if dataset == nil || dataset.FilesTotal != 2 {
		t.Errorf("dataset event = %+v, want FilesTotal 2", dataset)
	}
	if got := rec.count(types.StageFileDone); got != 2 {
		t.Errorf("got %d file-done events, want 2", got)
	}
	if rec.count(types.StageReferences) != 1 {
		t.Error("expected a references event")
	}
	// --skip-schema is honoured for ZIP datasets
	if rec.count(types.StageSchema) != 0 {
		t.Error("schema validation ran although SkipSchema is set")
	}
}

func TestMaxFindingsMarksResultTruncated(t *testing.T) {
	files := map[string]string{}
	for _, name := range []string{"a.xml", "b.xml", "c.xml", "d.xml"} {
		files[name] = ruleSetTestXML
	}
	zipPath := writeZip(t, files)

	options := testOptions(t).WithSkipSchema(true).WithMaxFindings(1).WithConcurrentFiles(1)
	v, err := NewWithOptions(options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.ValidateZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || len(result.ValidationReportEntries) != 1 {
		t.Errorf("Truncated = %v with %d entries, want true with 1", result.Truncated, len(result.ValidationReportEntries))
	}

	single, err := v.ValidateContent([]byte(ruleSetTestXML), "one.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !single.Truncated {
		t.Error("single-file result should be marked truncated")
	}
}

func TestFailedFilesInJSONReport(t *testing.T) {
	zipPath := writeZip(t, map[string]string{
		"good.xml":   ruleSetTestXML,
		"broken.xml": `<?xml version="1.0"?><PublicationDelivery xmlns="http://www.netex.org.uk/netex"><dataObjects><Line></dataObjects>`,
	})
	v, err := NewWithOptions(testOptions(t).WithSkipSchema(true))
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.ValidateZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	out, err := result.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"failedFiles"`) || !strings.Contains(string(out), "broken.xml") {
		t.Errorf("JSON report does not mention the failed file:\n%s", out)
	}
}
