package validator

import (
	"github.com/theoremus-urban-solutions/netex-validator/logging"
	"github.com/theoremus-urban-solutions/netex-validator/rules"
	"github.com/theoremus-urban-solutions/netex-validator/types"
)

// RuleSet is a JSON rule file loaded under a namespace. See rules.RuleSet.
type RuleSet = rules.RuleSet

// ValidationOptions configures NetEX validation behavior.
//
// This struct allows fine-grained control over validation processing,
// including rule selection, output formats, and performance optimizations.
//
// Use DefaultValidationOptions() to get a base configuration, then chain
// With* methods to customize specific settings:
//
//	options := DefaultValidationOptions().
//		WithCodespace("NO").
//		WithVerbose(true).
//		WithSkipSchema(true)
//
// All With* methods return the same ValidationOptions instance for method chaining.
type ValidationOptions struct {
	// Codespace identifies the authority/operator responsible for the NetEX data.
	// This is used for validation context and should match the data provider's
	// identifier (e.g., "NO" for Norway, "SE" for Sweden, "DK" for Denmark).
	Codespace string

	// ConfigFile specifies the path to a YAML configuration file for rule customization.
	// The config file can enable/disable rule namespaces or specific rules and
	// override their severity levels.
	ConfigFile string

	// SkipSchema bypasses XML schema validation for faster processing.
	// When true, only business rule validation is performed. Useful when you trust
	// the XML structure and want to focus on NetEX-specific business logic.
	SkipSchema bool

	// SkipValidators bypasses all XPath business rule validation.
	// When true, only schema validation is performed. Useful for basic
	// XML structure checking without business logic validation.
	SkipValidators bool

	// RuleSets lists the JSON rule files to run as XPath business rules, in order.
	// No rules are compiled into the validator: if RuleSets is empty, no XPath
	// business rules are run. A single rule set with an empty Name is loaded
	// without a namespace (rule IDs like "line/2"); otherwise each rule set is
	// namespaced by its Name, defaulting to the file base name ("eu/line/2").
	RuleSets []RuleSet

	// MaxSchemaErrors limits the number of schema validation errors reported.
	// Set to 0 to use the configuration default (typically 100).
	// Higher values provide more comprehensive error reporting but may impact performance.
	MaxSchemaErrors int

	// Verbose enables detailed logging during validation processing.
	// When true, validation progress, rule execution, and detailed error
	// information is logged to help with debugging and monitoring.
	Verbose bool

	// RuleOverrides allows selective enabling/disabling of validation rules.
	// Map key is the full rule ID (e.g., "line/2", "eu/line/2"),
	// value indicates whether the rule should be executed.
	RuleOverrides map[string]bool

	// SeverityOverrides allows changing the severity level of specific rules.
	// Map key is the full rule ID, value is the new severity level to apply.
	// Useful for treating warnings as errors or vice versa based on local requirements.
	SeverityOverrides map[string]types.Severity

	// OutputFormat specifies the preferred output format for structured results.
	// Supported values: "json" (default), "html" (interactive report), "text" (plain text).
	// This primarily affects CLI output; library users can call specific To* methods.
	OutputFormat string

	// LogLevel sets the minimum logging level for validation operations.
	// Available levels: DEBUG, INFO, WARN, ERROR. Default is INFO.
	LogLevel logging.LogLevel

	// LogFormat specifies the log output format. Logs are written to stderr.
	// Supported values: "plain" (message only), "text" (key=value), "json" (structured). Default is "plain".
	LogFormat string

	// Logger allows custom logger injection. If nil, a default logger is created
	// based on LogLevel and LogFormat settings.
	Logger *logging.Logger

	// Profile is deprecated and ignored; select rules with RuleSets.
	Profile string

	// MaxFindings limits the total number of validation findings to collect (0 = unlimited).
	MaxFindings int

	// AllowSchemaNetwork enables downloading schemas from network for XSD validation.
	// Default is false: downloaded schemas are not used yet (XSD validation is a
	// placeholder), and unreachable schema hosts can block validation for minutes.
	AllowSchemaNetwork bool

	// SchemaCacheDir specifies where downloaded schemas are cached.
	SchemaCacheDir string

	// SchemaTimeoutSeconds sets HTTP timeout for schema downloads.
	SchemaTimeoutSeconds int

	// UseLibxml2XSD enables real XSD validation using libxml2 bindings when available.
	// Default is false; when true, the validator will attempt libxml2 and fall back on failure.
	UseLibxml2XSD bool

	// ConcurrentFiles sets the number of files to process in parallel when validating ZIP datasets.
	// 0 means use configuration default.
	ConcurrentFiles int

	// EnableValidationCache enables in-memory caching of validation results by file hash
	EnableValidationCache bool

	// CacheMaxEntries sets the maximum number of validation results to cache (default: 1000)
	CacheMaxEntries int

	// CacheMaxMemoryMB sets the approximate maximum memory usage for cache in MB (default: 50)
	CacheMaxMemoryMB int

	// CacheTTLHours sets how long cached results remain valid (default: 24 hours)
	CacheTTLHours int

	// Progress, if set, receives progress events while validating. It may be
	// called concurrently when files of a dataset are validated in parallel.
	Progress types.ProgressFunc
}

