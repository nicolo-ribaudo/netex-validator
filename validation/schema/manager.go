package schema

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/nicolo-ribaudo/netex-validator/logging"
	"github.com/nicolo-ribaudo/netex-validator/utils"
)

const (
	// Constants for repeated strings
	defaultVersion = "1.4"
	latestVersion  = "1.16"
)

// SchemaManager manages NetEX schema download, caching, and validation
type SchemaManager struct {
	cacheDir      string
	httpClient    *utils.OptimizedHTTPClient
	schemaMutex   sync.RWMutex
	schemaCache   map[string]*CachedSchema
	enableNetwork bool
	maxCacheAge   time.Duration
}

// CachedSchema represents a cached XSD schema
type CachedSchema struct {
	FilePath string
	Version  string
	URL      string
	Content  []byte
	CachedAt time.Time
	LastUsed time.Time
}

// NetEXSchemaInfo contains information about NetEX schema versions
type NetEXSchemaInfo struct {
	Version    string
	SchemaURLs map[string]string // schema name -> URL
}

// DefaultSchemaVersions contains known NetEX schema versions and their URLs
var DefaultSchemaVersions = map[string]*NetEXSchemaInfo{
	"1.0": {
		Version: "1.0",
		SchemaURLs: map[string]string{
			"netex_publication": "http://www.netex.org.uk/schema/1.0/xsd/netex_publication.xsd",
			"NeTEx_publication": "http://www.netex.org.uk/schema/1.0/xsd/NeTEx_publication.xsd",
		},
	},
	"1.1": {
		Version: "1.1",
		SchemaURLs: map[string]string{
			"netex_publication": "http://www.netex.org.uk/schema/1.1/xsd/netex_publication.xsd",
			"NeTEx_publication": "http://www.netex.org.uk/schema/1.1/xsd/NeTEx_publication.xsd",
		},
	},
	"1.2.2": {
		Version: "1.2.2",
		SchemaURLs: map[string]string{
			"netex_publication": "http://www.netex.org.uk/schema/1.2.2/xsd/netex_publication.xsd",
			"NeTEx_publication": "http://www.netex.org.uk/schema/1.2.2/xsd/NeTEx_publication.xsd",
		},
	},
	"1.4": {
		Version: "1.4",
		SchemaURLs: map[string]string{
			"netex_publication": "https://raw.githubusercontent.com/NeTEx-CEN/NeTEx/master/xsd/netex_publication.xsd",
			"NeTEx_publication": "https://raw.githubusercontent.com/NeTEx-CEN/NeTEx/master/xsd/NeTEx_publication.xsd",
		},
	},
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(cacheDir string) *SchemaManager {
	if cacheDir == "" {
		// Use default cache directory
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".netex-validator", "schemas")
	}

	// Ensure cache directory exists
	_ = os.MkdirAll(cacheDir, 0o750)

	// Create optimized HTTP client for schema downloads
	httpClient := utils.NewOptimizedHTTPClient(utils.DefaultHTTPClientOptions())

	return &SchemaManager{
		cacheDir:      cacheDir,
		httpClient:    httpClient,
		schemaCache:   make(map[string]*CachedSchema),
		enableNetwork: true,
		maxCacheAge:   24 * time.Hour, // Cache schemas for 24 hours
	}
}

// SetNetworkEnabled enables or disables network access for schema download
func (sm *SchemaManager) SetNetworkEnabled(enabled bool) {
	sm.enableNetwork = enabled
}

// SetMaxCacheAge sets the maximum age for cached schemas
func (sm *SchemaManager) SetMaxCacheAge(maxAge time.Duration) {
	sm.maxCacheAge = maxAge
}

// SetHttpTimeout sets the HTTP timeout for schema downloads
func (sm *SchemaManager) SetHttpTimeout(timeout time.Duration) {
	// Create new optimized client with custom timeout
	opts := utils.DefaultHTTPClientOptions()
	opts.Timeout = timeout
	sm.httpClient = utils.NewOptimizedHTTPClient(opts)
}

// DetectSchemaVersion detects the NetEX schema version from XML content
func (sm *SchemaManager) DetectSchemaVersion(xmlContent []byte) (string, error) {
	// Parse XML to detect schema version
	doc, err := xmlquery.Parse(strings.NewReader(string(xmlContent)))
	if err != nil {
		return "", fmt.Errorf("failed to parse XML: %w", err)
	}

	// Look for version attribute on PublicationDelivery
	versionNode := xmlquery.FindOne(doc, "//@version")
	if versionNode != nil {
		version := versionNode.InnerText()
		if version != "" {
			return normalizeVersion(version), nil
		}
	}

	// Look for schema location hints
	schemaLocationNode := xmlquery.FindOne(doc, "//@xsi:schemaLocation")
	if schemaLocationNode != nil {
		schemaLocation := schemaLocationNode.InnerText()
		version := extractVersionFromSchemaLocation(schemaLocation)
		if version != "" {
			return version, nil
		}
	}

	// Look for xmlns declarations that might contain version info
	rootNode := xmlquery.FindOne(doc, "/*")
	if rootNode != nil {
		for _, attr := range rootNode.Attr {
			if strings.Contains(attr.Name.Local, "xmlns") && strings.Contains(attr.Value, "netex") {
				version := extractVersionFromNamespace(attr.Value)
				if version != "" {
					return version, nil
				}
			}
		}
	}

	// Default to latest known version if detection fails
	return defaultVersion, nil
}

