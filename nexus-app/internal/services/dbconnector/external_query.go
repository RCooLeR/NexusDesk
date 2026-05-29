package dbconnector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "modernc.org/sqlite"
)

type ConnectorProfileStatus struct {
	ProfileID string
	Name      string
	Kind      string
	Engine    string
	ReadOnly  bool
	Message   string
}

type ConnectorQueryRequest struct {
	ProfileID      string
	SQL            string
	RequestID      string
	ResultLimit    int
	TimeoutSeconds int
}

type ConnectorQueryResult struct {
	ProfileID      string
	Name           string
	Kind           string
	Engine         string
	SQL            string
	Columns        []string
	Rows           [][]string
	TotalRows      int
	Truncated      bool
	ResultLimit    int
	TimeoutSeconds int
	DurationMs     int64
	Message        string
}

func NormalizeConnectorQueryRequest(request ConnectorQueryRequest) ConnectorQueryRequest {
	if request.ResultLimit <= 0 {
		request.ResultLimit = defaultConnectorResultLimit
	}
	if request.ResultLimit > maxConnectorResultLimit {
		request.ResultLimit = maxConnectorResultLimit
	}
	if request.TimeoutSeconds <= 0 {
		request.TimeoutSeconds = defaultConnectorTimeoutSeconds
	}
	if request.TimeoutSeconds > maxConnectorTimeoutSeconds {
		request.TimeoutSeconds = maxConnectorTimeoutSeconds
	}
	request.ProfileID = strings.TrimSpace(request.ProfileID)
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.SQL = strings.TrimSpace(request.SQL)
	return request
}

func (s *Service) TestConnectorProfile(profile ConnectorProfile) (ConnectorProfileStatus, error) {
	profile = normalizeConnectorProfile(profile)
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      profile.ID,
		ResultLimit:    profile.ResultLimit,
		TimeoutSeconds: profile.TimeoutSeconds,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	return s.TestConnectorProfileContext(ctx, profile)
}

func (s *Service) TestConnectorProfileContext(ctx context.Context, profile ConnectorProfile) (ConnectorProfileStatus, error) {
	profile = normalizeConnectorProfile(profile)
	timeoutSeconds := profile.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultConnectorTimeoutSeconds
	}
	switch strings.ToLower(profile.Kind) {
	case "postgres":
		return s.testPostgresProfile(ctx, profile, timeoutSeconds)
	case "mysql", "mariadb":
		return s.testMySQLProfile(ctx, profile, timeoutSeconds)
	case "sqlserver":
		return s.testSQLServerProfile(ctx, profile, timeoutSeconds)
	case "duckdb":
		return testDuckDBProfile(ctx, profile, timeoutSeconds)
	case "sqlite":
		return testSQLiteProfile(ctx, profile, timeoutSeconds)
	default:
		return ConnectorProfileStatus{}, fmt.Errorf("connector kind %q is not runnable yet", profile.Kind)
	}
}

func (s *Service) QueryConnectorProfile(profile ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	request = NormalizeConnectorQueryRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	return s.QueryConnectorProfileContext(ctx, profile, request)
}

func (s *Service) QueryConnectorProfileContext(ctx context.Context, profile ConnectorProfile, request ConnectorQueryRequest) (ConnectorQueryResult, error) {
	profile = normalizeConnectorProfile(profile)
	request = NormalizeConnectorQueryRequest(request)
	query, err := NormalizeExternalReadOnlySQLForKind(profile.Kind, request.SQL)
	if err != nil {
		return ConnectorQueryResult{}, err
	}
	switch strings.ToLower(profile.Kind) {
	case "postgres":
		return s.queryPostgresProfileContext(ctx, profile, request, query)
	case "mysql", "mariadb":
		return s.queryMySQLProfileContext(ctx, profile, request, query)
	case "sqlserver":
		return s.querySQLServerProfileContext(ctx, profile, request, query)
	case "duckdb":
		return queryDuckDBProfileContext(ctx, profile, request, query)
	case "sqlite":
		return querySQLiteProfileContext(ctx, profile, request, query)
	default:
		return ConnectorQueryResult{}, fmt.Errorf("connector kind %q is not queryable yet", profile.Kind)
	}
}

