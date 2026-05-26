//go:build duckdb

package dbconnector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"NexusAugenticStudio/internal/storage"
	_ "github.com/duckdb/duckdb-go/v2"
)

func TestDuckDBProfile(profile storage.ConnectorProfile) (ConnectorProfileStatus, error) {
	if err := requireDuckDBProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openDuckDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRowContext(ctx, "select version()").Scan(&version); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	return ConnectorProfileStatus{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Kind:      profile.Kind,
		Engine:    duckDBEngineName,
		ReadOnly:  true,
		Message:   fmt.Sprintf("DuckDB read-only connection succeeded for %s.", profile.Name),
	}, nil
}

func QueryDuckDBProfile(profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	request = NormalizeConnectorQueryRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	return QueryDuckDBProfileContext(ctx, profile, request)
}

func QueryDuckDBProfileContext(ctx context.Context, profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	if err := requireDuckDBProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	request = NormalizeConnectorQueryRequest(request)
	query, err := normalizeReadOnlyConnectorSQL(request.SQL)
	if err != nil {
		return ConnectorQueryResult{}, err
	}

	db, err := openDuckDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
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
		row, err := scanRowAsStrings(rows, scanners)
		if err != nil {
			return ConnectorQueryResult{}, connectorError(err)
		}
		resultRows = append(resultRows, row)
		totalRows++
	}
	if err := rows.Err(); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}

	message := fmt.Sprintf("Read-only DuckDB query returned %d rows from %s.", totalRows, profile.Name)
	if truncated {
		message = fmt.Sprintf("Read-only DuckDB query reached the %d row cap for %s.", request.ResultLimit, profile.Name)
	}
	return ConnectorQueryResult{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         duckDBEngineName,
		SQL:            query,
		Columns:        columns,
		Rows:           resultRows,
		TotalRows:      totalRows,
		Truncated:      truncated,
		ResultLimit:    request.ResultLimit,
		TimeoutSeconds: request.TimeoutSeconds,
		Message:        message,
	}, nil
}

