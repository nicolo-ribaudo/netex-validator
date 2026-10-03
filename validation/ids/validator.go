package ids

import (
	"fmt"
	"strings"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

// NetexReferenceValidator validates references between NetEX elements
type NetexReferenceValidator struct {
	idRepository      *NetexIdRepository
	validEntityTypes  map[string]bool
	allowedReferences map[string][]string // Map of element type -> allowed reference types
	ignoredReferences map[string]bool     // References to ignore (like service journey interchanges)
}

// NewNetexReferenceValidator creates a new reference validator
func NewNetexReferenceValidator(idRepository *NetexIdRepository) *NetexReferenceValidator {
	validator := &NetexReferenceValidator{
		idRepository:      idRepository,
		validEntityTypes:  getValidEntityTypes(),
		allowedReferences: getAllowedReferences(),
		ignoredReferences: getIgnoredReferences(),
	}
	return validator
}

// getValidEntityTypes returns the set of valid NetEX entity types
func getValidEntityTypes() map[string]bool {
	entityTypes := []string{
		"Line", "FlexibleLine", "Route", "JourneyPattern", "ServiceJourney", "DatedServiceJourney",
		"Operator", "Authority", "Network", "ScheduledStopPoint", "StopPlace", "Quay",
		"TariffZone", "FareZone", "GroupOfLines", "GroupOfServices", "Block", "CourseOfJourneys",
		"DeadRun", "Interchange", "Notice", "FlexibleService", "VehicleJourney", "PassengerStopAssignment",
		"TimingPoint", "ServiceLink", "RouteLink", "RoutePoint", "PointOnRoute",
		"AvailabilityCondition", "ValidityCondition", "DayType", "OperatingDay", "OperatingPeriod",
		"ServiceCalendar", "TypeOfService", "Direction", "DestinationDisplay", "Via",
		"AccessibilityAssessment", "PlaceEquipment", "ServiceFacilitySet", "AccommodationFacilitySet",
	}

	result := make(map[string]bool)
	for _, entityType := range entityTypes {
		result[entityType] = true
	}
	return result
}

// getAllowedReferences defines which entity types can reference which other types
func getAllowedReferences() map[string][]string {
	return map[string][]string{
		"ServiceJourney": {
			"JourneyPattern", "Line", "Operator", "Block", "DayType", "Notice",
		},
		"DatedServiceJourney": {
			"ServiceJourney", "OperatingDay",
		},
		"JourneyPattern": {
			"Route", "DestinationDisplay", "ScheduledStopPoint", "TimingPoint",
		},
		"Route": {
			"Line", "Direction", "PointOnRoute",
		},
		"Line": {
			"Network", "Operator", "Authority", "GroupOfLines", "Notice",
		},
		// Add more as needed based on NetEX specification
	}
}

// getIgnoredReferences returns references that should be ignored during validation
func getIgnoredReferences() map[string]bool {
	ignored := []string{
		"ServiceJourneyInterchange", // These have special handling
		"BlockJourney",              // Block journey references have special rules
		"InterchangeRule",           // Interchange rules have their own validation
	}

	result := make(map[string]bool)
	for _, ref := range ignored {
		result[ref] = true
	}
	return result
}

// ValidateReferences validates all references in the repository
func (v *NetexReferenceValidator) ValidateReferences() []types.ValidationIssue {
	var issues []types.ValidationIssue

	// First validate that all references point to existing IDs
	unresolved := v.idRepository.ValidateReferences()
	issues = append(issues, unresolved...)

	// Validate reference types
	entityTypeIssues := v.validateReferenceEntityTypes()
	issues = append(issues, entityTypeIssues...)

	// Validate cross-file reference consistency
	crossFileIssues := v.validateCrossFileReferences()
	issues = append(issues, crossFileIssues...)

	return issues
}

// validateReferenceEntityTypes validates that references point to valid entity types
func (v *NetexReferenceValidator) validateReferenceEntityTypes() []types.ValidationIssue {
	var issues []types.ValidationIssue

	allIds := v.idRepository.GetAllIds()

	for id, idVersion := range allIds {
		entityType := v.extractEntityType(id)
		if entityType == "" {
			// Cannot determine entity type - might be malformed ID
			continue
		}

		if !v.validEntityTypes[entityType] {
			issues = append(issues, types.ValidationIssue{
				Rule: types.ValidationRule{
					Code:     "NETEX_ID_INVALID_ENTITY_TYPE",
					Name:     "NeTEx ID references invalid entity type",
					Message:  fmt.Sprintf("NetEX ID references unknown entity type '%s'", entityType),
					Severity: types.WARNING,
				},
				Location: types.DataLocation{
					FileName:  idVersion.FileName,
					ElementID: id,
				},
				Message: fmt.Sprintf("NetEX ID '%s' references unknown entity type '%s'", id, entityType),
			})
		}
	}

	return issues
}

// validateCrossFileReferences validates consistency of references across files
func (v *NetexReferenceValidator) validateCrossFileReferences() []types.ValidationIssue {
	var issues []types.ValidationIssue

	// This would implement more complex cross-file validation logic
	// For now, we rely on the basic reference validation from the repository

	return issues
}

// extractEntityType extracts the entity type from a NetEX ID
func (v *NetexReferenceValidator) extractEntityType(id string) string {
	// NetEX ID format: Codespace:EntityType:Identifier
	// Handle cases with multiple colons by looking for the second segment

	if !strings.Contains(id, ":") {
		return ""
	}

	// Split and handle empty segments (double colons)
	raw := strings.Split(id, ":")
	tokens := make([]string, 0, len(raw))
	for _, t := range raw {
		if t != "" {
			tokens = append(tokens, t)
		}
	}

	if len(tokens) < 2 {
		return ""
	}

	return tokens[1] // Entity type is the second non-empty token
}

// ValidateEntityTypeReferences validates that references are to appropriate entity types
func (v *NetexReferenceValidator) ValidateEntityTypeReferences(elementType string, references []types.IdVersion) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Check if we have rules for this element type
	allowedTypes, hasRules := v.allowedReferences[elementType]
	if !hasRules {
		// No specific rules defined - allow all valid entity types
		return issues
	}

	allowedMap := make(map[string]bool)
	for _, allowedType := range allowedTypes {
		allowedMap[allowedType] = true
	}

	for _, ref := range references {
		refEntityType := v.extractEntityType(ref.ID)
		if refEntityType == "" {
			continue // Skip if we can't determine the type
		}

		if !allowedMap[refEntityType] {
			issues = append(issues, types.ValidationIssue{
				Rule: types.ValidationRule{
					Code:     "NETEX_ID_INVALID_REFERENCE_TYPE",
					Name:     "NeTEx ID references inappropriate entity type",
					Message:  fmt.Sprintf("%s should not reference %s", elementType, refEntityType),
					Severity: types.WARNING,
				},
				Location: types.DataLocation{
					FileName:  ref.FileName,
					ElementID: ref.ID,
				},
				Message: fmt.Sprintf("%s '%s' references inappropriate entity type %s in '%s'",
					elementType, ref.ID, refEntityType, ref.FileName),
			})
		}
	}

	return issues
}

