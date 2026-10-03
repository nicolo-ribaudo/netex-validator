# Changelog

All notable changes to the NetEX Validator project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2026-10-03

First release of this fork, based on [theoremus-urban-solutions/netex-validator](https://github.com/theoremus-urban-solutions/netex-validator) v1.0.5.

### Added
- Rule files can include other files under nested namespaces with `"extends"`.
- Rules can be kept but disabled with `"skip": true`. The 7 rules using the unsupported XPath
  function `current()`, which never ran, are marked this way.
- `rulesets/extended.json`: the rules that were previously defined but filtered out by the
  hard-coded EU profile.
- Report entries now include the rule ID (`ruleId`), and grouped JSON notices list the IDs of
  the rules they contain (`ruleIds`).
- A colored summary on stderr after each CLI run: findings per rule with level and count, totals
  per severity, how many rules found no issues, files whose rule checks were skipped or that could
  not be validated. Colors are disabled with `--no-color`, `NO_COLOR` or when stderr is not a
  terminal.
- A progress bar on stderr while the CLI is validating (only on a terminal; `--no-progress`
  disables it). Library users can receive the same events with `ValidationOptions.Progress` /
  `WithProgress`.
- `ValidationResult.Rules`, `RuleChecksSkipped`, `FailedFiles` and `Truncated` (set when
  `--max-findings` stopped validation early). The JSON reports include `failedFiles`,
  `ruleChecksSkipped`, `truncated` and `error` when set.

### Changed
- **Breaking:** XPath business rules are no longer compiled into the binary. They are loaded from
  JSON rule files passed with `--rules <file>` / `--rules:<name>=<file>` (repeatable), or with
  `ValidationOptions.RuleSets` / `WithRuleSet` in the library. `--rules` is required unless
  `--skip-validators` is given. The previous default rules ship as `rulesets/eu.json`.
- **Breaking:** Rule IDs are now `<namespace>/<key>` (e.g. `line/2` instead of `LINE_2`). The
  namespace is the `--rules:<name>`, or the file base name when several rule files are passed; a
  single unnamed rule file is not namespaced. `RuleOverrides`, `SeverityOverrides` and the YAML
  `rules.categories` configuration use these IDs.
- **Breaking:** 15 rule codes that were defined twice for different checks were disambiguated with
  a `_B` suffix (e.g. `journey_pattern/1_B`).
- **Breaking:** `rules.NewRuleRegistry` now takes the rules to hold; `RuleRegistry.WithProfile` was
  removed.
- **Breaking:** `config.IsRuleEnabled` and `config.GetRuleSeverity` take a namespace and rule key;
  the YAML `enabled` fields of categories and rules are now optional (unset = enabled); custom
  rules still need `enabled: true`. Entries and overrides that match no loaded rule are reported
  as warnings.
- **Breaking:** `ValidationResult.IsValid()` now returns false when files of a ZIP dataset could not
  be validated (`FailedFiles`); before, such files were silently ignored.
- Logs are written to stderr instead of stdout, so stdout only carries the report. The default
  log format is the new `"plain"` (message only); `"text"` and `"json"` are still available via
  `LogFormat` / `WithLogFormat`.
- Without `--verbose`, the CLI only logs warnings and errors.
- Schema download from the network is now off by default (`--allow-schema-network`,
  `AllowSchemaNetwork`). Downloaded schemas were not used (XSD validation is not implemented),
  and unreachable schema hosts could block validation for minutes.

### Fixed
- `--skip-validators` was ignored by the CLI.
- ZIP validation could hang forever when an entry could not be read or a validation panicked,
  and kept validating files after `--max-findings` was reached.
- ZIP datasets ignored `--skip-schema` / `--skip-validators`, reported 1 processed file, and
  silently dropped files that failed to parse (they are now listed in `FailedFiles` and make the
  result invalid).
- Warnings and errors in the plain log format now include their context (file, error).
- The CLI no longer prints the usage text after a validation run that found errors.

[Unreleased]: https://github.com/nicolo-ribaudo/netex-validator/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/nicolo-ribaudo/netex-validator/releases/tag/v0.0.1
