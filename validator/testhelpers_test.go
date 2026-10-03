package validator

import (
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/testutil"
)

// testOptions returns the default options with the shipped EU rule set loaded
// (without a namespace, so rule IDs look like "line/2").
func testOptions(tb testing.TB) *ValidationOptions {
	tb.Helper()
	return DefaultValidationOptions().WithRuleSet("", testutil.RuleSetPath(tb, "eu"))
}
