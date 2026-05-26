package dataset

import (
	"archive/zip"
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
	Profiles  []workspace.ColumnProfile `json:"profiles"`
	UpdatedAt string                    `json:"updatedAt"`
	Message   string                    `json:"message"`
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
		sheets, err := inspectXLSXSheets(absTarget)
		if err != nil {
			return Profile{}, err
		}
		profile.Kind = "xlsx"
		profile.Sheets = sheets
		profile.Message = "Excel workbook profile persisted."
	default:
		return Profile{}, errors.New("dataset profiles currently support CSV, TSV, JSON, NDJSON, and XLSX files")
	}

	if err := saveProfile(absRoot, profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
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

func inspectXLSXSheets(path string) ([]string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "xl/workbook.xml" {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer handle.Close()

		var workbook xlsxWorkbook
		if err := xml.NewDecoder(handle).Decode(&workbook); err != nil {
			return nil, err
		}
		sheets := make([]string, 0, len(workbook.Sheets.Items))
		for _, sheet := range workbook.Sheets.Items {
			if sheet.Name != "" {
				sheets = append(sheets, sheet.Name)
			}
		}
		return sheets, nil
	}

	return nil, errors.New("XLSX workbook metadata not found")
}

type xlsxWorkbook struct {
	Sheets xlsxSheets `xml:"sheets"`
}

type xlsxSheets struct {
	Items []xlsxSheet `xml:"sheet"`
}

type xlsxSheet struct {
	Name string `xml:"name,attr"`
}
