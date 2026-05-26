package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"NexusAugenticStudio/internal/analytics"
	"NexusAugenticStudio/internal/appmeta"
	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/dbconnector"
	"NexusAugenticStudio/internal/workspace"
)

type DatasetService struct {
	workspaceRoot           func() string
	mirrorMetadataStore     func(root string, create bool) (appmeta.SQLiteStatus, error)
	persistMetadata         func(root string, relPath string)
	recordDatasetDependency func(root string, relPath string, kind string, query string, target string, artifactRelPath string)
	recordSQLRun            func(root string, relPath string, sqlText string, engine string, rows int, artifactRelPath string, status string, message string)
	recordApproval          func(action string, target string, risk string, message string) string
	sqliteQueryCancels      map[string]context.CancelFunc
	sqliteQueryCancelsMu    sync.Mutex
}

func NewDatasetService(
	workspaceRoot func() string,
	mirrorMetadataStore func(string, bool) (appmeta.SQLiteStatus, error),
	persistMetadata func(string, string),
	recordDatasetDependency func(string, string, string, string, string, string),
	recordSQLRun func(string, string, string, string, int, string, string, string),
	recordApproval func(string, string, string, string) string,
) *DatasetService {
	return &DatasetService{
		workspaceRoot:           workspaceRoot,
		mirrorMetadataStore:     mirrorMetadataStore,
		persistMetadata:         persistMetadata,
		recordDatasetDependency: recordDatasetDependency,
		recordSQLRun:            recordSQLRun,
		recordApproval:          recordApproval,
		sqliteQueryCancels:      map[string]context.CancelFunc{},
	}
}

func (s *DatasetService) Profile(relPath string) (dataset.Profile, error) {
	root, err := s.requireRoot("profiling datasets")
	if err != nil {
		return dataset.Profile{}, err
	}
	return dataset.Build(root, relPath)
}

func (s *DatasetService) ListProfiles() ([]dataset.Profile, error) {
	root := s.workspaceRoot()
	if root == "" {
		return []dataset.Profile{}, nil
	}
	return dataset.List(root)
}

func (s *DatasetService) Query(relPath string, query string) (workspace.DatasetQueryResult, error) {
	root, err := s.requireRoot("querying datasets")
	if err != nil {
		return workspace.DatasetQueryResult{}, err
	}
	return workspace.QueryCSV(root, relPath, query)
}

func (s *DatasetService) QuerySQL(request analytics.SQLQueryRequest) (analytics.SQLQueryResult, error) {
	root, err := s.requireRoot("querying datasets")
	if err != nil {
		return analytics.SQLQueryResult{}, err
	}
	result, err := analytics.QueryCSVSQL(root, request)
	if err != nil {
		s.recordSQL(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "", 0, "", "failed", sanitizeProviderMessage(err.Error()))
		return analytics.SQLQueryResult{}, err
	}
	s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "completed", sanitizeProviderMessage(result.Message))
	s.recordDependency(root, result.RelPath, "sql-run", sanitizeQueryForMetadata(result.SQL), result.Engine, "")
	return result, nil
}

func (s *DatasetService) SaveQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	root, err := s.requireRoot("saving dataset queries")
	if err != nil {
		return dataset.SavedQuery{}, err
	}
	saved, err := dataset.SaveQuery(root, relPath, query, label)
	if err == nil {
		s.recordDependency(root, saved.RelPath, "filter-snippet", saved.Query, saved.Label, "")
	}
	return saved, err
}

func (s *DatasetService) ListQueries(relPath string) ([]dataset.SavedQuery, error) {
	root := s.workspaceRoot()
	if root == "" {
		return []dataset.SavedQuery{}, nil
	}
	return dataset.ListSavedQueries(root, relPath)
}

func (s *DatasetService) SaveSQLQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	root, err := s.requireRoot("saving SQL snippets")
	if err != nil {
		return dataset.SavedQuery{}, err
	}
	saved, err := dataset.SaveQueryKind(root, relPath, query, label, "sql")
	if err == nil {
		s.recordDependency(root, saved.RelPath, "sql-snippet", saved.Query, saved.Label, "")
	}
	return saved, err
}

func (s *DatasetService) SaveSQLiteConnectorQuery(relPath string, query string, label string) (dataset.SavedQuery, error) {
	root, err := s.requireRoot("saving SQLite connector queries")
	if err != nil {
		return dataset.SavedQuery{}, err
	}
	saved, err := dataset.SaveQueryKind(root, relPath, query, label, "sqlite-sql")
	if err == nil {
		s.recordDependency(root, saved.RelPath, "sqlite-query-snippet", saved.Query, saved.Label, "")
	}
	return saved, err
}

