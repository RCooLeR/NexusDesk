package dbconnector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"NexusAugenticStudio/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type ConnectorProfileStatus struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Engine    string `json:"engine"`
	ReadOnly  bool   `json:"readOnly"`
	Message   string `json:"message"`
}

type ConnectorQueryRequest struct {
	ProfileID      string `json:"profileId"`
	SQL            string `json:"sql"`
	ResultLimit    int    `json:"resultLimit"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type ConnectorQueryResult struct {
	ProfileID      string     `json:"profileId"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	Engine         string     `json:"engine"`
	SQL            string     `json:"sql"`
	Columns        []string   `json:"columns"`
	Rows           [][]string `json:"rows"`
	TotalRows      int        `json:"totalRows"`
	Truncated      bool       `json:"truncated"`
	ResultLimit    int        `json:"resultLimit"`
	TimeoutSeconds int        `json:"timeoutSeconds"`
	Message        string     `json:"message"`
}

func TestPostgresProfile(profile storage.ConnectorProfile) (ConnectorProfileStatus, error) {
	if err := requirePostgresProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openPostgresDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer db.Close()

	conn, err := preparePostgresReadOnlyConn(ctx, db, request.TimeoutSeconds)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer conn.Close()

	var version string
	if err := conn.QueryRowContext(ctx, "select version()").Scan(&version); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	return ConnectorProfileStatus{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Kind:      profile.Kind,
		Engine:    "postgres-readonly",
		ReadOnly:  true,
		Message:   fmt.Sprintf("PostgreSQL read-only connection succeeded for %s.", profile.Name),
	}, nil
}

