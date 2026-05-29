package artifacts

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestValidateOfficePackageRejectsMissingRequiredPart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.docx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	if err := addZipText(writer, "[Content_Types].xml", `<?xml version="1.0"?><Types/>`); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	validation, err := ValidateOfficePackage(path, documentExportFormat, nil)
	if err == nil {
		t.Fatal("expected missing required DOCX parts to fail validation")
	}
	if validation.Valid || len(validation.MissingFiles) == 0 || !strings.Contains(validation.Message, "missing required parts") {
		t.Fatalf("unexpected validation result: %#v", validation)
	}
}

func TestValidateOfficePackageRejectsBrokenPPTXSlideRelationship(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pptx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, content := range map[string]string{
		"[Content_Types].xml":             `<?xml version="1.0"?><Types/>`,
		"_rels/.rels":                     `<?xml version="1.0"?><Relationships/>`,
		"docProps/core.xml":               `<?xml version="1.0"?><core/>`,
		"docProps/app.xml":                `<?xml version="1.0"?><app/>`,
		"ppt/presentation.xml":            `<?xml version="1.0"?><presentation/>`,
		"ppt/_rels/presentation.xml.rels": `<?xml version="1.0"?><Relationships/>`,
		"ppt/slides/slide1.xml":           `<?xml version="1.0"?><slide/>`,
	} {
		if err := addZipText(writer, name, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	validation, err := ValidateOfficePackage(path, presentationDeckFormat, []string{"ppt/slides/slide1.xml"})
	if err == nil {
		t.Fatal("expected missing PPTX slide relationship to fail validation")
	}
	if validation.Valid || !strings.Contains(validation.Message, "relationship is missing target slides/slide1.xml") {
		t.Fatalf("unexpected validation result: %#v", validation)
	}
}

func TestValidateOfficePackageRejectsTooManyZipMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "too-many.pptx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for index := 0; index <= maxOfficeZipFiles; index++ {
		if err := addZipText(writer, filepath.ToSlash(filepath.Join("ppt", "extra", "part"+strconv.Itoa(index)+".txt")), "x"); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	validation, err := ValidateOfficePackage(path, presentationDeckFormat, nil)
	if err == nil {
		t.Fatal("expected ZIP member cap to fail validation")
	}
	if validation.Valid || !strings.Contains(validation.Message, "ZIP safety limits") {
		t.Fatalf("unexpected validation result: %#v", validation)
	}
}