// DefaultValidationOptions returns a ValidationOptions instance with sensible defaults.
//
// Default configuration:
//   - Codespace: "Default" (should be overridden with actual codespace)
//   - Schema validation: enabled
//   - Business rule validation: enabled
//   - Maximum schema errors: 100
//   - Verbose logging: disabled
//   - Output format: JSON
//   - No rule or severity overrides
//
// Example:
//
//	options := netexvalidator.DefaultValidationOptions()
//	options.Codespace = "NO"  // Or use WithCodespace("NO")
func DefaultValidationOptions() *ValidationOptions {
	return &ValidationOptions{
		Codespace:             "Default",
		ConfigFile:            "",
		SkipSchema:            false,
		SkipValidators:        false,
		MaxSchemaErrors:       100,
		Verbose:               false,
		RuleOverrides:         make(map[string]bool),
		SeverityOverrides:     make(map[string]types.Severity),
		OutputFormat:          "json",
		LogLevel:              logging.LevelInfo,
		LogFormat:             "plain",
		Logger:                nil, // Will be created automatically
		Profile:               "",
		MaxFindings:           0,
		AllowSchemaNetwork:    false,
		SchemaCacheDir:        "",
		SchemaTimeoutSeconds:  30,
		UseLibxml2XSD:         false,
		ConcurrentFiles:       0,
		EnableValidationCache: false,
		CacheMaxEntries:       1000,
		CacheMaxMemoryMB:      50,
		CacheTTLHours:         24, // 1 day default
	}
}

// WithCodespace sets the validation codespace and returns the options for chaining.
//
// The codespace identifies the authority or operator responsible for the NetEX data.
// Common values include country codes like "NO", "SE", "DK", or operator-specific
// identifiers like "RUT", "SL", "ATB".
//
// Example:
//
//	options := DefaultValidationOptions().WithCodespace("NO")
func (o *ValidationOptions) WithCodespace(codespace string) *ValidationOptions {
	o.Codespace = codespace
	return o
}

// WithConfigFile sets the path to a YAML configuration file and returns the options for chaining.
//
// The configuration file allows customizing validation rules, their severity levels,
// and other validation parameters. If not specified, rules run as defined in their files.
//
// Example:
//
//	options := DefaultValidationOptions().WithConfigFile("custom-rules.yaml")
func (o *ValidationOptions) WithConfigFile(configFile string) *ValidationOptions {
	o.ConfigFile = configFile
	return o
}

// WithRuleSet adds a JSON rule file to run and returns the options for chaining.
//
// The name is the namespace for the rules in the file; pass "" to use the file
// base name, or to load the file without a namespace when it is the only rule set.
//
// Example:
//
//	options := DefaultValidationOptions().WithRuleSet("", "rulesets/eu.json")
func (o *ValidationOptions) WithRuleSet(name, path string) *ValidationOptions {
	set := RuleSet{Name: name, Path: path}
	for _, existing := range o.RuleSets {
		if existing == set {
			return o // already added, e.g. when reusing options
		}
	}
	o.RuleSets = append(o.RuleSets, set)
	return o
}

// WithSkipSchema configures whether to skip XML schema validation and returns the options for chaining.
//
// When skip is true, schema validation is bypassed for faster processing.
// This is useful when you trust the XML structure and only need business rule validation.
//
// Example:
//
//	options := DefaultValidationOptions().WithSkipSchema(true)  // Skip for speed
func (o *ValidationOptions) WithSkipSchema(skip bool) *ValidationOptions {
	o.SkipSchema = skip
	return o
}

// WithVerbose enables or disables verbose logging and returns the options for chaining.
//
// When verbose is true, detailed validation progress and error information
// is logged, which is helpful for debugging and monitoring validation processes.
//
// Example:
//
//	options := DefaultValidationOptions().WithVerbose(true)  // Enable debug output
func (o *ValidationOptions) WithVerbose(verbose bool) *ValidationOptions {
	o.Verbose = verbose
	return o
}

// WithRuleOverride enables or disables a specific validation rule and returns the options for chaining.
//
// This allows fine-grained control over which rules are executed during validation.
// Rule codes can be found in the documentation or by examining validation results.
//
// Parameters:
//   - ruleCode: The full rule ID (e.g., "line/2", "eu/service_journey/4")
//   - enabled: Whether the rule should be executed
//
// Example:
//
//	options := DefaultValidationOptions().
//		WithRuleOverride("line/2", false).     // Disable this rule
//		WithRuleOverride("route/3", true)      // Explicitly enable this rule
func (o *ValidationOptions) WithRuleOverride(ruleCode string, enabled bool) *ValidationOptions {
	if o.RuleOverrides == nil {
		o.RuleOverrides = make(map[string]bool)
	}
	o.RuleOverrides[ruleCode] = enabled
	return o
}