// GetSchema retrieves a schema for the given version, downloading if necessary
func (sm *SchemaManager) GetSchema(version string) (*CachedSchema, error) {
	sm.schemaMutex.Lock()
	defer sm.schemaMutex.Unlock()

	cacheKey := fmt.Sprintf("netex_%s", version)

	// Check if schema is already cached and fresh
	if cached, exists := sm.schemaCache[cacheKey]; exists {
		if time.Since(cached.CachedAt) < sm.maxCacheAge {
			cached.LastUsed = time.Now()
			return cached, nil
		}
	}

	// Try to load from disk cache
	if schema, err := sm.loadFromDiskCache(version); err == nil {
		sm.schemaCache[cacheKey] = schema
		schema.LastUsed = time.Now()
		return schema, nil
	}

	// Download schema if network is enabled
	if sm.enableNetwork {
		if schema, err := sm.downloadSchema(version); err == nil {
			sm.schemaCache[cacheKey] = schema
			schema.LastUsed = time.Now()
			return schema, nil
		}
	}

	// If all else fails, try to find any cached version
	if cached, exists := sm.schemaCache[cacheKey]; exists {
		cached.LastUsed = time.Now()
		return cached, nil
	}

	return nil, fmt.Errorf("no schema available for version %s", version)
}

// loadFromDiskCache loads a schema from disk cache
func (sm *SchemaManager) loadFromDiskCache(version string) (*CachedSchema, error) {
	filename := fmt.Sprintf("netex_%s.xsd", sanitizeVersion(version))
	filePath := filepath.Join(sm.cacheDir, filename)

	// Check if file exists and is not too old
	if info, err := os.Stat(filePath); err == nil {
		if time.Since(info.ModTime()) < sm.maxCacheAge {
			content, err := os.ReadFile(filePath) //nolint:gosec // Path is constructed safely using filepath.Join
			if err != nil {
				return nil, err
			}

			return &CachedSchema{
				FilePath: filePath,
				Version:  version,
				Content:  content,
				CachedAt: info.ModTime(),
				LastUsed: time.Now(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no valid cached schema for version %s", version)
}

// downloadSchema downloads a schema from the internet
func (sm *SchemaManager) downloadSchema(version string) (*CachedSchema, error) {
	schemaInfo, exists := DefaultSchemaVersions[version]
	if !exists {
		// Try to find the closest supported version
		mappedVersion := sm.mapToSupportedVersion(version)
		if mappedVersion != version {
			schemaInfo = DefaultSchemaVersions[mappedVersion]
		}

		if schemaInfo == nil {
			return nil, fmt.Errorf("no supported schema version for %s (mapped to %s)", version, mappedVersion)
		}
	}

	// Try different schema URLs
	var lastErr error
	for _, schemaURL := range schemaInfo.SchemaURLs {
		content, err := sm.downloadFromURL(schemaURL)
		if err != nil {
			lastErr = err
			continue
		}

		// Save to disk cache
		filename := fmt.Sprintf("netex_%s.xsd", sanitizeVersion(version))
		filePath := filepath.Join(sm.cacheDir, filename)

		if err := os.WriteFile(filePath, content, 0o600); err != nil {
			// Log warning but continue
			logging.GetDefaultLogger().Warn("Failed to cache schema", "path", filePath, "error", err)
		}

		return &CachedSchema{
			FilePath: filePath,
			Version:  version,
			URL:      schemaURL,
			Content:  content,
			CachedAt: time.Now(),
			LastUsed: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("failed to download schema for version %s: %w", version, lastErr)
}

// downloadFromURL downloads content from a URL using the optimized HTTP client
func (sm *SchemaManager) downloadFromURL(url string) ([]byte, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use optimized HTTP client with retry logic
	resp, err := sm.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to download schema from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Basic validation - ensure it looks like an XSD
	if !strings.Contains(string(content), "xmlns:xs=") && !strings.Contains(string(content), "xmlns:xsd=") {
		return nil, fmt.Errorf("downloaded content does not appear to be an XSD schema")
	}

	return content, nil
}

// ValidateWithSchema validates XML content against a schema
func (sm *SchemaManager) ValidateWithSchema(xmlContent []byte, schema *CachedSchema) error {
	// This would integrate with the actual XSD validation implementation
	// For now, return nil to indicate successful validation
	// In a real implementation, this would use libxml2 or similar

	if schema == nil {
		return fmt.Errorf("no schema provided for validation")
	}

	if len(xmlContent) == 0 {
		return fmt.Errorf("no XML content provided for validation")
	}

	// Placeholder for actual XSD validation
	// This would be replaced with real libxml2 validation
	return nil
}

// ClearCache clears all cached schemas
func (sm *SchemaManager) ClearCache() error {
	sm.schemaMutex.Lock()
	defer sm.schemaMutex.Unlock()

	// Clear memory cache
	sm.schemaCache = make(map[string]*CachedSchema)

	// Clear disk cache
	return sm.clearDiskCache()
}

// clearDiskCache removes all cached schema files from disk
func (sm *SchemaManager) clearDiskCache() error {
	entries, err := os.ReadDir(sm.cacheDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".xsd") {
			filePath := filepath.Join(sm.cacheDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				// Log warning but continue
				logging.GetDefaultLogger().Warn("Failed to remove cached schema", "path", filePath, "error", err)
			}
		}
	}

	return nil
}

// GetCacheStats returns statistics about the schema cache
func (sm *SchemaManager) GetCacheStats() map[string]interface{} {
	sm.schemaMutex.RLock()
	defer sm.schemaMutex.RUnlock()

	stats := map[string]interface{}{
		"cacheDir":       sm.cacheDir,
		"cachedSchemas":  len(sm.schemaCache),
		"networkEnabled": sm.enableNetwork,
		"maxCacheAge":    sm.maxCacheAge.String(),
	}

	// Add HTTP client stats
	if sm.httpClient != nil {
		httpStats := sm.httpClient.GetStats()
		stats["httpClient"] = httpStats
	}

	var schemaList []map[string]interface{}
	for key, schema := range sm.schemaCache {
		schemaList = append(schemaList, map[string]interface{}{
			"key":      key,
			"version":  schema.Version,
			"cachedAt": schema.CachedAt,
			"lastUsed": schema.LastUsed,
			"size":     len(schema.Content),
		})
	}
	stats["schemas"] = schemaList

	return stats
}

// Close closes the schema manager and cleans up resources
func (sm *SchemaManager) Close() error {
	sm.schemaMutex.Lock()
	defer sm.schemaMutex.Unlock()

	// Close HTTP client and clean up connections
	if sm.httpClient != nil {
		sm.httpClient.Close()
		sm.httpClient = nil
	}

	// Clear memory cache
	sm.schemaCache = make(map[string]*CachedSchema)

	return nil
}

// Helper functions

// normalizeVersion normalizes a version string
func normalizeVersion(version string) string {
	// Remove common prefixes and suffixes
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	// Handle semantic versioning
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?`)
	if matches := versionRegex.FindStringSubmatch(version); matches != nil {
		if matches[3] != "" {
			return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
		}
		return fmt.Sprintf("%s.%s", matches[1], matches[2])
	}

	return version
}

// extractVersionFromSchemaLocation extracts version from schema location
func extractVersionFromSchemaLocation(schemaLocation string) string {
	// Look for version patterns in schema URLs
	versionRegex := regexp.MustCompile(`/(\d+\.\d+(?:\.\d+)?)/`)
	if matches := versionRegex.FindStringSubmatch(schemaLocation); matches != nil {
		return matches[1]
	}
	return ""
}

// extractVersionFromNamespace extracts version from XML namespace
func extractVersionFromNamespace(namespace string) string {
	// Look for version patterns in namespace URLs
	versionRegex := regexp.MustCompile(`/(\d+\.\d+(?:\.\d+)?)/?$`)
	if matches := versionRegex.FindStringSubmatch(namespace); matches != nil {
		return matches[1]
	}
	return ""
}

// sanitizeVersion sanitizes a version string for use in filenames
func sanitizeVersion(version string) string {
	// Replace characters that are not safe for filenames
	safe := regexp.MustCompile(`[^a-zA-Z0-9.\-_]`)
	return safe.ReplaceAllString(version, "_")
}

// mapToSupportedVersion maps unsupported versions to the closest supported version
func (sm *SchemaManager) mapToSupportedVersion(version string) string {
	// Normalize the version first
	normalized := normalizeVersion(version)

	// Define supported versions in order
	supportedVersions := []string{"1.0", "1.1", "1.2.2", "1.4", "1.15", "1.16"}

	// If it's already supported, return as-is
	for _, supported := range supportedVersions {
		if normalized == supported {
			return supported
		}
	}

	// Parse version components
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?`)
	matches := versionRegex.FindStringSubmatch(normalized)
	if matches == nil {
		// Can't parse version, default to latest
		return "1.16"
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	// Handle version 1.x mapping
	if major == 1 {
		switch {
		case minor <= 0:
			return "1.0"
		case minor <= 1:
			return "1.1"
		case minor <= 2:
			return "1.2.2"
		case minor <= 4:
			return "1.4"
		case minor <= 15:
			return "1.15"
		default:
			return "1.16"
		}
	}

	// For other major versions, default to latest
	return latestVersion
}
