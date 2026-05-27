package dbconnector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const maxSQLiteSampleRows = 5

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) InspectWorkspaceSQLite(root string, relPath string) (SQLiteMetadata, error) {
	_, absPath, cleanRel, err := resolveSQLitePath(root, relPath)
	if err != nil {
		return SQLiteMetadata{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(absPath)+"?mode=ro")
	if err != nil {
		return SQLiteMetadata{}, connectorError(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "pragma query_only = on"); err != nil {
		return SQLiteMetadata{}, connectorError(err)
	}

	objects, err := listSQLiteObjects(ctx, db)
	if err != nil {
		return SQLiteMetadata{}, connectorError(err)
	}
	metadata := SQLiteMetadata{
		ID:       "sqlite:" + cleanRel,
		RelPath:  cleanRel,
		Name:     filepath.Base(cleanRel),
		Engine:   "sqlite-readonly",
		ReadOnly: true,
		Tables:   []SQLiteObject{},
		Views:    []SQLiteObject{},
		Indexes:  []SQLiteIndex{},
	}
	for _, object := range objects {
		table, err := inspectSQLiteObject(ctx, db, object.name, object.kind)
		if err != nil {
			return SQLiteMetadata{}, connectorError(err)
		}
		if object.kind == "view" {
			metadata.Views = append(metadata.Views, table)
			continue
		}
		metadata.Tables = append(metadata.Tables, table)
		metadata.Indexes = append(metadata.Indexes, table.Indexes...)
	}
	metadata.Relationships, err = sqliteRelationships(ctx, db, metadata.Tables)
	if err != nil {
		return SQLiteMetadata{}, connectorError(err)
	}
	metadata.Message = fmt.Sprintf("SQLite connector inspected %d table(s), %d view(s), and %d relationship hint(s).", len(metadata.Tables), len(metadata.Views), len(metadata.Relationships))
	return metadata, nil
}

type sqliteObject struct {
	name string
	kind string
}

func listSQLiteObjects(ctx context.Context, db *sql.DB) ([]sqliteObject, error) {
	rows, err := db.QueryContext(ctx, `select name, type from sqlite_master where type in ('table', 'view') and name not like 'sqlite_%' order by type, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	objects := []sqliteObject{}
	for rows.Next() {
		var object sqliteObject
		if err := rows.Scan(&object.name, &object.kind); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, rows.Err()
}

func inspectSQLiteObject(ctx context.Context, db *sql.DB, name string, kind string) (SQLiteObject, error) {
	columns, err := sqliteColumns(ctx, db, name)
	if err != nil {
		return SQLiteObject{}, err
	}
	rowCount, err := sqliteRowCount(ctx, db, name)
	if err != nil {
		return SQLiteObject{}, err
	}
	sampleRows, err := sqliteSampleRows(ctx, db, name)
	if err != nil {
		return SQLiteObject{}, err
	}
	indexes := []SQLiteIndex{}
	if kind == "table" {
		indexes, err = sqliteIndexes(ctx, db, name)
		if err != nil {
			return SQLiteObject{}, err
		}
	}
	return SQLiteObject{
		Name:       name,
		Type:       kind,
		RowCount:   rowCount,
		Columns:    columns,
		Indexes:    indexes,
		SampleRows: sampleRows,
	}, nil
}

func sqliteColumns(ctx context.Context, db *sql.DB, tableName string) ([]SQLiteColumn, error) {
	rows, err := db.QueryContext(ctx, "pragma table_info("+quoteSQLiteIdent(tableName)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := []SQLiteColumn{}
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns = append(columns, SQLiteColumn{
			Name:       name,
			Type:       columnType,
			Nullable:   notNull == 0 && primaryKey == 0,
			PrimaryKey: primaryKey > 0,
			Default:    defaultValue.String,
		})
	}
	return columns, rows.Err()
}

func sqliteIndexes(ctx context.Context, db *sql.DB, tableName string) ([]SQLiteIndex, error) {
	rows, err := db.QueryContext(ctx, "pragma index_list("+quoteSQLiteIdent(tableName)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	indexes := []SQLiteIndex{}
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}
		_ = seq
		_ = origin
		_ = partial
		columns, err := sqliteIndexColumns(ctx, db, name)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, SQLiteIndex{Name: name, Table: tableName, Unique: unique != 0, Columns: columns})
	}
	return indexes, rows.Err()
}

func sqliteIndexColumns(ctx context.Context, db *sql.DB, indexName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "pragma index_info("+quoteSQLiteIdent(indexName)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := []string{}
	for rows.Next() {
		var seqno int
		var cid int
		var name string
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}
		if name != "" {
			columns = append(columns, name)
		}
	}
	return columns, rows.Err()
}

func sqliteRelationships(ctx context.Context, db *sql.DB, tables []SQLiteObject) ([]SQLiteRelationship, error) {
	relationships := []SQLiteRelationship{}
	seen := map[string]bool{}
	tableByLowerName := map[string]SQLiteObject{}
	for _, table := range tables {
		tableByLowerName[strings.ToLower(table.Name)] = table
	}
	for _, table := range tables {
		explicit, err := sqliteForeignKeys(ctx, db, table)
		if err != nil {
			return nil, err
		}
		for _, relationship := range explicit {
			if relationship.ToColumn == "" {
				if target, ok := tableByLowerName[strings.ToLower(relationship.ToTable)]; ok {
					relationship.ToColumn = primaryKeyColumn(target)
				}
			}
			key := relationshipKey(relationship)
			if !seen[key] {
				seen[key] = true
				relationships = append(relationships, relationship)
			}
		}
	}
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
			relationship := SQLiteRelationship{
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
	return relationships, nil
}

func sqliteForeignKeys(ctx context.Context, db *sql.DB, table SQLiteObject) ([]SQLiteRelationship, error) {
	rows, err := db.QueryContext(ctx, "pragma foreign_key_list("+quoteSQLiteIdent(table.Name)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	relationships := []SQLiteRelationship{}
	for rows.Next() {
		var id int
		var seq int
		var toTable string
		var fromColumn string
		var toColumn string
		var onUpdate string
		var onDelete string
		var match string
		if err := rows.Scan(&id, &seq, &toTable, &fromColumn, &toColumn, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		_ = id
		_ = seq
		_ = onUpdate
		_ = onDelete
		_ = match
		relationships = append(relationships, SQLiteRelationship{
			Kind:       "foreign-key",
			FromTable:  table.Name,
			FromColumn: fromColumn,
			ToTable:    toTable,
			ToColumn:   toColumn,
			Confidence: "high",
			Reason:     "Declared by SQLite foreign_key_list metadata.",
		})
	}
	return relationships, rows.Err()
}

func findLikelySQLiteTargetTable(stem string, tableByLowerName map[string]SQLiteObject) (SQLiteObject, bool) {
	candidates := []string{stem, stem + "s", stem + "es"}
	if strings.HasSuffix(stem, "y") {
		candidates = append(candidates, strings.TrimSuffix(stem, "y")+"ies")
	}
	for _, candidate := range candidates {
		if table, ok := tableByLowerName[candidate]; ok {
			return table, true
		}
	}
	return SQLiteObject{}, false
}

func primaryKeyColumn(table SQLiteObject) string {
	for _, column := range table.Columns {
		if column.PrimaryKey {
			return column.Name
		}
	}
	return "id"
}

func relationshipKey(relationship SQLiteRelationship) string {
	return strings.ToLower(strings.Join([]string{
		relationship.FromTable,
		relationship.FromColumn,
		relationship.ToTable,
		relationship.ToColumn,
	}, "\x00"))
}

func sqliteRowCount(ctx context.Context, db *sql.DB, name string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "select count(*) from "+quoteSQLiteIdent(name)).Scan(&count)
	return count, err
}

func sqliteSampleRows(ctx context.Context, db *sql.DB, name string) ([][]string, error) {
	rows, err := db.QueryContext(ctx, "select * from "+quoteSQLiteIdent(name)+" limit ?1", maxSQLiteSampleRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowsAsStrings(rows)
}

func quoteSQLiteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func connectorError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("connector query was canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("connector query timed out")
	}
	return fmt.Errorf("%s", err.Error())
}
