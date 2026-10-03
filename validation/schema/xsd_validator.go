package schema

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/logging"
	errors "github.com/nicolo-ribaudo/netex-validator/reporting"
)

const (
	// Constants for repeated strings
	xsdLatestVersion = "1.16"
)

// XSDValidator provides XML Schema (XSD) validation capabilities for NetEX files.
type XSDValidator struct {
	schemaCache      map[string]*XSDSchema
	schemaManager    *SchemaManager
	cacheDir         string
	client           *http.Client
	mutex            sync.RWMutex
	logger           *logging.Logger
	allowNetwork     bool
	cacheExpiryHours int
	// useLibxml2 controls whether to attempt libxml2-backed validation when available
	useLibxml2 bool
}

// XSDSchema represents a cached XSD schema with metadata.
type XSDSchema struct {
	Version   string
	Content   []byte
	URL       string
	CachedAt  time.Time
	ExpiresAt time.Time
}

// XSDValidationOptions configures XSD validation behavior.
type XSDValidationOptions struct {
	// AllowNetworkDownload enables downloading schemas from remote URLs
	AllowNetworkDownload bool
	// CacheDirectory specifies where to cache downloaded schemas
	CacheDirectory string
	// CacheExpiryHours sets how long cached schemas remain valid
	CacheExpiryHours int
	// StrictMode fails validation on any schema-related errors
	StrictMode bool
	// MaxSchemaSize limits the size of schema files (in bytes)
	MaxSchemaSize int64
	// HttpTimeoutSeconds controls the schema download timeout in seconds
	HttpTimeoutSeconds int
	// UseLibxml2 enables libxml2-backed XSD validation when the build has libxml2 bindings
	UseLibxml2 bool
}

// DefaultXSDValidationOptions returns sensible defaults for XSD validation.
func DefaultXSDValidationOptions() *XSDValidationOptions {
	return &XSDValidationOptions{
		AllowNetworkDownload: false,
		CacheDirectory:       filepath.Join(os.TempDir(), "netex-schemas"),
		CacheExpiryHours:     24 * 7, // 1 week
		StrictMode:           false,
		MaxSchemaSize:        50 * 1024 * 1024, // 50MB
		HttpTimeoutSeconds:   30,
		UseLibxml2:           false,
	}
}

