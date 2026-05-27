package spreadsheets

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func ParseXLSX(content []byte, options Options) (Workbook, error) {
	if options.MaxRows <= 0 {
		options.MaxRows = DefaultOptions().MaxRows
	}
	if options.MaxColumns <= 0 {
		options.MaxColumns = DefaultOptions().MaxColumns
	}
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return Workbook{}, err
	}
	files := mapZipFiles(reader.File)
	workbookFile := files["xl/workbook.xml"]
	if workbookFile == nil {
		return Workbook{}, errors.New("xlsx workbook metadata was not found")
	}
	workbookXML, err := parseWorkbookXML(workbookFile)
	if err != nil {
		return Workbook{}, err
	}
	relationships, err := parseRelationships(files["xl/_rels/workbook.xml.rels"], "xl")
	if err != nil {
		return Workbook{}, err
	}
	sharedStrings, err := parseSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return Workbook{}, err
	}
	workbook := Workbook{Sheets: make([]Sheet, 0, len(workbookXML.Sheets.Items))}
	for _, item := range workbookXML.Sheets.Items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		sheet := Sheet{Name: strings.TrimSpace(item.Name), Path: relationships[item.RelID]}
		if sheet.Path != "" && files[sheet.Path] != nil {
			parsed, err := parseWorksheet(files[sheet.Path], sharedStrings, options)
			if err != nil {
				return Workbook{}, fmt.Errorf("parse sheet %q: %w", sheet.Name, err)
			}
			sheet.Rows = parsed.Rows
			sheet.Truncated = parsed.Truncated
		}
		workbook.Sheets = append(workbook.Sheets, sheet)
	}
	if len(workbook.Sheets) == 0 {
		return Workbook{}, errors.New("xlsx workbook did not contain visible sheets")
	}
	return workbook, nil
}

func mapZipFiles(files []*zip.File) map[string]*zip.File {
	mapped := map[string]*zip.File{}
	for _, file := range files {
		mapped[filepath.ToSlash(file.Name)] = file
	}
	return mapped
}

func readZipFile(file *zip.File) ([]byte, error) {
	body, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}
