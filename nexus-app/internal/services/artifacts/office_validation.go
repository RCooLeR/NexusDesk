package artifacts

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxOfficeXMLValidationBytes = 4 * 1024 * 1024

func ValidateOfficePackage(absPath string, format string, requiredFiles []string) (PackageValidation, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	validation := PackageValidation{
		Format:        format,
		CheckedAt:     time.Now().UTC(),
		RequiredFiles: uniqueOfficePackageFiles(requiredOfficePackageFiles(format, requiredFiles)),
	}
	if format != documentExportFormat && format != presentationDeckFormat {
		return validation, fmt.Errorf("unsupported office package format %q", format)
	}
	reader, err := zip.OpenReader(absPath)
	if err != nil {
		validation.Message = "Office package is not a readable ZIP archive."
		return validation, err
	}
	defer reader.Close()

	parts := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		parts[filepath.ToSlash(file.Name)] = file
	}
	validation.CheckedFiles = len(parts)
	for _, required := range validation.RequiredFiles {
		if _, ok := parts[required]; !ok {
			validation.MissingFiles = append(validation.MissingFiles, required)
		}
	}
	if len(validation.MissingFiles) > 0 {
		validation.Message = "Office package is missing required parts: " + strings.Join(validation.MissingFiles, ", ") + "."
		return validation, errors.New(validation.Message)
	}

	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if !officePartShouldValidateXML(name) {
			continue
		}
		if err := validateOfficeXMLPart(file); err != nil {
			validation.Message = fmt.Sprintf("Office package XML part %s is invalid: %v.", name, err)
			return validation, errors.New(validation.Message)
		}
		validation.XMLFiles++
	}
	if format == presentationDeckFormat {
		slideCount, err := validatePresentationPackageRelationships(parts)
		if err != nil {
			validation.Message = err.Error()
			return validation, err
		}
		validation.SlideCount = slideCount
	}
	validation.Valid = true
	switch format {
	case documentExportFormat:
		validation.Message = fmt.Sprintf("DOCX package validation passed with %d required parts and %d XML parts.", len(validation.RequiredFiles), validation.XMLFiles)
	case presentationDeckFormat:
		validation.Message = fmt.Sprintf("PPTX package validation passed with %d required parts, %d XML parts, and %d slide(s).", len(validation.RequiredFiles), validation.XMLFiles, validation.SlideCount)
	}
	return validation, nil
}

func requiredOfficePackageFiles(format string, requiredFiles []string) []string {
	required := append([]string{}, requiredFiles...)
	switch strings.ToLower(strings.TrimSpace(format)) {
	case documentExportFormat:
		required = append(required,
			"[Content_Types].xml",
			"_rels/.rels",
			"docProps/core.xml",
			"docProps/app.xml",
			"word/document.xml",
			"word/styles.xml",
		)
	case presentationDeckFormat:
		required = append(required,
			"[Content_Types].xml",
			"_rels/.rels",
			"docProps/core.xml",
			"docProps/app.xml",
			"ppt/presentation.xml",
			"ppt/_rels/presentation.xml.rels",
		)
	}
	return required
}

func uniqueOfficePackageFiles(files []string) []string {
	seen := make(map[string]struct{}, len(files))
	result := make([]string, 0, len(files))
	for _, file := range files {
		file = filepath.ToSlash(strings.TrimSpace(file))
		if file == "" {
			continue
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		result = append(result, file)
	}
	sort.Strings(result)
	return result
}

func officePartShouldValidateXML(name string) bool {
	name = strings.ToLower(filepath.ToSlash(strings.TrimSpace(name)))
	return strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".rels")
}

func validateOfficeXMLPart(file *zip.File) error {
	if file.UncompressedSize64 > maxOfficeXMLValidationBytes {
		return fmt.Errorf("XML part is too large to validate (%d bytes)", file.UncompressedSize64)
	}
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	decoder := xml.NewDecoder(io.LimitReader(reader, maxOfficeXMLValidationBytes+1))
	for {
		if _, err := decoder.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func validatePresentationPackageRelationships(parts map[string]*zip.File) (int, error) {
	rels, err := readOfficeZipText(parts, "ppt/_rels/presentation.xml.rels")
	if err != nil {
		return 0, err
	}
	slideCount := 0
	for name := range parts {
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			slideCount++
			target := strings.TrimPrefix(name, "ppt/")
			if !strings.Contains(rels, `Target="`+target+`"`) {
				return 0, fmt.Errorf("PPTX package relationship is missing target %s", target)
			}
		}
	}
	if slideCount == 0 {
		return 0, errors.New("PPTX package contains no slide parts")
	}
	return slideCount, nil
}

func readOfficeZipText(parts map[string]*zip.File, name string) (string, error) {
	file, ok := parts[name]
	if !ok {
		return "", fmt.Errorf("Office package part %s is missing", name)
	}
	if file.UncompressedSize64 > maxOfficeXMLValidationBytes {
		return "", fmt.Errorf("Office package part %s is too large to validate", name)
	}
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxOfficeXMLValidationBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxOfficeXMLValidationBytes {
		return "", fmt.Errorf("Office package part %s is too large to validate", name)
	}
	return string(data), nil
}
