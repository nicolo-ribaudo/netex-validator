package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Logger provides structured logging capabilities for the NetEX validator.
type Logger struct {
	*slog.Logger
	level slog.Level
}

// LogLevel represents different logging levels.
type LogLevel int

const (
	// LevelDebug provides detailed debugging information.
	LevelDebug LogLevel = iota
	// LevelInfo provides general informational messages.
	LevelInfo
	// LevelWarn provides warning messages for potentially problematic situations.
	LevelWarn
	// LevelError provides error messages for serious problems.
	LevelError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel converts LogLevel to slog.Level.
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LoggerConfig holds configuration for logger creation.
type LoggerConfig struct {
	// Level sets the minimum log level.
	Level LogLevel
	// Format specifies the output format: "plain" (message only, the default),
	// "text" (key=value) or "json".
	Format string
	// Output specifies the output destination (default: stderr, so logs do not
	// mix with reports written to stdout).
	Output io.Writer
	// IncludeSource adds source code information to log entries.
	IncludeSource bool
	// Component identifies the logging component.
	Component string
}

// NewLogger creates a new structured logger with the specified configuration.
func NewLogger(config LoggerConfig) *Logger {
	if config.Output == nil {
		config.Output = os.Stderr
	}

	if config.Format == "" {
		config.Format = "plain"
	}

	if config.Component == "" {
		config.Component = "netex-validator"
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     config.Level.ToSlogLevel(),
		AddSource: config.IncludeSource,
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(config.Output, opts)
	case "plain":
		handler = newPlainHandler(config.Output, opts.Level)
	default:
		handler = slog.NewTextHandler(config.Output, opts)
	}

	// Add component context to all log entries
	logger := slog.New(handler).With("component", config.Component)

	return &Logger{
		Logger: logger,
		level:  config.Level.ToSlogLevel(),
	}
}

// NewDefaultLogger creates a logger with sensible defaults.
func NewDefaultLogger() *Logger {
	return NewLogger(LoggerConfig{
		Level:         LevelInfo,
		Format:        "plain",
		Output:        os.Stderr,
		IncludeSource: false,
		Component:     "netex-validator",
	})
}

// NewJSONLogger creates a logger that outputs JSON format.
func NewJSONLogger(level LogLevel) *Logger {
	return NewLogger(LoggerConfig{
		Level:         level,
		Format:        "json",
		Output:        os.Stderr,
		IncludeSource: false,
		Component:     "netex-validator",
	})
}

// NewDebugLogger creates a logger with debug level and source information.
func NewDebugLogger() *Logger {
	return NewLogger(LoggerConfig{
		Level:         LevelDebug,
		Format:        "text",
		Output:        os.Stderr,
		IncludeSource: true,
		Component:     "netex-validator",
	})
}

// WithContext returns a logger with context values.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		l.With("context", ctx.Value("request_id")),
		l.level,
	}
}

// WithFile returns a logger with file context.
func (l *Logger) WithFile(filename string) *Logger {
	return &Logger{
		l.With("file", filename),
		l.level,
	}
}

// WithValidation returns a logger with validation context.
func (l *Logger) WithValidation(validationID, codespace string) *Logger {
	return &Logger{
		l.With(
			"validation_id", validationID,
			"codespace", codespace,
		),
		l.level,
	}
}

// WithRule returns a logger with validation rule context.
func (l *Logger) WithRule(ruleCode, ruleName string) *Logger {
	return &Logger{
		l.With(
			"rule_code", ruleCode,
			"rule_name", ruleName,
		),
		l.level,
	}
}

// WithError returns a logger with error context.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		l.With("error", err.Error()),
		l.level,
	}
}

// WithDuration returns a logger with duration context.
func (l *Logger) WithDuration(operation string, duration time.Duration) *Logger {
	return &Logger{
		l.With(
			"operation", operation,
			"duration_ms", duration.Milliseconds(),
		),
		l.level,
	}
}

// WithMetrics returns a logger with performance metrics.
func (l *Logger) WithMetrics(filesProcessed, issuesFound int, processingTime time.Duration) *Logger {
	return &Logger{
		l.With(
			"files_processed", filesProcessed,
			"issues_processed", issuesFound,
			"processing_time_ms", processingTime.Milliseconds(),
		),
		l.level,
	}
}

