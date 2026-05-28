package shell

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"

	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestFormatConnectorProfiles(t *testing.T) {
	text := formatConnectorProfiles([]dbconnectorSvc.ConnectorProfile{
		{
			Name:           "Warehouse",
			Kind:           "postgres",
			Host:           "db.internal",
			Port:           5432,
			Database:       "analytics",
			Username:       "analyst",
			SSLMode:        "require",
			ReadOnly:       true,
			WorkspaceScope: "c:/workspaces/app",
			ResultLimit:    1000,
			TimeoutSeconds: 30,
			UpdatedAt:      "2026-05-27T12:00:00Z",
		},
	})
	if !strings.Contains(text, "Warehouse [postgres]") {
		t.Fatalf("missing profile header in output: %q", text)
	}
	if !strings.Contains(text, "Cap: 1000 rows") {
		t.Fatalf("missing cap line in output: %q", text)
	}
	if !strings.Contains(text, "Scope: c:/workspaces/app") {
		t.Fatalf("missing scope line in output: %q", text)
	}
}

func TestFormatConnectorProfilesEmpty(t *testing.T) {
	text := formatConnectorProfiles(nil)
	if !strings.Contains(text, "No external connector profiles configured yet") {
		t.Fatalf("unexpected empty output: %q", text)
	}
}

func TestValueOrDefault(t *testing.T) {
	if got := valueOrDefault(0, 42); got != 42 {
		t.Fatalf("expected fallback 42, got %d", got)
	}
	if got := valueOrDefault(7, 42); got != 7 {
		t.Fatalf("expected explicit value 7, got %d", got)
	}
}

func TestConnectorQueryJobLabel(t *testing.T) {
	if got := connectorQueryJobLabel(dbconnectorSvc.ConnectorProfile{Name: "Warehouse"}); got != "Connector query (Warehouse)" {
		t.Fatalf("unexpected connector query label: %q", got)
	}
	if got := connectorQueryJobLabel(dbconnectorSvc.ConnectorProfile{ID: "profile-1"}); got != "Connector query (profile-1)" {
		t.Fatalf("unexpected fallback connector query label: %q", got)
	}
}

func TestConnectorProfileActionJobLabels(t *testing.T) {
	profile := dbconnectorSvc.ConnectorProfile{Name: "Warehouse"}
	if got := connectorProfileTestJobLabel(profile); got != "Connector test (Warehouse)" {
		t.Fatalf("unexpected connector profile test label: %q", got)
	}
	if got := connectorProfileInspectJobLabel(profile); got != "Connector inspect (Warehouse)" {
		t.Fatalf("unexpected connector profile inspect label: %q", got)
	}
}

func TestFormatConnectorMetadata(t *testing.T) {
	text := formatConnectorMetadata(dbconnectorSvc.ConnectorMetadata{
		Name:     "Warehouse",
		Kind:     "postgres",
		Engine:   "postgres-readonly",
		ReadOnly: true,
		Tables: []dbconnectorSvc.ConnectorTable{
			{Name: "public.orders", RowCount: 10, Columns: []dbconnectorSvc.ConnectorColumn{{Name: "id"}}, Indexes: []dbconnectorSvc.ConnectorIndex{{Name: "idx_orders_id"}}},
		},
		Relationships: []dbconnectorSvc.ConnectorRelationship{
			{FromTable: "public.orders", FromColumn: "customer_id", ToTable: "public.customers", ToColumn: "id", Kind: "foreign-key"},
		},
		Message: "ok",
	})
	if !strings.Contains(text, "External connector metadata") {
		t.Fatalf("missing header: %q", text)
	}
	if !strings.Contains(text, "public.orders") {
		t.Fatalf("missing table details: %q", text)
	}
	if !strings.Contains(text, "customer_id") {
		t.Fatalf("missing relationship details: %q", text)
	}
}

