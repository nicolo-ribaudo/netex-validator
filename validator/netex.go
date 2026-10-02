// Package netexvalidator provides a comprehensive NetEX validation library for Go applications.
//
// This package validates NetEX (Network Exchange) files against the EU NeTEx Profile,
// covering schema validation, XPath business rules loaded from JSON rule files,
// and cross-file ID validation.
//
// Basic usage:
//
//	import "github.com/theoremus-urban-solutions/netex-validator"
//
//	// Simple validation
//	options := netexvalidator.DefaultValidationOptions().
//		WithCodespace("MyCodespace").
//		WithRuleSet("", "rulesets/eu.json")
//	result, err := netexvalidator.ValidateFile("data.xml", options)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Check results
//	if result.IsValid() {
//		fmt.Println("Validation passed!")
//	} else {
//		fmt.Printf("Found %d issues\n", len(result.ValidationReportEntries))
//	}
//
//	// Generate HTML report
//	htmlReport, err := result.ToHTML()
//	if err != nil {
//		log.Fatal(err)
//	}
//
// The library supports multiple validation modes:
//   - Single XML file validation
//   - ZIP dataset validation (multiple files)
//   - In-memory content validation
//   - Business rules from JSON rule files (see ValidationOptions.RuleSets)
//   - Rule customization via YAML
//
// Output formats include JSON, HTML (with interactive interface), and plain text.
package validator

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/xmlquery"
	antxpath "github.com/antchfx/xpath"
	"github.com/theoremus-urban-solutions/netex-validator/config"
	"github.com/theoremus-urban-solutions/netex-validator/interfaces"
	"github.com/theoremus-urban-solutions/netex-validator/logging"
	"github.com/theoremus-urban-solutions/netex-validator/rules"
	"github.com/theoremus-urban-solutions/netex-validator/types"
	"github.com/theoremus-urban-solutions/netex-validator/utils"
	"github.com/theoremus-urban-solutions/netex-validator/validation/context"
	"github.com/theoremus-urban-solutions/netex-validator/validation/engine"
	"github.com/theoremus-urban-solutions/netex-validator/validation/ids"
	xsdpkg "github.com/theoremus-urban-solutions/netex-validator/validation/schema"
)

// NetexSchemaValidatorAdapter adapts XSDValidator to SchemaValidator interface
type NetexSchemaValidatorAdapter struct {
	xsdValidator *xsdpkg.XSDValidator
	maxFindings  int
}

// NewNetexSchemaValidatorAdapter creates a new schema validator adapter
func NewNetexSchemaValidatorAdapter(xsdValidator *xsdpkg.XSDValidator, maxFindings int) *NetexSchemaValidatorAdapter {
	return &NetexSchemaValidatorAdapter{
		xsdValidator: xsdValidator,
		maxFindings:  maxFindings,
	}
}