// ValidationStart logs the start of a validation operation.
func (l *Logger) ValidationStart(filename, codespace string) {
	l.Info("Starting validation",
		"file", filename,
		"codespace", codespace,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// ValidationComplete logs the completion of a validation operation.
func (l *Logger) ValidationComplete(filename string, duration time.Duration, issuesFound int, isValid bool) {
	l.Info("Validation completed",
		"file", filename,
		"duration_ms", duration.Milliseconds(),
		"issues_found", issuesFound,
		"is_valid", isValid,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// ValidationError logs a validation error.
func (l *Logger) ValidationError(filename string, err error) {
	l.Error("Validation error",
		"file", filename,
		"error", err.Error(),
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// RuleViolation logs a validation rule violation.
func (l *Logger) RuleViolation(filename, ruleCode, ruleName, message string, line int) {
	l.Warn("Rule violation",
		"file", filename,
		"rule_code", ruleCode,
		"rule_name", ruleName,
		"message", message,
		"line", line,
	)
}

// PerformanceWarning logs a performance warning.
func (l *Logger) PerformanceWarning(operation string, duration time.Duration, threshold time.Duration) {
	l.Warn("Performance warning",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"threshold_ms", threshold.Milliseconds(),
		"exceeded_by_ms", (duration - threshold).Milliseconds(),
	)
}

// BatchValidationStart logs the start of batch validation.
func (l *Logger) BatchValidationStart(fileCount int) {
	l.Info("Starting batch validation",
		"file_count", fileCount,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// BatchValidationComplete logs batch validation completion.
func (l *Logger) BatchValidationComplete(fileCount, validFiles, totalIssues int, duration time.Duration) {
	l.Info("Batch validation completed",
		"file_count", fileCount,
		"valid_files", validFiles,
		"invalid_files", fileCount-validFiles,
		"total_issues", totalIssues,
		"duration_ms", duration.Milliseconds(),
		"avg_duration_per_file_ms", duration.Milliseconds()/int64(fileCount),
	)
}

// ConfigurationLoaded logs successful configuration loading.
func (l *Logger) ConfigurationLoaded(configPath string, ruleCount int) {
	l.Info("Configuration loaded",
		"config_path", configPath,
		"rule_count", ruleCount,
	)
}

// SchemaValidationStart logs the start of schema validation.
func (l *Logger) SchemaValidationStart(filename string) {
	l.Debug("Starting schema validation", "file", filename)
}

// SchemaValidationComplete logs schema validation completion.
func (l *Logger) SchemaValidationComplete(filename string, duration time.Duration, valid bool) {
	l.Debug("Schema validation completed",
		"file", filename,
		"duration_ms", duration.Milliseconds(),
		"valid", valid,
	)
}

// XPathValidationStart logs the start of XPath rule validation.
func (l *Logger) XPathValidationStart(filename string, ruleCount int) {
	l.Debug("Starting XPath validation",
		"file", filename,
		"rule_count", ruleCount,
	)
}

// XPathValidationComplete logs XPath validation completion.
func (l *Logger) XPathValidationComplete(filename string, duration time.Duration, violationCount int) {
	l.Debug("XPath validation completed",
		"file", filename,
		"duration_ms", duration.Milliseconds(),
		"violations", violationCount,
	)
}

// MemoryUsage logs current memory usage statistics.
func (l *Logger) MemoryUsage(operation string, allocMB, sysMB float64) {
	l.Debug("Memory usage",
		"operation", operation,
		"alloc_mb", allocMB,
		"sys_mb", sysMB,
	)
}

// IsLevelEnabled checks if a log level is enabled.
func (l *Logger) IsLevelEnabled(level LogLevel) bool {
	return l.level <= level.ToSlogLevel()
}

// Global logger instance for convenience.
var (
	defaultLogger *Logger = NewDefaultLogger()
	loggerMu      sync.RWMutex
)

// SetDefaultLogger sets the global default logger.
func SetDefaultLogger(logger *Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	defaultLogger = logger
}

// GetDefaultLogger returns the global default logger.
func GetDefaultLogger() *Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return defaultLogger
}

// Convenience functions for global logger.

// Debug logs a debug message using the default logger.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message using the default logger.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message using the default logger.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message using the default logger.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// ValidationStart logs validation start using the default logger.
func ValidationStart(filename, codespace string) {
	defaultLogger.ValidationStart(filename, codespace)
}

// ValidationComplete logs validation completion using the default logger.
func ValidationComplete(filename string, duration time.Duration, issuesFound int, isValid bool) {
	defaultLogger.ValidationComplete(filename, duration, issuesFound, isValid)
}

// ValidationError logs validation error using the default logger.
func ValidationError(filename string, err error) {
	defaultLogger.ValidationError(filename, err)
}

// RuleViolation logs rule violation using the default logger.
func RuleViolation(filename, ruleCode, ruleName, message string, line int) {
	defaultLogger.RuleViolation(filename, ruleCode, ruleName, message, line)
}