func (s *DatasetService) ListSQLQueries(relPath string) ([]dataset.SavedQuery, error) {
	root := s.workspaceRoot()
	if root == "" {
		return []dataset.SavedQuery{}, nil
	}
	return dataset.ListSavedQueriesKind(root, relPath, "sql")
}

func (s *DatasetService) ListSQLiteConnectorQueries(relPath string) ([]dataset.SavedQuery, error) {
	root := s.workspaceRoot()
	if root == "" {
		return []dataset.SavedQuery{}, nil
	}
	return dataset.ListSavedQueriesKind(root, relPath, "sqlite-sql")
}

func (s *DatasetService) ListDependencies(relPath string) ([]appmeta.DatasetDependency, error) {
	root := s.workspaceRoot()
	if root == "" || !appmeta.Exists(root) {
		return []appmeta.DatasetDependency{}, nil
	}
	return appmeta.ListDatasetDependencies(root, relPath)
}

func (s *DatasetService) ListSQLRuns(relPath string) ([]appmeta.SQLRun, error) {
	root := s.workspaceRoot()
	if root == "" || !appmeta.Exists(root) {
		return []appmeta.SQLRun{}, nil
	}
	return appmeta.ListSQLRuns(root, relPath)
}

func (s *DatasetService) SearchMetadata(query string) ([]appmeta.MetadataSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []appmeta.MetadataSearchResult{}, nil
	}
	root := s.workspaceRoot()
	if root == "" {
		return []appmeta.MetadataSearchResult{}, nil
	}
	if s.mirrorMetadataStore != nil {
		if _, err := s.mirrorMetadataStore(root, true); err != nil {
			return []appmeta.MetadataSearchResult{}, err
		}
	}
	return appmeta.Search(root, query, 40)
}

func (s *DatasetService) QueryWorkspaceSQLite(request dbconnector.SQLiteQueryRequest) (dbconnector.SQLiteQueryResult, error) {
	root, err := s.requireRoot("querying SQLite files")
	if err != nil {
		return dbconnector.SQLiteQueryResult{}, err
	}
	request = dbconnector.NormalizeSQLiteQueryRequest(request)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(request.TimeoutSeconds)*time.Second)
	if request.RequestID != "" {
		s.registerSQLiteQueryCancel(request.RequestID, cancel)
		defer s.unregisterSQLiteQueryCancel(request.RequestID)
	}
	defer cancel()
	result, err := dbconnector.QuerySQLiteContext(ctx, root, request)
	if err != nil {
		s.recordSQL(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "sqlite-readonly", 0, "", "failed", dbconnector.RedactConnectorError(err.Error()))
		return dbconnector.SQLiteQueryResult{}, err
	}
	s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "completed", dbconnector.RedactConnectorError(result.Message))
	s.recordDependency(root, result.RelPath, "sqlite-query", sanitizeQueryForMetadata(result.SQL), result.Engine, "")
	return result, nil
}

func (s *DatasetService) CancelWorkspaceSQLiteQuery(requestID string) bool {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return false
	}
	s.sqliteQueryCancelsMu.Lock()
	cancel := s.sqliteQueryCancels[requestID]
	s.sqliteQueryCancelsMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (s *DatasetService) registerSQLiteQueryCancel(requestID string, cancel context.CancelFunc) {
	s.sqliteQueryCancelsMu.Lock()
	defer s.sqliteQueryCancelsMu.Unlock()
	s.sqliteQueryCancels[requestID] = cancel
}

func (s *DatasetService) unregisterSQLiteQueryCancel(requestID string) {
	s.sqliteQueryCancelsMu.Lock()
	defer s.sqliteQueryCancelsMu.Unlock()
	delete(s.sqliteQueryCancels, requestID)
}

func (s *DatasetService) InspectWorkspaceSQLite(relPath string) (dbconnector.ConnectorMetadata, error) {
	root, err := s.requireRoot("inspecting SQLite metadata")
	if err != nil {
		return dbconnector.ConnectorMetadata{}, err
	}
	metadata, err := dbconnector.InspectSQLite(root, relPath)
	if err != nil {
		return dbconnector.ConnectorMetadata{}, err
	}
	s.recordDependency(root, metadata.RelPath, "sqlite-metadata", "inspect schema", metadata.Engine, "")
	return metadata, nil
}

