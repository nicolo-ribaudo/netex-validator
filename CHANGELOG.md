# Changelog

All notable changes to the NetEX Validator project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed (breaking)
- XPath business rules are no longer compiled into the binary. They are loaded from JSON rule
  files passed with `--rules <file>` / `--rules:<name>=<file>` (repeatable), or with
  `ValidationOptions.RuleSets` / `WithRuleSet` in the library. `--rules` is required unless
  `--skip-validators` is given. The previous default rules ship as `rulesets/eu.json`.
- Rule IDs are now `<namespace>/<key>` (e.g. `line/2` instead of `LINE_2`). The namespace is the
  `--rules:<name>`, or the file base name when several rule files are passed; a single unnamed
  rule file is not namespaced. `RuleOverrides`, `SeverityOverrides` and the YAML `rules.categories`
  configuration use these IDs.
- 15 rule codes that were defined twice for different checks were disambiguated with a `_B`
  suffix (e.g. `journey_pattern/1_B`).
- `rules.NewRuleRegistry` now takes the rules to hold; `RuleRegistry.WithProfile` was removed.
- `config.IsRuleEnabled` and `config.GetRuleSeverity` take a namespace and rule key; the YAML
  `enabled` fields of categories and rules are now optional (unset = enabled); custom rules still
  need `enabled: true`. Entries and overrides that match no loaded rule are reported as warnings.

### Added
- Rule files can include other files under nested namespaces with `"extends"`.
- Rules can be kept but disabled with `"skip": true`. The 7 rules using the unsupported XPath
  function `current()`, which never ran, are marked this way.
- `rulesets/extended.json`: the rules that were previously defined but filtered out by the
  hard-coded EU profile.
- Report entries now include the rule ID (`ruleId`), and grouped JSON notices list the IDs of
  the rules they contain (`ruleIds`).

### Fixed
- `--skip-validators` was ignored by the CLI.

## [1.0.2] - 2025-09-01

### Added
- Grouped JSON output format for better user experience
- Consistent grouping between HTML and JSON reports
- JSON output now groups validation issues by:
  - File: Issues organized by source file
  - Severity: Issues organized by severity level (Critical/Error/Warning/Info)
  - Rule: Issues organized by validation rule name
- Enhanced JSON statistics with top issues and severity percentages
- Backwards compatibility with flat JSON format via ToFlatJSON() method

### Changed
- JSON output now uses grouped format by default
- Improved consistency between HTML and JSON report structures

### Added
- Initial release of NetEX Validator
- Support for NetEX XML validation against EU NeTEx Profile
- Over 200 built-in validation rules covering:
  - Schema validation
  - Business logic rules
  - Cross-file ID validation
  - Transport mode validation
  - Service journey validation
  - Stop point validation
  - Timing validation
  - Reference integrity checks
- Multiple validation modes:
  - Single XML file validation
  - ZIP dataset validation (multiple files)
  - In-memory content validation
- Configurable validation options:
  - Skip schema validation for faster processing
  - Custom codespace configuration
  - Rule severity overrides via YAML configuration
  - Selective rule enable/disable
- Multiple output formats:
  - JSON for programmatic processing
  - HTML with interactive interface
  - Plain text for CLI output
- Performance features:
  - Validation caching with TTL
  - Concurrent file processing
  - Memory-efficient processing for large datasets
- Comprehensive test coverage including:
  - Unit tests for all components
  - Integration tests
  - Performance benchmarks
  - Concurrent validation stress tests
- Go library API for integration into other applications
- Command-line interface with rich options

### Fixed
- Race conditions in concurrent validation scenarios
- XPath expression thread-safety issues
- Global logger concurrent access synchronization
- Validation cache result modification race conditions

### Security
- Safe XML parsing with protection against XXE attacks
- Secure file path handling
- Input validation and sanitization

## [0.1.0] - TBD

_Initial release - see Unreleased section above for features_

[Unreleased]: https://github.com/theoremus-urban-solutions/netex-validator/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/theoremus-urban-solutions/netex-validator/releases/tag/v0.1.0