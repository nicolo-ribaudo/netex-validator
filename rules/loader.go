package rules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	antxpath "github.com/antchfx/xpath"
	"github.com/nicolo-ribaudo/netex-validator/types"
)

// RuleSet is a JSON rule file loaded under a namespace.
type RuleSet struct {
	// Name is the namespace for the rules in the file. When empty, it defaults
	// to the file base name (see DefaultRuleSetName), unless the rule set is
	// the only one being loaded, in which case its rules are not namespaced.
	Name string
	// Path is the path to the JSON rule file.
	Path string
}

// namespacePattern restricts rule set names, extends names and rule keys.
var namespacePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ruleFileRule is a single rule as written in a JSON rule file.
type ruleFileRule struct {
	Name        string          `json:"name"`
	Message     string          `json:"message"`
	Severity    *types.Severity `json:"severity"`
	XPath       string          `json:"xpath"`
	Description string          `json:"description"`
	Skip        bool            `json:"skip"`
}

// ruleFileFields are the JSON field names of ruleFileRule.
var ruleFileFields = map[string]bool{
	"name": true, "message": true, "severity": true, "xpath": true, "description": true, "skip": true,
}

// DefaultRuleSetName returns the namespace used for a rule file when none is
// given: its base name without extension.
func DefaultRuleSetName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ValidateRuleSetName returns an error if name cannot be used as a namespace.
func ValidateRuleSetName(name string) error {
	if !namespacePattern.MatchString(name) {
		return fmt.Errorf("invalid rule set name %q: must match %s", name, namespacePattern)
	}
	return nil
}

// LoadRuleSets loads the given JSON rule files, in order.
//
// Each rule's Code is its namespace path and key joined with "/", e.g. "line/2",
// or "eu/line/2" when line.json is reached through an "extends" entry named
// "line" in a file loaded under the namespace "eu". A single rule set with an
// empty Name is loaded without a top-level namespace.
func LoadRuleSets(sets []RuleSet) ([]Rule, error) {
	var all []Rule
	seenNames := make(map[string]string)
	seenCodes := make(map[string]bool)

	for _, set := range sets {
		ns := set.Name
		if ns == "" && len(sets) > 1 {
			ns = DefaultRuleSetName(set.Path)
		}
		if ns != "" {
			if err := ValidateRuleSetName(ns); err != nil {
				if set.Name == "" {
					return nil, fmt.Errorf("%s: %w (derived from the file name; use --rules:<name>=%s to choose one)", set.Path, err, set.Path)
				}
				return nil, fmt.Errorf("%s: %w", set.Path, err)
			}
			if prev, ok := seenNames[ns]; ok {
				return nil, fmt.Errorf("rule set name %q used by both %s and %s; use --rules:<other-name>=%s to rename one", ns, prev, set.Path, set.Path)
			}
			seenNames[ns] = set.Path
		}

		loaded, err := loadRuleFile(set.Path, ns, nil)
		if err != nil {
			return nil, err
		}
		for _, r := range loaded {
			if seenCodes[r.Code] {
				return nil, fmt.Errorf("duplicate rule %q", r.Code)
			}
			seenCodes[r.Code] = true
		}
		all = append(all, loaded...)
	}

	return all, nil
}

