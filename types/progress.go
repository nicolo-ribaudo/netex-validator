package types

// ProgressStage identifies what a validation is currently doing.
type ProgressStage string

const (
	// StageDataset announces a dataset (ZIP) and how many files it contains.
	StageDataset ProgressStage = "dataset"
	// StageSchema means schema validation of a file started.
	StageSchema ProgressStage = "schema"
	// StageParsing means a file is being parsed for rule checks.
	StageParsing ProgressStage = "parsing"
	// StageRules reports rule checks of a file: RulesDone of RulesTotal are done.
	StageRules ProgressStage = "rules"
	// StageFileDone means a file is completely validated (or failed).
	StageFileDone ProgressStage = "file-done"
	// StageReferences means cross-file reference checks are running.
	StageReferences ProgressStage = "references"
)

// ProgressEvent reports validation progress. Events can arrive concurrently
// from several goroutines when files of a dataset are validated in parallel.
type ProgressEvent struct {
	Stage ProgressStage
	// File is the file the event is about (empty for dataset-wide stages).
	File string
	// FilesTotal is set for StageDataset.
	FilesTotal int
	// RulesDone and RulesTotal are set for StageRules.
	RulesDone  int
	RulesTotal int
}

// ProgressFunc receives progress events. It must be safe for concurrent use.
type ProgressFunc func(ProgressEvent)