func TestConnectorSQLRunRecordCapturesFailure(t *testing.T) {
	started := time.Now().UTC().Add(-2 * time.Second)
	record := connectorSQLRunRecord(
		dbconnectorSvc.ConnectorQueryResult{},
		dbconnectorSvc.ConnectorProfile{ID: "warehouse", Kind: "postgres"},
		"select * from users",
		started,
		errors.New("blocked"),
	)
	if record.Status != "failed" {
		t.Fatalf("expected failed status, got %q", record.Status)
	}
	if record.Error != "blocked" {
		t.Fatalf("expected error text to be captured, got %q", record.Error)
	}
	if record.RelPath != "connector:warehouse" {
		t.Fatalf("expected connector rel path, got %q", record.RelPath)
	}
	if record.DurationMs <= 0 {
		t.Fatalf("expected positive duration, got %d", record.DurationMs)
	}
}

func TestConnectorSQLRunRecordCapturesCanceledStatus(t *testing.T) {
	started := time.Now().UTC().Add(-time.Second)
	record := connectorSQLRunRecord(
		dbconnectorSvc.ConnectorQueryResult{},
		dbconnectorSvc.ConnectorProfile{ID: "warehouse", Kind: "postgres"},
		"select * from users",
		started,
		context.Canceled,
	)
	if record.Status != "canceled" || record.Error == "" {
		t.Fatalf("expected canceled connector SQL record, got %#v", record)
	}
}

func TestConnectorDependencyRecordCarriesConnectorMetadata(t *testing.T) {
	record := connectorDependencyRecord(
		dbconnectorSvc.ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres"},
		metadataRunForTest("sql-run-1", "select 1", "postgres-readonly"),
	)
	if record.SourcePath != "connector:warehouse" {
		t.Fatalf("unexpected source path: %q", record.SourcePath)
	}
	if record.DependentKind != "connector-query" || record.Relation != "reads" {
		t.Fatalf("unexpected dependency kind/relation: %#v", record)
	}
	if record.Metadata["profile"] != "Warehouse" || record.Metadata["engine"] != "postgres-readonly" {
		t.Fatalf("unexpected dependency metadata: %#v", record.Metadata)
	}
}

func TestConnectorProfileTestDependencyRecordCapturesAuditMetadata(t *testing.T) {
	record := connectorProfileTestDependencyRecord(
		"job-0042",
		dbconnectorSvc.ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres"},
		dbconnectorSvc.ConnectorProfileStatus{
			Engine:  "postgres-readonly",
			Message: "PostgreSQL read-only connection succeeded for Warehouse.",
		},
		"success",
		nil,
	)
	if record.SourcePath != "connector:warehouse" {
		t.Fatalf("unexpected source path: %q", record.SourcePath)
	}
	if record.DependentKind != "connector-profile-test" || record.Relation != "checks" {
		t.Fatalf("unexpected dependency kind/relation: %#v", record)
	}
	if record.DependentRef != "job-0042" {
		t.Fatalf("expected job id dependent ref, got %q", record.DependentRef)
	}
	for key, expected := range map[string]string{
		"profile": "Warehouse",
		"kind":    "postgres",
		"engine":  "postgres-readonly",
		"status":  "success",
	} {
		if record.Metadata[key] != expected {
			t.Fatalf("unexpected metadata %s=%q (expected %q)", key, record.Metadata[key], expected)
		}
	}
}