// NewXSDValidator creates a new XSD validator instance.
func NewXSDValidator(options *XSDValidationOptions) (*XSDValidator, error) {
	if options == nil {
		options = DefaultXSDValidationOptions()
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(options.CacheDirectory, 0750); err != nil {
		return nil, fmt.Errorf("failed to create schema cache directory: %w", err)
	}

	// Create schema manager
	schemaManager := NewSchemaManager(options.CacheDirectory)
	schemaManager.SetNetworkEnabled(options.AllowNetworkDownload)
	schemaManager.SetMaxCacheAge(time.Duration(options.CacheExpiryHours) * time.Hour)

	// Set timeout for schema downloads - use the configured timeout or default to 10s
	schemaTimeout := time.Duration(options.HttpTimeoutSeconds) * time.Second
	if schemaTimeout <= 0 {
		schemaTimeout = 10 * time.Second // Much faster default than 30s
	}
	schemaManager.SetHttpTimeout(schemaTimeout)

	timeout := time.Duration(options.HttpTimeoutSeconds) * time.Second
	if options.HttpTimeoutSeconds <= 0 {
		timeout = 30 * time.Second
	}
	validator := &XSDValidator{
		schemaCache:   make(map[string]*XSDSchema),
		schemaManager: schemaManager,
		cacheDir:      options.CacheDirectory,
		client: &http.Client{
			Timeout: timeout,
		},
		logger:           logging.GetDefaultLogger(),
		allowNetwork:     options.AllowNetworkDownload,
		cacheExpiryHours: options.CacheExpiryHours,
		useLibxml2:       options.UseLibxml2,
	}

	// Load cached schemas from disk (legacy support)
	if err := validator.loadCachedSchemas(); err != nil {
		validator.logger.Warn("Failed to load cached schemas", "error", err.Error())
	}

	return validator, nil
}

// NetEX schema URLs for different versions
var netexSchemaURLs = map[string]string{
	"1.0":  "http://www.netex.org.uk/schema/1.0/xsd/NeTEx_publication.xsd",
	"1.1":  "http://www.netex.org.uk/schema/1.1/xsd/NeTEx_publication.xsd",
	"1.2":  "http://www.netex.org.uk/schema/1.2/xsd/NeTEx_publication.xsd",
	"1.3":  "http://www.netex.org.uk/schema/1.3/xsd/NeTEx_publication.xsd",
	"1.4":  "http://www.netex.org.uk/schema/1.4/xsd/NeTEx_publication.xsd",
	"1.15": "http://www.netex.org.uk/schema/1.15/xsd/NeTEx_publication.xsd",
	"1.16": "http://www.netex.org.uk/schema/1.16/xsd/NeTEx_publication.xsd",
}

// ValidateXML performs XSD validation on the provided XML content.
func (v *XSDValidator) ValidateXML(xmlContent []byte, filename string) ([]*errors.ValidationError, error) {
	logger := v.logger.WithFile(filename)
	logger.Debug("Starting XSD validation")

	// Detect NetEX version from XML content using the schema manager
	version, err := v.schemaManager.DetectSchemaVersion(xmlContent)
	if err != nil {
		logger.Warn("Error detecting NetEX version", "error", err.Error(), "detected_version", version)
	}

	logger.Debug("Detected NetEX version", "version", version)

	// Get schema using the schema manager
	var cachedSchema *CachedSchema
	var schema *XSDSchema
	if v.allowNetwork {
		cachedSchema, err = v.schemaManager.GetSchema(version)
		if err != nil {
			logger.Warn("Failed to get schema from schema manager; continuing with basic checks", "error", err.Error())
		} else {
			// Convert CachedSchema to XSDSchema for compatibility
			schema = &XSDSchema{
				Version:   cachedSchema.Version,
				Content:   cachedSchema.Content,
				URL:       cachedSchema.URL,
				CachedAt:  cachedSchema.CachedAt,
				ExpiresAt: cachedSchema.LastUsed.Add(24 * time.Hour), // Simple expiry logic
			}
		}
	} else {
		logger.Debug("Network download disabled; performing basic schema checks only")
	}

	// Perform XSD validation
	validationErrors, err := v.validateAgainstSchema(xmlContent, schema, filename)
	if err != nil {
		return nil, fmt.Errorf("XSD validation failed: %w", err)
	}

	logger.Debug("XSD validation completed", "errors_found", len(validationErrors))
	return validationErrors, nil
}

// detectNetexVersion extracts the NetEX version from XML content.
func (v *XSDValidator) detectNetexVersion(xmlContent []byte) (string, error) {
	// Parse XML and look at the first start element for a version attribute
	decoder := xml.NewDecoder(bytes.NewReader(xmlContent))
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to parse XML: %w", err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			// Check attributes for version
			for _, attr := range start.Attr {
				if strings.EqualFold(attr.Name.Local, "version") {
					version := attr.Value
					if _, exists := netexSchemaURLs[version]; exists {
						return version, nil
					}
					return v.mapToKnownVersion(version), nil
				}
			}
			// If no version attribute, check namespace presence
			for _, attr := range start.Attr {
				if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
					if strings.Contains(attr.Value, "http://www.netex.org.uk/netex") {
						return xsdLatestVersion, fmt.Errorf("NetEX namespace found but version unclear")
					}
				}
			}
			// Root element processed; stop
			break
		}
	}
	return "", fmt.Errorf("could not determine NetEX version from XML content")
}

