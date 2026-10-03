package ids

import (
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// ExternalReferenceValidator validates references to external NetEX objects
// This interface matches the Java implementation for compatibility
type ExternalReferenceValidator interface {
	// ValidateReferenceIds returns a set of IDs that are valid according to this external reference validator
	// The returned IDs will be removed from the unresolved references list
	ValidateReferenceIds(externalIdsToValidate []types.IdVersion) []types.IdVersion
}

// DefaultExternalReferenceValidator provides basic external reference validation
type DefaultExternalReferenceValidator struct {
	knownExternalIds map[string]bool
	ignoredPatterns  []string
}

// NewDefaultExternalReferenceValidator creates a new default external reference validator
func NewDefaultExternalReferenceValidator() *DefaultExternalReferenceValidator {
	return &DefaultExternalReferenceValidator{
		knownExternalIds: make(map[string]bool),
		ignoredPatterns: []string{
			// Common patterns for external references that should be ignored
			"NSR:", // Norwegian Stop Registry
			"RUT:", // Ruter (Oslo transport authority)
			"ENT:", // Entur platform references
			"FR:",  // French national references
		},
	}
}

// ValidateReferenceIds validates external reference IDs
func (v *DefaultExternalReferenceValidator) ValidateReferenceIds(externalIdsToValidate []types.IdVersion) []types.IdVersion {
	var validIds []types.IdVersion

	for _, idVersion := range externalIdsToValidate {
		if v.isValidExternalReference(idVersion.ID) {
			validIds = append(validIds, idVersion)
		}
	}

	return validIds
}

// isValidExternalReference checks if an external reference should be considered valid
func (v *DefaultExternalReferenceValidator) isValidExternalReference(id string) bool {
	// Check if it's in our known external IDs
	if v.knownExternalIds[id] {
		return true
	}

	// Check against ignored patterns - these are assumed to be valid external references
	for _, pattern := range v.ignoredPatterns {
		if len(id) >= len(pattern) && id[:len(pattern)] == pattern {
			return true
		}
	}

	// For now, be conservative - only validate IDs we explicitly know about or ignore
	return false
}

// AddKnownExternalId adds a known external ID that should be considered valid
func (v *DefaultExternalReferenceValidator) AddKnownExternalId(id string) {
	v.knownExternalIds[id] = true
}

// AddIgnoredPattern adds a pattern that should be ignored (treated as valid)
func (v *DefaultExternalReferenceValidator) AddIgnoredPattern(pattern string) {
	v.ignoredPatterns = append(v.ignoredPatterns, pattern)
}

// FrenchExternalReferenceValidator provides validation for French NetEX external references
type FrenchExternalReferenceValidator struct {
	*DefaultExternalReferenceValidator
}

// NewFrenchExternalReferenceValidator creates a validator configured for French NetEX datasets
func NewFrenchExternalReferenceValidator() *FrenchExternalReferenceValidator {
	base := NewDefaultExternalReferenceValidator()

	// Add French-specific ignored patterns
	base.AddIgnoredPattern("FR:")
	base.AddIgnoredPattern("MOBIITI:")
	base.AddIgnoredPattern("BISCARROSSE:")
	base.AddIgnoredPattern("GTFS:")

	return &FrenchExternalReferenceValidator{
		DefaultExternalReferenceValidator: base,
	}
}
