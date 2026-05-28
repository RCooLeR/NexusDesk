package dbconnector

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	DefaultSQLiteRows           = 100
	MaxSQLiteRows               = 10000
	DefaultSQLiteTimeoutSeconds = 30
	MaxSQLiteTimeoutSeconds     = 300
)

func (s *Service) QueryWorkspaceSQLite(root string, request SQLiteQueryRequest) (SQLiteQueryResult, error) {
	request = NormalizeSQLiteQueryRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	return s.QueryWorkspaceSQLiteContext(ctx, root, request)
}

func (s *Service) QueryWorkspaceSQLiteContext(ctx context.Context, root string, request SQLiteQueryRequest) (SQLiteQueryResult, error) {
	request = NormalizeSQLiteQueryRequest(request)
	_, absPath, cleanRel, err := resolveSQLitePath(root, request.RelPath)
	if err != nil {
		return SQLiteQueryResult{}, err
	}
	query, err := normalizeReadOnlySQLiteQuery(request.SQL)
	if err != nil {
		return SQLiteQueryResult{}, err
	}
	started := time.Now()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(absPath)+"?mode=ro")
	if err != nil {
		return SQLiteQueryResult{}, connectorError(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "pragma query_only = on"); err != nil {
		return SQLiteQueryResult{}, connectorError(err)
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return SQLiteQueryResult{}, connectorError(err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return SQLiteQueryResult{}, connectorError(err)
	}
	scanners := rowScanners(len(columns))
	resultRows := [][]string{}
	totalRows := 0
	truncated := false
	for rows.Next() {
		if totalRows >= request.ResultLimit {
			truncated = true
			break
		}
		select {
		case <-ctx.Done():
			return SQLiteQueryResult{}, connectorError(ctx.Err())
		default:
		}
		row, err := scanRowAsStrings(rows, scanners)
		if err != nil {
			return SQLiteQueryResult{}, connectorError(err)
		}
		resultRows = append(resultRows, row)
		totalRows++
	}
	if err := rows.Err(); err != nil {
		return SQLiteQueryResult{}, connectorError(err)
	}
	message := fmt.Sprintf("Read-only SQLite query returned %d row(s) from %s.", totalRows, cleanRel)
	if truncated {
		message = fmt.Sprintf("Read-only SQLite query reached the %d row cap for %s.", request.ResultLimit, cleanRel)
	}
	return SQLiteQueryResult{
		RelPath:        cleanRel,
		SQL:            query,
		Engine:         "sqlite-readonly",
		Columns:        columns,
		Rows:           resultRows,
		TotalRows:      totalRows,
		Truncated:      truncated,
		ResultLimit:    request.ResultLimit,
		TimeoutSeconds: request.TimeoutSeconds,
		DurationMs:     time.Since(started).Milliseconds(),
		Message:        message,
	}, nil
}

func NormalizeSQLiteQueryRequest(request SQLiteQueryRequest) SQLiteQueryRequest {
	request.RelPath = strings.TrimSpace(request.RelPath)
	request.SQL = strings.TrimSpace(request.SQL)
	if request.ResultLimit <= 0 {
		request.ResultLimit = DefaultSQLiteRows
	}
	if request.ResultLimit > MaxSQLiteRows {
		request.ResultLimit = MaxSQLiteRows
	}
	if request.TimeoutSeconds <= 0 {
		request.TimeoutSeconds = DefaultSQLiteTimeoutSeconds
	}
	if request.TimeoutSeconds > MaxSQLiteTimeoutSeconds {
		request.TimeoutSeconds = MaxSQLiteTimeoutSeconds
	}
	return request
}

func normalizeReadOnlySQLiteQuery(query string) (string, error) {
	return normalizeReadOnlySQL(query,
		"workspace SQLite connector only supports read-only SELECT queries",
		"workspace SQLite connector blocks mutating SQL",
		"sqlite",
	)
}
