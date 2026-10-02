# NetEX Validator - Go Library & CLI

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](LICENSE)
[![CI/CD](https://github.com/theoremus-urban-solutions/netex-validator/workflows/CI%2FCD/badge.svg)](https://github.com/theoremus-urban-solutions/netex-validator/actions)

A comprehensive Go implementation of the NetEX validator for validating NetEX datasets against the EU NeTEx Profile. Available as both a Go library and command-line tool with advanced features including concurrent processing, validation caching, and comprehensive reporting.

## ✨ Features

### 🏛️ Production-Ready
- **Thread-Safe Validation**: Concurrent validation with race condition protection
- **Validation Caching**: TTL-based caching for improved performance
- **Memory Efficient**: Optimized for large dataset processing
- **Comprehensive Logging**: Structured logging with configurable levels

### 🔍 Extensive Validation Coverage
- **237 Validation Rules**: NetEX business rules in JSON rule sets (73 in the EU set)
- **Schema Validation**: XML Schema (XSD) validation against NetEX specification
- **Cross-File Validation**: ID reference validation across multiple files
- **Transport Mode Validation**: Comprehensive transport mode and submode validation
- **Service Journey Validation**: Timing, passing times, and journey pattern validation
- **Reference Integrity**: ID uniqueness and reference consistency checks

### 📦 Multiple Input Formats
- **Single XML Files**: Individual NetEX XML file validation
- **ZIP Datasets**: Multi-file NetEX dataset validation
- **In-Memory Content**: Direct validation of XML content
- **Stream Processing**: Memory-efficient processing of large files

### 📊 Rich Output Options
- **JSON Reports**: Machine-readable validation results with optimized grouping
- **HTML Reports**: Interactive reports with tabbed interface and statistics
- **Plain Text**: Human-readable console output
- **Detailed Diagnostics**: Location information with XPath and line numbers
- **Element Statistics**: GTFS-style statistics showing counts of NetEX elements

### ⚙️ Flexible Configuration
- **JSON Rule Sets**: Business rules loaded from JSON files at runtime, selectable per run without recompiling
- **YAML Configuration**: Rule severity overrides and custom settings
- **Selective Validation**: Skip schema or business rule validation
- **Performance Tuning**: Configurable concurrency and cache settings
- **Custom Codespaces**: Support for organization-specific codespaces

## 🚀 Installation

### Prerequisites

- Go 1.21 or later

### Install via Go

```bash
go install github.com/theoremus-urban-solutions/netex-validator/cmd/netex-validator@latest
```

### Build from Source

```bash
git clone https://github.com/theoremus-urban-solutions/netex-validator.git
cd netex-validator
go build -o netex-validator cmd/netex-validator/main.go
```

### Download Pre-built Binaries

Download the latest release from the [GitHub Releases page](https://github.com/theoremus-urban-solutions/netex-validator/releases).

## 📖 Usage

### Command Line Interface

#### Basic Validation
```bash
# Validate a single NetEX XML file against the EU rule set
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json

# Validate a NetEX dataset (ZIP file)
./netex-validator -i dataset.zip -c "MyCodespace" --rules rulesets/eu.json

# Generate HTML report
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json --format html -o report.html
```

Business rules are not built into the binary: they are read from the JSON rule files passed
with `--rules` (see [Rule Sets](#-rule-sets)). `--rules` is required unless `--skip-validators` is given.

#### Advanced Options
```bash
# Run the EU rules plus extra rule files
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json --rules ./local-rules.json

# Skip schema validation for faster processing
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json --skip-schema

# Schema validation only
./netex-validator -i input.xml -c "MyCodespace" --skip-validators

# Enable validation caching with custom TTL (hours)
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json --enable-cache --cache-ttl 24

# Limit concurrent processing
./netex-validator -i dataset.zip -c "MyCodespace" --rules rulesets/eu.json --concurrent 2

# Verbose output with debug information
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json --verbose

# Custom configuration file
./netex-validator -i input.xml -c "MyCodespace" --rules rulesets/eu.json --config config.yaml
```

#### Configuration File Example

The YAML configuration can turn rule namespaces or single rules on and off and change their
severity. Namespaces and keys are the ones in the rule IDs (see [Rule IDs](#rule-ids)).

```yaml
# config.yaml
rules:
  categories:
    # Applies to line/* (or eu/line/* when loaded with --rules:eu=rulesets/eu.json)
    line:
      defaultSeverity: WARNING
      rules:
        "2":
          severity: ERROR
        "3":
          enabled: false
    # Disables every rule nested under transport_mode
    transport_mode:
      enabled: false
```

### Go Library API

#### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/theoremus-urban-solutions/netex-validator/validator"
)

func main() {
    // Create validator with the EU rule set
    options := validator.DefaultValidationOptions().
        WithCodespace("MyCodespace").
        WithRuleSet("", "rulesets/eu.json")
    v, err := validator.NewWithOptions(options)
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate a file
    result, err := v.ValidateFile("input.xml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Check results
    if result.IsValid() {
        fmt.Println("✅ Validation passed!")
    } else {
        fmt.Printf("❌ Found %d issues\\n", len(result.ValidationReportEntries))
        for _, entry := range result.ValidationReportEntries {
            fmt.Printf("- %s: %s\\n", entry.Severity, entry.Message)
        }
    }
}
```

#### Advanced Configuration

```go
package main

import (
    "github.com/theoremus-urban-solutions/netex-validator/types"
    "github.com/theoremus-urban-solutions/netex-validator/validator"
)

func main() {
    // Create validator with custom options
    options := validator.DefaultValidationOptions().
        WithCodespace("MyOrg").
        WithRuleSet("eu", "rulesets/eu.json").      // rule IDs like "eu/line/2"
        WithRuleSet("local", "local-rules.json").   // rule IDs like "local/my-rule"
        WithSeverityOverride("eu/line/3", types.ERROR).
        WithSkipSchema(false).
        WithValidationCache(true, 1000, 100, 24).
        WithConcurrentFiles(4).
        WithMaxFindings(500).
        WithVerbose(true)
    
    v, err := validator.NewWithOptions(options)
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate ZIP dataset
    result, err := v.ValidateZip("dataset.zip")
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate HTML report
    htmlReport, err := result.ToHTML()
    if err != nil {
        log.Fatal(err)
    }
    
    // Save report
    if err := os.WriteFile("report.html", []byte(htmlReport), 0644); err != nil {
        log.Fatal(err)
    }
}
```

#### Concurrent Validation

```go
package main

import (
    "sync"
    "github.com/theoremus-urban-solutions/netex-validator/validator"
)

func main() {
    v, _ := validator.NewWithOptions(validator.DefaultValidationOptions().WithRuleSet("", "rulesets/eu.json"))
    
    files := []string{"file1.xml", "file2.xml", "file3.xml"}
    results := make([]*validator.ValidationResult, len(files))
    
    var wg sync.WaitGroup
    for i, file := range files {
        wg.Add(1)
        go func(idx int, filename string) {
            defer wg.Done()
            result, err := v.ValidateFile(filename)
            if err == nil {
                results[idx] = result
            }
        }(i, file)
    }
    wg.Wait()
    
    // Process results...
}
```

## 🏗️ Architecture

The validator follows a modular architecture with clear separation of concerns:

```
netex-validator/
├── cmd/netex-validator/        # CLI application
├── validator/                  # Main validator library
├── validation/
│   ├── engine/                # Validation orchestration
│   ├── schema/                # XSD schema validation
│   ├── context/               # Validation context management
│   └── ids/                   # ID and reference validation
├── rules/                     # Rule loading (JSON rule sets) and registry
├── rulesets/                  # Shipped JSON rule sets (eu.json, extended.json)
├── reporting/                 # Error reporting and formatting
├── config/                    # Configuration management
├── logging/                   # Structured logging
├── utils/                     # Utilities (caching, XPath, etc.)
├── types/                     # Common type definitions
└── interfaces/                # Interface definitions
```

### Core Components

- **ValidationEngine**: Orchestrates the validation process with concurrent execution
- **SchemaValidator**: XML schema validation against NetEX XSD files
- **XPathValidator**: Business rule validation using XPath expressions
- **IDValidator**: Cross-file ID reference and uniqueness validation
- **ReportGenerator**: Creates detailed validation reports in multiple formats
- **CacheManager**: TTL-based validation result caching
- **ConfigManager**: YAML-based configuration handling

## 📚 Rule Sets

XPath business rules live in JSON files, not in the Go code, so they can be changed without
recompiling. Each file you pass with `--rules` is a filter set: run more or fewer rules by passing
more or fewer files.

### Rule file format

```json
{
  "extends": {
    "stop_points": "./stop_points.json"
  },
  "rules": {
    "2": {
      "name": "Line missing Name",
      "message": "Line is missing Name",
      "severity": "ERROR",
      "xpath": "//lines/Line[not(Name)]"
    },
    "uses-current": {
      "name": "Rule using an unsupported XPath function",
      "severity": "WARNING",
      "xpath": "//a[@ref = current()/b]",
      "skip": true
    }
  }
}
```

- `rules` is an object keyed by rule key. Each rule needs `name`, `severity`
  (`INFO`, `WARNING`, `ERROR` or `CRITICAL`) and `xpath`; `message` and `description` are optional.
  The XPath selects the offending nodes: every match is reported as a finding.
- `"skip": true` keeps a rule in the file without running it (its XPath is not compiled).
- `extends` includes other rule files under a nested namespace. Paths are relative to the file
  that contains the `extends` block.
- Files are checked strictly when loaded: unknown or duplicate fields, duplicate keys, invalid
  severities and XPaths that do not compile are errors that name the file and rule.

### Rule IDs

A rule's ID is its namespace path and key joined with `/`. The namespace of a file passed on the
command line is the name given with `--rules:<name>`, or the file base name otherwise. A single
`--rules` without a name is not namespaced:

| Command line | ID of rule `2` in `line.json`, extended by `eu.json` as `line` |
|---|---|
| `--rules rulesets/eu.json` | `line/2` |
| `--rules:foo rulesets/eu.json` (or `--rules:foo=rulesets/eu.json`) | `foo/line/2` |
| `--rules rulesets/eu.json --rules local.json` | `eu/line/2` (and `local/...`) |

The same namespace cannot be used twice; rename one of the files with `--rules:<name>=<path>`.
Rule IDs are used in reports, in `WithRuleOverride` / `WithSeverityOverride` and in the YAML configuration.
Because they depend on how the files are passed, YAML entries and overrides written for one invocation
(e.g. `line/2`) do not apply to another (`eu/line/2`); the validator warns about entries that match no
loaded rule. Rules defined directly in a single unnamed file have no namespace (ID `2`); address them
in YAML under the category `""`, or give the file a name with `--rules:<name>=<path>`.

### Shipped rule sets

- `rulesets/eu.json` extends one file per topic in `rulesets/eu/`: the EU NeTEx Profile rules that
  the validator has always run by default.
- `rulesets/extended.json` extends `rulesets/extended/`: additional rules (service journeys,
  booking, flexible services, frames, stop places, ...) that were previously defined but disabled.

Individual topic files can also be passed directly, e.g. `--rules rulesets/eu/line.json`.

## 📋 Validation Rules

The shipped rule sets contain 237 active validation rules (73 in `eu.json`, 164 in `extended.json`, plus 7 marked `skip`) covering:

### Schema Validation
- XML well-formedness and namespace validation
- NetEX XSD schema compliance
- Root element and structure validation

### Business Logic Rules
- **Transport Rules**: Mode and submode validation
- **Service Rules**: Journey patterns, passing times, timetables
- **Infrastructure Rules**: Stop places, quays, service links
- **Reference Rules**: ID formats, reference integrity
- **Frame Rules**: Proper frame organization and relationships
- **Timing Rules**: Validity periods, operating days
- **Group Rules**: Lines, services, and operator groups

### Cross-File Validation
- ID uniqueness across multiple files
- Reference consistency in ZIP datasets
- Common data file validation
- Circular reference detection

## 📊 Output Examples

### JSON Output (Optimized Grouping)
```json
{
  "codespace": "MyCodespace",
  "validationReportId": "validation-123",
  "creationDate": "2024-01-15T14:30:00Z",
  "generatedAt": "2024-01-15T14:30:01Z",
  "summary": {
    "totalIssues": 150,
    "uniqueIssueTypes": 3,
    "errorCount": 2,
    "warningCount": 148,
    "infoCount": 0,
    "filesProcessed": 5,
    "filesWithIssues": 3,
    "isValid": false
  },
  "notices": {
    "errors": [
      {
        "type": "Missing required TransportMode",
        "description": "TransportMode must be specified for this element",
        "count": 2,
        "severity": "ERROR",
        "affectedFiles": ["lines.xml"],
        "fileDetails": {
          "lines.xml": {
            "count": 2,
            "elementIds": ["MyCodespace:Line:123", "MyCodespace:Line:456"],
            "lineNumbers": [42, 85]
          }
        }
      }
    ],
    "warnings": [
      {
        "type": "Non-numeric NeTEx version",
        "description": "NetEX version should be numeric (e.g., '1.0', '1.4')",
        "count": 148,
        "severity": "WARNING",
        "affectedFiles": ["lines.xml", "stops.xml", "routes.xml"],
        "fileDetails": {
          "lines.xml": { "count": 50 },
          "stops.xml": { "count": 48 },
          "routes.xml": { "count": 50 }
        }
      }
    ]
  },
  "statistics": {
    "operatorCount": 2,
    "lineCount": 15,
    "routeCount": 8,
    "serviceJourneyCount": 245,
    "stopPlaceCount": 120,
    "stopPointCount": 180,
    "timetabledPassingTimeCount": 1240,
    "dayTypeCount": 7,
    "serviceCalendarCount": 3,
    "routePointCount": 340,
    "routeLinkCount": 280,
    "journeyPatternCount": 25,
    "fareZoneCount": 5,
    "tariffZoneCount": 8,
    "accessibilityCount": 45,
    "quayCount": 85,
    "entranceCount": 12,
    "pathLinkCount": 18,
    "totalElements": 2650,
    "totalFiles": 5
  },
  "filesProcessed": 5,
  "processingTimeMs": 1230
}
```

### Benefits of Optimized Grouping

The optimized JSON format provides:
- **Reduced Output Size**: Groups similar issues instead of listing thousands of repetitive entries
- **Better Analysis**: Shows issue patterns and affected files at a glance  
- **Rich Statistics**: NetEX element counts similar to GTFS validator statistics
- **Clean Output**: Focuses on grouped insights without repetitive individual entries

### HTML Report Features
- **Interactive Interface**: Tabbed navigation between issues, statistics, and files
- **Filtering**: Filter by severity, rule, or file
- **Statistics Dashboard**: Visual charts and metrics
- **Responsive Design**: Works on desktop and mobile devices
- **Export Options**: Print-friendly and shareable reports

## 🧪 Testing

### Run Tests
```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run specific test suites
go test ./validator
go test ./validation/engine
```

### Benchmarks
```bash
# Run performance benchmarks
go test -bench=. ./validator

# Memory profiling
go test -memprofile=mem.prof -bench=. ./validator
```

### Test Coverage
The project maintains high test coverage with:
- Unit tests for all components
- Integration tests for end-to-end validation
- Performance benchmarks and stress tests
- Race condition detection tests
- Cross-platform compatibility tests

## 🔧 Development

### Adding New Validation Rules

1. **Define Rule**: Add it to the relevant file in `rulesets/` (or a new file, added to the
   `extends` block of `rulesets/eu.json` or `rulesets/extended.json`)
```json
"MISSING_REQUIRED": {
  "name": "Missing required transport mode",
  "message": "TransportMode is required for all Lines and FlexibleLines",
  "severity": "ERROR",
  "xpath": "//lines/*[self::Line or self::FlexibleLine][not(TransportMode)]"
}
```

2. **Add Tests**: `go test ./rules/` checks that every shipped rule file loads
3. **Update Documentation**: Add to rule documentation

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with tests
4. Ensure all tests pass (`go test -race ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style
- Follow Go conventions and `gofmt` formatting
- Use meaningful variable and function names
- Add comprehensive tests for new functionality
- Include documentation for public APIs
- Ensure thread-safety for concurrent operations

## 📈 Performance

The Go implementation is optimized for performance:

- **Concurrent Processing**: Parallel validation of multiple files
- **Memory Efficiency**: Stream processing and memory pooling
- **Validation Caching**: TTL-based result caching with configurable limits
- **Fast Startup**: No JVM overhead, instant startup time
- **Optimized XPath**: Compiled XPath expressions with thread-safe access

### Benchmarks
- **Single File**: ~100ms for typical NetEX files
- **Large Datasets**: Linear scaling with concurrent processing
- **Memory Usage**: <50MB for most validation scenarios
- **Cache Hit Rate**: 80%+ cache hit rate in typical workflows

## 🔒 Security

- **Safe XML Processing**: Protection against XXE and billion laughs attacks
- **Path Validation**: Secure file path handling for ZIP datasets
- **Input Sanitization**: Validation of all user inputs
- **Memory Limits**: Configurable limits to prevent DoS attacks

## 📄 License

This project is licensed under the EUPL-1.2 License - see the [LICENSE](LICENSE) file for details.

## 🤝 Support & Community

- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/theoremus-urban-solutions/netex-validator/issues)
- **Discussions**: Join the conversation on [GitHub Discussions](https://github.com/theoremus-urban-solutions/netex-validator/discussions)
- **Documentation**: Additional documentation available in the [docs](docs/) directory

## 📝 Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed list of changes and version history.

---

**Made with ❤️ by the Theoremus Urban Solutions team**