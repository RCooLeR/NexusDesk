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
	_ "github.com/denisenkom/go-mssqldb"
)

func TestSQLServerProfile(profile storage.ConnectorProfile) (ConnectorProfileStatus, error) {
	if err := requireSQLServerProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openSQLServerDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer db.Close()

	conn, err := prepareSQLServerReadOnlyConn(ctx, db, request.TimeoutSeconds)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer conn.Close()

	var version string
	if err := conn.QueryRowContext(ctx, "select @@version").Scan(&version); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	return ConnectorProfileStatus{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Kind:      profile.Kind,
		Engine:    "sqlserver-readonly",
		ReadOnly:  true,
		Message:   fmt.Sprintf("SQL Server read-only connection succeeded for %s.", profile.Name),
	}, nil
}

func QuerySQLServerProfile(profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	request = NormalizeConnectorQueryRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	return QuerySQLServerProfileContext(ctx, profile, request)
}

func QuerySQLServerProfileContext(ctx context.Context, profile storage.ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	if err := requireSQLServerProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	request = NormalizeConnectorQueryRequest(request)
	query, err := normalizeReadOnlyConnectorSQL(request.SQL)
	if err != nil {
		return ConnectorQueryResult{}, err
	}

	db, err := openSQLServerDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer db.Close()

	conn, err := prepareSQLServerReadOnlyConn(ctx, db, request.TimeoutSeconds)
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

	message := fmt.Sprintf("Read-only SQL Server query returned %d rows from %s.", totalRows, profile.Name)
	if truncated {
		message = fmt.Sprintf("Read-only SQL Server query reached the %d row cap for %s.", request.ResultLimit, profile.Name)
	}
	return ConnectorQueryResult{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         "sqlserver-readonly",
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

func InspectSQLServerProfile(profile storage.ConnectorProfile) (ConnectorMetadata, error) {
	if err := requireSQLServerProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := openSQLServerDB(profile, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()

	conn, err := prepareSQLServerReadOnlyConn(ctx, db, request.TimeoutSeconds)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()

	objects, err := listSQLServerObjects(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	columns, err := listSQLServerColumns(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	indexes, err := listSQLServerIndexes(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	relationships, err := listSQLServerRelationships(ctx, conn)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}

	metadata := ConnectorMetadata{
		ID:            "sqlserver:" + profile.ID,
		RelPath:       profile.ID,
		Name:          profile.Name,
		Kind:          "sqlserver",
		Engine:        "sqlserver-readonly",
		ReadOnly:      true,
		Tables:        []ConnectorTable{},
		Views:         []ConnectorTable{},
		Indexes:       []ConnectorIndex{},
		Relationships: relationships,
	}
	for _, object := range objects {
		sampleRows := sqlServerSampleRows(ctx, conn, object.name)
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
	metadata.Message = fmt.Sprintf("SQL Server connector metadata inspected: %d tables, %d views, %d relationships.", len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

func requireSQLServerProfile(profile storage.ConnectorProfile) error {
	if strings.TrimSpace(profile.ID) == "" {
		return errors.New("connector profile id is required")
	}
	if strings.ToLower(strings.TrimSpace(profile.Kind)) != "sqlserver" {
		return errors.New("selected connector profile is not SQL Server")
	}
	if strings.TrimSpace(profile.Host) == "" {
		return errors.New("SQL Server profile needs a host")
	}
	if strings.TrimSpace(profile.Database) == "" {
		return errors.New("SQL Server profile needs a database")
	}
	return nil
}

func openSQLServerDB(profile storage.ConnectorProfile, timeoutSeconds int) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", sqlServerDSN(profile, timeoutSeconds))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	return db, nil
}

func prepareSQLServerReadOnlyConn(ctx context.Context, db *sql.DB, timeoutSeconds int) (*sql.Conn, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	timeoutMillis := timeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set transaction isolation level read committed"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "set lock_timeout "+strconv.Itoa(timeoutMillis)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func sqlServerDSN(profile storage.ConnectorProfile, timeoutSeconds int) string {
	port := profile.Port
	if port <= 0 {
		port = 1433
	}
	dsn := url.URL{
		Scheme: "sqlserver",
		Host:   net.JoinHostPort(profile.Host, strconv.Itoa(port)),
	}
	if profile.Username != "" {
		if profile.Password != "" {
			dsn.User = url.UserPassword(profile.Username, profile.Password)
		} else {
			dsn.User = url.User(profile.Username)
		}
	}
	query := dsn.Query()
	query.Set("database", profile.Database)
	query.Set("connection timeout", strconv.Itoa(timeoutSeconds))
	query.Set("app name", "NexusAugenticStudio")
	if encrypt := sqlServerEncryptMode(profile.SSLMode); encrypt != "" {
		query.Set("encrypt", encrypt)
	}
	if strings.EqualFold(strings.TrimSpace(profile.SSLMode), "skip-verify") {
		query.Set("TrustServerCertificate", "true")
	}
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func sqlServerEncryptMode(sslMode string) string {
	switch strings.ToLower(strings.TrimSpace(sslMode)) {
	case "disable", "false", "off":
		return "disable"
	case "require", "true", "on":
		return "true"
	case "prefer", "preferred":
		return "false"
	case "skip-verify":
		return "true"
	default:
		return ""
	}
}

type sqlServerObject struct {
	name string
	kind string
	rows int
}

func listSQLServerObjects(ctx context.Context, conn *sql.Conn) ([]sqlServerObject, error) {
	rows, err := conn.QueryContext(ctx, `
select s.name + '.' + o.name as object_name,
       case when o.type = 'V' then 'view' else 'table' end as kind,
       coalesce(sum(p.rows), 0) as estimated_rows
from sys.objects o
join sys.schemas s on s.schema_id = o.schema_id
left join sys.partitions p on p.object_id = o.object_id and p.index_id in (0, 1)
where o.type in ('U', 'V')
  and o.is_ms_shipped = 0
group by s.name, o.name, o.type
order by s.name, o.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	objects := []sqlServerObject{}
	for rows.Next() {
		var object sqlServerObject
		if err := rows.Scan(&object.name, &object.kind, &object.rows); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, rows.Err()
}

func listSQLServerColumns(ctx context.Context, conn *sql.Conn) (map[string][]ConnectorColumn, error) {
	rows, err := conn.QueryContext(ctx, `
select s.name + '.' + t.name as table_name,
       c.name,
       ty.name,
       c.is_nullable,
       coalesce(dc.definition, ''),
       case when pk.column_id is null then cast(0 as bit) else cast(1 as bit) end as is_primary_key
from sys.tables t
join sys.schemas s on s.schema_id = t.schema_id
join sys.columns c on c.object_id = t.object_id
join sys.types ty on ty.user_type_id = c.user_type_id
left join sys.default_constraints dc on dc.object_id = c.default_object_id
left join (
  select ic.object_id, ic.column_id
  from sys.indexes i
  join sys.index_columns ic on ic.object_id = i.object_id and ic.index_id = i.index_id
  where i.is_primary_key = 1
) pk on pk.object_id = c.object_id and pk.column_id = c.column_id
where t.is_ms_shipped = 0
order by s.name, t.name, c.column_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string][]ConnectorColumn{}
	for rows.Next() {
		var table string
		var name string
		var dataType string
		var nullable bool
		var defaultValue string
		var primaryKey bool
		if err := rows.Scan(&table, &name, &dataType, &nullable, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[table] = append(columns[table], ConnectorColumn{
			Name:       name,
			Type:       dataType,
			Nullable:   nullable && !primaryKey,
			PrimaryKey: primaryKey,
			Default:    defaultValue,
		})
	}
	return columns, rows.Err()
}

func listSQLServerIndexes(ctx context.Context, conn *sql.Conn) (map[string][]ConnectorIndex, error) {
	rows, err := conn.QueryContext(ctx, `
select s.name + '.' + t.name as table_name,
       i.name,
       i.is_unique,
       string_agg(c.name, ',') within group (order by ic.key_ordinal)
from sys.tables t
join sys.schemas s on s.schema_id = t.schema_id
join sys.indexes i on i.object_id = t.object_id
join sys.index_columns ic on ic.object_id = i.object_id and ic.index_id = i.index_id
join sys.columns c on c.object_id = t.object_id and c.column_id = ic.column_id
where t.is_ms_shipped = 0
  and i.name is not null
  and ic.key_ordinal > 0
group by s.name, t.name, i.name, i.is_unique
order by s.name, t.name, i.name`)
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
			Columns: splitSQLServerCSV(columnCSV),
		})
	}
	return indexes, rows.Err()
}

func listSQLServerRelationships(ctx context.Context, conn *sql.Conn) ([]ConnectorRelationship, error) {
	rows, err := conn.QueryContext(ctx, `
select fs.name + '.' + ft.name as from_table,
       fc.name as from_column,
       ts.name + '.' + tt.name as to_table,
       tc.name as to_column
from sys.foreign_keys fk
join sys.foreign_key_columns fkc on fkc.constraint_object_id = fk.object_id
join sys.tables ft on ft.object_id = fkc.parent_object_id
join sys.schemas fs on fs.schema_id = ft.schema_id
join sys.columns fc on fc.object_id = ft.object_id and fc.column_id = fkc.parent_column_id
join sys.tables tt on tt.object_id = fkc.referenced_object_id
join sys.schemas ts on ts.schema_id = tt.schema_id
join sys.columns tc on tc.object_id = tt.object_id and tc.column_id = fkc.referenced_column_id
order by fs.name, ft.name, fkc.constraint_column_id`)
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
			Reason:     "Declared by SQL Server sys.foreign_key_columns metadata.",
		})
	}
	return relationships, rows.Err()
}

func sqlServerSampleRows(ctx context.Context, conn *sql.Conn, table string) [][]string {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("select top (%d) * from %s", maxConnectorSampleRows, quoteSQLServerQualifiedName(table)))
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

func quoteSQLServerQualifiedName(name string) string {
	parts := splitQualifiedConnectorName(name)
	if len(parts) == 0 {
		return quoteSQLServerIdent(name)
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, quoteSQLServerIdent(part))
	}
	return strings.Join(quoted, ".")
}

func splitSQLServerCSV(value string) []string {
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
