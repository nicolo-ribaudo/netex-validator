package engine

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/nicolo-ribaudo/netex-validator/interfaces"
	"github.com/nicolo-ribaudo/netex-validator/logging"
	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
	"github.com/nicolo-ribaudo/netex-validator/validation/ids"
)

// EnhancedNetexValidatorsRunner orchestrates NetEX validation with improved architecture
type EnhancedNetexValidatorsRunner struct {
	schemaValidator    interfaces.SchemaValidator
	xpathValidators    []interfaces.XPathValidator
	jaxbValidators     []interfaces.JAXBValidator
	datasetValidators  []interfaces.DatasetValidator
	idValidator        interfaces.IdValidator
	reportEntryFactory interfaces.ValidationReportEntryFactory
	maxFindings        int
	concurrentFiles    int
	progress           types.ProgressFunc
}

// EnhancedNetexValidatorsRunnerBuilder builds enhanced validator instances
type EnhancedNetexValidatorsRunnerBuilder struct {
	schemaValidator    interfaces.SchemaValidator
	xpathValidators    []interfaces.XPathValidator
	jaxbValidators     []interfaces.JAXBValidator
	datasetValidators  []interfaces.DatasetValidator
	idValidator        interfaces.IdValidator
	reportEntryFactory interfaces.ValidationReportEntryFactory
	maxFindings        int
	concurrentFiles    int
	progress           types.ProgressFunc
}

// NewEnhancedNetexValidatorsRunnerBuilder creates a new enhanced builder
func NewEnhancedNetexValidatorsRunnerBuilder() *EnhancedNetexValidatorsRunnerBuilder {
	return &EnhancedNetexValidatorsRunnerBuilder{
		xpathValidators:   make([]interfaces.XPathValidator, 0),
		jaxbValidators:    make([]interfaces.JAXBValidator, 0),
		datasetValidators: make([]interfaces.DatasetValidator, 0),
		// Don't set default factory - let Build() require it explicitly
	}
}

// WithSchemaValidator sets the schema validator
func (b *EnhancedNetexValidatorsRunnerBuilder) WithSchemaValidator(validator interfaces.SchemaValidator) *EnhancedNetexValidatorsRunnerBuilder {
	b.schemaValidator = validator
	return b
}

// WithXPathValidators sets the XPath validators
func (b *EnhancedNetexValidatorsRunnerBuilder) WithXPathValidators(validators []interfaces.XPathValidator) *EnhancedNetexValidatorsRunnerBuilder {
	b.xpathValidators = validators
	return b
}

// WithJAXBValidators sets the JAXB validators
func (b *EnhancedNetexValidatorsRunnerBuilder) WithJAXBValidators(validators []interfaces.JAXBValidator) *EnhancedNetexValidatorsRunnerBuilder {
	b.jaxbValidators = validators
	return b
}

// WithDatasetValidators sets the dataset validators
func (b *EnhancedNetexValidatorsRunnerBuilder) WithDatasetValidators(validators []interfaces.DatasetValidator) *EnhancedNetexValidatorsRunnerBuilder {
	b.datasetValidators = validators
	return b
}

// WithValidationReportEntryFactory sets the report entry factory
func (b *EnhancedNetexValidatorsRunnerBuilder) WithValidationReportEntryFactory(factory interfaces.ValidationReportEntryFactory) *EnhancedNetexValidatorsRunnerBuilder {
	b.reportEntryFactory = factory
	return b
}

// WithIdValidator sets the ID validator
func (b *EnhancedNetexValidatorsRunnerBuilder) WithIdValidator(validator interfaces.IdValidator) *EnhancedNetexValidatorsRunnerBuilder {
	b.idValidator = validator
	return b
}

// WithMaxFindings sets a cap on total findings to collect
func (b *EnhancedNetexValidatorsRunnerBuilder) WithMaxFindings(limit int) *EnhancedNetexValidatorsRunnerBuilder {
	b.maxFindings = limit
	return b
}

// WithConcurrentFiles sets the number of files to validate concurrently for ZIP datasets
func (b *EnhancedNetexValidatorsRunnerBuilder) WithConcurrentFiles(n int) *EnhancedNetexValidatorsRunnerBuilder {
	if n < 1 {
		n = 1
	}
	b.concurrentFiles = n
	return b
}