func TestConnectorProfileInspectDependencyRecordCapturesCountsAndStatus(t *testing.T) {
	record := connectorProfileInspectDependencyRecord(
		"job-0100",
		dbconnectorSvc.ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres"},
		dbconnectorSvc.ConnectorMetadata{
			Engine:        "postgres-readonly",
			Tables:        []dbconnectorSvc.ConnectorTable{{Name: "public.orders"}},
			Views:         []dbconnectorSvc.ConnectorTable{{Name: "public.orders_view"}},
			Indexes:       []dbconnectorSvc.ConnectorIndex{{Name: "idx_orders_id"}},
			Relationships: []dbconnectorSvc.ConnectorRelationship{{FromTable: "public.orders", ToTable: "public.customers"}},
			Message:       "Metadata inspection succeeded.",
		},
		"failed",
		errors.New("network timeout"),
	)
	if record.DependentKind != "connector-profile-inspect" || record.Relation != "inspects" {
		t.Fatalf("unexpected dependency kind/relation: %#v", record)
	}
	for key, expected := range map[string]string{
		"status":        "failed",
		"tables":        "1",
		"views":         "1",
		"indexes":       "1",
		"relationships": "1",
	} {
		if record.Metadata[key] != expected {
			t.Fatalf("unexpected metadata %s=%q (expected %q)", key, record.Metadata[key], expected)
		}
	}
}

func metadataRunForTest(id string, sqlText string, engine string) metadataSvc.SQLRunRecord {
	now := time.Now().UTC()
	return metadataSvc.SQLRunRecord{
		ID:          id,
		SQL:         sqlText,
		Engine:      engine,
		StartedAt:   now,
		CompletedAt: now,
	}
}

func testMetadataStore(t *testing.T) *metadataSvc.Store {
	t.Helper()
	store, err := metadataSvc.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Ensure(); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	return store
}

func TestFinishConnectorProfileTestJobPersistsDependencyMetadata(t *testing.T) {
	store := testMetadataStore(t)
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("connector-test-audit")
	defer window.Close()
	view := New(window)
	view.metadataStore = store

	profile := dbconnectorSvc.ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres"}
	status := dbconnectorSvc.ConnectorProfileStatus{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Kind:      profile.Kind,
		Engine:    "postgres-readonly",
		ReadOnly:  true,
		Message:   "PostgreSQL read-only connection succeeded for Warehouse.",
	}

	view.finishConnectorProfileTestJob("job-2001", profile, status, nil)

	records, err := store.ListDatasetDependencies("connector:warehouse", 20)
	if err != nil {
		t.Fatalf("ListDatasetDependencies failed: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected connector profile test dependency record")
	}
	found := false
	for _, record := range records {
		if record.DependentKind != "connector-profile-test" {
			continue
		}
		found = true
		if record.Metadata["status"] != "success" {
			t.Fatalf("expected success status metadata, got %#v", record.Metadata)
		}
		if record.Metadata["engine"] != "postgres-readonly" {
			t.Fatalf("expected engine metadata, got %#v", record.Metadata)
		}
	}
	if !found {
		t.Fatalf("missing connector-profile-test dependency in %#v", records)
	}
}

func TestFinishConnectorProfileInspectJobPersistsCanceledDependencyMetadata(t *testing.T) {
	store := testMetadataStore(t)
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("connector-inspect-audit")
	defer window.Close()
	view := New(window)
	view.metadataStore = store

	profile := dbconnectorSvc.ConnectorProfile{ID: "warehouse", Name: "Warehouse", Kind: "postgres"}
	metadata := dbconnectorSvc.ConnectorMetadata{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Kind:      profile.Kind,
		Engine:    "postgres-readonly",
		ReadOnly:  true,
	}

	view.finishConnectorProfileInspectJob("job-2002", profile, metadata, context.Canceled)

	records, err := store.ListDatasetDependencies("connector:warehouse", 20)
	if err != nil {
		t.Fatalf("ListDatasetDependencies failed: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected connector profile inspect dependency record")
	}
	found := false
	for _, record := range records {
		if record.DependentKind != "connector-profile-inspect" {
			continue
		}
		found = true
		if record.Metadata["status"] != "canceled" {
			t.Fatalf("expected canceled status metadata, got %#v", record.Metadata)
		}
		if record.Relation != "inspects" {
			t.Fatalf("expected inspects relation, got %#v", record)
		}
	}
	if !found {
		t.Fatalf("missing connector-profile-inspect dependency in %#v", records)
	}
}