func QueryPostgresProfile(profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	if err := requirePostgresProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	request = NormalizeConnectorQueryRequest(request)
	query, err := normalizeReadOnlyConnectorSQL(request.SQL)
	if err != nil {
		return ConnectorQueryResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openPostgresDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer db.Close()

	conn, err := preparePostgresReadOnlyConn(ctx, db, request.TimeoutSeconds)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer conn.Close()

	rows, err := conn.QueryContext(ctx, query)
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

	message := fmt.Sprintf("Read-only PostgreSQL query returned %d rows from %s.", totalRows, profile.Name)
	if truncated {
		message = fmt.Sprintf("Read-only PostgreSQL query reached the %d row cap for %s.", request.ResultLimit, profile.Name)
	}
	return ConnectorQueryResult{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         "postgres-readonly",
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

func InspectPostgresProfile(profile storage.ConnectorProfile) (ConnectorMetadata, error) {
	if err := requirePostgresProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openPostgresDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()

	conn, err := preparePostgresReadOnlyConn(ctx, db, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()

	objects, err := listPostgresObjects(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	columns, err := listPostgresColumns(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	indexes, err := listPostgresIndexes(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	relationships, err := listPostgresRelationships(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}

	metadata := ConnectorMetadata{
		ID:            "postgres:" + profile.ID,
		RelPath:       profile.ID,
		Name:          profile.Name,
		Kind:          "postgres",
		Engine:        "postgres-readonly",
		ReadOnly:      true,
		Tables:        []ConnectorTable{},
		Views:         []ConnectorTable{},
		Indexes:       []ConnectorIndex{},
		Relationships: relationships,
	}
	for _, object := range objects {
		key := postgresObjectKey(object.schema, object.name)
		table := ConnectorTable{
			Name:       key,
			Type:       object.kind,
			RowCount:   object.rows,
			Columns:    columns[key],
			Indexes:    indexes[key],
			SampleRows: [][]string{},
		}
		if table.Type == "table" {
			metadata.Tables = append(metadata.Tables, table)
			metadata.Indexes = append(metadata.Indexes, table.Indexes...)
		} else {
			metadata.Views = append(metadata.Views, table)
		}
	}
	metadata.Relationships = append(metadata.Relationships, inferredConnectorRelationships(metadata.Tables)...)
	metadata.Message = fmt.Sprintf("PostgreSQL connector metadata inspected: %d tables, %d views, %d relationships.", len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

func NormalizeConnectorQueryRequest(request ConnectorQueryRequest) ConnectorQueryRequest {
	if request.ResultLimit <= 0 {
		request.ResultLimit = defaultSQLiteRows
	}
	if request.ResultLimit > maxSQLiteRows {
		request.ResultLimit = maxSQLiteRows
	}
	if request.TimeoutSeconds <= 0 {
		request.TimeoutSeconds = defaultSQLiteTimeoutSeconds
	}
	if request.TimeoutSeconds > maxSQLiteTimeoutSeconds {
		request.TimeoutSeconds = maxSQLiteTimeoutSeconds
	}
	request.ProfileID = strings.TrimSpace(request.ProfileID)
	return request
}

func normalizeReadOnlyConnectorSQL(query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("enter a read-only SELECT query")
	}
	if err := validateSingleStatement(query); err != nil {
		return "", err
	}
	for strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	lower := strings.ToLower(query)
	tokens := tokenizeSQL(lower)
	if len(tokens) == 0 || (tokens[0] != "select" && tokens[0] != "with") {
		return "", errors.New("external database connectors only support read-only SELECT queries")
	}
	if containsBlockedSQL(lower) {
		return "", errors.New("external database connector blocks mutating SQL")
	}
	return query, nil
}

func requirePostgresProfile(profile storage.ConnectorProfile) error {
	if strings.TrimSpace(profile.ID) == "" {
		return errors.New("connector profile id is required")
	}
	if strings.ToLower(strings.TrimSpace(profile.Kind)) != "postgres" {
		return errors.New("selected connector profile is not PostgreSQL")
	}
	if strings.TrimSpace(profile.Host) == "" {
		return errors.New("PostgreSQL profile needs a host")
	}
	if strings.TrimSpace(profile.Database) == "" {
		return errors.New("PostgreSQL profile needs a database")
	}
	return nil
}

func openPostgresDB(profile storage.ConnectorProfile, timeoutSeconds int) (*sql.DB, error) {
	db, err := sql.Open("pgx", postgresDSN(profile, timeoutSeconds))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	return db, nil
}

func preparePostgresReadOnlyConn(ctx context.Context, db *sql.DB, timeoutSeconds int) (*sql.Conn, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "set default_transaction_read_only = on"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("set statement_timeout = %d", timeoutSeconds*1000)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func postgresDSN(profile storage.ConnectorProfile, timeoutSeconds int) string {
	port := profile.Port
	if port <= 0 {
		port = 5432
	}
	sslMode := strings.TrimSpace(profile.SSLMode)
	if sslMode == "" {
		sslMode = "prefer"
	}
	dsn := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(profile.Host, strconv.Itoa(port)),
		Path:   profile.Database,
	}
	if profile.Username != "" {
		if profile.Password != "" {
			dsn.User = url.UserPassword(profile.Username, profile.Password)
		} else {
			dsn.User = url.User(profile.Username)
		}
	}
	query := dsn.Query()
	query.Set("sslmode", sslMode)
	query.Set("connect_timeout", strconv.Itoa(timeoutSeconds))
	query.Set("application_name", "NexusAugenticStudio")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

type postgresObject struct {
	schema string
	name   string
	kind   string
	rows   int
}

func listPostgresObjects(ctx context.Context, conn *sql.Conn) ([]postgresObject, error) {
	rows, err := conn.QueryContext(ctx, `
select n.nspname, c.relname,
       case when c.relkind = 'v' then 'view' else 'table' end as kind,
       greatest(c.reltuples::bigint, 0) as estimated_rows
from pg_class c
join pg_namespace n on n.oid = c.relnamespace
where c.relkind in ('r', 'p', 'v')
  and n.nspname not in ('pg_catalog', 'information_schema')
  and n.nspname not like 'pg_toast%'
order by n.nspname, c.relname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	objects := []postgresObject{}
	for rows.Next() {
		var object postgresObject
		if err := rows.Scan(&object.schema, &object.name, &object.kind, &object.rows); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, rows.Err()
}

func listPostgresColumns(ctx context.Context, conn *sql.Conn) (map[string][]ConnectorColumn, error) {
	rows, err := conn.QueryContext(ctx, `
select c.table_schema, c.table_name, c.column_name, c.data_type, c.is_nullable, coalesce(c.column_default, ''),
       case when kcu.column_name is null then false else true end as is_primary_key
from information_schema.columns c
left join information_schema.table_constraints tc
  on tc.table_schema = c.table_schema and tc.table_name = c.table_name and tc.constraint_type = 'PRIMARY KEY'
left join information_schema.key_column_usage kcu
  on kcu.constraint_schema = tc.constraint_schema and kcu.constraint_name = tc.constraint_name and kcu.table_schema = c.table_schema and kcu.table_name = c.table_name and kcu.column_name = c.column_name
where c.table_schema not in ('pg_catalog', 'information_schema')
order by c.table_schema, c.table_name, c.ordinal_position`)
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
		var primaryKey bool
		if err := rows.Scan(&schema, &table, &name, &dataType, &nullable, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		key := postgresObjectKey(schema, table)
		columns[key] = append(columns[key], ConnectorColumn{
			Name:       name,
			Type:       dataType,
			Nullable:   nullable == "YES" && !primaryKey,
			PrimaryKey: primaryKey,
			Default:    defaultValue,
		})
	}
	return columns, rows.Err()
}

func listPostgresIndexes(ctx context.Context, conn *sql.Conn) (map[string][]ConnectorIndex, error) {
	rows, err := conn.QueryContext(ctx, `
select schemaname, tablename, indexname, indexdef
from pg_indexes
where schemaname not in ('pg_catalog', 'information_schema')
order by schemaname, tablename, indexname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	indexes := map[string][]ConnectorIndex{}
	for rows.Next() {
		var schema string
		var table string
		var name string
		var definition string
		if err := rows.Scan(&schema, &table, &name, &definition); err != nil {
			return nil, err
		}
		key := postgresObjectKey(schema, table)
		indexes[key] = append(indexes[key], ConnectorIndex{
			Name:    name,
			Table:   key,
			Unique:  strings.Contains(strings.ToLower(definition), "unique index"),
			Columns: []string{definition},
		})
	}
	return indexes, rows.Err()
}

func listPostgresRelationships(ctx context.Context, conn *sql.Conn) ([]ConnectorRelationship, error) {
	rows, err := conn.QueryContext(ctx, `
select tc.table_schema, tc.table_name, kcu.column_name,
       ccu.table_schema, ccu.table_name, ccu.column_name
from information_schema.table_constraints tc
join information_schema.key_column_usage kcu
  on tc.constraint_name = kcu.constraint_name and tc.table_schema = kcu.table_schema
join information_schema.constraint_column_usage ccu
  on ccu.constraint_name = tc.constraint_name and ccu.constraint_schema = tc.constraint_schema
where tc.constraint_type = 'FOREIGN KEY'
  and tc.table_schema not in ('pg_catalog', 'information_schema')
order by tc.table_schema, tc.table_name, kcu.ordinal_position`)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		relationships = append(relationships, ConnectorRelationship{
			Kind:       "foreign-key",
			FromTable:  postgresObjectKey(fromSchema, fromTable),
			FromColumn: fromColumn,
			ToTable:    postgresObjectKey(toSchema, toTable),
			ToColumn:   toColumn,
			Confidence: "high",
			Reason:     "Declared by PostgreSQL information_schema foreign-key metadata.",
		})
	}
	return relationships, rows.Err()
}

func inferredConnectorRelationships(tables []ConnectorTable) []ConnectorRelationship {
	relationships := []ConnectorRelationship{}
	tableByLowerName := map[string]ConnectorTable{}
	for _, table := range tables {
		tableByLowerName[strings.ToLower(table.Name)] = table
		tableByLowerName[strings.ToLower(unqualifiedConnectorTableName(table.Name))] = table
	}
	seen := map[string]bool{}
	for _, table := range tables {
		for _, column := range table.Columns {
			lowerColumn := strings.ToLower(column.Name)
			if column.PrimaryKey || !strings.HasSuffix(lowerColumn, "_id") {
				continue
			}
			targetStem := strings.TrimSuffix(lowerColumn, "_id")
			target, ok := findLikelySQLiteTargetTable(targetStem, tableByLowerName)
			if !ok || strings.EqualFold(target.Name, table.Name) {
				continue
			}
			relationship := ConnectorRelationship{
				Kind:       "inferred",
				FromTable:  table.Name,
				FromColumn: column.Name,
				ToTable:    target.Name,
				ToColumn:   primaryKeyColumn(target),
				Confidence: "medium",
				Reason:     "Column name follows a *_id pattern that matches another table.",
			}
			key := relationshipKey(relationship)
			if !seen[key] {
				seen[key] = true
				relationships = append(relationships, relationship)
			}
		}
	}
	return relationships
}

func postgresObjectKey(schema string, table string) string {
	if schema == "" || schema == "public" {
		return table
	}
	return schema + "." + table
}

func unqualifiedConnectorTableName(name string) string {
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}
