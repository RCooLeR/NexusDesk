package dataset

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"NexusAugenticStudio/internal/workspace"
)

const profileDirRelPath = ".nexusdesk/datasets"
const profileStoreName = "profiles.json"

type Profile struct {
	RelPath   string                    `json:"relPath"`
	Name      string                    `json:"name"`
	Kind      string                    `json:"kind"`
	Rows      int                       `json:"rows"`
	Columns   int                       `json:"columns"`
	Sheets    []string                  `json:"sheets"`
	Workbook  WorkbookInfo              `json:"workbook"`
	Parquet   ParquetInfo               `json:"parquet"`
	Profiles  []workspace.ColumnProfile `json:"profiles"`
	UpdatedAt string                    `json:"updatedAt"`
	Message   string                    `json:"message"`
}

type WorkbookInfo struct {
	Sheets       []WorkbookSheetInfo `json:"sheets"`
	NamedRanges  []string            `json:"namedRanges"`
	TableRanges  []WorkbookTableInfo `json:"tableRanges"`
	PivotTables  []string            `json:"pivotTables"`
	FormulaCount int                 `json:"formulaCount"`
}

type WorkbookSheetInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Dimension    string `json:"dimension"`
	Rows         int    `json:"rows"`
	Columns      int    `json:"columns"`
	FormulaCount int    `json:"formulaCount"`
	TableCount   int    `json:"tableCount"`
}

type WorkbookTableInfo struct {
	Name  string `json:"name"`
	Sheet string `json:"sheet"`
	Ref   string `json:"ref"`
}

type ParquetInfo struct {
	FileSize            int64  `json:"fileSize"`
	FooterMetadataBytes int64  `json:"footerMetadataBytes"`
	DataBytes           int64  `json:"dataBytes"`
	Magic               string `json:"magic"`
	Message             string `json:"message"`
}

