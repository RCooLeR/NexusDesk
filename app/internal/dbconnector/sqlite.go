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
	query := strings.TrimSpace(strings.TrimSuffix(request.SQL, ";"))
	if query == "" {
		return SQLiteQueryResult{}, errors.New("enter a read-only SELECT query")
	}
	lower := strings.ToLower(query)
	if !strings.HasPrefix(lower, "select ") && !strings.HasPrefix(lower, "with ") {
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
	values := make([]sql.NullString, len(columns))
	dest := make([]any, len(columns))
	for index := range values {
		dest[index] = &values[index]
	}
	resultRows := [][]string{}
	totalRows := 0
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return SQLiteQueryResult{}, err
		}
		totalRows++
		if len(resultRows) >= maxSQLiteRows {
			continue
		}
		row := make([]string, len(columns))
		for index, value := range values {
			if value.Valid {
				row[index] = value.String
			}
		}
		resultRows = append(resultRows, row)
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
		Message:   fmt.Sprintf("Read-only SQLite query returned %d rows from %s.", len(resultRows), cleanRel),
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

func containsBlockedSQL(lower string) bool {
	for _, blocked := range []string{" insert ", " update ", " delete ", " drop ", " alter ", " truncate ", " create ", " attach ", " detach ", " replace ", " vacuum ", " pragma "} {
		if strings.Contains(" "+lower+" ", blocked) {
			return true
		}
	}
	return false
}