// mapToKnownVersion maps detected versions to known schema versions.
func (v *XSDValidator) mapToKnownVersion(detectedVersion string) string {
	// Prefer latest known version for the same major when minor is unknown/newer
	// Fallback to latest overall when major doesn't match
	type ver struct{ major, minor int }
	parse := func(s string) ver {
		parts := strings.SplitN(s, ".", 3)
		mv, nv := 0, 0
		if len(parts) > 0 {
			if _, err := fmt.Sscanf(parts[0], "%d", &mv); err != nil {
				v.logger.Warn("Failed to parse major version", "version", parts[0], "error", err.Error())
			}
		}
		if len(parts) > 1 {
			if _, err := fmt.Sscanf(parts[1], "%d", &nv); err != nil {
				v.logger.Warn("Failed to parse minor version", "version", parts[1], "error", err.Error())
			}
		}
		return ver{mv, nv}
	}

	detected := parse(detectedVersion)
	// Known versions in ascending order by minor within major 1
	knownOrdered := []string{"1.0", "1.1", "1.2", "1.3", "1.4", "1.15", "1.16"}

	// First collect all known for same major
	var sameMajor []string
	for _, kv := range knownOrdered {
		if parse(kv).major == detected.major {
			sameMajor = append(sameMajor, kv)
		}
	}
	if len(sameMajor) > 0 {
		// Pick the highest minor <= detected; otherwise highest known for that major
		candidate := sameMajor[len(sameMajor)-1]
		for _, kv := range sameMajor {
			k := parse(kv)
			if k.minor <= detected.minor {
				candidate = kv
			}
		}
		return candidate
	}

	// Fallback: latest known overall
	return xsdLatestVersion
}

// validateAgainstSchema performs the actual XSD validation.
func (v *XSDValidator) validateAgainstSchema(xmlContent []byte, schema *XSDSchema, filename string) ([]*errors.ValidationError, error) {
	var validationErrors []*errors.ValidationError

	// Try schema manager validation first if we have a schema
	if schema != nil {
		// Convert XSDSchema to CachedSchema for schema manager
		cachedSchema := &CachedSchema{
			Version:  schema.Version,
			Content:  schema.Content,
			URL:      schema.URL,
			CachedAt: schema.CachedAt,
			LastUsed: time.Now(),
		}

		// Use schema manager's validation
		if err := v.schemaManager.ValidateWithSchema(xmlContent, cachedSchema); err != nil {
			// Schema manager validation failed, convert to validation error
			validationErrors = append(validationErrors,
				errors.NewSchemaValidationError(filename, 1, err.Error()))
		} else {
			v.logger.Debug("Schema manager validation completed successfully")
		}
	}

	// If libxml2 backend is requested, try it as well
	if v.useLibxml2 && schema != nil {
		if libxml2Errs, err := v.validateWithLibxml2(xmlContent, schema, filename); err == nil && libxml2Errs != nil {
			validationErrors = append(validationErrors, libxml2Errs...)
		} else if err != nil {
			v.logger.Warn("libxml2 validation unavailable or failed", "error", err.Error())
		}
	}

	// Perform basic structural validation regardless
	basicErrors := v.performBasicValidation(xmlContent, filename)
	validationErrors = append(validationErrors, basicErrors...)

	v.logger.Debug("XSD validation completed", "errors", len(validationErrors))
	return validationErrors, nil
}

