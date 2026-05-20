package dbconnector

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const maxSQLiteRows = 100

type SQLiteQueryRequest struct {
	RelPath string `json:"relPath"`
	SQL     string `json:"sql"`
}

type SQLiteQueryResult struct {
	RelPath   string     `json:"relPath"`
	SQL       string     `json:"sql"`
	Engine    string     `json:"engine"`
	Columns   []string   `json:"columns"`
	Rows      [][]string `json:"rows"`
	TotalRows int        `json:"totalRows"`
	Message   string     `json:"message"`
}

func QuerySQLite(root string, request SQLiteQueryRequest) (SQLiteQueryResult, error) {
	absRoot, absPath, cleanRel, err := resolveSQLitePath(root, request.RelPath)
	if err != nil {
		return SQLiteQueryResult{}, err
	}
	_ = absRoot
	query := strings.TrimSpace(request.SQL)
	if query == "" {
		return SQLiteQueryResult{}, errors.New("enter a read-only SELECT query")
	}
	if err := validateSingleStatement(query); err != nil {
		return SQLiteQueryResult{}, err
	}
	for strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	if query == "" {
		return SQLiteQueryResult{}, errors.New("enter a read-only SELECT query")
	}

	lower := strings.ToLower(query)
	tokens := tokenizeSQL(lower)
	if len(tokens) == 0 || (tokens[0] != "select" && tokens[0] != "with") {
		return SQLiteQueryResult{}, errors.New("workspace SQLite connector only supports read-only SELECT queries")
	}
	if containsBlockedSQL(lower) {
		return SQLiteQueryResult{}, errors.New("workspace SQLite connector blocks mutating SQL")
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(absPath)+"?mode=ro")
	if err != nil {
		return SQLiteQueryResult{}, err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return SQLiteQueryResult{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return SQLiteQueryResult{}, err
	}
	scanners := rowScanners(len(columns))
	skipScanners := rowScanners(len(columns))
	resultRows := [][]string{}
	totalRows := 0
	for rows.Next() {
		if len(resultRows) >= maxSQLiteRows {
			if err := rows.Scan(skipScanners...); err != nil {
				return SQLiteQueryResult{}, err
			}
		} else {
			row, err := scanRowAsStrings(rows, scanners)
			if err != nil {
				return SQLiteQueryResult{}, err
			}
			resultRows = append(resultRows, row)
		}
		totalRows++
	}
	if err := rows.Err(); err != nil {
		return SQLiteQueryResult{}, err
	}

	return SQLiteQueryResult{
		RelPath:   cleanRel,
		SQL:       query,
		Engine:    "sqlite-readonly",
		Columns:   columns,
		Rows:      resultRows,
		TotalRows: totalRows,
		Message:   fmt.Sprintf("Read-only SQLite query returned %d rows from %s (showing %d).", totalRows, cleanRel, len(resultRows)),
	}, nil
}

func resolveSQLitePath(root string, relPath string) (string, string, string, error) {
	if strings.TrimSpace(root) == "" {
		return "", "", "", errors.New("open a workspace before querying SQLite files")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", err
	}
	cleanRel := filepath.Clean(strings.TrimSpace(relPath))
	if cleanRel == "." || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
		return "", "", "", errors.New("SQLite path must stay inside the workspace")
	}
	ext := strings.ToLower(filepath.Ext(cleanRel))
	if ext != ".sqlite" && ext != ".sqlite3" && ext != ".db" {
		return "", "", "", errors.New("selected file is not a SQLite database")
	}
	absPath := filepath.Join(absRoot, cleanRel)
	resolved, err := filepath.Abs(absPath)
	if err != nil {
		return "", "", "", err
	}
	if !strings.HasPrefix(strings.ToLower(resolved), strings.ToLower(absRoot)+string(filepath.Separator)) && !strings.EqualFold(resolved, absRoot) {
		return "", "", "", errors.New("SQLite path must stay inside the workspace")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", "", "", err
	}
	if info.IsDir() {
		return "", "", "", errors.New("SQLite connector target must be a file")
	}
	return absRoot, resolved, filepath.ToSlash(cleanRel), nil
}

func tokenizeSQL(query string) []string {
	if query == "" {
		return nil
	}
	normalized := strings.TrimSpace(strings.ToLower(query))
	normalized = strings.ReplaceAll(normalized, "(", " ")
	normalized = strings.ReplaceAll(normalized, ")", " ")
	normalized = strings.ReplaceAll(normalized, ";", " ")
	return strings.Fields(normalized)
}

func validateSingleStatement(query string) error {
	segments := splitTopLevelSQLStatements(query)
	if len(segments) == 0 {
		return errors.New("enter a read-only SELECT query")
	}
	nonEmptyCount := 0
	for index, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			if index < len(segments)-1 {
				return errors.New("query must contain a single SQL statement")
			}
			continue
		}
		nonEmptyCount++
		if nonEmptyCount > 1 {
			return errors.New("query must contain a single SQL statement")
		}
	}
	if nonEmptyCount == 0 {
		return errors.New("enter a read-only SELECT query")
	}
	return nil
}

func splitTopLevelSQLStatements(query string) []string {
	segments := []string{}
	var builder strings.Builder

	var quote rune
	lineComment := false
	blockComment := false

	runes := []rune(query)
	for index := 0; index < len(runes); index++ {
		current := runes[index]
		if lineComment {
			if current == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if current == '*' && index+1 < len(runes) && runes[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
		if quote != 0 {
			if current == quote {
				if index+1 < len(runes) && runes[index+1] == quote {
					builder.WriteRune(current)
					index++
					continue
				}
				quote = 0
			}
			builder.WriteRune(current)
			continue
		}

		switch current {
		case '\'':
			quote = '\''
			builder.WriteRune(current)
		case '"':
			quote = '"'
			builder.WriteRune(current)
		case '-':
			if index+1 < len(runes) && runes[index+1] == '-' {
				lineComment = true
				index++
				continue
			}
			builder.WriteRune(current)
		case '/':
			if index+1 < len(runes) && runes[index+1] == '*' {
				blockComment = true
				index++
				continue
			}
			builder.WriteRune(current)
		case ';':
			segments = append(segments, strings.TrimSpace(builder.String()))
			builder.Reset()
		default:
			builder.WriteRune(current)
		}
	}
	segments = append(segments, strings.TrimSpace(builder.String()))
	return segments
}

func containsBlockedSQL(lower string) bool {
	if lower == "" {
		return false
	}
	for _, token := range tokenizeSQL(lower) {
		switch token {
		case "insert", "update", "delete", "drop", "alter", "truncate", "create", "attach", "detach", "replace", "vacuum", "pragma":
			return true
		}
	}
	return false
}

func rowScanners(columnCount int) []any {
	scanners := make([]any, columnCount)
	for index := range scanners {
		var value any
		scanners[index] = &value
	}
	return scanners
}

func scanRowAsStrings(rows *sql.Rows, scanners []any) ([]string, error) {
	if err := rows.Scan(scanners...); err != nil {
		return nil, err
	}
	row := make([]string, len(scanners))
	for index, scanner := range scanners {
		switch value := scanner.(*any); {
		case value == nil || *value == nil:
			row[index] = ""
		default:
			row[index] = stringifyValue(*value)
		}
	}
	return row, nil
}

func stringifyValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
