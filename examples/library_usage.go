package netexvalidator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validator"
)

// ExampleBasicValidation demonstrates the simplest way to validate a NetEX file.
func ExampleBasicValidation() {
	// Validate a NetEX file against the shipped EU rule set
	options := validator.DefaultValidationOptions().WithRuleSet("", "rulesets/eu.json")
	result, err := validator.ValidateFile("example.xml", options)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	if result.IsValid() {
		fmt.Println("File is valid!")
	} else {
		fmt.Printf("File has %d validation issues\n", len(result.ValidationReportEntries))
		for _, entry := range result.ValidationReportEntries {
			fmt.Printf("- %s: %s\n", entry.Name, entry.Message)
		}
	}
}

// ExampleValidationWithOptions demonstrates validation with custom options.
func ExampleValidationWithOptions() {
	// Create options with method chaining
	options := validator.DefaultValidationOptions().
		WithCodespace("NO").
		WithRuleSet("eu", "rulesets/eu.json").    // rule IDs like "eu/line/2"
		WithRuleSet("local", "local-rules.json"). // your own rules, as "local/<key>"
		WithVerbose(true).
		WithSkipSchema(false)

	result, err := validator.ValidateFile("norwegian_data.xml", options)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	summary := result.Summary()
	fmt.Printf("Validation completed in %v\n", summary.ProcessingTime)
	fmt.Printf("Files processed: %d\n", summary.FilesProcessed)
	fmt.Printf("Total issues: %d\n", summary.TotalIssues)
	fmt.Printf("Has errors: %t\n", summary.HasErrors)

	// Print issues by severity
	for severity, count := range summary.IssuesBySeverity {
		if count > 0 {
			fmt.Printf("%v issues: %d\n", severity, count)
		}
	}
}

