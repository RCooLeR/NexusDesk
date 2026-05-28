package dbconnector

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const maxConnectorSampleRows = 5

type ConnectorMetadata struct {
	ProfileID      string
	Name           string
	Kind           string
	Engine         string
	ReadOnly       bool
	Tables         []ConnectorTable
	Views          []ConnectorTable
	Indexes        []ConnectorIndex
	Relationships  []ConnectorRelationship
	InspectedAtUTC time.Time
	Message        string
}

type ConnectorTable struct {
	Name       string
	Type       string
	RowCount   int
	Columns    []ConnectorColumn
	Indexes    []ConnectorIndex
	SampleRows [][]string
}

type ConnectorColumn struct {
	Name       string
	Type       string
	Nullable   bool
	PrimaryKey bool
	Default    string
}

type ConnectorIndex struct {
	Name    string
	Table   string
	Unique  bool
	Columns []string
}

type ConnectorRelationship struct {
	Kind       string
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
	Confidence string
	Reason     string
}

func (s *Service) InspectConnectorProfile(profile ConnectorProfile) (ConnectorMetadata, error) {
	profile = normalizeConnectorProfile(profile)
	timeout := profile.TimeoutSeconds
	if timeout <= 0 {
		timeout = defaultConnectorTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	return s.InspectConnectorProfileContext(ctx, profile)
}

func (s *Service) InspectConnectorProfileContext(ctx context.Context, profile ConnectorProfile) (ConnectorMetadata, error) {
	profile = normalizeConnectorProfile(profile)
	timeout := profile.TimeoutSeconds
	if timeout <= 0 {
		timeout = defaultConnectorTimeoutSeconds
	}
	switch strings.ToLower(profile.Kind) {
	case "postgres":
		return inspectPostgresProfile(ctx, profile, timeout)
	case "mysql", "mariadb":
		return inspectMySQLProfile(ctx, profile, timeout)
	case "sqlserver":
		return inspectSQLServerProfile(ctx, profile, timeout)
	case "duckdb":
		return inspectDuckDBProfile(ctx, profile, timeout)
	case "sqlite":
		return inspectSQLiteProfile(ctx, profile, timeout)
	default:
		return ConnectorMetadata{}, fmt.Errorf("connector kind %q is not inspectable yet", profile.Kind)
	}
}

func inspectPostgresProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorMetadata, error) {
	if err := requirePostgresProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	db, err := sql.Open("pgx", postgresDSN(profile, timeoutSeconds))
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "set default_transaction_read_only = on"); err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("set statement_timeout = %d", timeoutSeconds*1000)); err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	return inspectGenericInformationSchema(ctx, conn, profile, "postgres-readonly", "PostgreSQL", postgresInformationSchemaQueries())
}

func inspectMySQLProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorMetadata, error) {
	if err := requireMySQLProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	db, err := sql.Open("mysql", mysqlDSN(profile, timeoutSeconds))
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "set session transaction read only"); err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	return inspectGenericInformationSchema(ctx, conn, profile, mysqlEngine(profile), mysqlDisplayName(profile), mysqlInformationSchemaQueries())
}

func inspectSQLServerProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorMetadata, error) {
	if err := requireSQLServerProfile(profile); err != nil {
		return ConnectorMetadata{}, err
	}
	db, err := sql.Open("sqlserver", sqlServerDSN(profile, timeoutSeconds))
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()
	timeoutMillis := timeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set transaction isolation level read committed"); err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	if _, err := conn.ExecContext(ctx, "set lock_timeout "+fmt.Sprintf("%d", timeoutMillis)); err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	return inspectGenericInformationSchema(ctx, conn, profile, "sqlserver-readonly", "SQL Server", sqlServerInformationSchemaQueries())
}

func inspectSQLiteProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorMetadata, error) {
	_ = timeoutSeconds
	path, err := sqliteProfilePath(profile)
	if err != nil {
		return ConnectorMetadata{}, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "pragma query_only = on"); err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	objects, err := listSQLiteObjects(ctx, db)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	metadata := ConnectorMetadata{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         "sqlite-readonly",
		ReadOnly:       true,
		Tables:         []ConnectorTable{},
		Views:          []ConnectorTable{},
		Indexes:        []ConnectorIndex{},
		Relationships:  []ConnectorRelationship{},
		InspectedAtUTC: time.Now().UTC(),
	}
	for _, object := range objects {
		table, err := inspectSQLiteObject(ctx, db, object.name, object.kind)
		if err != nil {
			return ConnectorMetadata{}, connectorError(err)
		}
		mapped := ConnectorTable{
			Name:       table.Name,
			Type:       table.Type,
			RowCount:   table.RowCount,
			Columns:    mapSQLiteColumns(table.Columns),
			Indexes:    mapSQLiteIndexes(table.Indexes),
			SampleRows: table.SampleRows,
		}
		if object.kind == "view" {
			metadata.Views = append(metadata.Views, mapped)
			continue
		}
		metadata.Tables = append(metadata.Tables, mapped)
		metadata.Indexes = append(metadata.Indexes, mapped.Indexes...)
	}
	rels, err := sqliteRelationships(ctx, db, mapConnectorTablesToSQLite(metadata.Tables))
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	metadata.Relationships = mapSQLiteRelationships(rels)
	metadata.Message = fmt.Sprintf("SQLite connector metadata inspected: %d tables, %d views, %d relationships.", len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

func inspectDuckDBProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorMetadata, error) {
	path, err := duckDBProfilePath(profile)
	if err != nil {
		return ConnectorMetadata{}, err
	}
	if err := ensureDuckDBDriverEnabled(); err != nil {
		return ConnectorMetadata{}, err
	}
	db, err := sql.Open("duckdb", duckDBDSN(path))
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	defer conn.Close()
	return inspectGenericInformationSchema(ctx, conn, profile, "duckdb-readonly", "DuckDB", duckDBInformationSchemaQueries())
}

type informationSchemaQueries struct {
	objectsSQL       string
	columnsSQL       string
	indexesSQL       string
	relationshipsSQL string
}

func inspectGenericInformationSchema(ctx context.Context, conn *sql.Conn, profile ConnectorProfile, engine string, displayName string, queries informationSchemaQueries) (ConnectorMetadata, error) {
	objects, err := readConnectorObjects(ctx, conn, queries.objectsSQL)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	columns, err := readConnectorColumns(ctx, conn, queries.columnsSQL)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	indexes, err := readConnectorIndexes(ctx, conn, queries.indexesSQL)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	relationships, err := readConnectorRelationships(ctx, conn, queries.relationshipsSQL)
	if err != nil {
		return ConnectorMetadata{}, connectorError(err)
	}
	metadata := ConnectorMetadata{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         engine,
		ReadOnly:       true,
		Tables:         []ConnectorTable{},
		Views:          []ConnectorTable{},
		Indexes:        []ConnectorIndex{},
		Relationships:  relationships,
		InspectedAtUTC: time.Now().UTC(),
	}
	for _, object := range objects {
		table := ConnectorTable{
			Name:       object.Name,
			Type:       object.Type,
			RowCount:   object.RowCount,
			Columns:    columns[object.Name],
			Indexes:    indexes[object.Name],
			SampleRows: sampleConnectorRows(ctx, conn, object.Name),
		}
		if table.Type == "table" {
			metadata.Tables = append(metadata.Tables, table)
			metadata.Indexes = append(metadata.Indexes, table.Indexes...)
		} else {
			metadata.Views = append(metadata.Views, table)
		}
	}
	metadata.Relationships = append(metadata.Relationships, inferredConnectorRelationships(metadata.Tables)...)
	metadata.Message = fmt.Sprintf("%s connector metadata inspected: %d tables, %d views, %d relationships.", displayName, len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

type connectorObject struct {
	Name     string
	Type     string
	RowCount int
}

func readConnectorObjects(ctx context.Context, conn *sql.Conn, query string) ([]connectorObject, error) {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	objects := []connectorObject{}
	for rows.Next() {
		var object connectorObject
		if err := rows.Scan(&object.Name, &object.Type, &object.RowCount); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, rows.Err()
}

func readConnectorColumns(ctx context.Context, conn *sql.Conn, query string) (map[string][]ConnectorColumn, error) {
	rows, err := conn.QueryContext(ctx, query)
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

func readConnectorIndexes(ctx context.Context, conn *sql.Conn, query string) (map[string][]ConnectorIndex, error) {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	indexes := map[string][]ConnectorIndex{}
	for rows.Next() {
		var table string
		var name string
		var unique bool
		var csv string
		if err := rows.Scan(&table, &name, &unique, &csv); err != nil {
			return nil, err
		}
		indexes[table] = append(indexes[table], ConnectorIndex{
			Name:    name,
			Table:   table,
			Unique:  unique,
			Columns: splitConnectorCSV(csv),
		})
	}
	return indexes, rows.Err()
}

func readConnectorRelationships(ctx context.Context, conn *sql.Conn, query string) ([]ConnectorRelationship, error) {
	rows, err := conn.QueryContext(ctx, query)
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
			Reason:     "Declared by connector metadata.",
		})
	}
	return relationships, rows.Err()
}

func sampleConnectorRows(ctx context.Context, conn *sql.Conn, table string) [][]string {
	query := "select * from " + quoteConnectorQualifiedName(table) + " limit ?"
	rows, err := conn.QueryContext(ctx, query, maxConnectorSampleRows)
	if err != nil {
		return [][]string{}
	}
	defer rows.Close()
	samples, err := scanRowsAsStrings(rows)
	if err != nil {
		return [][]string{}
	}
	return samples
}

func splitConnectorCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func quoteConnectorQualifiedName(name string) string {
	parts := strings.Split(strings.TrimSpace(name), ".")
	if len(parts) == 0 {
		return name
	}
	for index := range parts {
		parts[index] = `"` + strings.ReplaceAll(parts[index], `"`, `""`) + `"`
	}
	return strings.Join(parts, ".")
}

func postgresInformationSchemaQueries() informationSchemaQueries {
	return informationSchemaQueries{
		objectsSQL: `
select n.nspname || '.' || c.relname as object_name,
       case when c.relkind = 'v' then 'view' else 'table' end as kind,
       greatest(c.reltuples::bigint, 0) as estimated_rows
from pg_class c
join pg_namespace n on n.oid = c.relnamespace
where c.relkind in ('r', 'p', 'v')
  and n.nspname not in ('pg_catalog', 'information_schema')
  and n.nspname not like 'pg_toast%'
order by n.nspname, c.relname`,
		columnsSQL: `
select c.table_schema || '.' || c.table_name as table_name,
       c.column_name,
       c.data_type,
       (c.is_nullable = 'YES') as is_nullable,
       coalesce(c.column_default, '') as default_value,
       case when kcu.column_name is null then false else true end as is_primary_key
from information_schema.columns c
left join information_schema.table_constraints tc
  on tc.table_schema = c.table_schema and tc.table_name = c.table_name and tc.constraint_type = 'PRIMARY KEY'
left join information_schema.key_column_usage kcu
  on kcu.constraint_schema = tc.constraint_schema and kcu.constraint_name = tc.constraint_name and kcu.table_schema = c.table_schema and kcu.table_name = c.table_name and kcu.column_name = c.column_name
where c.table_schema not in ('pg_catalog', 'information_schema')
order by c.table_schema, c.table_name, c.ordinal_position`,
		indexesSQL: `
select schemaname || '.' || tablename as table_name,
       indexname,
       (position('UNIQUE' in upper(indexdef)) > 0) as is_unique,
       regexp_replace(regexp_replace(indexdef, '.*\((.*)\).*', '\1'), '\s+', '', 'g') as index_columns
from pg_indexes
where schemaname not in ('pg_catalog', 'information_schema')
order by schemaname, tablename, indexname`,
		relationshipsSQL: `
select tc.table_schema || '.' || tc.table_name as from_table,
       kcu.column_name as from_column,
       ccu.table_schema || '.' || ccu.table_name as to_table,
       ccu.column_name as to_column
from information_schema.table_constraints tc
join information_schema.key_column_usage kcu
  on kcu.constraint_schema = tc.constraint_schema and kcu.constraint_name = tc.constraint_name
join information_schema.constraint_column_usage ccu
  on ccu.constraint_schema = tc.constraint_schema and ccu.constraint_name = tc.constraint_name
where tc.constraint_type = 'FOREIGN KEY'
order by tc.table_schema, tc.table_name, kcu.ordinal_position`,
	}
}

func mysqlInformationSchemaQueries() informationSchemaQueries {
	return informationSchemaQueries{
		objectsSQL: `
select table_name,
       case when table_type = 'VIEW' then 'view' else 'table' end as kind,
       coalesce(table_rows, 0) as estimated_rows
from information_schema.tables
where table_schema = database()
  and table_type in ('BASE TABLE', 'VIEW')
order by table_name`,
		columnsSQL: `
select table_name,
       column_name,
       column_type,
       (is_nullable = 'YES') as is_nullable,
       coalesce(column_default, '') as default_value,
       (column_key = 'PRI') as is_primary_key
from information_schema.columns
where table_schema = database()
order by table_name, ordinal_position`,
		indexesSQL: `
select table_name,
       index_name,
       (non_unique = 0) as is_unique,
       group_concat(column_name order by seq_in_index separator ',') as index_columns
from information_schema.statistics
where table_schema = database()
group by table_name, index_name, non_unique
order by table_name, index_name`,
		relationshipsSQL: `
select table_name as from_table,
       column_name as from_column,
       referenced_table_name as to_table,
       referenced_column_name as to_column
from information_schema.key_column_usage
where table_schema = database()
  and referenced_table_name is not null
order by table_name, ordinal_position`,
	}
}

func sqlServerInformationSchemaQueries() informationSchemaQueries {
	return informationSchemaQueries{
		objectsSQL: `
select s.name + '.' + o.name as object_name,
       case when o.type = 'V' then 'view' else 'table' end as kind,
       coalesce(sum(p.rows), 0) as estimated_rows
from sys.objects o
join sys.schemas s on s.schema_id = o.schema_id
left join sys.partitions p on p.object_id = o.object_id and p.index_id in (0, 1)
where o.type in ('U', 'V')
  and o.is_ms_shipped = 0
group by s.name, o.name, o.type
order by s.name, o.name`,
		columnsSQL: `
select s.name + '.' + t.name as table_name,
       c.name,
       ty.name,
       c.is_nullable,
       coalesce(dc.definition, '') as default_value,
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
order by s.name, t.name, c.column_id`,
		indexesSQL: `
select s.name + '.' + t.name as table_name,
       i.name,
       i.is_unique,
       string_agg(c.name, ',') within group (order by ic.key_ordinal) as index_columns
from sys.tables t
join sys.schemas s on s.schema_id = t.schema_id
join sys.indexes i on i.object_id = t.object_id
join sys.index_columns ic on ic.object_id = i.object_id and ic.index_id = i.index_id
join sys.columns c on c.object_id = t.object_id and c.column_id = ic.column_id
where t.is_ms_shipped = 0
  and i.name is not null
  and ic.key_ordinal > 0
group by s.name, t.name, i.name, i.is_unique
order by s.name, t.name, i.name`,
		relationshipsSQL: `
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
order by fs.name, ft.name, fkc.constraint_column_id`,
	}
}

func duckDBInformationSchemaQueries() informationSchemaQueries {
	return informationSchemaQueries{
		objectsSQL: `
select table_schema || '.' || table_name as object_name,
       case when table_type = 'VIEW' then 'view' else 'table' end as kind,
       0 as estimated_rows
from information_schema.tables
where table_schema not in ('information_schema', 'pg_catalog')
  and table_type in ('BASE TABLE', 'VIEW', 'LOCAL TEMPORARY')
order by table_schema, table_name`,
		columnsSQL: `
select table_schema || '.' || table_name as table_name,
       column_name,
       data_type,
       (is_nullable = 'YES') as is_nullable,
       coalesce(column_default, '') as default_value,
       false as is_primary_key
from information_schema.columns
where table_schema not in ('information_schema', 'pg_catalog')
order by table_schema, table_name, ordinal_position`,
		indexesSQL: `
select '' as table_name,
       '' as index_name,
       false as is_unique,
       '' as index_columns
where 1 = 0`,
		relationshipsSQL: `
select '' as from_table,
       '' as from_column,
       '' as to_table,
       '' as to_column
where 1 = 0`,
	}
}

func inferredConnectorRelationships(tables []ConnectorTable) []ConnectorRelationship {
	relationships := []ConnectorRelationship{}
	seen := map[string]bool{}
	tableByLower := map[string]ConnectorTable{}
	for _, table := range tables {
		tableByLower[strings.ToLower(table.Name)] = table
	}
	for _, table := range tables {
		for _, column := range table.Columns {
			lowerColumn := strings.ToLower(column.Name)
			if column.PrimaryKey || !strings.HasSuffix(lowerColumn, "_id") {
				continue
			}
			stem := strings.TrimSuffix(lowerColumn, "_id")
			target, ok := findLikelyConnectorTargetTable(stem, tableByLower)
			if !ok || strings.EqualFold(target.Name, table.Name) {
				continue
			}
			relationship := ConnectorRelationship{
				Kind:       "inferred",
				FromTable:  table.Name,
				FromColumn: column.Name,
				ToTable:    target.Name,
				ToColumn:   primaryConnectorKey(target),
				Confidence: "medium",
				Reason:     "Column name follows a *_id pattern that matches another table.",
			}
			key := strings.ToLower(strings.Join([]string{
				relationship.FromTable,
				relationship.FromColumn,
				relationship.ToTable,
				relationship.ToColumn,
			}, "\x00"))
			if !seen[key] {
				seen[key] = true
				relationships = append(relationships, relationship)
			}
		}
	}
	return relationships
}

func findLikelyConnectorTargetTable(stem string, tableByLower map[string]ConnectorTable) (ConnectorTable, bool) {
	candidates := []string{stem, stem + "s", stem + "es"}
	if strings.HasSuffix(stem, "y") {
		candidates = append(candidates, strings.TrimSuffix(stem, "y")+"ies")
	}
	for _, candidate := range candidates {
		if table, ok := tableByLower[candidate]; ok {
			return table, true
		}
	}
	return ConnectorTable{}, false
}

func primaryConnectorKey(table ConnectorTable) string {
	for _, column := range table.Columns {
		if column.PrimaryKey {
			return column.Name
		}
	}
	return "id"
}

func mapSQLiteColumns(columns []SQLiteColumn) []ConnectorColumn {
	result := make([]ConnectorColumn, 0, len(columns))
	for _, column := range columns {
		result = append(result, ConnectorColumn{
			Name:       column.Name,
			Type:       column.Type,
			Nullable:   column.Nullable,
			PrimaryKey: column.PrimaryKey,
			Default:    column.Default,
		})
	}
	return result
}

func mapSQLiteIndexes(indexes []SQLiteIndex) []ConnectorIndex {
	result := make([]ConnectorIndex, 0, len(indexes))
	for _, index := range indexes {
		result = append(result, ConnectorIndex{
			Name:    index.Name,
			Table:   index.Table,
			Unique:  index.Unique,
			Columns: append([]string{}, index.Columns...),
		})
	}
	return result
}

func mapSQLiteRelationships(relationships []SQLiteRelationship) []ConnectorRelationship {
	result := make([]ConnectorRelationship, 0, len(relationships))
	for _, relationship := range relationships {
		result = append(result, ConnectorRelationship{
			Kind:       relationship.Kind,
			FromTable:  relationship.FromTable,
			FromColumn: relationship.FromColumn,
			ToTable:    relationship.ToTable,
			ToColumn:   relationship.ToColumn,
			Confidence: relationship.Confidence,
			Reason:     relationship.Reason,
		})
	}
	return result
}

func mapConnectorTablesToSQLite(tables []ConnectorTable) []SQLiteObject {
	result := make([]SQLiteObject, 0, len(tables))
	for _, table := range tables {
		columns := make([]SQLiteColumn, 0, len(table.Columns))
		for _, column := range table.Columns {
			columns = append(columns, SQLiteColumn{
				Name:       column.Name,
				Type:       column.Type,
				Nullable:   column.Nullable,
				PrimaryKey: column.PrimaryKey,
				Default:    column.Default,
			})
		}
		result = append(result, SQLiteObject{
			Name:       table.Name,
			Type:       table.Type,
			RowCount:   table.RowCount,
			Columns:    columns,
			Indexes:    []SQLiteIndex{},
			SampleRows: table.SampleRows,
		})
	}
	return result
}