func (s *Service) testPostgresProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorProfileStatus, error) {
	if err := requirePostgresProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	db, err := s.externalConnectorDB(profile, "pgx", postgresDSN(profile, timeoutSeconds))
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "set default_transaction_read_only = on"); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("set statement_timeout = %d", timeoutSeconds*1000)); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
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

func (s *Service) queryPostgresProfileContext(ctx context.Context, profile ConnectorProfile, request ConnectorQueryRequest, query string) (ConnectorQueryResult, error) {
	if err := requirePostgresProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	db, err := s.externalConnectorDB(profile, "pgx", postgresDSN(profile, request.TimeoutSeconds))
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "set default_transaction_read_only = on"); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("set statement_timeout = %d", request.TimeoutSeconds*1000)); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	return runConnectorQuery(ctx, conn, profile, request, query, "postgres-readonly", "PostgreSQL")
}

func (s *Service) testMySQLProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorProfileStatus, error) {
	if err := requireMySQLProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	db, err := s.externalConnectorDB(profile, "mysql", mysqlDSN(profile, timeoutSeconds))
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "set session transaction read only"); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	timeoutMillis := timeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set session max_execution_time = "+strconv.Itoa(timeoutMillis)); err != nil {
		_, _ = conn.ExecContext(ctx, "set session max_statement_time = "+strconv.FormatFloat(float64(timeoutSeconds), 'f', -1, 64))
	}
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

func (s *Service) queryMySQLProfileContext(ctx context.Context, profile ConnectorProfile, request ConnectorQueryRequest, query string) (ConnectorQueryResult, error) {
	if err := requireMySQLProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	db, err := s.externalConnectorDB(profile, "mysql", mysqlDSN(profile, request.TimeoutSeconds))
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "set session transaction read only"); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	timeoutMillis := request.TimeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set session max_execution_time = "+strconv.Itoa(timeoutMillis)); err != nil {
		_, _ = conn.ExecContext(ctx, "set session max_statement_time = "+strconv.FormatFloat(float64(request.TimeoutSeconds), 'f', -1, 64))
	}
	return runConnectorQuery(ctx, conn, profile, request, query, mysqlEngine(profile), mysqlDisplayName(profile))
}

func (s *Service) testSQLServerProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorProfileStatus, error) {
	if err := requireSQLServerProfile(profile); err != nil {
		return ConnectorProfileStatus{}, err
	}
	db, err := s.externalConnectorDB(profile, "sqlserver", sqlServerDSN(profile, timeoutSeconds))
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer conn.Close()
	timeoutMillis := timeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set transaction isolation level read committed"); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	if _, err := conn.ExecContext(ctx, "set lock_timeout "+strconv.Itoa(timeoutMillis)); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
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

func (s *Service) querySQLServerProfileContext(ctx context.Context, profile ConnectorProfile, request ConnectorQueryRequest, query string) (ConnectorQueryResult, error) {
	if err := requireSQLServerProfile(profile); err != nil {
		return ConnectorQueryResult{}, err
	}
	db, err := s.externalConnectorDB(profile, "sqlserver", sqlServerDSN(profile, request.TimeoutSeconds))
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer conn.Close()
	timeoutMillis := request.TimeoutSeconds * 1000
	if _, err := conn.ExecContext(ctx, "set transaction isolation level read committed"); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	if _, err := conn.ExecContext(ctx, "set lock_timeout "+strconv.Itoa(timeoutMillis)); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	return runConnectorQuery(ctx, conn, profile, request, query, "sqlserver-readonly", "SQL Server")
}

func testSQLiteProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorProfileStatus, error) {
	path, err := sqliteProfilePath(profile)
	if err != nil {
		return ConnectorProfileStatus{}, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "pragma query_only = on"); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	var version string
	if err := conn.QueryRowContext(ctx, "select sqlite_version()").Scan(&version); err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	return ConnectorProfileStatus{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Kind:      profile.Kind,
		Engine:    "sqlite-readonly",
		ReadOnly:  true,
		Message:   fmt.Sprintf("SQLite read-only connection succeeded for %s.", profile.Name),
	}, nil
}

