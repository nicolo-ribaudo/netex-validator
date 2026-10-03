package engine

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// EnhancedObjectRunner runs object model validation with data collection
type EnhancedObjectRunner struct {
	validators           *ObjectValidatorRegistry
	dataCollectors       *DataCollectorRegistry
	commonDataRepo       *context.CommonDataRepository
	enableDataCollection bool
}

// NewEnhancedObjectRunner creates a new enhanced object runner
func NewEnhancedObjectRunner() *EnhancedObjectRunner {
	runner := &EnhancedObjectRunner{
		validators:           NewObjectValidatorRegistry(),
		dataCollectors:       NewDataCollectorRegistry(),
		commonDataRepo:       context.NewCommonDataRepository(),
		enableDataCollection: true,
	}

	// Register default validators
	runner.registerDefaultValidators()

	// Register default data collectors
	runner.registerDefaultDataCollectors()

	return runner
}

// registerDefaultValidators registers the built-in object validators
func (r *EnhancedObjectRunner) registerDefaultValidators() {
	r.validators.RegisterValidator(NewServiceJourneyObjectValidator())
	r.validators.RegisterValidator(NewNetworkConsistencyValidator())
}

// registerDefaultDataCollectors registers the built-in data collectors
func (r *EnhancedObjectRunner) registerDefaultDataCollectors() {
	r.dataCollectors.RegisterCollector(NewCommonDataCollector())
	r.dataCollectors.RegisterCollector(NewNetworkTopologyCollector())
	r.dataCollectors.RegisterCollector(NewServiceFrequencyCollector())
}

// RegisterValidator adds a custom object validator
func (r *EnhancedObjectRunner) RegisterValidator(validator ObjectValidator) {
	r.validators.RegisterValidator(validator)
}

// RegisterDataCollector adds a custom data collector
func (r *EnhancedObjectRunner) RegisterDataCollector(collector DataCollector) {
	r.dataCollectors.RegisterCollector(collector)
}

// SetEnableDataCollection enables or disables data collection
func (r *EnhancedObjectRunner) SetEnableDataCollection(enabled bool) {
	r.enableDataCollection = enabled
}

// ValidateFile validates a single file using object model validation
func (r *EnhancedObjectRunner) ValidateFile(fileName, codespace, reportID string, xmlData []byte, xmlDoc *xmlquery.Node) ([]types.ValidationIssue, error) {
	// Create object validation context
	ctx, err := context.NewObjectValidationContext(fileName, codespace, reportID, xmlData, xmlDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to create object validation context: %w", err)
	}

	// Set shared common data repository
	ctx.SetCommonDataRepository(r.commonDataRepo)

	// Collect data if enabled
	if r.enableDataCollection {
		if err := r.dataCollectors.CollectFromAllFiles(ctx); err != nil {
			return nil, fmt.Errorf("data collection failed: %w", err)
		}

		// Update common data repository from collectors
		if commonCollector := r.dataCollectors.GetCollector("CommonDataCollector"); commonCollector != nil {
			if cdc, ok := commonCollector.(*CommonDataCollector); ok {
				r.commonDataRepo = cdc.GetCommonDataRepository()
				ctx.SetCommonDataRepository(r.commonDataRepo)
			}
		}
	}

	// Run object model validation
	issues := r.validators.ValidateAll(ctx)

	return issues, nil
}

