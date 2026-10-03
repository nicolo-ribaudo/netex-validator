package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/validator"
)

func TestExtractRuleArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantSets []validator.RuleSet
		wantRest []string
	}{
		{
			name:     "no rules",
			args:     []string{"-i", "data.xml", "-c", "X"},
			wantRest: []string{"-i", "data.xml", "-c", "X"},
		},
		{
			name:     "space separated",
			args:     []string{"--rules", "a.json", "-i", "data.xml"},
			wantSets: []validator.RuleSet{{Path: "a.json"}},
			wantRest: []string{"-i", "data.xml"},
		},
		{
			name:     "equals",
			args:     []string{"-i", "data.xml", "--rules=a.json"},
			wantSets: []validator.RuleSet{{Path: "a.json"}},
			wantRest: []string{"-i", "data.xml"},
		},
		{
			name:     "named with equals",
			args:     []string{"--rules:eu=rulesets/eu.json"},
			wantSets: []validator.RuleSet{{Name: "eu", Path: "rulesets/eu.json"}},
			wantRest: []string{},
		},
		{
			name:     "named space separated",
			args:     []string{"--rules:eu", "rulesets/eu.json", "--verbose"},
			wantSets: []validator.RuleSet{{Name: "eu", Path: "rulesets/eu.json"}},
			wantRest: []string{"--verbose"},
		},
		{
			name: "mixed, order preserved",
			args: []string{"--rules", "a.json", "-c", "X", "--rules:b=b.json", "--rules=c.json"},
			wantSets: []validator.RuleSet{
				{Path: "a.json"},
				{Name: "b", Path: "b.json"},
				{Path: "c.json"},
			},
			wantRest: []string{"-c", "X"},
		},
		{
			name:     "after double dash untouched",
			args:     []string{"--rules", "a.json", "--", "--rules", "b.json"},
			wantSets: []validator.RuleSet{{Path: "a.json"}},
			wantRest: []string{"--", "--rules", "b.json"},
		},
		{
			name:     "other flag with rules prefix",
			args:     []string{"--rulesx", "1"},
			wantRest: []string{"--rulesx", "1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sets, rest, err := extractRuleArgs(tc.args, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(sets, tc.wantSets) {
				t.Errorf("sets = %#v, want %#v", sets, tc.wantSets)
			}
			if !reflect.DeepEqual(rest, tc.wantRest) {
				t.Errorf("rest = %#v, want %#v", rest, tc.wantRest)
			}
		})
	}
}

func TestExtractRuleArgsErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--rules"},
		{"--rules:eu"},
		{"--rules", "--verbose"},
		{"--rules="},
		{"--rules:=a.json"},
		{"--rules:eu="},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, _, err := extractRuleArgs(args, nil); err == nil {
				t.Errorf("expected an error for %v", args)
			}
		})
	}
}

func TestExtractRuleArgsSkipsOtherFlagValues(t *testing.T) {
	takesValue := func(arg string) bool { return arg == "-o" || arg == "--output" }
	sets, rest, err := extractRuleArgs([]string{"-o", "--rules", "--rules", "a.json", "--output", "--rules:x=b.json", "-v"}, takesValue)
	if err != nil {
		t.Fatal(err)
	}
	wantSets := []validator.RuleSet{{Path: "a.json"}}
	wantRest := []string{"-o", "--rules", "--output", "--rules:x=b.json", "-v"}
	if !reflect.DeepEqual(sets, wantSets) {
		t.Errorf("sets = %#v, want %#v", sets, wantSets)
	}
	if !reflect.DeepEqual(rest, wantRest) {
		t.Errorf("rest = %#v, want %#v", rest, wantRest)
	}
}