// Validate implements the SchemaValidator interface
func (a *NetexSchemaValidatorAdapter) Validate(ctx context.SchemaValidationContext) ([]types.ValidationIssue, error) {
	validationErrors, err := a.xsdValidator.ValidateXML(ctx.FileContent, ctx.FileName)
	if err != nil {
		return nil, err
	}

	// Convert validation errors to validation issues
	issues := make([]types.ValidationIssue, 0, len(validationErrors))
	for _, verr := range validationErrors {
		if len(issues) >= a.maxFindings && a.maxFindings > 0 {
			break
		}

		issue := types.ValidationIssue{
			Rule: types.ValidationRule{
				Code:     "SCHEMA_ERROR",
				Name:     "Schema validation error",
				Message:  verr.Message,
				Severity: types.ERROR,
			},
			Location: types.DataLocation{
				FileName:   ctx.FileName,
				LineNumber: verr.Line,
			},
			Message: verr.Message,
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// GetRules implements the SchemaValidator interface
func (a *NetexSchemaValidatorAdapter) GetRules() []types.ValidationRule {
	// Schema validation has basic structural rules
	return []types.ValidationRule{
		{
			Code:     "SCHEMA_ERROR",
			Name:     "Schema validation error",
			Message:  "XML content does not conform to NetEX schema",
			Severity: types.ERROR,
		},
	}
}

// NetexValidator is the main library interface for NetEX validation.
// It encapsulates the validation configuration and provides methods for validating
// NetEX files and content against the EU NeTEx Profile.
//
// Use New() or NewWithOptions() to create a validator instance, then call
// ValidateFile(), ValidateContent(), or ValidateZip() to perform validation.
type NetexValidator struct {
	config          *config.ValidatorConfig
	runner          *engine.EnhancedNetexValidatorsRunner
	codespace       string
	validationCache utils.ValidationCache
	options         *ValidationOptions
}

// New creates a new NetexValidator instance with default configuration.
//
// The default configuration includes:
//   - No XPath business rules (use NewWithOptions with RuleSets to add some)
//   - Schema validation enabled
//   - Maximum 100 schema errors reported
//   - Verbose mode disabled
//
// Returns an error if the validator cannot be initialized.
//
// Example:
//
//	validator, err := netexvalidator.New()
//	if err != nil {
//		log.Fatal(err)
//	}
//	result, err := validator.ValidateFile("data.xml")
func New() (*NetexValidator, error) {
	return NewWithOptions(DefaultValidationOptions())
}

// NewWithOptions creates a new NetexValidator instance with custom options.
//
// This allows you to customize validation behavior such as:
//   - Skipping schema validation for faster processing
//   - Setting custom codespace for validation context
//   - Selecting JSON rule sets and configuring rule overrides via YAML files
//   - Enabling verbose output for debugging
//
// Parameters:
//   - opts: ValidationOptions containing configuration settings
//
// Returns a configured NetexValidator instance or an error if initialization fails.
//
// Example:
//
//	options := netexvalidator.DefaultValidationOptions().
//		WithCodespace("MyCodespace").
//		WithRuleSet("", "rulesets/eu.json").
//		WithSkipSchema(true)
//	validator, err := netexvalidator.NewWithOptions(options)
func NewWithOptions(opts *ValidationOptions) (*NetexValidator, error) {
	var cfg *config.ValidatorConfig
	var err error

	// Set up logging based on options
	logger := opts.GetLogger()
	logging.SetDefaultLogger(logger)

	logger.Info("Initializing NetEX validator",
		"codespace", opts.Codespace,
		"skip_schema", opts.SkipSchema,
		"skip_validators", opts.SkipValidators,
		"verbose", opts.Verbose,
	)

	// Load configuration
	if opts.ConfigFile != "" {
		logger.Info("Loading configuration from file", "config_file", opts.ConfigFile)
		cfg, err = config.LoadConfig(opts.ConfigFile)
		if err != nil {
			logger.Error("Failed to load configuration", "error", err.Error(), "config_file", opts.ConfigFile)
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		logger.Debug("Using default configuration")
		cfg = config.DefaultConfig()
	}

	// Apply option overrides
	if opts.MaxSchemaErrors > 0 {
		cfg.Validator.MaxSchemaErrors = opts.MaxSchemaErrors
		logger.Debug("Applied max schema errors override", "max_errors", opts.MaxSchemaErrors)
	}

	// Initialize validation cache if enabled
	var validationCache utils.ValidationCache
	if opts.EnableValidationCache {
		maxEntries := opts.CacheMaxEntries
		if maxEntries <= 0 {
			maxEntries = 1000
		}
		maxMemoryBytes := int64(opts.CacheMaxMemoryMB) << 20 // Convert MB to bytes
		if maxMemoryBytes <= 0 {
			maxMemoryBytes = 50 << 20 // 50MB default
		}

		cacheOpts := &utils.MemoryCacheOptions{
			MaxEntries: maxEntries,
			MaxBytes:   maxMemoryBytes,
		}
		memoryCache := utils.NewMemoryValidationCache(cacheOpts)
		validationCache = memoryCache
		logger.Info("Memory validation cache enabled", "max_entries", maxEntries, "max_memory_mb", opts.CacheMaxMemoryMB, "ttl_hours", opts.CacheTTLHours)
	}

	// Create validator
	validator := &NetexValidator{
		config:          cfg,
		codespace:       opts.Codespace,
		validationCache: validationCache,
		options:         opts,
	}

	// Initialize runner
	logger.Debug("Initializing validation runner")
	err = validator.initializeRunner(opts)
	if err != nil {
		logger.Error("Failed to initialize validator", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	logger.Info("NetEX validator initialized successfully")
	return validator, nil
}

// ValidateFile validates a single NetEX file using the provided options.
//
// This is a convenience function that creates a validator instance and validates
// the specified file in one call. For multiple validations, consider creating
// a validator instance with New() or NewWithOptions() and reusing it.
//
// Parameters:
//   - filePath: Path to the NetEX XML file to validate
//   - options: Validation configuration options
//
// Returns:
//   - ValidationResult containing all validation issues found
//   - Error if file cannot be read or validation fails
//
// Example:
//
//	options := netexvalidator.DefaultValidationOptions().WithCodespace("NO").WithRuleSet("", "rulesets/eu.json")
//	result, err := netexvalidator.ValidateFile("timetable.xml", options)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Found %d validation issues\n", len(result.ValidationReportEntries))
func ValidateFile(filePath string, options *ValidationOptions) (*ValidationResult, error) {
	validator, err := NewWithOptions(options)
	if err != nil {
		return nil, err
	}
	return validator.ValidateFile(filePath)
}

// ValidateContent validates NetEX content from memory.
//
// This function validates XML content that is already loaded in memory,
// useful for processing data from APIs, databases, or other sources
// without writing to disk first.
//
// Parameters:
//   - content: NetEX XML content as byte slice
//   - filename: Logical filename for reporting (used in validation results)
//   - options: Validation configuration options
//
// Returns:
//   - ValidationResult containing all validation issues found
//   - Error if content cannot be parsed or validation fails
//
// Example:
//
//	xmlContent := []byte(`<?xml version="1.0"?>...`)
//	options := netexvalidator.DefaultValidationOptions().WithCodespace("SE").WithRuleSet("", "rulesets/eu.json")
//	result, err := netexvalidator.ValidateContent(xmlContent, "in-memory.xml", options)
func ValidateContent(content []byte, filename string, options *ValidationOptions) (*ValidationResult, error) {
	validator, err := NewWithOptions(options)
	if err != nil {
		return nil, err
	}
	return validator.ValidateContent(content, filename)
}

// ValidateZip validates a ZIP dataset containing multiple NetEX files.
//
// NetEX datasets are often distributed as ZIP files containing multiple
// XML files with shared data and cross-references. This function validates
// all XML files in the ZIP and performs cross-file ID validation.
//
// Parameters:
//   - zipPath: Path to the ZIP file containing NetEX XML files
//   - options: Validation configuration options
//
// Returns:
//   - ValidationResult with combined results from all files in the ZIP
//   - Error if ZIP cannot be read or validation fails
//
// Example:
//
//	options := netexvalidator.DefaultValidationOptions().WithCodespace("DK").WithRuleSet("", "rulesets/eu.json")
//	result, err := netexvalidator.ValidateZip("dataset.zip", options)
//	if err != nil {
//		log.Fatal(err)
//	}
//	summary := result.Summary()
//	fmt.Printf("Validated %d files, found %d issues\n",
//		summary.FilesProcessed, summary.TotalIssues)
func ValidateZip(zipPath string, options *ValidationOptions) (*ValidationResult, error) {
	validator, err := NewWithOptions(options)
	if err != nil {
		return nil, err
	}
	return validator.ValidateZip(zipPath)
}

// ValidateFile validates a single NetEX file using this validator instance
func (v *NetexValidator) ValidateFile(filePath string) (*ValidationResult, error) {
	startTime := time.Now()

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &ValidationResult{
			Error:        fmt.Sprintf("file does not exist: %s", filePath),
			CreationDate: time.Now(),
		}, nil
	}

	// Clean the file path and check if it exists (allows legitimate relative paths)
	cleanPath := filepath.Clean(filePath)
	if !filepath.IsAbs(cleanPath) {
		// For relative paths, resolve them relative to current working directory
		var err error
		cleanPath, err = filepath.Abs(cleanPath)
		if err != nil {
			return &ValidationResult{
				Error:        fmt.Sprintf("failed to resolve file path %s: %v", filePath, err),
				CreationDate: time.Now(),
			}, nil
		}
	}

	// Read file content using the cleaned path
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return &ValidationResult{
			Error:        fmt.Sprintf("failed to read file: %v", err),
			CreationDate: time.Now(),
		}, nil
	}

	result, err := v.ValidateContent(content, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	result.ProcessingTime = time.Since(startTime)
	result.FilesProcessed = 1

	return result, nil
}

// ValidateContent validates NetEX content from memory using this validator instance
func (v *NetexValidator) ValidateContent(content []byte, filename string) (*ValidationResult, error) {
	startTime := time.Now()
	return v.validateContentWithCaching(content, filename, startTime)
}

// ValidateZip validates a ZIP dataset using this validator instance
func (v *NetexValidator) ValidateZip(zipPath string) (*ValidationResult, error) {
	startTime := time.Now()

	// Check if ZIP file exists
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return &ValidationResult{
			Error:        fmt.Sprintf("ZIP file does not exist: %s", zipPath),
			CreationDate: time.Now(),
		}, nil
	}

	// Extract raw content from ZIP for statistics before validation
	rawContents, err := v.extractZipContents(zipPath)
	if err != nil {
		return &ValidationResult{
			Error:        fmt.Sprintf("failed to extract ZIP contents: %v", err),
			CreationDate: time.Now(),
		}, nil
	}

	// Use the validator's built-in ZIP support
	report, err := v.runner.ValidateFile(zipPath, v.codespace, false, false)
	if err != nil {
		return &ValidationResult{
			Error:        fmt.Sprintf("ZIP validation failed: %v", err),
			CreationDate: time.Now(),
		}, nil
	}

	// Convert to result format
	result := v.createValidationResultFromReport(report, filepath.Base(zipPath), startTime)

	// Store raw content for statistics extraction
	for fileName, content := range rawContents {
		result.SetRawContent(fileName, content)
	}

	return result, nil
}

// extractZipContents extracts raw XML content from ZIP files for statistics
func (v *NetexValidator) extractZipContents(zipPath string) (map[string][]byte, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() { _ = zr.Close() }()

	contents := make(map[string][]byte)

	for _, f := range zr.File {
		// Only process XML files
		if strings.ToLower(filepath.Ext(f.Name)) != ".xml" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue // Skip files that can't be opened
		}

		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			continue // Skip files that can't be read
		}

		contents[filepath.Base(f.Name)] = content
	}

	return contents, nil
}

// ValidateReader validates NetEX content from an io.Reader
func (v *NetexValidator) ValidateReader(reader io.Reader, filename string) (*ValidationResult, error) {
	startTime := time.Now()

	// Read all content for validation
	content, err := io.ReadAll(reader)
	if err != nil {
		return &ValidationResult{
			Error:        fmt.Sprintf("failed to read content: %v", err),
			CreationDate: time.Now(),
		}, nil
	}

	return v.validateContentWithCaching(content, filename, startTime)
}

// validateContentWithCaching validates content with caching support
func (v *NetexValidator) validateContentWithCaching(content []byte, filename string, startTime time.Time) (*ValidationResult, error) {
	// Calculate file hash for caching
	var fileHash string
	var cacheHit bool

	if v.validationCache != nil {
		fileHash = utils.CalculateFileHash(content)

		// Check cache first
		if cachedInterface, found := v.validationCache.Get(fileHash); found {
			if cachedResult, ok := cachedInterface.(*ValidationResult); ok {
				// Create a copy to avoid concurrent modification of cached result
				resultCopy := *cachedResult
				resultCopy.ProcessingTime = time.Since(startTime)
				resultCopy.CacheHit = true
				resultCopy.FileHash = fileHash

				return &resultCopy, nil
			}
		}
	}

	// Perform validation
	report, err := v.runner.ValidateContent(filename, v.codespace, content, v.options.SkipSchema, v.options.SkipValidators)
	if err != nil {
		return &ValidationResult{
			Error:        fmt.Sprintf("validation failed: %v", err),
			CreationDate: time.Now(),
			FileHash:     fileHash,
		}, nil
	}

	// Convert to result format
	result := v.createValidationResultFromReport(report, filename, startTime)
	result.FilesProcessed = 1
	result.CacheHit = cacheHit
	result.FileHash = fileHash

	// Store raw content for statistics extraction
	result.SetRawContent(filename, content)

	// Cache the result if caching is enabled
	if v.validationCache != nil && fileHash != "" {
		ttl := time.Duration(v.options.CacheTTLHours) * time.Hour
		if err := v.validationCache.Set(fileHash, result, ttl); err != nil {
			// Log warning but don't fail validation
			logging.GetDefaultLogger().Warn("Failed to cache validation result", "error", err)
		}
	}

	return result, nil
}

// initializeRunner sets up the validation runner with rules
func (v *NetexValidator) initializeRunner(opts *ValidationOptions) error {
	// Create builder
	builder := engine.NewEnhancedNetexValidatorsRunnerBuilder()

	// Add schema validator if not skipped
	if !opts.SkipSchema {
		// Initialize schema validator honoring options
		xsdOpts := xsdpkg.DefaultXSDValidationOptions()
		xsdOpts.AllowNetworkDownload = opts.AllowSchemaNetwork
		if opts.SchemaCacheDir != "" {
			xsdOpts.CacheDirectory = opts.SchemaCacheDir
		}
		if opts.SchemaTimeoutSeconds > 0 {
			xsdOpts.HttpTimeoutSeconds = opts.SchemaTimeoutSeconds
		}
		// experimental libxml2 backend
		if opts.UseLibxml2XSD {
			xsdOpts.UseLibxml2 = true
		}
		xsdValidator, err := xsdpkg.NewXSDValidator(xsdOpts)
		if err != nil {
			return fmt.Errorf("failed to create XSD validator: %w", err)
		}
		schemaValidator := NewNetexSchemaValidatorAdapter(xsdValidator, v.config.Validator.MaxSchemaErrors)
		builder = builder.WithSchemaValidator(schemaValidator)
	}

	// Add XPath validators if not skipped
	if !opts.SkipValidators {
		logger := opts.GetLogger()
		if len(opts.RuleSets) == 0 && len(v.config.GetCustomRules()) == 0 {
			logger.Warn("No rule sets configured; XPath business rules will not run")
		}
		loaded, err := rules.LoadRuleSets(opts.RuleSets)
		if err != nil {
			return fmt.Errorf("failed to load rule sets: %w", err)
		}
		warnUnmatchedRuleConfig(logger, v.config, opts, loaded)
		// Create rule registry and get enabled rules
		ruleRegistry := rules.NewRuleRegistry(v.config, loaded)
		enabled := ruleRegistry.GetEnabledRules()
		// Apply in-memory rule overrides from options (in addition to config)
		if len(opts.RuleOverrides) > 0 {
			filtered := make([]rules.Rule, 0, len(enabled))
			for _, r := range enabled {
				if enabledFlag, ok := opts.RuleOverrides[r.Code]; ok {
					if !enabledFlag {
						continue
					}
				}
				filtered = append(filtered, r)
			}
			enabled = filtered
		}
		if len(opts.SeverityOverrides) > 0 {
			for i := range enabled {
				if sev, ok := opts.SeverityOverrides[enabled[i].Code]; ok {
					enabled[i].Severity = sev
				}
			}
		}
		// Wrap rules as XPathValidationRule implementations
		xrules := make([]utils.XPathValidationRule, 0, len(enabled))
		for _, r := range enabled {
			xrules = append(xrules, NewSimpleXPathRule(r))
		}
		if len(xrules) > 0 {
			xpathValidator := utils.NewXPathRuleValidator(xrules)
			builder = builder.WithXPathValidators([]interfaces.XPathValidator{xpathValidator})
		}
	}

	// Add ID validator
	idRepo := ids.NewNetexIdRepository()
	idExtractor := ids.NewNetexIdExtractor()
	idValidator := ids.NewNetexIdValidator(idRepo, idExtractor)
	builder = builder.WithIdValidator(idValidator)

	// Apply max findings if set
	if opts.MaxFindings > 0 {
		builder = builder.WithMaxFindings(opts.MaxFindings)
	}

	// Apply concurrency from config
	concurrent := v.config.Validator.ConcurrentFiles
	if opts.ConcurrentFiles > 0 {
		concurrent = opts.ConcurrentFiles
	}
	if concurrent > 0 {
		builder = builder.WithConcurrentFiles(concurrent)
	}

	// Set validation report entry factory
	builder = builder.WithValidationReportEntryFactory(engine.NewDefaultValidationReportEntryFactory())

	// Build runner
	runner, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build validator runner: %w", err)
	}
	v.runner = runner

	return nil
}

// createValidationResultFromReport converts a validation report to library result format
func (v *NetexValidator) createValidationResultFromReport(report *types.ValidationReport, reportID string, startTime time.Time) *ValidationResult {
	// Convert entries to library format
	var resultEntries []ValidationReportEntry
	for _, entry := range report.ValidationReportEntries {
		resultEntries = append(resultEntries, ValidationReportEntry{
			RuleID:   entry.RuleID,
			Name:     entry.Name,
			Message:  entry.Message,
			Severity: entry.Severity,
			FileName: entry.FileName,
			Location: ValidationReportLocation{
				FileName:   entry.Location.FileName,
				LineNumber: entry.Location.LineNumber,
				XPath:      entry.Location.XPath,
				ElementID:  entry.Location.ElementID,
			},
		})
	}

	// Convert int64 map to int map
	entriesPerRule := make(map[string]int)
	for k, v := range report.NumberOfValidationEntriesPerRule {
		entriesPerRule[k] = int(v)
	}

	return &ValidationResult{
		Codespace:                        report.Codespace,
		ValidationReportID:               report.ValidationReportID,
		CreationDate:                     report.CreationDate,
		ValidationReportEntries:          resultEntries,
		NumberOfValidationEntriesPerRule: entriesPerRule,
		ProcessingTime:                   time.Since(startTime),
	}
}

// SimpleXPathRule is a minimal adapter to execute a rule's XPath and produce issues
type SimpleXPathRule struct {
	rule     rules.Rule
	compiled *antxpath.Expr
	mu       sync.Mutex // Protects compiled XPath expression
}

// NewSimpleXPathRule creates a new adapter from a rules.Rule
func NewSimpleXPathRule(rule rules.Rule) *SimpleXPathRule {
	r := &SimpleXPathRule{rule: rule}
	if rule.XPath != "" {
		// Check for unsupported XPath functions before compilation
		if !r.hasUnsupportedFunctions(rule.XPath) {
			if expr, err := antxpath.Compile(rule.XPath); err == nil {
				r.compiled = expr
			}
		}
	}
	return r
}

// Validate executes the XPath and returns issues for matched nodes
func (r *SimpleXPathRule) Validate(ctx context.XPathValidationContext) ([]types.ValidationIssue, error) {
	var issues []types.ValidationIssue
	if ctx.Document == nil || r.rule.XPath == "" {
		return issues, nil
	}

	// Evaluate XPath with proper error handling
	var nodes []*xmlquery.Node
	var evalErr error

	// Use a recovery mechanism to handle unsupported XPath functions
	defer func() {
		if rec := recover(); rec != nil {
			// Log the unsupported XPath function and skip this rule
			evalErr = fmt.Errorf("unsupported XPath function in rule %s: %v", r.rule.Code, rec)
		}
	}()

	if r.compiled != nil {
		// Synchronize access to shared compiled XPath expression
		r.mu.Lock()
		nav := xmlquery.CreateXPathNavigator(ctx.Document)
		v := r.compiled.Evaluate(nav)
		r.mu.Unlock()
		if v != nil {
			if iter, ok := v.(*antxpath.NodeIterator); ok {
				for iter.MoveNext() {
					if n, ok := iter.Current().(*xmlquery.NodeNavigator); ok {
						nodes = append(nodes, n.Current())
					}
				}
			}
		}
	} else {
		// Fallback to Find with error handling
		nodes = r.safeXPathFind(ctx.Document, r.rule.XPath)
	}

	// If there was an evaluation error, return it as a warning
	if evalErr != nil {
		return issues, evalErr
	}
	for _, node := range nodes {
		// Try to enrich with element id/ref if available
		elementID := node.SelectAttr("id")
		if elementID == "" {
			// Many NetEX references are on @ref
			elementID = node.SelectAttr("ref")
		}
		// Build message with helpful context
		baseMsg := r.rule.Message
		if baseMsg == "" {
			baseMsg = r.rule.Name
		}
		if elementID != "" {
			baseMsg = fmt.Sprintf("%s (element=%s, id=%s)", baseMsg, node.Data, elementID)
		} else {
			baseMsg = fmt.Sprintf("%s (element=%s)", baseMsg, node.Data)
		}
		issue := types.ValidationIssue{
			Rule: types.ValidationRule{
				Code:     r.rule.Code,
				Name:     r.rule.Name,
				Message:  r.rule.Message,
				Severity: r.rule.Severity,
			},
			Location: types.DataLocation{
				FileName:  ctx.GetFileName(),
				XPath:     computeNodeXPath(node),
				ElementID: elementID,
			},
			Message: baseMsg,
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// GetRule returns the underlying rule metadata
func (r *SimpleXPathRule) GetRule() types.ValidationRule {
	return types.ValidationRule{
		Code:     r.rule.Code,
		Name:     r.rule.Name,
		Message:  r.rule.Message,
		Severity: r.rule.Severity,
	}
}

// GetXPath returns the XPath expression
func (r *SimpleXPathRule) GetXPath() string { return r.rule.XPath }

// hasUnsupportedFunctions checks if XPath contains functions not supported by antchfx/xmlquery
func (r *SimpleXPathRule) hasUnsupportedFunctions(xpath string) bool {
	unsupportedFunctions := []string{
		"current()",
		"document()",
		"key()",
		"format-number()",
		"generate-id()",
		"system-property()",
		"element-available()",
		"function-available()",
	}

	xpathLower := strings.ToLower(xpath)
	for _, fn := range unsupportedFunctions {
		if strings.Contains(xpathLower, strings.ToLower(fn)) {
			return true
		}
	}
	return false
}

// safeXPathFind safely executes XPath query with error recovery
func (r *SimpleXPathRule) safeXPathFind(doc *xmlquery.Node, xpath string) []*xmlquery.Node {
	defer func() {
		if rec := recover(); rec != nil {
			// Log the error but don't crash
			logging.GetDefaultLogger().Warn("Skipping rule: unsupported XPath function", "rule", r.rule.Code, "error", rec)
		}
	}()

	// Check for unsupported functions before executing
	if r.hasUnsupportedFunctions(xpath) {
		logging.GetDefaultLogger().Warn("Skipping rule: unsupported XPath function", "rule", r.rule.Code)
		return nil
	}

	return xmlquery.Find(doc, xpath)
}

// computeNodeXPath builds a simple XPath-like location for a node
func computeNodeXPath(n *xmlquery.Node) string {
	if n == nil {
		return ""
	}
	var parts []string
	for cur := n; cur != nil && cur.Type != xmlquery.DocumentNode; cur = cur.Parent {
		// position among siblings with same name
		pos := 1
		for sib := cur.PrevSibling; sib != nil; sib = sib.PrevSibling {
			if sib.Type == xmlquery.ElementNode && sib.Data == cur.Data {
				pos++
			}
		}
		parts = append(parts, fmt.Sprintf("/%s[%d]", cur.Data, pos))
	}
	// reverse
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "")
}

// warnUnmatchedRuleConfig logs YAML configuration entries and rule/severity
// overrides that refer to no loaded rule. Rule IDs depend on how rule sets are
// loaded (e.g. "line/2" vs "eu/line/2"), so this catches stale or mistyped IDs
// that would otherwise be silently ignored.
func warnUnmatchedRuleConfig(logger *logging.Logger, cfg *config.ValidatorConfig, opts *ValidationOptions, loaded []rules.Rule) {
	byNamespace := make(map[string][]string)
	known := make(map[string]bool)
	for _, r := range loaded {
		byNamespace[r.Category] = append(byNamespace[r.Category], r.Key)
		known[r.Code] = true
	}
	for _, custom := range cfg.Rules.Custom {
		known[custom.Code] = true
	}

	for _, problem := range cfg.UnmatchedRuleConfig(byNamespace) {
		logger.Warn("Configuration entry has no effect: " + problem)
	}
	for id := range opts.RuleOverrides {
		if !known[id] {
			logger.Warn(fmt.Sprintf("Rule override has no effect: no loaded rule has ID %q", id))
		}
	}
	for id := range opts.SeverityOverrides {
		if !known[id] {
			logger.Warn(fmt.Sprintf("Severity override has no effect: no loaded rule has ID %q", id))
		}
	}
}
