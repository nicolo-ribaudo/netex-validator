package ids

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/nicolo-ribaudo/netex-validator/interfaces"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// NetexIdExtractor extracts NetEX IDs and references from XML content
type NetexIdExtractor struct{}

// NewNetexIdExtractor creates a new ID extractor
func NewNetexIdExtractor() interfaces.IdExtractor {
	return &NetexIdExtractor{}
}

// ExtractIds extracts all NetEX IDs from XML content
func (e *NetexIdExtractor) ExtractIds(fileName string, content []byte) ([]types.IdVersion, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var ids []types.IdVersion

	// Find all elements with @id attribute
	nodes := xmlquery.Find(doc, "//*[@id]")
	for _, node := range nodes {
		id := node.SelectAttr("id")
		version := node.SelectAttr("version")

		if id != "" {
			ids = append(ids, types.NewIdVersion(id, version, fileName))
		}
	}

	return ids, nil
}

// ExtractReferences extracts all NetEX ID references from XML content
func (e *NetexIdExtractor) ExtractReferences(fileName string, content []byte) ([]types.IdVersion, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var references []types.IdVersion

	// Common NetEX reference patterns
	referencePatterns := []string{
		"//*[@ref]",                    // Generic ref attribute
		"//LineRef",                    // Line references
		"//RouteRef",                   // Route references
		"//JourneyPatternRef",          // Journey pattern references
		"//ServiceJourneyRef",          // Service journey references
		"//OperatorRef",                // Operator references
		"//AuthorityRef",               // Authority references
		"//NetworkRef",                 // Network references
		"//ScheduledStopPointRef",      // Stop point references
		"//StopPlaceRef",               // Stop place references
		"//RoutePointRef",              // Route point references
		"//PassengerStopAssignmentRef", // Stop assignment references
		"//DayTypeRef",                 // Day type references
		"//ValidityConditionsRef",      // Validity conditions references
		"//RepresentedByGroupRef",      // Group references
		"//FlexibleLineRef",            // Flexible line references
		"//TariffZoneRef",              // Tariff zone references
		"//ResponsibilitySetRef",       // Responsibility set references
		"//TypeOfServiceRef",           // Type of service references
		"//TypeOfProductCategoryRef",   // Product category references
		"//TransportOrganisationRef",   // Transport organization references
		"//GroupOfLinesRef",            // Group of lines references
		"//GroupOfServicesRef",         // Group of services references
		"//BlockRef",                   // Block references
		"//CourseOfJourneysRef",        // Course of journeys references
		"//DeadRunRef",                 // Dead run references
		"//InterchangeRef",             // Interchange references
		"//NoticeRef",                  // Notice references
		"//NoticeAssignmentRef",        // Notice assignment references
	}

	for _, pattern := range referencePatterns {
		nodes := xmlquery.Find(doc, pattern)
		for _, node := range nodes {
			var refId, version string

			// Try @ref attribute first
			if refAttr := node.SelectAttr("ref"); refAttr != "" {
				refId = refAttr
				version = node.SelectAttr("version")
			} else {
				// For some elements, the reference might be the text content
				refId = strings.TrimSpace(node.InnerText())
			}

			if refId != "" {
				references = append(references, types.NewIdVersion(refId, version, fileName))
			}
		}
	}

	// Extract references from specific elements that contain ID references
	textReferencePatterns := []string{
		"//JourneyPatternRef",
		"//ServiceJourneyRef",
		"//LineRef",
		"//RouteRef",
		"//OperatorRef",
		"//AuthorityRef",
		"//NetworkRef",
		"//ScheduledStopPointRef",
		"//DayTypeRef",
	}

	for _, pattern := range textReferencePatterns {
		nodes := xmlquery.Find(doc, pattern)
		for _, node := range nodes {
			refId := strings.TrimSpace(node.InnerText())
			version := node.SelectAttr("version")

			if refId != "" {
				references = append(references, types.NewIdVersion(refId, version, fileName))
			}
		}
	}

	return references, nil
}

// NetexIdValidator validates NetEX IDs using a repository
type NetexIdValidator struct {
	repository interfaces.IdRepository
	extractor  interfaces.IdExtractor
}

// NewNetexIdValidator creates a new ID validator
func NewNetexIdValidator(repository interfaces.IdRepository, extractor interfaces.IdExtractor) interfaces.IdValidator {
	return &NetexIdValidator{
		repository: repository,
		extractor:  extractor,
	}
}

// ValidateIds validates all registered IDs and references
func (v *NetexIdValidator) ValidateIds() ([]types.ValidationIssue, error) {
	allIssues := make([]types.ValidationIssue, 0) // Initialize as empty slice, not nil

	// Validate references
	referenceIssues := v.repository.ValidateReferences()
	allIssues = append(allIssues, referenceIssues...)

	// Validate ID formats
	formatIssues := v.repository.ValidateIdFormat()
	allIssues = append(allIssues, formatIssues...)

	// Validate versions
	versionIssues := v.repository.ValidateVersions()
	allIssues = append(allIssues, versionIssues...)

	// Validate for duplicates
	duplicateIssues := v.repository.GetDuplicateIds()
	allIssues = append(allIssues, duplicateIssues...)

	// Validate version consistency across files
	if repo, ok := v.repository.(*NetexIdRepository); ok {
		consistency := repo.ValidateVersionConsistencyAcrossFiles()
		allIssues = append(allIssues, consistency...)
	}

	return allIssues, nil
}

// ExtractIds extracts IDs from XML content and registers them
func (v *NetexIdValidator) ExtractIds(fileName string, content []byte) error {
	ids, err := v.extractor.ExtractIds(fileName, content)
	if err != nil {
		return fmt.Errorf("failed to extract IDs: %w", err)
	}

	for _, id := range ids {
		if err := v.repository.AddId(id.ID, id.Version, id.FileName); err != nil {
			// Log error but continue processing
			// In production, might want to collect these errors
			continue
		}
	}

	return nil
}

// ExtractReferences extracts references from XML content and registers them
func (v *NetexIdValidator) ExtractReferences(fileName string, content []byte) error {
	references, err := v.extractor.ExtractReferences(fileName, content)
	if err != nil {
		return fmt.Errorf("failed to extract references: %w", err)
	}

	for _, ref := range references {
		v.repository.AddReference(ref.ID, ref.Version, ref.FileName)
	}

	return nil
}

// GetRepository returns the underlying ID repository
func (v *NetexIdValidator) GetRepository() interfaces.IdRepository {
	return v.repository
}