// WithProgress sets a callback that receives progress events
func (b *EnhancedNetexValidatorsRunnerBuilder) WithProgress(progress types.ProgressFunc) *EnhancedNetexValidatorsRunnerBuilder {
	b.progress = progress
	return b
}

// Build creates the EnhancedNetexValidatorsRunner
func (b *EnhancedNetexValidatorsRunnerBuilder) Build() (*EnhancedNetexValidatorsRunner, error) {
	if b.reportEntryFactory == nil {
		return nil, fmt.Errorf("validation report entry factory is required")
	}

	// Create default ID validator if none provided
	if b.idValidator == nil {
		// Use the NetEX ID validator with repository and extractor
		idRepo := ids.NewNetexIdRepository()
		idExtractor := ids.NewNetexIdExtractor()
		b.idValidator = ids.NewNetexIdValidator(idRepo, idExtractor)
	}

	return &EnhancedNetexValidatorsRunner{
		schemaValidator:    b.schemaValidator,
		xpathValidators:    b.xpathValidators,
		jaxbValidators:     b.jaxbValidators,
		datasetValidators:  b.datasetValidators,
		idValidator:        b.idValidator,
		reportEntryFactory: b.reportEntryFactory,
		maxFindings:        b.maxFindings,
		concurrentFiles:    b.concurrentFiles,
		progress:           b.progress,
	}, nil
}

// ValidateFile validates a single NetEX file (XML or ZIP)
func (r *EnhancedNetexValidatorsRunner) ValidateFile(filePath, codespace string, skipSchema, skipValidators bool) (*types.ValidationReport, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".zip") {
		return r.validateZipDataset(filePath, codespace, skipSchema, skipValidators)
	}
	return r.validateSingleXMLFile(filePath, codespace, skipSchema, skipValidators)
}

// emit sends a progress event if a progress callback is set
func (r *EnhancedNetexValidatorsRunner) emit(event types.ProgressEvent) {
	if r.progress != nil {
		r.progress(event)
	}
}