// ValidateDataset validates a complete dataset with cross-file validation
func (r *EnhancedObjectRunner) ValidateDataset(files []FileData) ([]types.ValidationIssue, error) {
	var allIssues []types.ValidationIssue

	// Reset data collectors for new dataset
	if r.enableDataCollection {
		r.dataCollectors.ResetAll()
		r.commonDataRepo = context.NewCommonDataRepository()
	}

	// First pass: collect data from all files
	var contexts []*context.ObjectValidationContext

	for _, file := range files {
		// Parse XML document
		xmlDoc, err := xmlquery.Parse(strings.NewReader(string(file.Content)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML for file %s: %w", file.FileName, err)
		}

		// Create object validation context
		ctx, err := context.NewObjectValidationContext(file.FileName, file.Codespace, file.ReportID, file.Content, xmlDoc)
		if err != nil {
			return nil, fmt.Errorf("failed to create context for file %s: %w", file.FileName, err)
		}

		contexts = append(contexts, ctx)

		// Collect data
		if r.enableDataCollection {
			if err := r.dataCollectors.CollectFromAllFiles(ctx); err != nil {
				return nil, fmt.Errorf("data collection failed for file %s: %w", file.FileName, err)
			}
		}
	}

	// Update common data repository
	if r.enableDataCollection {
		if commonCollector := r.dataCollectors.GetCollector("CommonDataCollector"); commonCollector != nil {
			if cdc, ok := commonCollector.(*CommonDataCollector); ok {
				r.commonDataRepo = cdc.GetCommonDataRepository()
			}
		}
	}

	// Second pass: validate all files with complete dataset context
	for _, ctx := range contexts {
		// Set the updated common data repository
		ctx.SetCommonDataRepository(r.commonDataRepo)

		// Run validation
		issues := r.validators.ValidateAll(ctx)
		allIssues = append(allIssues, issues...)
	}

	// Run cross-file validation
	if r.enableDataCollection {
		crossFileIssues := r.validateCrossFileConsistency(contexts)
		allIssues = append(allIssues, crossFileIssues...)
	}

	return allIssues, nil
}

// validateCrossFileConsistency performs validation across multiple files
func (r *EnhancedObjectRunner) validateCrossFileConsistency(contexts []*context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Get topology collector for cross-file validation
	topologyCollector := r.dataCollectors.GetCollector("NetworkTopologyCollector")
	if topologyCollector == nil {
		return issues
	}

	ntc, ok := topologyCollector.(*NetworkTopologyCollector)
	if !ok {
		return issues
	}

	// Validate network connectivity across files
	issues = append(issues, r.validateNetworkConnectivity(contexts, ntc)...)

	// Validate service consistency across files
	issues = append(issues, r.validateServiceConsistency(contexts)...)

	return issues
}

// validateNetworkConnectivity validates network connectivity across files
func (r *EnhancedObjectRunner) validateNetworkConnectivity(contexts []*context.ObjectValidationContext, ntc *NetworkTopologyCollector) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Check for lines without routes
	for _, ctx := range contexts {
		for _, line := range ctx.Lines() {
			routes := ntc.GetRoutesForLine(line.ID)
			if len(routes) == 0 {
				issues = append(issues, types.ValidationIssue{
					Rule: types.ValidationRule{
						Code:     "CROSS_FILE_1",
						Name:     "Line without routes",
						Message:  "Line has no associated routes across all files",
						Severity: types.WARNING,
					},
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: line.ID,
					},
					Message: fmt.Sprintf("Line '%s' has no routes defined across the dataset", line.ID),
				})
			}
		}
	}

	// Check for routes without journey patterns
	for _, ctx := range contexts {
		for _, route := range ctx.Routes() {
			patterns := ntc.GetPatternsForRoute(route.ID)
			if len(patterns) == 0 {
				issues = append(issues, types.ValidationIssue{
					Rule: types.ValidationRule{
						Code:     "CROSS_FILE_2",
						Name:     "Route without journey patterns",
						Message:  "Route has no associated journey patterns across all files",
						Severity: types.WARNING,
					},
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: route.ID,
					},
					Message: fmt.Sprintf("Route '%s' has no journey patterns defined across the dataset", route.ID),
				})
			}
		}
	}

	return issues
}

// validateServiceConsistency validates service consistency across files
func (r *EnhancedObjectRunner) validateServiceConsistency(contexts []*context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Check for service journeys without corresponding lines in any file
	allLineIDs := make(map[string]bool)

	// Collect all line IDs across files
	for _, ctx := range contexts {
		for _, line := range ctx.Lines() {
			allLineIDs[line.ID] = true
		}
		for _, flexLine := range ctx.FlexibleLines() {
			allLineIDs[flexLine.ID] = true
		}
	}

	// Check service journeys
	for _, ctx := range contexts {
		for _, sj := range ctx.ServiceJourneys() {
			if sj.LineRef != nil && !allLineIDs[sj.LineRef.Ref] {
				issues = append(issues, types.ValidationIssue{
					Rule: types.ValidationRule{
						Code:     "CROSS_FILE_3",
						Name:     "Service journey references unknown line",
						Message:  "Service journey references line not found in any file",
						Severity: types.ERROR,
					},
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: sj.ID,
					},
					Message: fmt.Sprintf("Service journey '%s' references line '%s' not found in dataset", sj.ID, sj.LineRef.Ref),
				})
			}
		}
	}

	return issues
}

// GetCollectedData returns collected data from a specific collector
func (r *EnhancedObjectRunner) GetCollectedData(collectorName string) interface{} {
	collector := r.dataCollectors.GetCollector(collectorName)
	if collector != nil {
		return collector.GetCollectedData()
	}
	return nil
}

// GetAllValidationRules returns all validation rules from object validators
func (r *EnhancedObjectRunner) GetAllValidationRules() []types.ValidationRule {
	return r.validators.GetAllRules()
}

// FileData represents file data for dataset validation
type FileData struct {
	FileName  string
	Codespace string
	ReportID  string
	Content   []byte
}
