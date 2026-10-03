package main

import (
	"fmt"
	"strings"

	"github.com/nicolo-ribaudo/netex-validator/validator"
)

const rulesFlag = "--rules"

// extractRuleArgs removes every --rules flag from args and returns the rule
// sets they specify, in order, along with the remaining arguments.
//
// pflag cannot parse the namespaced form, so this runs before cobra. Supported forms:
//
//	--rules:<name>=<path>   --rules:<name> <path>
//	--rules=<path>          --rules <path>
//
// Without :<name>, the rule set has an empty Name, so it is loaded without a
// namespace when it is the only one, and under its file base name otherwise.
// Arguments after "--" are left untouched, and so is the value of any other
// flag: takesValue reports whether a flag argument (e.g. "-o") consumes the
// next argument, so that in "-o --rules" the "--rules" is left as -o's value.
func extractRuleArgs(args []string, takesValue func(arg string) bool) ([]validator.RuleSet, []string, error) {
	var sets []validator.RuleSet
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		if !strings.HasPrefix(arg, rulesFlag) {
			rest = append(rest, arg)
			if takesValue != nil && takesValue(arg) && i+1 < len(args) {
				i++
				rest = append(rest, args[i])
			}
			continue
		}

		spec := strings.TrimPrefix(arg, rulesFlag)
		var name, path string
		hasValue := false
		switch {
		case spec == "":
		case strings.HasPrefix(spec, "="):
			path, hasValue = spec[1:], true
		case strings.HasPrefix(spec, ":"):
			name, path, hasValue = strings.Cut(spec[1:], "=")
			if name == "" {
				return nil, nil, fmt.Errorf("%s: missing rule set name after ':'", arg)
			}
		default:
			// Some other flag that happens to start with --rules
			rest = append(rest, arg)
			continue
		}

		if !hasValue {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return nil, nil, fmt.Errorf("%s: missing rule file path", arg)
			}
			i++
			path = args[i]
		}
		if path == "" {
			return nil, nil, fmt.Errorf("%s: missing rule file path", arg)
		}
		sets = append(sets, validator.RuleSet{Name: name, Path: path})
	}

	return sets, rest, nil
}