// ValidateContent validates NetEX content directly
func (r *EnhancedNetexValidatorsRunner) ValidateContent(fileName, codespace string, content []byte, skipSchema, skipValidators bool) (*types.ValidationReport, error) {
	startTime := time.Now()
	logger := logging.GetDefaultLogger().WithFile(fileName).WithValidation(generateReportID(fileName), codespace)

	logger.ValidationStart(fileName, codespace)
	defer r.emit(types.ProgressEvent{Stage: types.StageFileDone, File: fileName})
	defer func() {
		duration := time.Since(startTime)
		if duration > 5*time.Second {
			logger.PerformanceWarning("total_validation", duration, 5*time.Second)
		}
	}()

	reportID := generateReportID(fileName)
	report := types.NewValidationReport(codespace, reportID)

	// Step 1: Schema validation (blocking)
	if r.schemaValidator != nil && !skipSchema {
		r.emit(types.ProgressEvent{Stage: types.StageSchema, File: fileName})
		schemaStart := time.Now()
		logger.SchemaValidationStart(fileName)

		schemaContext := context.NewSchemaValidationContext(fileName, codespace, content)
		schemaIssues, err := r.schemaValidator.Validate(*schemaContext)

		schemaDuration := time.Since(schemaStart)
		logger.SchemaValidationComplete(fileName, schemaDuration, err == nil && len(schemaIssues) == 0)

		if err != nil {
			logger.ValidationError(fileName, err)
			return nil, fmt.Errorf("schema validation error: %w", err)
		}

		entries := r.convertIssuesToEntries(schemaIssues)
		r.addEntriesWithCap(report, entries)

		if len(schemaIssues) > 0 {
			logger.Info("Schema validation issues found", "count", len(schemaIssues))
		}

		if report.HasError() || r.reachedCap(report) {
			logger.Info("Stopping validation due to schema errors")
			// A cap stop is reported through report.Truncated instead
			if report.HasError() && !skipValidators && len(r.xpathValidators) > 0 {
				report.RuleChecksSkipped = append(report.RuleChecksSkipped, fileName)
			}
			return report, nil // Stop on schema errors
		}
	}

	if skipValidators || len(r.xpathValidators) == 0 {
		return report, nil
	}

	// Step 2: Prepare XPath validation context
	r.emit(types.ProgressEvent{Stage: types.StageParsing, File: fileName})
	xpathContext, err := r.prepareXPathValidationContext(reportID, codespace, fileName, content)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare XPath context: %w", err)
	}

	// Step 3: XPath validation (blocking)
	xpathStart := time.Now()
	logger.XPathValidationStart(fileName, len(r.xpathValidators))

	xpathIssues, err := r.runXPathValidators(*xpathContext)

	xpathDuration := time.Since(xpathStart)
	logger.XPathValidationComplete(fileName, xpathDuration, len(xpathIssues))

	if err != nil {
		logger.ValidationError(fileName, err)
		return nil, fmt.Errorf("XPath validation error: %w", err)
	}

	entries := r.convertIssuesToEntries(xpathIssues)
	r.addEntriesWithCap(report, entries)

	if len(xpathIssues) > 0 {
		logger.Info("XPath validation issues found", "count", len(xpathIssues))
	}

	if report.HasError() || r.reachedCap(report) {
		logger.Info("Stopping validation due to XPath errors")
		return report, nil // Stop on XPath errors
	}

	// Step 4: JAXB validation (non-blocking)
	if len(r.jaxbValidators) > 0 {
		jaxbContext := r.prepareJAXBValidationContext(reportID, codespace, fileName, content, xpathContext.LocalIDs)
		jaxbIssues, err := r.runJAXBValidators(*jaxbContext)
		if err != nil {
			return nil, fmt.Errorf("JAXB validation error: %w", err)
		}

		entries := r.convertIssuesToEntries(jaxbIssues)
		r.addEntriesWithCap(report, entries)
	}

	// Step 5: ID validation (extract IDs and references for later validation)
	if r.idValidator != nil {
		// Extract IDs and references from content
		if err := r.idValidator.ExtractIds(fileName, content); err != nil {
			logger.Warn("ID extraction failed", "error", err.Error())
		}
		if err := r.idValidator.ExtractReferences(fileName, content); err != nil {
			logger.Warn("Reference extraction failed", "error", err.Error())
		}
	}

	totalDuration := time.Since(startTime)
	issuesFound := len(report.ValidationReportEntries)
	logger.ValidationComplete(fileName, totalDuration, issuesFound, !report.HasError())

	return report, nil
}

// FinalizeIdValidation performs cross-file ID validation and returns issues
func (r *EnhancedNetexValidatorsRunner) FinalizeIdValidation() ([]types.ValidationIssue, error) {
	if r.idValidator == nil {
		return []types.ValidationIssue{}, nil // Return empty slice instead of nil
	}

	return r.idValidator.ValidateIds()
}