func testDuckDBProfile(ctx context.Context, profile ConnectorProfile, timeoutSeconds int) (ConnectorProfileStatus, error) {
	path, err := duckDBProfilePath(profile)
	if err != nil {
		return ConnectorProfileStatus{}, err
	}
	if err := ensureDuckDBDriverEnabled(); err != nil {
		return ConnectorProfileStatus{}, err
	}
	db, err := sql.Open("duckdb", duckDBDSN(path))
	if err != nil {
		return ConnectorProfileStatus{}, connectorError(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(timeoutSeconds) * time.Second)
	conn, err := db.Conn(ctx)
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
		Engine:    "duckdb-readonly",
		ReadOnly:  true,
		Message:   fmt.Sprintf("DuckDB read-only connection succeeded for %s.", profile.Name),
	}, nil
}

func querySQLiteProfileContext(ctx context.Context, profile ConnectorProfile, request ConnectorQueryRequest, query string) (ConnectorQueryResult, error) {
	path, err := sqliteProfilePath(profile)
	if err != nil {
		return ConnectorQueryResult{}, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(request.TimeoutSeconds) * time.Second)
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "pragma query_only = on"); err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	return runConnectorQuery(ctx, conn, profile, request, query, "sqlite-readonly", "SQLite")
}

func queryDuckDBProfileContext(ctx context.Context, profile ConnectorProfile, request ConnectorQueryRequest, query string) (ConnectorQueryResult, error) {
	path, err := duckDBProfilePath(profile)
	if err != nil {
		return ConnectorQueryResult{}, err
	}
	if err := ensureDuckDBDriverEnabled(); err != nil {
		return ConnectorQueryResult{}, err
	}
	db, err := sql.Open("duckdb", duckDBDSN(path))
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Duration(request.TimeoutSeconds) * time.Second)
	conn, err := db.Conn(ctx)
	if err != nil {
		return ConnectorQueryResult{}, connectorError(err)
	}
	defer conn.Close()
	return runConnectorQuery(ctx, conn, profile, request, query, "duckdb-readonly", "DuckDB")
}