func (s *DatasetService) PreviewChart(request workspace.DatasetChartRequest) (workspace.DatasetChartResult, error) {
	root, err := s.requireRoot("previewing dataset charts")
	if err != nil {
		return workspace.DatasetChartResult{}, err
	}
	return workspace.BuildCSVChart(root, request)
}

func (s *DatasetService) CreateChartArtifact(request workspace.DatasetChartRequest) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("creating dataset charts")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	chart, err := workspace.BuildCSVChart(root, request)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetChartSVG(root, chart, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.recordDependency(root, chart.RelPath, "chart", chart.CategoryColumn, chart.ValueColumn, report.RelPath)
	s.record("artifact.chart", report.RelPath, "low", report.Message)
	return report, nil
}

func (s *DatasetService) CreateQueryArtifact(relPath string, query string) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("exporting dataset queries")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	result, err := workspace.QueryCSV(root, relPath, query)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetQueryCSV(root, result, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.recordDependency(root, result.RelPath, "filter-export", result.Query, "", report.RelPath)
	s.record("artifact.query", report.RelPath, "low", report.Message)
	return report, nil
}

func (s *DatasetService) CreateSQLArtifact(request analytics.SQLQueryRequest) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("exporting SQL results")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	result, err := analytics.QueryCSVSQL(root, request)
	if err != nil {
		s.recordSQL(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "", 0, "", "failed", sanitizeProviderMessage(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetSQLMarkdown(root, result, time.Now())
	if err != nil {
		s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "failed", sanitizeProviderMessage(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, report.RelPath, "completed", sanitizeProviderMessage(result.Message))
	s.recordDependency(root, result.RelPath, "sql-report", sanitizeQueryForMetadata(result.SQL), result.Engine, report.RelPath)
	s.record("artifact.dataset_sql.create", report.RelPath, "medium", fmt.Sprintf("Created SQL result artifact from %s using %s.", result.RelPath, result.Engine))
	return report, nil
}

func (s *DatasetService) CreateSQLiteQueryCSVArtifact(request dbconnector.SQLiteQueryRequest) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("exporting SQLite query results")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	result, err := dbconnector.QuerySQLite(root, request)
	if err != nil {
		s.recordSQL(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "sqlite-readonly", 0, "", "failed", dbconnector.RedactConnectorError(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateSQLiteQueryCSV(root, result, time.Now())
	if err != nil {
		s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "failed", dbconnector.RedactConnectorError(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, report.RelPath, "completed", dbconnector.RedactConnectorError(result.Message))
	s.recordDependency(root, result.RelPath, "sqlite-query-csv", sanitizeQueryForMetadata(result.SQL), result.Engine, report.RelPath)
	s.record("artifact.sqlite_query_csv.create", report.RelPath, "low", fmt.Sprintf("Created SQLite query CSV artifact from %s.", result.RelPath))
	return report, nil
}

func (s *DatasetService) CreateSQLiteQueryMarkdownArtifact(request dbconnector.SQLiteQueryRequest) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("exporting SQLite query reports")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	result, err := dbconnector.QuerySQLite(root, request)
	if err != nil {
		s.recordSQL(root, request.RelPath, sanitizeQueryForMetadata(request.SQL), "sqlite-readonly", 0, "", "failed", dbconnector.RedactConnectorError(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateSQLiteQueryMarkdown(root, result, time.Now())
	if err != nil {
		s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "failed", dbconnector.RedactConnectorError(err.Error()))
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, report.RelPath, "completed", dbconnector.RedactConnectorError(result.Message))
	s.recordDependency(root, result.RelPath, "sqlite-query-report", sanitizeQueryForMetadata(result.SQL), result.Engine, report.RelPath)
	s.record("artifact.sqlite_query_report.create", report.RelPath, "low", fmt.Sprintf("Created SQLite query report artifact from %s.", result.RelPath))
	return report, nil
}

func (s *DatasetService) CreateSummaryArtifact(relPath string) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("creating dataset summaries")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: 1024 * 1024})
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateDatasetSummaryMarkdown(root, preview, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.recordDependency(root, relPath, "summary", "dataset summary", "", report.RelPath)
	s.record("artifact.dataset-summary", report.RelPath, "low", report.Message)
	return report, nil
}

func (s *DatasetService) RebuildDependency(id string) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("rebuilding dataset artifacts")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	dependency, err := appmeta.GetDatasetDependency(root, id)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	rebuildDependency := func() error {
		if strings.TrimSpace(dependency.Artifact) == "" {
			return nil
		}
		if _, err := artifact.Delete(root, dependency.Artifact); err != nil {
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		return nil
	}

	now := time.Now()
	switch dependency.Kind {
	case "filter-export":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		query := strings.TrimSpace(dependency.Query)
		if query == "" {
			return artifact.MarkdownReport{}, errors.New("cannot rebuild filter export without query text")
		}
		result, err := workspace.QueryCSV(root, dependency.RelPath, query)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetQueryCSV(root, result, now)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.persist(root, report.RelPath)
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.record("artifact.dataset_query.rebuild", report.RelPath, "low", fmt.Sprintf("Rebuilt dataset export for %s.", dependency.RelPath))
		return report, nil

	case "sql-report":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		query := strings.TrimSpace(dependency.Query)
		if query == "" {
			return artifact.MarkdownReport{}, errors.New("cannot rebuild SQL report without SQL text")
		}
		result, err := analytics.QueryCSVSQL(root, analytics.SQLQueryRequest{
			RelPath: dependency.RelPath,
			SQL:     query,
		})
		if err != nil {
			s.recordSQL(root, dependency.RelPath, sanitizeQueryForMetadata(query), "unknown", 0, "", "failed", sanitizeProviderMessage(err.Error()))
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetSQLMarkdown(root, result, now)
		if err != nil {
			s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, "", "failed", sanitizeProviderMessage(err.Error()))
			return artifact.MarkdownReport{}, err
		}
		s.persist(root, report.RelPath)
		s.recordSQL(root, result.RelPath, sanitizeQueryForMetadata(result.SQL), result.Engine, result.TotalRows, report.RelPath, "completed", sanitizeProviderMessage(result.Message))
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.record("artifact.dataset_sql.rebuild", report.RelPath, "medium", fmt.Sprintf("Rebuilt SQL result artifact for %s using %s.", result.RelPath, result.Engine))
		return report, nil

	case "chart":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		category := strings.TrimSpace(dependency.Target)
		if category == "" {
			return artifact.MarkdownReport{}, errors.New("cannot rebuild chart without a category column")
		}
		chart, err := workspace.BuildCSVChart(root, workspace.DatasetChartRequest{
			RelPath:        dependency.RelPath,
			ChartType:      "bar",
			CategoryColumn: category,
			ValueColumn:    strings.TrimSpace(dependency.Query),
		})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetChartSVG(root, chart, now)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.persist(root, report.RelPath)
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.record("artifact.chart.rebuild", report.RelPath, "low", fmt.Sprintf("Rebuilt chart for %s from category %s.", dependency.RelPath, dependency.Target))
		return report, nil

	case "summary":
		if err := rebuildDependency(); err != nil {
			return artifact.MarkdownReport{}, err
		}
		preview, err := workspace.Preview(root, dependency.RelPath, workspace.PreviewOptions{MaxBytes: 1024 * 1024})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		report, err := artifact.CreateDatasetSummaryMarkdown(root, preview, now)
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.persist(root, report.RelPath)
		if _, err := appmeta.UpdateDatasetDependencyRefresh(root, dependency.ID, report.RelPath); err != nil {
			return artifact.MarkdownReport{}, err
		}
		s.record("artifact.dataset-summary.rebuild", report.RelPath, "low", fmt.Sprintf("Rebuilt dataset summary for %s.", dependency.RelPath))
		return report, nil
	}

	return artifact.MarkdownReport{}, fmt.Errorf("cannot rebuild dependency kind %q", dependency.Kind)
}

func (s *DatasetService) requireRoot(action string) (string, error) {
	root := s.workspaceRoot()
	if root == "" {
		return "", errors.New("open a workspace before " + action)
	}
	return root, nil
}

func (s *DatasetService) persist(root string, relPath string) {
	if s.persistMetadata != nil {
		s.persistMetadata(root, relPath)
	}
}

func (s *DatasetService) recordDependency(root string, relPath string, kind string, query string, target string, artifactRelPath string) {
	if s.recordDatasetDependency != nil {
		s.recordDatasetDependency(root, relPath, kind, query, target, artifactRelPath)
	}
}

func (s *DatasetService) recordSQL(root string, relPath string, sqlText string, engine string, rows int, artifactRelPath string, status string, message string) {
	if s.recordSQLRun != nil {
		s.recordSQLRun(root, relPath, sqlText, engine, rows, artifactRelPath, status, message)
	}
}

func (s *DatasetService) record(action string, target string, risk string, message string) {
	if s.recordApproval != nil {
		s.recordApproval(action, target, risk, message)
	}
}
