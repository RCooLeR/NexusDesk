package dbconnector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"NexusAugenticStudio/internal/storage"
)

const duckDBEngineName = "duckdb-readonly"

func requireDuckDBProfile(profile storage.ConnectorProfile) error {
	if strings.TrimSpace(profile.ID) == "" {
		return errors.New("connector profile id is required")
	}
	if strings.ToLower(strings.TrimSpace(profile.Kind)) != "duckdb" {
		return errors.New("selected connector profile is not DuckDB")
	}
	path := duckDBProfilePath(profile)
	if path == "" {
		return errors.New("DuckDB profile needs a database file path in Host or Database")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("DuckDB database file is unavailable: %w", err)
	}
	if info.IsDir() {
		return errors.New("DuckDB connector target must be a file")
	}
	return nil
}

func duckDBProfilePath(profile storage.ConnectorProfile) string {
	if database := strings.TrimSpace(profile.Database); database != "" {
		return filepath.Clean(database)
	}
	host := strings.TrimSpace(profile.Host)
	if host == "" {
		return ""
	}
	return filepath.Clean(host)
}

func duckDBDSN(profile storage.ConnectorProfile) string {
	path := filepath.ToSlash(duckDBProfilePath(profile))
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "access_mode=read_only"
}

func duckDBObjectKey(schema string, name string) string {
	schema = strings.TrimSpace(schema)
	if schema == "" || schema == "main" || schema == "temp" {
		return name
	}
	return schema + "." + name
}