// loadRuleFile loads one rule file under the namespace path ns, recursing into
// its "extends" entries. stack holds the resolved paths of the files currently
// being loaded, to detect cycles.
func loadRuleFile(path, ns string, stack []string) ([]Rule, error) {
	// Resolve symlinks so that relative "extends" paths are relative to the
	// real file, and so that cycles through links are detected.
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, p := range stack {
		if p == absPath {
			return nil, fmt.Errorf("%s: extends cycle: %s", path, strings.Join(append(stack, absPath), " -> "))
		}
	}
	stack = append(stack, absPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}

	var extends []namedValue
	var ruleEntries []namedValue
	hasExtends, hasRules := false, false

	dec := json.NewDecoder(bytes.NewReader(data))
	err = readObject(dec, func(field string, dec *json.Decoder) error {
		var fieldErr error
		switch field {
		case "extends":
			hasExtends = true
			extends, fieldErr = readNamedValues(dec, "extends")
			return fieldErr
		case "rules":
			hasRules = true
			ruleEntries, fieldErr = readNamedValues(dec, "rules")
			return fieldErr
		default:
			return fmt.Errorf("unknown field %q", field)
		}
	})
	if len(bytes.TrimSpace(data)) == 0 {
		err = errors.New("file is empty")
	} else if err == nil {
		if _, tokErr := dec.Token(); !errors.Is(tokErr, io.EOF) {
			err = errors.New("unexpected data after top-level object")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if !hasExtends && !hasRules {
		return nil, fmt.Errorf("%s: rule file must have \"rules\" or \"extends\"", path)
	}

	var result []Rule

	for _, ext := range extends {
		if err := ValidateRuleSetName(ext.name); err != nil {
			return nil, fmt.Errorf("%s: extends: %w", path, err)
		}
		var target string
		if err := json.Unmarshal(ext.value, &target); err != nil {
			return nil, fmt.Errorf("%s: extends %q: expected a file path string", path, ext.name)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(absPath), target)
		}
		nested, err := loadRuleFile(target, joinID(ns, ext.name), stack)
		if err != nil {
			return nil, fmt.Errorf("%s: extends %q: %w", path, ext.name, err)
		}
		result = append(result, nested...)
	}

	for _, entry := range ruleEntries {
		rule, skip, err := parseRule(ns, entry)
		if err != nil {
			return nil, fmt.Errorf("%s: rule %q: %w", path, entry.name, err)
		}
		if !skip {
			result = append(result, rule)
		}
	}

	return result, nil
}

// parseRule decodes and validates a single rule definition. It reports
// whether the rule is marked as skipped.
func parseRule(ns string, entry namedValue) (Rule, bool, error) {
	if !namespacePattern.MatchString(entry.name) {
		return Rule{}, false, fmt.Errorf("invalid rule key: must match %s", namespacePattern)
	}

	// encoding/json matches field names case-insensitively and accepts
	// duplicates, so check the exact field names first.
	fields, err := readNamedValues(json.NewDecoder(bytes.NewReader(entry.value)), "rule")
	if err != nil {
		return Rule{}, false, err
	}
	for _, f := range fields {
		if !ruleFileFields[f.name] {
			return Rule{}, false, fmt.Errorf("unknown field %q", f.name)
		}
	}

	var def ruleFileRule
	if err := json.Unmarshal(entry.value, &def); err != nil {
		return Rule{}, false, err
	}

	switch {
	case def.Name == "":
		return Rule{}, false, errors.New("missing \"name\"")
	case def.Severity == nil:
		return Rule{}, false, errors.New("missing \"severity\"")
	case def.XPath == "":
		return Rule{}, false, errors.New("missing \"xpath\"")
	}
	if def.Skip {
		return Rule{}, true, nil
	}
	if _, err := antxpath.Compile(def.XPath); err != nil {
		return Rule{}, false, fmt.Errorf("invalid xpath: %w", err)
	}

	return Rule{
		Code:        joinID(ns, entry.name),
		Key:         entry.name,
		Name:        def.Name,
		Message:     def.Message,
		Severity:    *def.Severity,
		XPath:       def.XPath,
		Category:    ns,
		Description: def.Description,
	}, false, nil
}

func joinID(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "/" + name
}

// namedValue is an object member whose value is kept undecoded.
type namedValue struct {
	name  string
	value json.RawMessage
}

// readNamedValues reads a JSON object as an ordered list of members, failing
// on duplicate keys (which encoding/json would silently accept).
func readNamedValues(dec *json.Decoder, field string) ([]namedValue, error) {
	var values []namedValue
	seen := make(map[string]bool)
	err := readObject(dec, func(name string, dec *json.Decoder) error {
		if seen[name] {
			return fmt.Errorf("duplicate key %q in %q", name, field)
		}
		seen[name] = true
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		values = append(values, namedValue{name: name, value: raw})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%q: %w", field, err)
	}
	return values, nil
}

// readObject reads a JSON object from dec, calling member for each key with
// the decoder positioned at the corresponding value.
func readObject(dec *json.Decoder, member func(name string, dec *json.Decoder) error) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return errors.New("expected a JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name, ok := tok.(string)
		if !ok {
			return errors.New("expected an object key")
		}
		if err := member(name, dec); err != nil {
			return err
		}
	}
	_, err = dec.Token() // closing '}'
	return err
}
