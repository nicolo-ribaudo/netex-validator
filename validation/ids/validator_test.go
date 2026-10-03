package ids

import (
	"testing"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

func TestIdVersionBasic(t *testing.T) {
	t.Run("Create IdVersion", func(t *testing.T) {
		id := types.NewIdVersion("TEST:Line:1", "1", "test.xml")

		if id.ID != "TEST:Line:1" {
			t.Errorf("Expected ID TEST:Line:1, got %s", id.ID)
		}

		if id.Version != "1" {
			t.Errorf("Expected version 1, got %s", id.Version)
		}

		if id.FileName != "test.xml" {
			t.Errorf("Expected filename test.xml, got %s", id.FileName)
		}
	})
}

func TestNetexIdRepositoryBasic(t *testing.T) {
	t.Run("Create repository", func(t *testing.T) {
		repo := NewNetexIdRepository()

		if repo == nil {
			t.Error("Expected non-nil repository")
		}
	})

	t.Run("Add ID", func(t *testing.T) {
		repo := NewNetexIdRepository()

		err := repo.AddId("TEST:Line:1", "1", "test.xml")
		if err != nil {
			t.Errorf("Expected no error adding ID, got: %v", err)
		}
	})

	t.Run("Add reference", func(t *testing.T) {
		repo := NewNetexIdRepository()

		// Should not panic
		repo.AddReference("TEST:Line:1", "1", "test.xml")
	})
}

func TestNetexIdExtractorBasic(t *testing.T) {
	t.Run("Create extractor", func(t *testing.T) {
		extractor := NewNetexIdExtractor()

		if extractor == nil {
			t.Error("Expected non-nil extractor")
		}
	})

	t.Run("Extract from valid XML", func(t *testing.T) {
		extractor := NewNetexIdExtractor()

		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<PublicationDelivery xmlns="http://www.netex.org.uk/netex" version="1.15">
	<PublicationTimestamp>2023-01-01T00:00:00</PublicationTimestamp>
	<ParticipantRef>TEST</ParticipantRef>
	<dataObjects>
		<ServiceFrame id="TEST:ServiceFrame:1" version="1">
			<lines>
				<Line id="TEST:Line:1" version="1">
					<Name>Test Line</Name>
				</Line>
			</lines>
		</ServiceFrame>
	</dataObjects>
</PublicationDelivery>`

		ids, err := extractor.ExtractIds("test.xml", []byte(xmlContent))
		if err != nil {
			t.Errorf("Expected no error extracting IDs, got: %v", err)
		}

		if len(ids) == 0 {
			t.Error("Expected to extract some IDs")
		}

		t.Logf("Extracted %d IDs", len(ids))
	})
}

func TestNetexIdValidatorBasic(t *testing.T) {
	t.Run("Create validator", func(t *testing.T) {
		repo := NewNetexIdRepository()
		extractor := NewNetexIdExtractor()
		validator := NewNetexIdValidator(repo, extractor)

		if validator == nil {
			t.Error("Expected non-nil validator")
		}
	})

	t.Run("Validate IDs", func(t *testing.T) {
		repo := NewNetexIdRepository()
		extractor := NewNetexIdExtractor()
		validator := NewNetexIdValidator(repo, extractor)

		issues, err := validator.ValidateIds()
		if err != nil {
			t.Errorf("Expected no error validating IDs, got: %v", err)
		}

		// Should return a non-nil slice (even if empty)
		// Note: ValidateIds returns empty slice when no issues found

		t.Logf("Found %d validation issues", len(issues))
	})
}