// ValidateExternalReferences validates references to external files
func (v *NetexReferenceValidator) ValidateExternalReferences(fileName string, externalRefs []types.IdVersion) []types.ValidationIssue {
	var issues []types.ValidationIssue

	for _, ref := range externalRefs {
		// Check if the external reference uses a different codespace
		if v.isDifferentCodespace(fileName, ref.ID) {
			// External references should have version information
			if ref.Version == "" || ref.Version == "any" {
				issues = append(issues, types.ValidationIssue{
					Rule: types.ValidationRule{
						Code:     "NETEX_ID_EXTERNAL_MISSING_VERSION",
						Name:     "NeTEx ID external reference missing version",
						Message:  fmt.Sprintf("External reference to '%s' is missing version", ref.ID),
						Severity: types.WARNING,
					},
					Location: types.DataLocation{
						FileName:  fileName,
						ElementID: ref.ID,
					},
					Message: fmt.Sprintf("External reference to '%s' should include version information", ref.ID),
				})
			}
		}
	}

	return issues
}

// isDifferentCodespace checks if the reference ID uses a different codespace than the current file
func (v *NetexReferenceValidator) isDifferentCodespace(fileName string, refId string) bool {
	// This is a simplified check - in practice, you would need to track the codespace
	// of each file and compare it with the codespace in the reference ID

	// Extract codespace from reference ID
	if !strings.Contains(refId, ":") {
		return false
	}

	parts := strings.Split(refId, ":")
	if len(parts) < 1 {
		return false
	}

	refCodespace := parts[0]

	// For now, we'll assume different if the codespace is not empty and contains specific patterns
	// This could be enhanced to track actual codespaces per file
	externalPatterns := []string{
		"ENT", "RUT", "NSR", "OST", "VKT", // Common Norwegian codespaces
		"EXTERNAL", "EXT", "OTHER",
	}

	for _, pattern := range externalPatterns {
		if strings.Contains(strings.ToUpper(refCodespace), pattern) {
			return true
		}
	}

	return false
}

// AddIgnoredReferenceType adds a reference type to be ignored during validation
func (v *NetexReferenceValidator) AddIgnoredReferenceType(referenceType string) {
	v.ignoredReferences[referenceType] = true
}

// RemoveIgnoredReferenceType removes a reference type from the ignored list
func (v *NetexReferenceValidator) RemoveIgnoredReferenceType(referenceType string) {
	delete(v.ignoredReferences, referenceType)
}

// IsIgnoredReference checks if a reference type should be ignored
func (v *NetexReferenceValidator) IsIgnoredReference(referenceType string) bool {
	return v.ignoredReferences[referenceType]
}

// ValidateVersionConsistency checks version consistency for references
func (v *NetexReferenceValidator) ValidateVersionConsistency() []types.ValidationIssue {
	// This delegates to the repository's validation
	return v.idRepository.ValidateVersionConsistencyAcrossFiles()
}