// WithSeverityOverride changes the severity level of a specific rule and returns the options for chaining.
//
// This allows treating warnings as errors, or reducing the severity of certain rules
// based on local requirements or data quality considerations.
//
// Parameters:
//   - ruleCode: The full rule ID (e.g., "line/2", "eu/service_journey/4")
//   - severity: The new severity level (types.INFO, types.WARNING, types.ERROR, types.CRITICAL)
//
// Example:
//
//	options := DefaultValidationOptions().
//		WithSeverityOverride("line/3", types.ERROR).     // Treat warning as error
//		WithSeverityOverride("route/2", types.WARNING)   // Reduce error to warning
func (o *ValidationOptions) WithSeverityOverride(ruleCode string, severity types.Severity) *ValidationOptions {
	if o.SeverityOverrides == nil {
		o.SeverityOverrides = make(map[string]types.Severity)
	}
	o.SeverityOverrides[ruleCode] = severity
	return o
}

// WithLogLevel sets the logging level and returns the options for chaining.
//
// Available log levels:
//   - logging.LevelDebug: Detailed debugging information
//   - logging.LevelInfo: General informational messages (default)
//   - logging.LevelWarn: Warning messages for potentially problematic situations
//   - logging.LevelError: Error messages for serious problems
//
// Example:
//
//	options := DefaultValidationOptions().WithLogLevel(logging.LevelDebug)
func (o *ValidationOptions) WithLogLevel(level logging.LogLevel) *ValidationOptions {
	o.LogLevel = level
	return o
}

// WithLogFormat sets the log output format and returns the options for chaining.
//
// Supported formats:
//   - "plain": Message only, one per line (default)
//   - "text": key=value format with timestamps, levels and context
//   - "json": Structured JSON format for machine processing
//
// Example:
//
//	options := DefaultValidationOptions().WithLogFormat("json")
func (o *ValidationOptions) WithLogFormat(format string) *ValidationOptions {
	o.LogFormat = format
	return o
}

// WithLogger sets a custom logger instance and returns the options for chaining.
//
// When a custom logger is provided, LogLevel and LogFormat settings are ignored.
// The provided logger is used as-is for all validation logging operations.
//
// Example:
//
//	customLogger := logging.NewJSONLogger(logging.LevelDebug)
//	options := DefaultValidationOptions().WithLogger(customLogger)
func (o *ValidationOptions) WithLogger(logger *logging.Logger) *ValidationOptions {
	o.Logger = logger
	return o
}

// WithProfile sets the validation profile (e.g., "eu", "custom").
func (o *ValidationOptions) WithProfile(profile string) *ValidationOptions {
	o.Profile = profile
	return o
}

// WithMaxFindings caps the number of collected findings (0 = unlimited)
func (o *ValidationOptions) WithMaxFindings(n int) *ValidationOptions {
	o.MaxFindings = n
	return o
}

// WithAllowSchemaNetwork toggles schema network download
func (o *ValidationOptions) WithAllowSchemaNetwork(allow bool) *ValidationOptions {
	o.AllowSchemaNetwork = allow
	return o
}

// WithSchemaCacheDir sets schema cache directory
func (o *ValidationOptions) WithSchemaCacheDir(dir string) *ValidationOptions {
	o.SchemaCacheDir = dir
	return o
}

// WithSchemaTimeoutSeconds sets schema HTTP timeout
func (o *ValidationOptions) WithSchemaTimeoutSeconds(seconds int) *ValidationOptions {
	o.SchemaTimeoutSeconds = seconds
	return o
}

// WithUseLibxml2XSD toggles libxml2-backed XSD validation (experimental)
func (o *ValidationOptions) WithUseLibxml2XSD(use bool) *ValidationOptions {
	o.UseLibxml2XSD = use
	return o
}

// WithConcurrentFiles sets the parallelism for ZIP processing
func (o *ValidationOptions) WithConcurrentFiles(n int) *ValidationOptions {
	o.ConcurrentFiles = n
	return o
}

// WithValidationCache enables caching of validation results by file hash with memory limits
func (o *ValidationOptions) WithValidationCache(enabled bool, maxEntries int, maxMemoryMB int, ttlHours int) *ValidationOptions {
	o.EnableValidationCache = enabled
	o.CacheMaxEntries = maxEntries
	o.CacheMaxMemoryMB = maxMemoryMB
	o.CacheTTLHours = ttlHours
	return o
}

// GetLogger returns the logger instance to use for validation operations.
//
// If a custom logger was set via WithLogger(), it is returned directly.
// Otherwise, a new logger is created based on LogLevel and LogFormat settings.
func (o *ValidationOptions) GetLogger() *logging.Logger {
	if o.Logger != nil {
		return o.Logger
	}

	// Create logger based on configuration
	config := logging.LoggerConfig{
		Level:         o.LogLevel,
		Format:        o.LogFormat,
		Component:     "netex-validator",
		IncludeSource: o.LogLevel == logging.LevelDebug,
	}

	return logging.NewLogger(config)
}

// WithProgress sets a callback that receives progress events and returns the options for chaining.
func (o *ValidationOptions) WithProgress(progress types.ProgressFunc) *ValidationOptions {
	o.Progress = progress
	return o
}
