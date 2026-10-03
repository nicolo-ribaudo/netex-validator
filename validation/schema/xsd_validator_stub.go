//go:build !libxml2
// +build !libxml2

package schema

import (
	"fmt"

	verrors "github.com/nicolo-ribaudo/netex-validator/reporting"
)

// validateWithLibxml2 is a stub when libxml2 build tag is not enabled
func (v *XSDValidator) validateWithLibxml2(xmlContent []byte, schema *XSDSchema, filename string) ([]*verrors.ValidationError, error) {
	return nil, fmt.Errorf("libxml2 backend not integrated")
}
