package validator

import (
	"path/filepath"
	"testing"
)

// BenchmarkValidateFile_ValidMinimal benchmarks validation of a valid minimal NetEX file.
func BenchmarkValidateFile_ValidMinimal(b *testing.B) {
	testFile := filepath.Join("..", "..", "testdata", "valid_minimal.xml")
	options := testOptions(b).WithCodespace("BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateFile(testFile, options)
		if err != nil {
			b.Fatalf("ValidateFile() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateFile_InvalidMissingElements benchmarks validation of a file with missing elements.
func BenchmarkValidateFile_InvalidMissingElements(b *testing.B) {
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).WithCodespace("BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateFile(testFile, options)
		if err != nil {
			b.Fatalf("ValidateFile() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateFile_InvalidTransportModes benchmarks validation of a file with invalid transport modes.
func BenchmarkValidateFile_InvalidTransportModes(b *testing.B) {
	testFile := filepath.Join("..", "..", "testdata", "invalid_transport_modes.xml")
	options := testOptions(b).WithCodespace("BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateFile(testFile, options)
		if err != nil {
			b.Fatalf("ValidateFile() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateFile_WithSchemaSkip benchmarks validation with schema validation disabled.
func BenchmarkValidateFile_WithSchemaSkip(b *testing.B) {
	testFile := filepath.Join("..", "..", "testdata", "valid_minimal.xml")
	options := testOptions(b).
		WithCodespace("BENCH").
		WithSkipSchema(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateFile(testFile, options)
		if err != nil {
			b.Fatalf("ValidateFile() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateFile_Verbose benchmarks validation with verbose output enabled.
func BenchmarkValidateFile_Verbose(b *testing.B) {
	testFile := filepath.Join("..", "..", "testdata", "valid_minimal.xml")
	options := testOptions(b).
		WithCodespace("BENCH").
		WithVerbose(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateFile(testFile, options)
		if err != nil {
			b.Fatalf("ValidateFile() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateFile_StopOnFirstError benchmarks validation with stop-on-first-error enabled.
func BenchmarkValidateFile_StopOnFirstError(b *testing.B) {
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).
		WithCodespace("BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateFile(testFile, options)
		if err != nil {
			b.Fatalf("ValidateFile() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateData_ValidMinimal benchmarks in-memory validation of valid NetEX data.
func BenchmarkValidateData_ValidMinimal(b *testing.B) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.0">
	<PublicationTimestamp>2023-01-01T12:00:00</PublicationTimestamp>
	<ParticipantRef>BENCH</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="BENCH:CompositeFrame:1" version="1">
			<Name>Benchmark Test Frame</Name>
			<ServiceFrame id="BENCH:ServiceFrame:1" version="1">
				<Name>Benchmark Service Frame</Name>
				<lines>
					<Line id="BENCH:Line:1" version="1">
						<Name>Benchmark Line</Name>
						<PublicCode>B1</PublicCode>
						<TransportMode>bus</TransportMode>
						<TransportSubmode>localBus</TransportSubmode>
						<OperatorRef ref="BENCH:Operator:1"/>
					</Line>
				</lines>
			</ServiceFrame>
			<ResourceFrame id="BENCH:ResourceFrame:1" version="1">
				<Name>Benchmark Resource Frame</Name>
				<organisations>
					<Operator id="BENCH:Operator:1" version="1">
						<Name>Benchmark Operator</Name>
					</Operator>
				</organisations>
			</ResourceFrame>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`)

	options := testOptions(b).WithCodespace("BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateContent(xmlData, "bench-valid.xml", options)
		if err != nil {
			b.Fatalf("ValidateData() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidateData_InvalidData benchmarks in-memory validation of invalid NetEX data.
func BenchmarkValidateData_InvalidData(b *testing.B) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.0">
	<PublicationTimestamp>2023-01-01T12:00:00</PublicationTimestamp>
	<ParticipantRef>BENCH</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="BENCH:CompositeFrame:1" version="1">
			<Name>Invalid Benchmark Test Frame</Name>
			<ServiceFrame id="BENCH:ServiceFrame:1" version="1">
				<Name>Invalid Benchmark Service Frame</Name>
				<lines>
					<Line id="BENCH:Line:1" version="1">
						<!-- Missing Name -->
						<PublicCode>B1</PublicCode>
						<!-- Missing TransportMode -->
						<OperatorRef ref="BENCH:Operator:1"/>
					</Line>
				</lines>
			</ServiceFrame>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`)

	options := testOptions(b).WithCodespace("BENCH")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ValidateContent(xmlData, "bench-invalid.xml", options)
		if err != nil {
			b.Fatalf("ValidateData() failed: %v", err)
		}
		_ = result // Prevent optimization
	}
}

// BenchmarkValidationOptions_Creation benchmarks the creation of validation options.
func BenchmarkValidationOptions_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		options := testOptions(b).
			WithCodespace("BENCH").
			WithVerbose(true).
			WithSkipSchema(false)
		_ = options // Prevent optimization
	}
}

// BenchmarkValidationResult_Summary benchmarks the generation of validation result summary.
func BenchmarkValidationResult_Summary(b *testing.B) {
	// Pre-validate a file to get a result
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).WithCodespace("BENCH")
	result, err := ValidateFile(testFile, options)
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summary := result.Summary()
		_ = summary // Prevent optimization
	}
}

// BenchmarkValidationResult_ToJSON benchmarks JSON serialization of validation results.
func BenchmarkValidationResult_ToJSON(b *testing.B) {
	// Pre-validate a file to get a result
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).WithCodespace("BENCH")
	result, err := ValidateFile(testFile, options)
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonData, err := result.ToJSON()
		if err != nil {
			b.Fatalf("ToJSON() failed: %v", err)
		}
		_ = jsonData // Prevent optimization
	}
}

// BenchmarkValidationResult_ToHTML benchmarks HTML generation of validation results.
func BenchmarkValidationResult_ToHTML(b *testing.B) {
	// Pre-validate a file to get a result
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).WithCodespace("BENCH")
	result, err := ValidateFile(testFile, options)
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		htmlData, err := result.ToHTML()
		if err != nil {
			b.Fatalf("ToHTML() failed: %v", err)
		}
		_ = htmlData // Prevent optimization
	}
}

// BenchmarkValidationResult_String benchmarks string representation of validation results.
func BenchmarkValidationResult_String(b *testing.B) {
	// Pre-validate a file to get a result
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).WithCodespace("BENCH")
	result, err := ValidateFile(testFile, options)
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str := result.String()
		_ = str // Prevent optimization
	}
}

// BenchmarkValidationResult_IsValid benchmarks the IsValid check.
func BenchmarkValidationResult_IsValid(b *testing.B) {
	// Pre-validate a file to get a result
	testFile := filepath.Join("..", "..", "testdata", "invalid_missing_elements.xml")
	options := testOptions(b).WithCodespace("BENCH")
	result, err := ValidateFile(testFile, options)
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid := result.IsValid()
		_ = valid // Prevent optimization
	}
}

// BenchmarkComparativeValidation compares performance of different validation approaches.
func BenchmarkComparativeValidation(b *testing.B) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.0">
	<PublicationTimestamp>2023-01-01T12:00:00</PublicationTimestamp>
	<ParticipantRef>COMP</ParticipantRef>
	<dataObjects>
		<CompositeFrame id="COMP:CompositeFrame:1" version="1">
			<Name>Comparative Test Frame</Name>
		</CompositeFrame>
	</dataObjects>
</PublicationDelivery>`)

	b.Run("DefaultOptions", func(b *testing.B) {
		options := testOptions(b).WithCodespace("COMP")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := ValidateContent(xmlData, "comp-default.xml", options)
			if err != nil {
				b.Fatalf("ValidateData() failed: %v", err)
			}
			_ = result
		}
	})

	b.Run("SkipSchema", func(b *testing.B) {
		options := testOptions(b).
			WithCodespace("COMP").
			WithSkipSchema(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := ValidateContent(xmlData, "comp-skip.xml", options)
			if err != nil {
				b.Fatalf("ValidateData() failed: %v", err)
			}
			_ = result
		}
	})

	b.Run("Verbose", func(b *testing.B) {
		options := testOptions(b).
			WithCodespace("COMP").
			WithVerbose(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := ValidateContent(xmlData, "comp-verbose.xml", options)
			if err != nil {
				b.Fatalf("ValidateData() failed: %v", err)
			}
			_ = result
		}
	})

	b.Run("StopOnFirstError", func(b *testing.B) {
		options := testOptions(b).
			WithCodespace("COMP")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := ValidateContent(xmlData, "comp-stop.xml", options)
			if err != nil {
				b.Fatalf("ValidateData() failed: %v", err)
			}
			_ = result
		}
	})
}