func Build(root string, relPath string) (Profile, error) {
	absRoot, absTarget, cleanRel, err := resolveDatasetPath(root, relPath)
	if err != nil {
		return Profile{}, err
	}

	extension := strings.ToLower(filepath.Ext(cleanRel))
	profile := Profile{
		RelPath:   filepath.ToSlash(cleanRel),
		Name:      filepath.Base(cleanRel),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	switch extension {
	case ".csv", ".tsv", ".json", ".jsonl", ".ndjson":
		preview, err := workspace.Preview(absRoot, cleanRel, workspace.PreviewOptions{})
		if err != nil {
			return Profile{}, err
		}
		if preview.Table == nil {
			return Profile{}, errors.New("dataset profile could not parse a table")
		}
		profile.Kind = datasetKindFromExtension(extension)
		profile.Rows = preview.Table.TotalRows
		profile.Columns = len(preview.Table.Columns)
		profile.Profiles = preview.Table.Profiles
		profile.Message = strings.ToUpper(profile.Kind) + " dataset profile persisted."
	case ".xlsx":
		workbook, err := inspectXLSXWorkbook(absTarget)
		if err != nil {
			return Profile{}, err
		}
		profile.Kind = "xlsx"
		profile.Sheets = workbookSheetNames(workbook.Sheets)
		profile.Workbook = workbook
		profile.Rows = workbookRows(workbook.Sheets)
		profile.Columns = workbookColumns(workbook.Sheets)
		profile.Message = "Excel workbook profile persisted with sheet, formula, named range, table, and pivot metadata."
	case ".parquet":
		parquet, err := inspectParquetFile(absTarget)
		if err != nil {
			return Profile{}, err
		}
		profile.Kind = "parquet"
		profile.Parquet = parquet
		profile.Message = parquet.Message
	case ".xls":
		return Profile{}, errors.New("legacy binary XLS profiling is not available yet; convert the workbook to XLSX or CSV before profiling")
	default:
		return Profile{}, errors.New("dataset profiles currently support CSV, TSV, JSON, NDJSON, XLSX, and Parquet files")
	}

	if err := saveProfile(absRoot, profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func inspectParquetFile(path string) (ParquetInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return ParquetInfo{}, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return ParquetInfo{}, err
	}
	size := stat.Size()
	if size < 12 {
		return ParquetInfo{}, errors.New("parquet file is too small to contain a valid footer")
	}

	header := make([]byte, 4)
	if _, err := file.ReadAt(header, 0); err != nil {
		return ParquetInfo{}, err
	}
	if string(header) != "PAR1" {
		return ParquetInfo{}, errors.New("parquet file magic header not found")
	}

	footer := make([]byte, 8)
	if _, err := file.ReadAt(footer, size-8); err != nil {
		return ParquetInfo{}, err
	}
	if string(footer[4:]) != "PAR1" {
		return ParquetInfo{}, errors.New("parquet file magic footer not found")
	}

	metadataBytes := int64(binary.LittleEndian.Uint32(footer[:4]))
	if metadataBytes < 0 || metadataBytes+8 > size {
		return ParquetInfo{}, errors.New("parquet footer metadata length is invalid")
	}

	return ParquetInfo{
		FileSize:            size,
		FooterMetadataBytes: metadataBytes,
		DataBytes:           size - metadataBytes - 8,
		Magic:               "PAR1",
		Message:             "Parquet footer metadata inspected; schema decoding is planned.",
	}, nil
}

func datasetKindFromExtension(extension string) string {
	switch extension {
	case ".jsonl", ".ndjson":
		return "ndjson"
	default:
		return strings.TrimPrefix(extension, ".")
	}
}

func List(root string) ([]Profile, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	items, err := readProfiles(absRoot)
	if err != nil {
		return nil, err
	}
	profiles := make([]Profile, 0, len(items))
	for _, profile := range items {
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func saveProfile(absRoot string, profile Profile) error {
	items, err := readProfiles(absRoot)
	if err != nil {
		return err
	}
	items[profile.RelPath] = profile

	dir := filepath.Join(absRoot, filepath.FromSlash(profileDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, profileStoreName), append(data, '\n'), 0o644)
}

func readProfiles(absRoot string) (map[string]Profile, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(profileDirRelPath), profileStoreName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]Profile{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func resolveDatasetPath(root string, relPath string) (string, string, string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", err
	}
	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if cleanRel == "." || filepath.IsAbs(cleanRel) || strings.Contains(cleanRel, ".."+string(filepath.Separator)) {
		return "", "", "", errors.New("dataset path must stay inside the workspace")
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, cleanRel))
	if err != nil {
		return "", "", "", err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", "", "", errors.New("dataset path must stay inside the workspace")
	}
	return absRoot, absTarget, cleanRel, nil
}

func inspectXLSXWorkbook(path string) (WorkbookInfo, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return WorkbookInfo{}, err
	}
	defer reader.Close()

	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}

	workbookFile := files["xl/workbook.xml"]
	if workbookFile == nil {
		return WorkbookInfo{}, errors.New("XLSX workbook metadata not found")
	}
	workbook, err := decodeZipXML[xlsxWorkbook](workbookFile)
	if err != nil {
		return WorkbookInfo{}, err
	}

	rels := mapRelationshipTargets(files["xl/_rels/workbook.xml.rels"], "xl")
	tablesByPath := inspectXLSXTables(files)
	pivotTables := inspectXLSXPivotTables(files)

	info := WorkbookInfo{
		Sheets:      make([]WorkbookSheetInfo, 0, len(workbook.Sheets.Items)),
		NamedRanges: workbookDefinedNames(workbook.DefinedNames.Items),
		TableRanges: make([]WorkbookTableInfo, 0),
		PivotTables: pivotTables,
	}
	for _, sheet := range workbook.Sheets.Items {
		if sheet.Name == "" {
			continue
		}
		sheetPath := rels[sheet.RelID]
		sheetInfo := WorkbookSheetInfo{Name: sheet.Name, Path: sheetPath}
		if sheetPath != "" {
			worksheet, err := decodeZipXML[xlsxWorksheet](files[sheetPath])
			if err == nil {
				sheetInfo.Dimension = worksheet.Dimension.Ref
				sheetInfo.Rows = len(worksheet.SheetData.Rows)
				sheetInfo.FormulaCount = worksheetFormulaCount(worksheet)
				sheetInfo.TableCount = len(worksheet.TableParts.Items)
				if rows, columns, ok := dimensionSize(worksheet.Dimension.Ref); ok {
					sheetInfo.Rows = maxInt(sheetInfo.Rows, rows)
					sheetInfo.Columns = columns
				} else {
					sheetInfo.Columns = worksheetColumnCount(worksheet)
				}
				for _, tableRel := range worksheet.TableParts.Items {
					tablePath := mapRelationshipTargets(files[relationshipPath(sheetPath)], filepath.Dir(sheetPath))[tableRel.RelID]
					table := tablesByPath[tablePath]
					if table.Name != "" || table.Ref != "" {
						table.Sheet = sheet.Name
						info.TableRanges = append(info.TableRanges, table)
					}
				}
			}
		}
		info.FormulaCount += sheetInfo.FormulaCount
		info.Sheets = append(info.Sheets, sheetInfo)
	}

	return info, nil
}

type xlsxWorkbook struct {
	Sheets       xlsxSheets       `xml:"sheets"`
	DefinedNames xlsxDefinedNames `xml:"definedNames"`
}

type xlsxSheets struct {
	Items []xlsxSheet `xml:"sheet"`
}

type xlsxSheet struct {
	Name  string `xml:"name,attr"`
	RelID string `xml:"id,attr"`
}

type xlsxDefinedNames struct {
	Items []xlsxDefinedName `xml:"definedName"`
}

type xlsxDefinedName struct {
	Name string `xml:"name,attr"`
	Text string `xml:",chardata"`
}

type xlsxRelationships struct {
	Items []xlsxRelationship `xml:"Relationship"`
}

type xlsxRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
	Type   string `xml:"Type,attr"`
}

type xlsxWorksheet struct {
	Dimension  xlsxDimension  `xml:"dimension"`
	SheetData  xlsxSheetData  `xml:"sheetData"`
	TableParts xlsxTableParts `xml:"tableParts"`
}