// ExampleMemoryValidation demonstrates validating NetEX data from memory.
func ExampleMemoryValidation() {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.0">
	<PublicationTimestamp>2023-01-01T12:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="TEST:CompositeFrame:1" version="1">
			<Name>Test Frame</Name>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`)

	options := validator.DefaultValidationOptions().WithCodespace("TEST").WithRuleSet("", "rulesets/eu.json")
	result, err := validator.ValidateContent(xmlData, "in-memory.xml", options)
	if err != nil {
		log.Fatalf("Memory validation failed: %v", err)
	}

	fmt.Printf("Memory validation result: %s\n", result.String())
}

// ExampleBatchValidation demonstrates validating multiple files.
func ExampleBatchValidation() {
	files := []string{
		"data/file1.xml",
		"data/file2.xml",
		"data/file3.xml",
	}

	options := validator.DefaultValidationOptions().WithCodespace("BATCH").WithRuleSet("", "rulesets/eu.json")

	for i, file := range files {
		fmt.Printf("Validating file %d: %s\n", i+1, file)

		result, err := validator.ValidateFile(file, options)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		summary := result.Summary()
		fmt.Printf("  Result: %d issues found\n", summary.TotalIssues)

		if !result.IsValid() {
			fmt.Printf("  First 3 issues:\n")
			for i, entry := range result.ValidationReportEntries {
				if i >= 3 {
					break
				}
				fmt.Printf("    - %s\n", entry.Name)
			}
		}
	}
}

// ExampleOutputFormats demonstrates different output formats.
func ExampleOutputFormats() {
	result, err := validator.ValidateFile("example.xml", validator.DefaultValidationOptions().WithRuleSet("", "rulesets/eu.json"))
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// JSON output
	jsonData, err := result.ToJSON()
	if err != nil {
		log.Printf("JSON generation failed: %v", err)
	} else {
		fmt.Printf("JSON output (%d bytes):\n%s\n", len(jsonData), string(jsonData))
	}

	// HTML output
	htmlData, err := result.ToHTML()
	if err != nil {
		log.Printf("HTML generation failed: %v", err)
	} else {
		// Save HTML to file
		if err := os.WriteFile("validation_report.html", htmlData, 0600); err != nil {
			log.Printf("Failed to write HTML file: %v", err)
		} else {
			fmt.Printf("HTML report saved to validation_report.html (%d bytes)\n", len(htmlData))
		}
	}

	// String output
	fmt.Printf("String output:\n%s\n", result.String())
}

// ExampleErrorHandling demonstrates proper error handling patterns.
func ExampleErrorHandling() {
	options := validator.DefaultValidationOptions().WithCodespace("ERROR_TEST").WithRuleSet("", "rulesets/eu.json")

	result, err := validator.ValidateFile("nonexistent.xml", options)
	if err != nil {
		// Handle validation setup errors
		fmt.Printf("Validation setup error: %v\n", err)
		return
	}

	if result.Error != "" {
		// Handle validation runtime errors
		fmt.Printf("Validation runtime error: %s\n", result.Error)
	}

	// Check for specific validation issues
	if !result.IsValid() {
		criticalErrors := 0
		for _, entry := range result.ValidationReportEntries {
			if entry.Severity >= types.ERROR {
				criticalErrors++
			}
		}

		if criticalErrors > 0 {
			fmt.Printf("File has %d critical errors that must be fixed\n", criticalErrors)
		} else {
			fmt.Println("File has warnings but no critical errors")
		}
	}
}

// ExampleAdvancedValidation demonstrates advanced validation scenarios.
func ExampleAdvancedValidation() {
	// Validate with performance monitoring
	options := validator.DefaultValidationOptions().
		WithCodespace("ADVANCED").
		WithRuleSet("", "rulesets/eu.json").
		WithVerbose(true)

	result, err := validator.ValidateFile("large_file.xml", options)
	if err != nil {
		log.Fatalf("Advanced validation failed: %v", err)
	}

	summary := result.Summary()

	// Performance analysis
	fmt.Printf("Performance Analysis:\n")
	fmt.Printf("  Processing time: %v\n", summary.ProcessingTime)
	fmt.Printf("  Files processed: %d\n", summary.FilesProcessed)
	fmt.Printf("  Average time per file: %v\n",
		summary.ProcessingTime/time.Duration(summary.FilesProcessed))

	// Rule analysis
	fmt.Printf("\nRule Violations:\n")
	for rule, count := range result.NumberOfValidationEntriesPerRule {
		if count > 0 {
			fmt.Printf("  %s: %d violations\n", rule, count)
		}
	}

	// File analysis
	fmt.Printf("\nFile Analysis:\n")
	// NumberOfValidationEntriesPerFile is not available in this Go port; compute on demand
	issuesByFile := result.GetIssuesByFile()
	for filename, entries := range issuesByFile {
		fmt.Printf("  %s: %d issues\n", filename, len(entries))
	}

	// Generate detailed reports for different audiences
	if summary.HasErrors {
		// Technical report for developers
		fmt.Printf("\nTechnical Report:\n")
		for _, entry := range result.ValidationReportEntries {
			if entry.Severity >= types.ERROR {
				fmt.Printf("ERROR [%s:%d]: %s - %s\n",
					entry.FileName, entry.Location.LineNumber, entry.Name, entry.Message)
			}
		}

		// Summary report for managers
		fmt.Printf("\nExecutive Summary:\n")
		fmt.Printf("Data quality assessment shows %d issues requiring attention.\n",
			summary.TotalIssues)
		fmt.Printf("Critical errors: %d, Recommendations: %d\n",
			summary.IssuesBySeverity[types.ERROR], summary.IssuesBySeverity[types.INFO])
	}
}

// ExampleDirectoryValidation demonstrates validating all XML files in a directory.
func ExampleDirectoryValidation() {
	directory := "netex_data"
	options := validator.DefaultValidationOptions().WithCodespace("DIR").WithRuleSet("", "rulesets/eu.json")

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-XML files
		if filepath.Ext(path) != ".xml" {
			return nil
		}

		fmt.Printf("Validating: %s\n", path)
		result, err := validator.ValidateFile(path, options)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			return nil // Continue with other files
		}

		summary := result.Summary()
		if result.IsValid() {
			fmt.Printf("  ✓ Valid (%d informational messages)\n",
				summary.IssuesBySeverity[types.INFO])
		} else {
			fmt.Printf("  ✗ Invalid (%d errors, %d warnings)\n",
				summary.IssuesBySeverity[types.ERROR],
				summary.IssuesBySeverity[types.WARNING])
		}

		return nil
	})

	if err != nil {
		log.Printf("Directory walk failed: %v", err)
	}
}

// ExampleValidationPipeline demonstrates using validation as part of a data processing pipeline.
func ExampleValidationPipeline() {
	type ProcessingResult struct {
		Filename string
		Valid    bool
		Issues   int
		Summary  string
	}

	files := []string{"input1.xml", "input2.xml", "input3.xml"}
	results := make([]ProcessingResult, 0, len(files))
	options := validator.DefaultValidationOptions().WithCodespace("PIPELINE").WithRuleSet("", "rulesets/eu.json")

	// Validation phase
	for _, file := range files {
		result, err := validator.ValidateFile(file, options)
		if err != nil {
			results = append(results, ProcessingResult{
				Filename: file,
				Valid:    false,
				Issues:   -1,
				Summary:  fmt.Sprintf("Validation failed: %v", err),
			})
			continue
		}

		summary := result.Summary()
		results = append(results, ProcessingResult{
			Filename: file,
			Valid:    result.IsValid(),
			Issues:   summary.TotalIssues,
			Summary:  result.String(),
		})
	}

	// Processing phase
	validFiles := 0
	for _, result := range results {
		if result.Valid {
			fmt.Printf("✓ Processing %s (valid)\n", result.Filename)
			// Here you would process the valid file
			validFiles++
		} else {
			fmt.Printf("✗ Skipping %s (%d issues)\n", result.Filename, result.Issues)
			// Log details for later review
			log.Printf("Validation issues for %s: %s", result.Filename, result.Summary)
		}
	}

	fmt.Printf("\nPipeline completed: %d/%d files processed successfully\n",
		validFiles, len(files))
}

// ExampleCustomValidationWorkflow demonstrates a complete custom validation workflow.
func ExampleCustomValidationWorkflow() {
	// Configuration
	type Config struct {
		InputDir     string
		OutputDir    string
		Codespace    string
		StrictMode   bool
		GenerateHTML bool
	}

	config := Config{
		InputDir:     "input",
		OutputDir:    "reports",
		Codespace:    "WORKFLOW",
		StrictMode:   true,
		GenerateHTML: true,
	}

	// Setup validation options
	options := validator.DefaultValidationOptions().
		WithCodespace(config.Codespace).
		WithRuleSet("", "rulesets/eu.json")

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Find all XML files
	files := []string{}
	err := filepath.Walk(config.InputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".xml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to scan input directory: %v", err)
	}

	fmt.Printf("Found %d XML files to validate\n", len(files))

	// Validate each file
	for i, file := range files {
		fmt.Printf("[%d/%d] Validating %s\n", i+1, len(files), file)

		result, err := validator.ValidateFile(file, options)
		if err != nil {
			log.Printf("Validation failed for %s: %v", file, err)
			continue
		}

		// Generate reports
		baseName := filepath.Base(file)
		nameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]

		// JSON report
		jsonData, err := result.ToJSON()
		if err != nil {
			log.Printf("JSON generation failed for %s: %v", file, err)
			continue
		}
		jsonFile := filepath.Join(config.OutputDir, nameWithoutExt+"_report.json")
		if err := os.WriteFile(jsonFile, jsonData, 0600); err != nil {
			log.Printf("Failed to write JSON file for %s: %v", file, err)
			continue
		}

		// HTML report (if enabled)
		if config.GenerateHTML {
			htmlData, err := result.ToHTML()
			if err != nil {
				log.Printf("HTML generation failed for %s: %v", file, err)
			} else {
				htmlFile := filepath.Join(config.OutputDir, nameWithoutExt+"_report.html")
				if err := os.WriteFile(htmlFile, htmlData, 0600); err != nil {
					log.Printf("Failed to write HTML file for %s: %v", file, err)
				}
			}
		}

		// Summary
		summary := result.Summary()
		status := "VALID"
		if !result.IsValid() {
			status = "INVALID"
		}

		fmt.Printf("  Result: %s (%d issues, %v processing time)\n",
			status, summary.TotalIssues, summary.ProcessingTime)
	}

	fmt.Printf("Validation workflow completed. Reports saved to %s\n", config.OutputDir)
}