func runConnectorQuery(ctx context.Context, conn *sql.Conn, profile ConnectorProfile, request ConnectorQueryRequest, query string, engine string, displayName string) (ConnectorQueryResult, error) {
	started := time.Now()
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
		select {
		case <-ctx.Done():
			return ConnectorQueryResult{}, connectorError(ctx.Err())
		default:
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
	message := fmt.Sprintf("Read-only %s query returned %d rows from %s.", displayName, totalRows, profile.Name)
	if truncated {
		message = fmt.Sprintf("Read-only %s query reached the %d row cap for %s.", displayName, request.ResultLimit, profile.Name)
	}
	return ConnectorQueryResult{
		ProfileID:      profile.ID,
		Name:           profile.Name,
		Kind:           profile.Kind,
		Engine:         engine,
		SQL:            query,
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

func requirePostgresProfile(profile ConnectorProfile) error {
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

func postgresDSN(profile ConnectorProfile, timeoutSeconds int) string {
	port := profile.Port
	if port <= 0 {
		port = 5432
	}
	sslMode := postgresSSLMode(NormalizeConnectorSSLMode("postgres", profile.SSLMode))
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
	query.Set("application_name", "NexusDesk")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func postgresSSLMode(sslMode string) string {
	switch NormalizeConnectorSSLMode("postgres", sslMode) {
	case ConnectorSSLModeDevelopmentPlaintext:
		return "disable"
	case "verify-ca", "verify-full":
		return strings.ToLower(strings.TrimSpace(sslMode))
	default:
		return ConnectorSSLModeRequire
	}
}

func requireMySQLProfile(profile ConnectorProfile) error {
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

func mysqlDSN(profile ConnectorProfile, timeoutSeconds int) string {
	port := profile.Port
	if port <= 0 {
		port = 3306
	}
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
	config.Params = map[string]string{"charset": "utf8mb4"}
	if tlsMode := mysqlTLSMode(profile.SSLMode); tlsMode != "" {
		config.TLSConfig = tlsMode
	}
	return config.FormatDSN()
}

func mysqlTLSMode(sslMode string) string {
	switch NormalizeConnectorSSLMode("mysql", sslMode) {
	case ConnectorSSLModeDevelopmentPlaintext:
		return "false"
	case ConnectorSSLModeRequire:
		return "true"
	case ConnectorSSLModeSkipVerify:
		return "skip-verify"
	default:
		return "true"
	}
}

func mysqlEngine(profile ConnectorProfile) string {
	if strings.EqualFold(profile.Kind, "mariadb") {
		return "mariadb-readonly"
	}
	return "mysql-readonly"
}

func mysqlDisplayName(profile ConnectorProfile) string {
	if strings.EqualFold(profile.Kind, "mariadb") {
		return "MariaDB"
	}
	return "MySQL"
}

func requireSQLServerProfile(profile ConnectorProfile) error {
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

func sqlServerDSN(profile ConnectorProfile, timeoutSeconds int) string {
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
	query.Set("app name", "NexusDesk")
	query.Set("encrypt", sqlServerEncryptMode(profile.SSLMode))
	if NormalizeConnectorSSLMode("sqlserver", profile.SSLMode) == ConnectorSSLModeSkipVerify {
		query.Set("TrustServerCertificate", "true")
	}
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func sqlServerEncryptMode(sslMode string) string {
	switch NormalizeConnectorSSLMode("sqlserver", sslMode) {
	case ConnectorSSLModeDevelopmentPlaintext:
		return "disable"
	default:
		return "true"
	}
}

func sqliteProfilePath(profile ConnectorProfile) (string, error) {
	candidate := strings.TrimSpace(profile.Database)
	if candidate == "" {
		candidate = strings.TrimSpace(profile.Host)
	}
	if candidate == "" {
		return "", errors.New("SQLite profile needs a database file path in Host or Database")
	}
	path := filepath.Clean(candidate)
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".sqlite" && ext != ".sqlite3" && ext != ".db" {
		return "", errors.New("SQLite profile database file must end with .sqlite, .sqlite3, or .db")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("SQLite connector target is unavailable: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("SQLite connector target must be a file")
	}
	return path, nil
}

func duckDBProfilePath(profile ConnectorProfile) (string, error) {
	candidate := strings.TrimSpace(profile.Database)
	if candidate == "" {
		candidate = strings.TrimSpace(profile.Host)
	}
	if candidate == "" {
		return "", errors.New("DuckDB profile needs a database file path in Host or Database")
	}
	path := filepath.Clean(candidate)
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".duckdb" && ext != ".db" {
		return "", errors.New("DuckDB profile database file must end with .duckdb or .db")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("DuckDB connector target is unavailable: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("DuckDB connector target must be a file")
	}
	return path, nil
}

func duckDBDSN(path string) string {
	return filepath.ToSlash(path) + "?access_mode=read_only"
}

func ensureDuckDBDriverEnabled() error {
	if hasSQLDriver("duckdb") {
		return nil
	}
	return errors.New("DuckDB driver is not enabled in this build; add the DuckDB Go driver dependency before using DuckDB connector profiles")
}

func hasSQLDriver(name string) bool {
	name = strings.TrimSpace(name)
	for _, driver := range sql.Drivers() {
		if strings.EqualFold(strings.TrimSpace(driver), name) {
			return true
		}
	}
	return false
}
