package datasets

import (
	"errors"
	"sort"
	"strings"
)

const queryResultMaxRows = 50

func (s *Service) Query(root string, relPath string, query string) (QueryResult, error) {
	preview, err := s.workspace.PreviewFile(root, relPath)
	if err != nil {
		return QueryResult{}, err
	}
	columns, rows, format, sourceTruncated, err := queryableRows(preview.RelPath, preview.Text)
	if err != nil {
		return QueryResult{}, err
	}
	if len(rows) == 0 {
		return QueryResult{}, errors.New("dataset is empty")
	}

	filter := parseQueryFilter(query, columns)
	matchedRows := filterRows(rows, filter)
	sortRows(matchedRows, filter)
	displayRows := limitRows(matchedRows, queryDisplayLimit(filter), len(columns))
	truncated := sourceTruncated || len(matchedRows) > len(displayRows)

	return QueryResult{
		RelPath:     preview.RelPath,
		Query:       strings.TrimSpace(query),
		Format:      format,
		Columns:     columns,
		Rows:        displayRows,
		TotalRows:   len(rows),
		MatchedRows: len(matchedRows),
		Truncated:   truncated,
		Message:     queryMessage(preview.RelPath, len(rows), len(matchedRows), len(displayRows), filter, columns, sourceTruncated),
	}, nil
}

func filterRows(rows [][]string, filter queryFilter) [][]string {
	matchedRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		if filter.matches(row) {
			matchedRows = append(matchedRows, row)
		}
	}
	return matchedRows
}

func sortRows(rows [][]string, filter queryFilter) {
	if filter.orderIndex < 0 {
		return
	}
	sort.SliceStable(rows, func(i, j int) bool {
		result := compareValues(valueAt(rows[i], filter.orderIndex), valueAt(rows[j], filter.orderIndex))
		if filter.orderDesc {
			return result > 0
		}
		return result < 0
	})
}

func queryDisplayLimit(filter queryFilter) int {
	if filter.limit > 0 && filter.limit < queryResultMaxRows {
		return filter.limit
	}
	return queryResultMaxRows
}

func limitRows(rows [][]string, limit int, width int) [][]string {
	displayRows := make([][]string, 0, minInt(limit, len(rows)))
	for _, row := range rows {
		if len(displayRows) >= limit {
			break
		}
		displayRows = append(displayRows, trimRowWidth(row, width))
	}
	return displayRows
}