// validateZipDataset validates a ZIP dataset
func (r *EnhancedNetexValidatorsRunner) validateZipDataset(zipPath, codespace string, skipSchema, skipValidators bool) (*types.ValidationReport, error) {
	logger := logging.GetDefaultLogger().WithFile(zipPath).WithValidation(generateReportID(zipPath), codespace)
	report := types.NewValidationReport(codespace, generateReportID(zipPath))

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() { _ = zr.Close() }()

	// Count XML files first
	expectedFiles := 0
	for _, f := range zr.File {
		if strings.ToLower(filepath.Ext(f.Name)) == ".xml" {
			expectedFiles++
		}
	}

	if expectedFiles == 0 {
		logger.Info("No XML files found in ZIP", "file", zipPath)
		return report, nil
	}
	r.emit(types.ProgressEvent{Stage: types.StageDataset, File: zipPath, FilesTotal: expectedFiles})

	// Every XML entry produces exactly one outcome, including entries that
	// cannot be read and validations that panic, so the collector never waits
	// for an outcome that will not come.
	type job struct {
		name    string
		content []byte
		err     error // set if the entry could not be read
	}
	type outcome struct {
		report *types.ValidationReport
		err    error
	}

	jobs := make(chan job)
	outcomes := make(chan outcome, expectedFiles) // never blocks workers
	done := make(chan struct{})                   // closed when collection stops early

	validate := func(j job) (o outcome) {
		defer func() {
			if rec := recover(); rec != nil {
				o = outcome{err: fmt.Errorf("%s: panic during validation: %v", j.name, rec)}
			}
		}()
		if j.err != nil {
			r.emit(types.ProgressEvent{Stage: types.StageFileDone, File: j.name})
			return outcome{err: fmt.Errorf("%s: %w", j.name, j.err)}
		}
		subReport, err := r.ValidateContent(j.name, codespace, j.content, skipSchema, skipValidators)
		if err != nil {
			return outcome{err: fmt.Errorf("%s: %w", j.name, err)}
		}
		return outcome{report: subReport}
	}

	workerCount := r.concurrentFiles
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > expectedFiles {
		workerCount = expectedFiles
	}

	var workers sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := range jobs {
				outcomes <- validate(j)
			}
		}()
	}

	// Enqueue xml entries, stopping early if collection stops
	go func() {
		defer close(jobs)
		for _, f := range zr.File {
			if strings.ToLower(filepath.Ext(f.Name)) != ".xml" {
				continue
			}
			j := job{name: f.Name}
			if rc, err := f.Open(); err != nil {
				j.err = fmt.Errorf("failed to open zip entry: %w", err)
			} else {
				j.content, err = io.ReadAll(rc)
				_ = rc.Close()
				if err != nil {
					j.err = fmt.Errorf("failed to read zip entry: %w", err)
				}
			}
			select {
			case jobs <- j:
			case <-done:
				return
			}
		}
	}()

	// Collect results
	for i := 0; i < expectedFiles; i++ {
		o := <-outcomes
		if o.err != nil {
			logger.ValidationError(zipPath, o.err)
			report.FailedFiles = append(report.FailedFiles, o.err.Error())
			continue
		}
		report.RuleChecksSkipped = append(report.RuleChecksSkipped, o.report.RuleChecksSkipped...)
		report.Truncated = report.Truncated || o.report.Truncated
		if entries := o.report.ValidationReportEntries; len(entries) > 0 {
			r.addEntriesWithCap(report, entries)
		}
		if r.reachedCap(report) {
			report.Truncated = true
			break
		}
	}
	// Stop handing out files and let the validations in progress finish, so
	// nothing runs concurrently with the cross-file checks below
	close(done)
	workers.Wait()

	// Cross-file ID validation at the end
	r.emit(types.ProgressEvent{Stage: types.StageReferences, File: zipPath})
	if idIssues, err := r.FinalizeIdValidation(); err == nil && len(idIssues) > 0 {
		r.addEntriesWithCap(report, r.convertIssuesToEntries(idIssues))
	} else if err != nil {
		logger.ValidationError(zipPath, fmt.Errorf("ID finalization failed: %w", err))
	}

	return report, nil
}

// validateSingleXMLFile validates a single XML file
func (r *EnhancedNetexValidatorsRunner) validateSingleXMLFile(filePath, codespace string, skipSchema, skipValidators bool) (*types.ValidationReport, error) {
	// Validate file path to prevent path traversal
	if !filepath.IsAbs(filePath) && strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("invalid file path: %s", filePath)
	}

	// Read file content and delegate to ValidateContent
	data, err := os.ReadFile(filePath) //nolint:gosec // Path is validated above
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return r.ValidateContent(filepath.Base(filePath), codespace, data, skipSchema, skipValidators)
}

