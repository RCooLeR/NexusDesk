package datasets

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexusdesk/internal/domain"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const maxSamplesPerColumn = 3

type Service struct {
	workspace *workspaceSvc.Service
}

func New(workspace *workspaceSvc.Service) *Service {
	if workspace == nil {
		workspace = workspaceSvc.New()
	}
	return &Service{workspace: workspace}
}

func (s *Service) Profile(root string, relPath string) (Profile, error) {
	preview, err := s.workspace.PreviewFile(root, relPath)
	if err != nil {
		return Profile{}, err
	}
	switch strings.ToLower(filepath.Ext(preview.RelPath)) {
	case ".csv", ".tsv", ".xlsx":
		return s.profileTable(preview)
	case ".json":
		return s.profileJSON(preview)
	default:
		return Profile{}, fmt.Errorf("unsupported dataset type %q", filepath.Ext(preview.RelPath))
	}
}

func (s *Service) profileTable(preview domain.FilePreview) (Profile, error) {
	if preview.Table == nil {
		return Profile{}, errors.New("table preview is unavailable")
	}
	return Profile{
		RelPath:   preview.RelPath,
		Format:    tableFormatForPreview(preview),
		MediaType: preview.MediaType,
		Size:      preview.Size,
		Rows:      len(preview.Table.Rows),
		Columns:   profileRows(preview.Table.Headers, preview.Table.Rows),
		Sheet:     preview.Table.Sheet,
		Sheets:    append([]string{}, preview.Table.Sheets...),
		Truncated: preview.Table.Truncated,
	}, nil
}

func (s *Service) profileJSON(preview domain.FilePreview) (Profile, error) {
	if strings.TrimSpace(preview.Text) == "" {
		return Profile{}, errors.New("JSON file is empty")
	}
	decoder := json.NewDecoder(strings.NewReader(preview.Text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return Profile{}, err
	}
	fields, jsonProfile := profileJSONValue(value)
	return Profile{
		RelPath:     preview.RelPath,
		Format:      "JSON",
		MediaType:   preview.MediaType,
		Size:        preview.Size,
		Rows:        jsonProfile.Count,
		Columns:     fields,
		JSONProfile: &jsonProfile,
	}, nil
}

func profileRows(headers []string, rows [][]string) []ColumnProfile {
	width := len(headers)
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	columns := make([]ColumnProfile, 0, width)
	for index := 0; index < width; index++ {
		name := columnName(headers, index)
		values := make([]string, 0, len(rows))
		for _, row := range rows {
			if index >= len(row) {
				values = append(values, "")
				continue
			}
			values = append(values, row[index])
		}
		columns = append(columns, profileValues(name, values))
	}
	return columns
}

func profileJSONValue(value any) ([]ColumnProfile, JSONProfile) {
	switch typed := value.(type) {
	case []any:
		fields := profileJSONArrayFields(typed)
		return fields, JSONProfile{TopLevel: "array", Count: len(typed), Notes: jsonNotesForArray(typed, fields)}
	case map[string]any:
		keys := sortedKeys(typed)
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			values = append(values, scalarSummary(typed[key]))
		}
		return profileRows(keys, [][]string{values}), JSONProfile{TopLevel: "object", Count: len(keys), Notes: []string{"Object keys are profiled as fields."}}
	default:
		return nil, JSONProfile{TopLevel: jsonTopLevel(value), Count: 1, Notes: []string{"Top-level JSON value is scalar."}}
	}
}

func profileJSONArrayFields(values []any) []ColumnProfile {
	keySet := map[string]struct{}{}
	objectRows := []map[string]any{}
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			continue
		}
		objectRows = append(objectRows, object)
		for key := range object {
			keySet[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(objectRows))
	for _, object := range objectRows {
		row := make([]string, len(keys))
		for index, key := range keys {
			row[index] = scalarSummary(object[key])
		}
		rows = append(rows, row)
	}
	return profileRows(keys, rows)
}

func profileValues(name string, values []string) ColumnProfile {
	profile := ColumnProfile{Name: name}
	types := map[string]int{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			profile.Empty++
			continue
		}
		profile.NonEmpty++
		if len(profile.Samples) < maxSamplesPerColumn && !contains(profile.Samples, value) {
			profile.Samples = append(profile.Samples, value)
		}
		types[guessType(value)]++
	}
	profile.Type = dominantType(types)
	return profile
}

func columnName(headers []string, index int) string {
	if index < len(headers) && strings.TrimSpace(headers[index]) != "" {
		return strings.TrimSpace(headers[index])
	}
	return fmt.Sprintf("column_%d", index+1)
}

func tableFormat(delimiter string) string {
	if delimiter == "\t" {
		return "TSV"
	}
	return "CSV"
}

func tableFormatForPreview(preview domain.FilePreview) string {
	if strings.EqualFold(filepath.Ext(preview.RelPath), ".xlsx") {
		return "XLSX"
	}
	return tableFormat(preview.Table.Delimiter)
}

func jsonNotesForArray(values []any, fields []ColumnProfile) []string {
	if len(values) == 0 {
		return []string{"Array is empty."}
	}
	if len(fields) == 0 {
		return []string{"Array does not contain objects, so no field columns are available."}
	}
	return []string{"Array object fields are profiled across object elements."}
}

func jsonTopLevel(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case json.Number, float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "value"
	}
}

func scalarSummary(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case []any:
		return fmt.Sprintf("array[%d]", len(typed))
	case map[string]any:
		return fmt.Sprintf("object{%d}", len(typed))
	default:
		return fmt.Sprint(typed)
	}
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func guessType(value string) string {
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "integer"
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "number"
	}
	if _, err := strconv.ParseBool(strings.ToLower(value)); err == nil {
		return "boolean"
	}
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return "datetime"
	}
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return "date"
	}
	return "text"
}

func dominantType(types map[string]int) string {
	if len(types) == 0 {
		return "empty"
	}
	if types["number"] > 0 && types["integer"] > 0 {
		types["number"] += types["integer"]
		delete(types, "integer")
	}
	order := []string{"integer", "number", "boolean", "datetime", "date", "text"}
	bestType := "text"
	bestCount := -1
	for _, candidate := range order {
		if types[candidate] > bestCount {
			bestType = candidate
			bestCount = types[candidate]
		}
	}
	return bestType
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
