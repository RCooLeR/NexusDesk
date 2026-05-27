package datasets

import (
	"encoding/json"
	"fmt"
	"strings"
)

func profileNDJSON(relPath string, text string, mediaType string, size int64) (Profile, error) {
	columns, rows, notes, err := parseNDJSONRows(text)
	if err != nil {
		return Profile{}, err
	}
	return Profile{
		RelPath:   relPath,
		Format:    "NDJSON",
		MediaType: mediaType,
		Size:      size,
		Rows:      len(rows),
		Columns:   profileRows(columns, rows),
		Notes:     notes,
		JSONProfile: &JSONProfile{
			TopLevel: "ndjson",
			Count:    len(rows),
			Notes:    notes,
		},
	}, nil
}

func parseNDJSONRows(text string) ([]string, [][]string, []string, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	values := []any{}
	skipped := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			skipped++
			continue
		}
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, nil, fmt.Errorf("invalid NDJSON line %d: %w", len(values)+skipped+1, err)
		}
		values = append(values, value)
	}
	columns, rows := rowsFromJSONArray(values)
	notes := []string{"Each non-empty line was parsed as one JSON value."}
	if skipped > 0 {
		notes = append(notes, fmt.Sprintf("%d blank line(s) ignored.", skipped))
	}
	if len(columns) == 1 && columns[0] == "value" {
		notes = append(notes, "Lines are not all objects; values are profiled in a single value column.")
	}
	return columns, rows, notes, nil
}