type xlsxDimension struct {
	Ref string `xml:"ref,attr"`
}

type xlsxSheetData struct {
	Rows []xlsxRow `xml:"row"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref     string `xml:"r,attr"`
	Formula string `xml:"f"`
}

type xlsxTableParts struct {
	Items []xlsxTablePart `xml:"tablePart"`
}

type xlsxTablePart struct {
	RelID string `xml:"id,attr"`
}

type xlsxTable struct {
	Name        string `xml:"name,attr"`
	DisplayName string `xml:"displayName,attr"`
	Ref         string `xml:"ref,attr"`
}

type xlsxPivotTableDefinition struct {
	Name string `xml:"name,attr"`
}

func decodeZipXML[T any](file *zip.File) (T, error) {
	var value T
	if file == nil {
		return value, errors.New("XLSX metadata part not found")
	}
	handle, err := file.Open()
	if err != nil {
		return value, err
	}
	defer handle.Close()
	if err := xml.NewDecoder(handle).Decode(&value); err != nil {
		return value, err
	}
	return value, nil
}

func mapRelationshipTargets(file *zip.File, baseDir string) map[string]string {
	targets := map[string]string{}
	rels, err := decodeZipXML[xlsxRelationships](file)
	if err != nil {
		return targets
	}
	for _, rel := range rels.Items {
		if rel.ID == "" || rel.Target == "" {
			continue
		}
		target := filepath.ToSlash(filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(rel.Target))))
		targets[rel.ID] = target
	}
	return targets
}

func relationshipPath(partPath string) string {
	dir, name := filepath.Split(filepath.ToSlash(partPath))
	return filepath.ToSlash(filepath.Join(dir, "_rels", name+".rels"))
}

func inspectXLSXTables(files map[string]*zip.File) map[string]WorkbookTableInfo {
	tables := map[string]WorkbookTableInfo{}
	for path, file := range files {
		if !strings.HasPrefix(path, "xl/tables/") || !strings.HasSuffix(path, ".xml") {
			continue
		}
		table, err := decodeZipXML[xlsxTable](file)
		if err != nil {
			continue
		}
		name := table.DisplayName
		if name == "" {
			name = table.Name
		}
		tables[path] = WorkbookTableInfo{Name: name, Ref: table.Ref}
	}
	return tables
}

func inspectXLSXPivotTables(files map[string]*zip.File) []string {
	pivots := []string{}
	for path, file := range files {
		if !strings.HasPrefix(path, "xl/pivotTables/") || !strings.HasSuffix(path, ".xml") {
			continue
		}
		pivot, err := decodeZipXML[xlsxPivotTableDefinition](file)
		if err != nil || pivot.Name == "" {
			pivots = append(pivots, filepath.Base(path))
			continue
		}
		pivots = append(pivots, pivot.Name)
	}
	return pivots
}

func workbookDefinedNames(items []xlsxDefinedName) []string {
	names := []string{}
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		target := strings.TrimSpace(item.Text)
		if name == "" {
			continue
		}
		if target != "" {
			name += "=" + target
		}
		names = append(names, name)
	}
	return names
}

func workbookSheetNames(sheets []WorkbookSheetInfo) []string {
	names := make([]string, 0, len(sheets))
	for _, sheet := range sheets {
		names = append(names, sheet.Name)
	}
	return names
}

func workbookRows(sheets []WorkbookSheetInfo) int {
	rows := 0
	for _, sheet := range sheets {
		rows += sheet.Rows
	}
	return rows
}

func workbookColumns(sheets []WorkbookSheetInfo) int {
	columns := 0
	for _, sheet := range sheets {
		columns = maxInt(columns, sheet.Columns)
	}
	return columns
}

func worksheetFormulaCount(worksheet xlsxWorksheet) int {
	count := 0
	for _, row := range worksheet.SheetData.Rows {
		for _, cell := range row.Cells {
			if strings.TrimSpace(cell.Formula) != "" {
				count++
			}
		}
	}
	return count
}

func worksheetColumnCount(worksheet xlsxWorksheet) int {
	maxColumn := 0
	for _, row := range worksheet.SheetData.Rows {
		for _, cell := range row.Cells {
			if column := columnIndexFromCellRef(cell.Ref); column > maxColumn {
				maxColumn = column
			}
		}
	}
	return maxColumn
}

func dimensionSize(ref string) (int, int, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, 0, false
	}
	parts := strings.Split(ref, ":")
	last := parts[len(parts)-1]
	row := rowIndexFromCellRef(last)
	column := columnIndexFromCellRef(last)
	return row, column, row > 0 || column > 0
}

func rowIndexFromCellRef(ref string) int {
	digits := ""
	for _, value := range ref {
		if value >= '0' && value <= '9' {
			digits += string(value)
		}
	}
	var row int
	for _, value := range digits {
		row = row*10 + int(value-'0')
	}
	return row
}

func columnIndexFromCellRef(ref string) int {
	column := 0
	for _, value := range strings.ToUpper(ref) {
		if value < 'A' || value > 'Z' {
			continue
		}
		column = column*26 + int(value-'A') + 1
	}
	return column
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
