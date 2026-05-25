package main

import (
	"strings"
	"testing"

	"NexusAugenticStudio/internal/appmeta"
)

func TestRebuildDatasetDependencyFilterExportRoundTrip(t *testing.T) {
	root := t.TempDir()
	if _, err := appmeta.Ensure(root); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	writeAppTestFile(t, root, "data/sales.csv", "channel,spend\nPaid,120\nOrganic,80\nPaid,20\n")

	app := NewApp()
	app.setWorkspaceRoot(root)

	report, err := app.CreateDatasetQueryArtifact("data/sales.csv", "channel=Paid")
	if err != nil {
		t.Fatalf("CreateDatasetQueryArtifact returned error: %v", err)
	}
	if report.RelPath == "" {
		t.Fatalf("expected exported artifact with relPath")
	}

	dependencies, err := app.ListDatasetDependencies("")
	if err != nil {
		t.Fatalf("ListDatasetDependencies returned error: %v", err)
	}
	if len(dependencies) == 0 {
		t.Fatalf("expected dataset dependency to be recorded")
	}
	dependency := dependencies[0]
	if dependency.Kind != "filter-export" {
		t.Fatalf("expected filter-export dependency, got %q", dependency.Kind)
	}
	if dependency.ID == "" {
		t.Fatalf("expected dependency ID")
	}

	rebuilt, err := app.RebuildDatasetDependency(dependency.ID)
	if err != nil {
		t.Fatalf("RebuildDatasetDependency returned error: %v", err)
	}
	if !strings.HasSuffix(rebuilt.RelPath, ".csv") {
		t.Fatalf("expected rebuilt artifact path with .csv suffix, got %q", rebuilt.RelPath)
	}

	refreshed, err := appmeta.GetDatasetDependency(root, dependency.ID)
	if err != nil {
		t.Fatalf("GetDatasetDependency returned error: %v", err)
	}
	if refreshed.LastRefresh == "" {
		t.Fatalf("expected last_refresh to be recorded")
	}
}

func TestRebuildDatasetDependencyRejectsMissingId(t *testing.T) {
	root := t.TempDir()
	if _, err := appmeta.Ensure(root); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	writeAppTestFile(t, root, "data/sales.csv", "channel,spend\nPaid,120\n")

	app := NewApp()
	app.setWorkspaceRoot(root)

	_, err := app.RebuildDatasetDependency("missing-id")
	if err == nil || !strings.Contains(err.Error(), "dataset dependency not found") {
		t.Fatalf("expected dependency-not-found error, got %v", err)
	}
}

func TestRebuildDatasetDependencyRejectsUnsupportedKind(t *testing.T) {
	root := t.TempDir()
	if _, err := appmeta.Ensure(root); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	writeAppTestFile(t, root, "data/sales.csv", "channel,spend\nPaid,120\n")

	app := NewApp()
	app.setWorkspaceRoot(root)
	if err := appmeta.RecordDatasetDependency(root, appmeta.DatasetDependency{
		ID:        "dep-unsupported",
		RelPath:   "data/sales.csv",
		Kind:      "sqlite-query",
		Query:     "select * from sales",
		CreatedAt: "2026-05-16T00:00:00Z",
	}); err != nil {
		t.Fatalf("RecordDatasetDependency returned error: %v", err)
	}

	_, err := app.RebuildDatasetDependency("dep-unsupported")
	if err == nil || !strings.Contains(err.Error(), "cannot rebuild dependency kind") {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}