// prepareXPathValidationContext prepares the XPath validation context
func (r *EnhancedNetexValidatorsRunner) prepareXPathValidationContext(
	validationReportID, codespace, filename string,
	fileContent []byte,
) (*context.XPathValidationContext, error) {

	// Parse XML document using xmlquery
	document, err := xmlquery.Parse(bytes.NewReader(fileContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Use the idValidator to extract IDs and references
	// Create a temporary NetEX ID extractor
	extractor := ids.NewNetexIdExtractor()

	// Extract local IDs
	localIDsList, err := extractor.ExtractIds(filename, fileContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract local IDs: %w", err)
	}

	// Convert to map
	localIDsMap := make(map[string]types.IdVersion)
	for _, id := range localIDsList {
		localIDsMap[id.ID] = id
	}

	// Extract local references
	localRefs, err := extractor.ExtractReferences(filename, fileContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract local references: %w", err)
	}

	return context.NewXPathValidationContext(filename, codespace, validationReportID, document, localIDsMap, localRefs), nil
}

// prepareJAXBValidationContext prepares the JAXB validation context
func (r *EnhancedNetexValidatorsRunner) prepareJAXBValidationContext(
	validationReportID, codespace, filename string,
	fileContent []byte,
	localIDMap map[string]types.IdVersion,
) *context.JAXBValidationContext {

	return context.NewJAXBValidationContext(validationReportID, codespace, filename, localIDMap)
}

// runXPathValidators executes XPath validators with parallel rule execution
func (r *EnhancedNetexValidatorsRunner) runXPathValidators(ctx context.XPathValidationContext) ([]types.ValidationIssue, error) {
	if len(r.xpathValidators) == 0 {
		return nil, nil
	}

	// For small numbers of validators, run sequentially to avoid overhead
	if len(r.xpathValidators) == 1 {
		return r.xpathValidators[0].Validate(ctx)
	}

	// Use parallel execution for multiple validators
	return r.runXPathValidatorsParallel(ctx)
}

// runXPathValidatorsParallel executes XPath validators in parallel
func (r *EnhancedNetexValidatorsRunner) runXPathValidatorsParallel(ctx context.XPathValidationContext) ([]types.ValidationIssue, error) {
	type validatorResult struct {
		issues []types.ValidationIssue
		err    error
		index  int
	}

	// Create channels for communication
	results := make(chan validatorResult, len(r.xpathValidators))

	// Launch goroutines for each validator
	for i, validator := range r.xpathValidators {
		go func(idx int, v interfaces.XPathValidator) {
			defer func() {
				// Recover from panics in individual validators
				if r := recover(); r != nil {
					results <- validatorResult{
						issues: nil,
						err:    fmt.Errorf("validator panic at index %d: %v", idx, r),
						index:  idx,
					}
				}
			}()

			issues, err := v.Validate(ctx)
			results <- validatorResult{
				issues: issues,
				err:    err,
				index:  idx,
			}
		}(i, validator)
	}

	// Collect results
	var allIssues []types.ValidationIssue
	var errors []error

	for i := 0; i < len(r.xpathValidators); i++ {
		result := <-results

		if result.err != nil {
			errors = append(errors, fmt.Errorf("validator %d error: %w", result.index, result.err))
			continue
		}

		if len(result.issues) > 0 {
			allIssues = append(allIssues, result.issues...)
		}

		// Check if we've exceeded the max findings limit to short-circuit
		if r.maxFindings > 0 && len(allIssues) >= r.maxFindings {
			break
		}
	}

	// If we have errors, return the first one
	if len(errors) > 0 {
		return allIssues, errors[0]
	}

	return allIssues, nil
}

// runJAXBValidators executes JAXB validators with parallel execution
func (r *EnhancedNetexValidatorsRunner) runJAXBValidators(ctx context.JAXBValidationContext) ([]types.ValidationIssue, error) {
	if len(r.jaxbValidators) == 0 {
		return nil, nil
	}

	// For small numbers of validators, run sequentially to avoid overhead
	if len(r.jaxbValidators) == 1 {
		return r.jaxbValidators[0].Validate(ctx)
	}

	// Use parallel execution for multiple validators
	return r.runJAXBValidatorsParallel(ctx)
}

// runJAXBValidatorsParallel executes JAXB validators in parallel
func (r *EnhancedNetexValidatorsRunner) runJAXBValidatorsParallel(ctx context.JAXBValidationContext) ([]types.ValidationIssue, error) {
	type validatorResult struct {
		issues []types.ValidationIssue
		err    error
		index  int
	}

	// Create channels for communication
	results := make(chan validatorResult, len(r.jaxbValidators))

	// Launch goroutines for each validator
	for i, validator := range r.jaxbValidators {
		go func(idx int, v interfaces.JAXBValidator) {
			defer func() {
				// Recover from panics in individual validators
				if r := recover(); r != nil {
					results <- validatorResult{
						issues: nil,
						err:    fmt.Errorf("JAXB validator panic at index %d: %v", idx, r),
						index:  idx,
					}
				}
			}()

			issues, err := v.Validate(ctx)
			results <- validatorResult{
				issues: issues,
				err:    err,
				index:  idx,
			}
		}(i, validator)
	}

	// Collect results
	var allIssues []types.ValidationIssue
	var errors []error

	for i := 0; i < len(r.jaxbValidators); i++ {
		result := <-results

		if result.err != nil {
			errors = append(errors, fmt.Errorf("JAXB validator %d error: %w", result.index, result.err))
			continue
		}

		if len(result.issues) > 0 {
			allIssues = append(allIssues, result.issues...)
		}

		// Check if we've exceeded the max findings limit to short-circuit
		if r.maxFindings > 0 && len(allIssues) >= r.maxFindings {
			break
		}
	}

	// If we have errors, return the first one
	if len(errors) > 0 {
		return allIssues, errors[0]
	}

	return allIssues, nil
}

// convertIssuesToEntries converts validation issues to report entries
func (r *EnhancedNetexValidatorsRunner) convertIssuesToEntries(issues []types.ValidationIssue) []types.ValidationReportEntry {
	var entries []types.ValidationReportEntry
	for _, issue := range issues {
		entry := r.reportEntryFactory.CreateValidationReportEntry(issue)
		entries = append(entries, entry)
	}
	return entries
}

// addEntriesWithCap adds entries to report respecting maxFindings cap
func (r *EnhancedNetexValidatorsRunner) addEntriesWithCap(report *types.ValidationReport, entries []types.ValidationReportEntry) {
	if r.maxFindings <= 0 {
		report.AddAllValidationReportEntries(entries)
		return
	}
	// Once the cap is reached, validation stops early, so later findings are lost
	defer func() { report.Truncated = report.Truncated || r.reachedCap(report) }()
	remaining := r.maxFindings - len(report.ValidationReportEntries)
	if remaining <= 0 {
		return
	}
	if len(entries) <= remaining {
		report.AddAllValidationReportEntries(entries)
		return
	}
	report.AddAllValidationReportEntries(entries[:remaining])
}

// reachedCap returns true if max findings cap has been reached
func (r *EnhancedNetexValidatorsRunner) reachedCap(report *types.ValidationReport) bool {
	return r.maxFindings > 0 && len(report.ValidationReportEntries) >= r.maxFindings
}

// generateReportID generates a report ID from filename
func generateReportID(fileName string) string {
	// Remove extension and use as report ID
	if idx := strings.LastIndex(fileName, "."); idx > 0 {
		return fileName[:idx]
	}
	return fileName
}

// DefaultValidationReportEntryFactory is the default implementation
type DefaultValidationReportEntryFactory struct{}

// NewDefaultValidationReportEntryFactory creates a new default factory
func NewDefaultValidationReportEntryFactory() interfaces.ValidationReportEntryFactory {
	return &DefaultValidationReportEntryFactory{}
}

// CreateValidationReportEntry creates a validation report entry from an issue
func (f *DefaultValidationReportEntryFactory) CreateValidationReportEntry(issue types.ValidationIssue) types.ValidationReportEntry {
	return types.ValidationReportEntry{
		RuleID:   issue.Rule.Code,
		Name:     issue.Rule.Name,
		Message:  issue.Message,
		Severity: issue.Rule.Severity,
		FileName: issue.Location.FileName,
		Location: issue.Location,
	}
}

// TemplateValidationReportEntry creates a template entry from a rule
func (f *DefaultValidationReportEntryFactory) TemplateValidationReportEntry(rule types.ValidationRule) types.ValidationReportEntry {
	return types.ValidationReportEntry{
		RuleID:   rule.Code,
		Name:     rule.Name,
		Message:  rule.Message,
		Severity: rule.Severity,
	}
}