// performBasicValidation performs basic structural validation
func (v *XSDValidator) performBasicValidation(xmlContent []byte, filename string) []*errors.ValidationError {
	var validationErrors []*errors.ValidationError

	// 1. Check for required root element - only when a NetEX namespace is present
	hasNetexNs := bytes.Contains(xmlContent, []byte("http://www.netex.org.uk/netex")) ||
		bytes.Contains(xmlContent, []byte("http://netex-cen.eu/netex"))
	missingRoot := hasNetexNs && !bytes.Contains(xmlContent, []byte("PublicationDelivery"))
	if missingRoot {
		validationErrors = append(validationErrors,
			errors.NewSchemaValidationError(filename, 1,
				"Missing required root element 'PublicationDelivery'"))
		// If the root element is missing, return early to avoid cascading errors
		return validationErrors
	}

	// 2. Check for required namespace
	if !hasNetexNs {
		validationErrors = append(validationErrors,
			errors.NewSchemaValidationError(filename, 1,
				"Missing required NetEX namespace (expected http://www.netex.org.uk/netex or http://netex-cen.eu/netex)"))
	}

	// 3. Check for basic required elements
	requiredElements := []string{
		"PublicationTimestamp",
		"ParticipantRef",
		"dataObjects",
	}

	for _, element := range requiredElements {
		if !bytes.Contains(xmlContent, []byte("<"+element)) {
			validationErrors = append(validationErrors,
				errors.NewSchemaValidationError(filename, 0,
					fmt.Sprintf("Missing required element '%s'", element)))
		}
	}

	return validationErrors
}

// loadCachedSchemas loads previously cached schemas from disk.
func (v *XSDValidator) loadCachedSchemas() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	entries, err := os.ReadDir(v.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache dir: %w", err)
	}
	now := time.Now()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Expect filenames like NeTEx_publication_<version>.xsd
		if !strings.HasPrefix(name, "NeTEx_publication_") || !strings.HasSuffix(name, ".xsd") {
			continue
		}
		version := strings.TrimSuffix(strings.TrimPrefix(name, "NeTEx_publication_"), ".xsd")
		path := filepath.Join(v.cacheDir, name)
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(path) //nolint:gosec // Path is constructed safely using filepath.Join
		if err != nil {
			continue
		}
		cachedAt := fi.ModTime()
		expiresAt := cachedAt.Add(time.Duration(v.cacheExpiryHours) * time.Hour)
		v.schemaCache[version] = &XSDSchema{
			Version:   version,
			Content:   content,
			URL:       "",
			CachedAt:  cachedAt,
			ExpiresAt: expiresAt,
		}
		if expiresAt.After(now) {
			v.logger.Debug("Loaded cached schema", "version", version, "path", path)
		}
	}
	return nil
}

// findStringSubmatch is a simple regex-like function for finding version patterns.
// In a real implementation, you'd use the regexp package.
func findStringSubmatch(pattern, text string) []string {
	// This is a simplified implementation for version detection
	// A real implementation would use proper regex matching

	if strings.Contains(pattern, `version="([0-9]+\.[0-9]+[0-9]*)"`) {
		// Look for version="X.Y" pattern
		start := strings.Index(text, `version="`)
		if start == -1 {
			return nil
		}
		start += len(`version="`)
		end := strings.Index(text[start:], `"`)
		if end == -1 {
			return nil
		}
		version := text[start : start+end]
		return []string{`version="` + version + `"`, version}
	}

	return nil
}

// GetSupportedVersions returns the list of supported NetEX versions.
func (v *XSDValidator) GetSupportedVersions() []string {
	versions := make([]string, 0, len(netexSchemaURLs))
	for version := range netexSchemaURLs {
		versions = append(versions, version)
	}
	return versions
}

// ClearCache clears the in-memory schema cache.
func (v *XSDValidator) ClearCache() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.schemaCache = make(map[string]*XSDSchema)
	v.logger.Info("Schema cache cleared")
}

// GetCacheStats returns statistics about the schema cache.
func (v *XSDValidator) GetCacheStats() map[string]interface{} {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	stats := map[string]interface{}{
		"cached_schemas":  len(v.schemaCache),
		"cache_directory": v.cacheDir,
	}

	for version, schema := range v.schemaCache {
		stats[fmt.Sprintf("schema_%s_size", version)] = len(schema.Content)
		stats[fmt.Sprintf("schema_%s_cached_at", version)] = schema.CachedAt
		stats[fmt.Sprintf("schema_%s_expires_at", version)] = schema.ExpiresAt
	}

	return stats
}