func InspectDuckDBProfile(profile storage.ConnectorProfile) (ConnectorMetadata, error) {
	if err := requireDuckDBProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openDuckDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()

	objects, err := listDuckDBObjects(ctx, db)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	columns, err := listDuckDBColumns(ctx, db)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	primaryKeys := listDuckDBPrimaryKeys(ctx, db)

	metadata := ConnectorMetadata{
		ID:            "duckdb:" + profile.ID,
		RelPath:       profile.ID,
		Name:          profile.Name,
		Kind:          "duckdb",
		Engine:        duckDBEngineName,
		ReadOnly:      true,
		Tables:        []ConnectorTable{},
		Views:         []ConnectorTable{},
		Indexes:       []ConnectorIndex{},
		Relationships: listDuckDBRelationships(ctx, db),
	}
	for _, object := range objects {
		tableColumns := columns[object.name]
		applyDuckDBPrimaryKeys(tableColumns, primaryKeys[object.name])
		table := ConnectorTable{
			Name:       object.name,
			Type:       object.kind,
			RowCount:   duckDBRowCount(ctx, db, object.name),
			Columns:    tableColumns,
			Indexes:    []ConnectorIndex{},
			SampleRows: duckDBSampleRows(ctx, db, object.name),
		}
		if table.Type == "table" {
			metadata.Tables = append(metadata.Tables, table)
		} else {
			metadata.Views = append(metadata.Views, table)
		}
	}
	metadata.Relationships = append(metadata.Relationships, inferredConnectorRelationships(metadata.Tables)...)
	metadata.Message = fmt.Sprintf("DuckDB connector metadata inspected: %d tables, %d views, %d relationships.", len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

func openDuckDB(profile storage.ConnectorProfile, timeoutSeconds int) (*sql.DB, error) {
	db, err := sql.Open("duckdb", duckDBDSN(profile))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	return db, nil
}

type duckDBObject struct {
	name string
	kind string
}

func listDuckDBObjects(ctx context.Context, db *sql.DB) ([]duckDBObject, error) {
	rows, err := db.QueryContext(ctx, `
select table_schema, table_name,
       case when table_type = 'VIEW' then 'view' else 'table' end as kind
from information_schema.tables
where table_schema not in ('pg_catalog', 'information_schema')
  and table_type in ('BASE TABLE', 'VIEW')
order by table_schema, table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	objects := []duckDBObject{}
	for rows.Next() {
		var schema string
		var name string
		var kind string
		if err := rows.Scan(&schema, &name, &kind); err != nil {
			return nil, err
		}
		objects = append(objects, duckDBObject{
			name: duckDBObjectKey(schema, name),
			kind: kind,
		})
	}
	return objects, rows.Err()
}

func listDuckDBColumns(ctx context.Context, db *sql.DB) (map[string][]ConnectorColumn, error) {
	rows, err := db.QueryContext(ctx, `
select table_schema, table_name, column_name, data_type, is_nullable, coalesce(column_default, '')
from information_schema.columns
where table_schema not in ('pg_catalog', 'information_schema')
order by table_schema, table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string][]ConnectorColumn{}
	for rows.Next() {
		var schema string
		var table string
		var name string
		var dataType string
		var nullable string
		var defaultValue string
		if err := rows.Scan(&schema, &table, &name, &dataType, &nullable, &defaultValue); err != nil {
			return nil, err
		}
		key := duckDBObjectKey(schema, table)
		columns[key] = append(columns[key], ConnectorColumn{
			Name:     name,
			Type:     dataType,
			Nullable: nullable == "YES",
			Default:  defaultValue,
		})
	}
	return columns, rows.Err()
}

func listDuckDBPrimaryKeys(ctx context.Context, db *sql.DB) map[string]map[string]bool {
	rows, err := db.QueryContext(ctx, `
select tc.table_schema, tc.table_name, kcu.column_name
from information_schema.table_constraints tc
join information_schema.key_column_usage kcu
  on tc.constraint_name = kcu.constraint_name
 and tc.table_schema = kcu.table_schema
 and tc.table_name = kcu.table_name
where tc.constraint_type = 'PRIMARY KEY'
  and tc.table_schema not in ('pg_catalog', 'information_schema')`)
	if err != nil {
		return map[string]map[string]bool{}
	}
	defer rows.Close()
	keys := map[string]map[string]bool{}
	for rows.Next() {
		var schema string
		var table string
		var column string
		if err := rows.Scan(&schema, &table, &column); err != nil {
			return map[string]map[string]bool{}
		}
		key := duckDBObjectKey(schema, table)
		if keys[key] == nil {
			keys[key] = map[string]bool{}
		}
		keys[key][column] = true
	}
	if rows.Err() != nil {
		return map[string]map[string]bool{}
	}
	return keys
}

func listDuckDBRelationships(ctx context.Context, db *sql.DB) []ConnectorRelationship {
	rows, err := db.QueryContext(ctx, `
select tc.table_schema, tc.table_name, kcu.column_name,
       ccu.table_schema, ccu.table_name, ccu.column_name
from information_schema.table_constraints tc
join information_schema.key_column_usage kcu
  on tc.constraint_name = kcu.constraint_name
 and tc.table_schema = kcu.table_schema
join information_schema.constraint_column_usage ccu
  on ccu.constraint_name = tc.constraint_name
where tc.constraint_type = 'FOREIGN KEY'
  and tc.table_schema not in ('pg_catalog', 'information_schema')`)
	if err != nil {
		return []ConnectorRelationship{}
	}
	defer rows.Close()
	relationships := []ConnectorRelationship{}
	for rows.Next() {
		var fromSchema string
		var fromTable string
		var fromColumn string
		var toSchema string
		var toTable string
		var toColumn string
		if err := rows.Scan(&fromSchema, &fromTable, &fromColumn, &toSchema, &toTable, &toColumn); err != nil {
			return []ConnectorRelationship{}
		}
		relationships = append(relationships, ConnectorRelationship{
			Kind:       "foreign-key",
			FromTable:  duckDBObjectKey(fromSchema, fromTable),
			FromColumn: fromColumn,
			ToTable:    duckDBObjectKey(toSchema, toTable),
			ToColumn:   toColumn,
			Confidence: "high",
			Reason:     "Declared by DuckDB information_schema foreign-key metadata.",
		})
	}
	if rows.Err() != nil {
		return []ConnectorRelationship{}
	}
	return relationships
}

func duckDBRowCount(ctx context.Context, db *sql.DB, table string) int {
	var count int
	if err := db.QueryRowContext(ctx, "select count(*) from "+quoteDuckDBQualifiedName(table)).Scan(&count); err != nil {
		return 0
	}
	return count
}

func duckDBSampleRows(ctx context.Context, db *sql.DB, table string) [][]string {
	rows, err := db.QueryContext(ctx, "select * from "+quoteDuckDBQualifiedName(table)+" limit ?", maxConnectorSampleRows)
	if err != nil {
		return [][]string{}
	}
	defer rows.Close()
	samples, err := scanConnectorSampleRows(rows)
	if err != nil {
		return [][]string{}
	}
	return samples
}

func quoteDuckDBQualifiedName(name string) string {
	parts := splitQualifiedConnectorName(name)
	if len(parts) == 0 {
		return quoteDoubleIdent(name)
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, quoteDoubleIdent(part))
	}
	return strings.Join(quoted, ".")
}

func applyDuckDBPrimaryKeys(columns []ConnectorColumn, primaryKeys map[string]bool) {
	for index := range columns {
		if primaryKeys[columns[index].Name] {
			columns[index].PrimaryKey = true
			columns[index].Nullable = false
		}
	}
}
