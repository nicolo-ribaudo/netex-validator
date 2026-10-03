package context

import (
	"github.com/antchfx/xmlquery"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// ValidationContext represents the context for validation operations
type ValidationContext interface {
	GetCodespace() string
	GetFileName() string
	GetValidationReportID() string
}

// BaseValidationContext provides common validation context functionality
type BaseValidationContext struct {
	Codespace          string
	FileName           string
	ValidationReportID string
}

func (b *BaseValidationContext) GetCodespace() string {
	return b.Codespace
}

func (b *BaseValidationContext) GetFileName() string {
	return b.FileName
}

func (b *BaseValidationContext) GetValidationReportID() string {
	return b.ValidationReportID
}

// SchemaValidationContext represents context for XML schema validation
type SchemaValidationContext struct {
	BaseValidationContext
	FileContent []byte
}

func NewSchemaValidationContext(fileName, codespace string, fileContent []byte) *SchemaValidationContext {
	return &SchemaValidationContext{
		BaseValidationContext: BaseValidationContext{
			Codespace: codespace,
			FileName:  fileName,
		},
		FileContent: fileContent,
	}
}

// XPathValidationContext represents context for XPath-based validation
type XPathValidationContext struct {
	BaseValidationContext
	Document  *xmlquery.Node
	LocalIDs  map[string]types.IdVersion
	LocalRefs []types.IdVersion
}

func NewXPathValidationContext(fileName, codespace, reportID string, document *xmlquery.Node, localIDs map[string]types.IdVersion, localRefs []types.IdVersion) *XPathValidationContext {
	return &XPathValidationContext{
		BaseValidationContext: BaseValidationContext{
			Codespace:          codespace,
			FileName:           fileName,
			ValidationReportID: reportID,
		},
		Document:  document,
		LocalIDs:  localIDs,
		LocalRefs: localRefs,
	}
}

// JAXBValidationContext represents context for object model validation
type JAXBValidationContext struct {
	BaseValidationContext
	NetexEntities        interface{} // Will be replaced with proper NetEX entity index
	CommonDataRepository interface{} // Will be replaced with proper repository
	StopPlaceRepository  interface{} // Will be replaced with proper repository
	LocalIDMap           map[string]types.IdVersion
}

func NewJAXBValidationContext(reportID, codespace, fileName string, localIDMap map[string]types.IdVersion) *JAXBValidationContext {
	return &JAXBValidationContext{
		BaseValidationContext: BaseValidationContext{
			Codespace:          codespace,
			FileName:           fileName,
			ValidationReportID: reportID,
		},
		LocalIDMap: localIDMap,
	}
}

// IsCommonFile returns true if this context represents a common file (shared data)
func (j *JAXBValidationContext) IsCommonFile() bool {
	// Common files typically start with underscore in EU NeTEx profile
	if len(j.FileName) > 0 && j.FileName[0] == '_' {
		return true
	}
	return false
}
