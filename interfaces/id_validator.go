package interfaces

import (
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// IdValidator validates NetEX IDs and references
type IdValidator interface {
	// ValidateIds validates all registered IDs and references
	ValidateIds() ([]types.ValidationIssue, error)

	// ExtractIds extracts IDs from XML content
	ExtractIds(fileName string, content []byte) error

	// ExtractReferences extracts references from XML content
	ExtractReferences(fileName string, content []byte) error

	// GetRepository returns the underlying ID repository
	GetRepository() IdRepository
}

// IdRepository manages NetEX ID storage and validation
type IdRepository interface {
	// AddId registers a NetEX ID
	AddId(id, version, fileName string) error

	// AddReference registers a reference to a NetEX ID
	AddReference(refId, version, fileName string)

	// ValidateReferences validates all references against registered IDs
	ValidateReferences() []types.ValidationIssue

	// ValidateIdFormat validates NetEX ID format compliance
	ValidateIdFormat() []types.ValidationIssue

	// ValidateVersions validates version information on IDs
	ValidateVersions() []types.ValidationIssue

	// GetDuplicateIds returns IDs that appear in multiple files
	GetDuplicateIds() []types.ValidationIssue

	// GetIdsByFile returns all IDs registered for a specific file
	GetIdsByFile(fileName string) []string

	// GetAllIds returns all registered IDs
	GetAllIds() map[string]types.IdVersion

	// Clear resets the repository
	Clear()
}

// IdExtractor extracts NetEX IDs and references from XML content
type IdExtractor interface {
	// ExtractIds extracts all NetEX IDs from XML content
	ExtractIds(fileName string, content []byte) ([]types.IdVersion, error)

	// ExtractReferences extracts all NetEX ID references from XML content
	ExtractReferences(fileName string, content []byte) ([]types.IdVersion, error)
}
