package dbconnector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"NexusAugenticStudio/internal/storage"
	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestMySQLProfile(profile storage.ConnectorProfile) (ConnectorProfileStatus, error) {
	if err := requireMySQLProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openMySQLDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer db.Close()

	conn, err := prepareMySQLReadOnlyConn(ctx, db, request.TimeoutSeconds)
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
		Engine:    mysqlEngine(profile),
		ReadOnly:  true,
		Message:   fmt.Sprintf("%s read-only connection succeeded for %s.", mysqlDisplayName(profile), profile.Name),
	}, nil
}

func QueryMySQLProfile(profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	request = NormalizeConnectorQueryRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	return QueryMySQLProfileContext(ctx, profile, request)
}

func QueryMySQLProfileContext(ctx context.Context, profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	if err := requireMySQLProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	request = NormalizeConnectorQueryRequest(request)
	query, err := normalizeReadOnlyConnectorSQL(request.SQL)
	if err != nil {
		return ConnectorQueryResult{}, err
	}

	db, err := openMySQLDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer db.Close()

	conn, err := prepareMySQLReadOnlyConn(ctx, db, request.TimeoutSeconds)
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

	message := fmt.Sprintf("Read-only %s query returned %d rows from %s.", mysqlDisplayName(profile), totalRows, profile.Name)
	if truncated {
		message = fmt.Sprintf("Read-only %s query reached the %d row cap for %s.", mysqlDisplayName(profile), request.ResultLimit, profile.Name)
	}
	return ConnectorQueryResult{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         mysqlEngine(profile),
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

func InspectMySQLProfile(profile storage.ConnectorProfile) (ConnectorMetadata, error) {
	if err := requireMySQLProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openMySQLDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()

	conn, err := prepareMySQLReadOnlyConn(ctx, db, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()

	objects, err := listMySQLObjects(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	columns, err := listMySQLColumns(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	indexes, err := listMySQLIndexes(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	relationships, err := listMySQLRelationships(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}

	metadata := ConnectorMetadata{
		ID:            profile.Kind + ":" + profile.ID,
		RelPath:       profile.ID,
		Name:          profile.Name,
		Kind:          profile.Kind,
		Engine:        mysqlEngine(profile),
		ReadOnly:      true,
		Tables:        []ConnectorTable{},
		Views:         []ConnectorTable{},
		Indexes:       []ConnectorIndex{},
		Relationships: relationships,
	}
	for _, object := range objects {
		sampleRows := mysqlSampleRows(ctx, conn, object.name)
		table := ConnectorTable{
			Name:       object.name,
			Type:       object.kind,
			RowCount:   object.rows,
			Columns:    columns[object.name],
			Indexes:    indexes[object.name],
			SampleRows: sampleRows,
		}
		if table.Type == "table" {
			metadata.Tables = append(metadata.Tables, table)
			metadata.Indexes = append(metadata.Indexes, table.Indexes...)
		} else {
			metadata.Views = append(metadata.Views, table)
		}
	}
	metadata.Relationships = append(metadata.Relationships, inferredConnectorRelationships(metadata.Tables)...)
	metadata.Message = fmt.Sprintf("%s connector metadata inspected: %d tables, %d views, %d relationships.", mysqlDisplayName(profile), len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

func requireMySQLProfile(profile storage.ConnectorProfile) error {
	kind := strings.ToLower(strings.TrimSpace(profile.Kind))
	if strings.TrimSpace(profile.ID) == "" {
		return errors.New("connector profile id is required")
	}
	if kind != "mysql" && kind != "mariadb" {
		return errors.New("selected connector profile is not MySQL/MariaDB")
	}
	if strings.TrimSpace(profile.Host) == "" {
		return errors.New("MySQL/MariaDB profile needs a host")
	}
	if strings.TrimSpace(profile.Database) == "" {
		return errors.New("MySQL/MariaDB profile needs a database")
	}
	return nil
}

func openMySQLDB(profile storage.ConnectorProfile, timeoutSeconds int) (*sql.DB, error) {
	db, err := sql.Open("mysql", mysqlDSN(profile, timeoutSeconds))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	return db, nil
}

func prepareMySQLReadOnlyConn(ctx context.Context, db *sql.DB, timeoutSeconds int) (*sql.Conn, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "set session transaction read only"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	timeoutMillis := timeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set session max_execution_time = "+strconv.Itoa(timeoutMillis)); err != nil {
		_, _ = conn.ExecContext(ctx, "set session max_statement_time = "+strconv.FormatFloat(float64(timeoutSeconds), 'f', -1, 64))
	}
	return conn, nil
}

func mysqlDSN(profile storage.ConnectorProfile, timeoutSeconds int) string {
	port := profile.Port
	if port <= 0 {
		port = 3306
	}
	tlsMode := mysqlTLSMode(profile.SSLMode)
	config := mysqldriver.NewConfig()
	config.User = profile.Username
	config.Passwd = profile.Password
	config.Net = "tcp"
	config.Addr = profile.Host + ":" + strconv.Itoa(port)
	config.DBName = profile.Database
	config.Timeout = time.Duration(timeoutSeconds) * time.Second
	config.ReadTimeout = time.Duration(timeoutSeconds) * time.Second
	config.WriteTimeout = time.Duration(timeoutSeconds) * time.Second
	config.ParseTime = true
	config.Params = map[string]string{
		"charset": "utf8mb4",
	}
	if tlsMode != "" {
		config.TLSConfig = tlsMode
	}
	return config.FormatDSN()
}

func mysqlTLSMode(sslMode string) string {
	switch strings.ToLower(strings.TrimSpace(sslMode)) {
	case "disable", "false", "off":
		return "false"
	case "require", "true", "on":
		return "true"
	case "skip-verify":
		return "skip-verify"
	case "preferred", "prefer":
		return "preferred"
	default:
		return ""
	}
}

type mysqlObject struct {
	name string
	kind string
	rows int
}

func listMySQLObjects(ctx context.Context, conn *sql.Conn) ([]mysqlObject, error) {
	rows, err := conn.QueryContext(ctx, `
select table_name,
       case when table_type = 'VIEW' then 'view' else 'table' end as kind,
       coalesce(table_rows, 0) as estimated_rows
from information_schema.tables
where table_schema = database()
  and table_type in ('BASE TABLE', 'VIEW')
order by table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	objects := []mysqlObject{}
	for rows.Next() {
		var object mysqlObject
		if err := rows.Scan(&object.name, &object.kind, &object.rows); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, rows.Err()
}

func listMySQLColumns(ctx context.Context, conn *sql.Conn) (map[string][]ConnectorColumn, error) {
	rows, err := conn.QueryContext(ctx, `
select table_name, column_name, column_type, is_nullable, coalesce(column_default, ''), column_key = 'PRI'
from information_schema.columns
where table_schema = database()
order by table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string][]ConnectorColumn{}
	for rows.Next() {
		var table string
		var name string
		var dataType string
		var nullable string
		var defaultValue string
		var primaryKey bool
		if err := rows.Scan(&table, &name, &dataType, &nullable, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[table] = append(columns[table], ConnectorColumn{
			Name:       name,
			Type:       dataType,
			Nullable:   nullable == "YES" && !primaryKey,
			PrimaryKey: primaryKey,
			Default:    defaultValue,
		})
	}
	return columns, rows.Err()
}

func listMySQLIndexes(ctx context.Context, conn *sql.Conn) (map[string][]ConnectorIndex, error) {
	rows, err := conn.QueryContext(ctx, `
select table_name, index_name, non_unique = 0, group_concat(column_name order by seq_in_index separator ',')
from information_schema.statistics
where table_schema = database()
group by table_name, index_name, non_unique
order by table_name, index_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	indexes := map[string][]ConnectorIndex{}
	for rows.Next() {
		var table string
		var name string
		var unique bool
		var columnCSV string
		if err := rows.Scan(&table, &name, &unique, &columnCSV); err != nil {
			return nil, err
		}
		indexes[table] = append(indexes[table], ConnectorIndex{
			Name:    name,
			Table:   table,
			Unique:  unique,
			Columns: splitMySQLCSV(columnCSV),
		})
	}
	return indexes, rows.Err()
}

func listMySQLRelationships(ctx context.Context, conn *sql.Conn) ([]ConnectorRelationship, error) {
	rows, err := conn.QueryContext(ctx, `
select table_name, column_name, referenced_table_name, referenced_column_name
from information_schema.key_column_usage
where table_schema = database()
  and referenced_table_name is not null
order by table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	relationships := []ConnectorRelationship{}
	for rows.Next() {
		var fromTable string
		var fromColumn string
		var toTable string
		var toColumn string
		if err := rows.Scan(&fromTable, &fromColumn, &toTable, &toColumn); err != nil {
			return nil, err
		}
		relationships = append(relationships, ConnectorRelationship{
			Kind:       "foreign-key",
			FromTable:  fromTable,
			FromColumn: fromColumn,
			ToTable:    toTable,
			ToColumn:   toColumn,
			Confidence: "high",
			Reason:     "Declared by MySQL/MariaDB information_schema foreign-key metadata.",
		})
	}
	return relationships, rows.Err()
}

func mysqlSampleRows(ctx context.Context, conn *sql.Conn, table string) [][]string {
	rows, err := conn.QueryContext(ctx, "select * from "+quoteMySQLQualifiedName(table)+" limit ?", maxConnectorSampleRows)
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

func quoteMySQLQualifiedName(name string) string {
	parts := splitQualifiedConnectorName(name)
	if len(parts) == 0 {
		return quoteBacktickIdent(name)
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, quoteBacktickIdent(part))
	}
	return strings.Join(quoted, ".")
}

func splitMySQLCSV(value string) []string {
	parts := strings.Split(value, ",")
	cleaned := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}

func mysqlEngine(profile storage.ConnectorProfile) string {
	if strings.EqualFold(profile.Kind, "mariadb") {
		return "mariadb-readonly"
	}
	return "mysql-readonly"
}

func mysqlDisplayName(profile storage.ConnectorProfile) string {
	if strings.EqualFold(profile.Kind, "mariadb") {
		return "MariaDB"
	}
	return "MySQL"
}
