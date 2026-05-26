package dbconnector

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

const maxConnectorSampleRows = 5

type ConnectorMetadata struct {
	ID       string           `json:"id"`
	RelPath  string           `json:"relPath"`
	Name     string           `json:"name"`
	Kind     string           `json:"kind"`
	Engine   string           `json:"engine"`
	ReadOnly bool             `json:"readOnly"`
	Tables   []ConnectorTable `json:"tables"`
	Views    []ConnectorTable `json:"views"`
	Indexes  []ConnectorIndex `json:"indexes"`
	Message  string           `json:"message"`
}

type ConnectorTable struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	RowCount   int               `json:"rowCount"`
	Columns    []ConnectorColumn `json:"columns"`
	Indexes    []ConnectorIndex  `json:"indexes"`
	SampleRows [][]string        `json:"sampleRows"`
}

type ConnectorColumn struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	PrimaryKey bool   `json:"primaryKey"`
	Default    string `json:"default"`
}

type ConnectorIndex struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Unique  bool     `json:"unique"`
	Columns []string `json:"columns"`
}

func InspectSQLite(root string, relPath string) (ConnectorMetadata, error) {
	_, absPath, cleanRel, err := resolveSQLitePath(root, relPath)
	if err != nil {
		return ConnectorMetadata{}, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(absPath)+"?mode=ro")
	if err != nil {
		return ConnectorMetadata{}, err
	}
	defer db.Close()

	objects, err := listSQLiteObjects(db)
	if err != nil {
		return ConnectorMetadata{}, err
	}
	metadata := ConnectorMetadata{
		ID:       "sqlite:" + cleanRel,
		RelPath:  cleanRel,
		Name:     filepath.Base(cleanRel),
		Kind:     "sqlite",
		Engine:   "sqlite-readonly",
		ReadOnly: true,
		Tables:   []ConnectorTable{},
		Views:    []ConnectorTable{},
		Indexes:  []ConnectorIndex{},
	}
	for _, object := range objects {
		table, err := inspectSQLiteObject(db, object.name, object.kind)
		if err != nil {
			return ConnectorMetadata{}, err
		}
		if object.kind == "view" {
			metadata.Views = append(metadata.Views, table)
		} else {
			metadata.Tables = append(metadata.Tables, table)
			metadata.Indexes = append(metadata.Indexes, table.Indexes...)
		}
	}
	metadata.Message = fmt.Sprintf("SQLite connector metadata inspected: %d tables, %d views.", len(metadata.Tables), len(metadata.Views))
	return metadata, nil
}

type sqliteObject struct {
	name string
	kind string
}

func listSQLiteObjects(db *sql.DB) ([]sqliteObject, error) {
	rows, err := db.Query(`select name, type from sqlite_master where type in ('table', 'view') and name not like 'sqlite_%' order by type, name`)
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

func inspectSQLiteObject(db *sql.DB, name string, kind string) (ConnectorTable, error) {
	columns, err := sqliteColumns(db, name)
	if err != nil {
		return ConnectorTable{}, err
	}
	rowCount, err := sqliteRowCount(db, name)
	if err != nil {
		return ConnectorTable{}, err
	}
	sampleRows, err := sqliteSampleRows(db, name)
	if err != nil {
		return ConnectorTable{}, err
	}
	indexes := []ConnectorIndex{}
	if kind == "table" {
		indexes, err = sqliteIndexes(db, name)
		if err != nil {
			return ConnectorTable{}, err
		}
	}
	return ConnectorTable{
		Name:       name,
		Type:       kind,
		RowCount:   rowCount,
		Columns:    columns,
		Indexes:    indexes,
		SampleRows: sampleRows,
	}, nil
}

func sqliteColumns(db *sql.DB, tableName string) ([]ConnectorColumn, error) {
	rows, err := db.Query("pragma table_info(" + quoteSQLiteIdent(tableName) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := []ConnectorColumn{}
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
		_ = cid
		columns = append(columns, ConnectorColumn{
			Name:       name,
			Type:       columnType,
			Nullable:   notNull == 0 && primaryKey == 0,
			PrimaryKey: primaryKey > 0,
			Default:    defaultValue.String,
		})
	}
	return columns, rows.Err()
}

func sqliteIndexes(db *sql.DB, tableName string) ([]ConnectorIndex, error) {
	rows, err := db.Query("pragma index_list(" + quoteSQLiteIdent(tableName) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	indexes := []ConnectorIndex{}
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
		columns, err := sqliteIndexColumns(db, name)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, ConnectorIndex{
			Name:    name,
			Table:   tableName,
			Unique:  unique != 0,
			Columns: columns,
		})
	}
	return indexes, rows.Err()
}

func sqliteIndexColumns(db *sql.DB, indexName string) ([]string, error) {
	rows, err := db.Query("pragma index_info(" + quoteSQLiteIdent(indexName) + ")")
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
		_ = seqno
		_ = cid
		if name != "" {
			columns = append(columns, name)
		}
	}
	return columns, rows.Err()
}

func sqliteRowCount(db *sql.DB, name string) (int, error) {
	var count int
	err := db.QueryRow("select count(*) from " + quoteSQLiteIdent(name)).Scan(&count)
	return count, err
}

func sqliteSampleRows(db *sql.DB, name string) ([][]string, error) {
	rows, err := db.Query("select * from "+quoteSQLiteIdent(name)+" limit ?1", maxConnectorSampleRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	scanners := rowScanners(len(columns))
	samples := [][]string{}
	for rows.Next() {
		row, err := scanRowAsStrings(rows, scanners)
		if err != nil {
			return nil, err
		}
		samples = append(samples, row)
	}
	return samples, rows.Err()
}

func quoteSQLiteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
