package perf

import (
	"context"
	"testing"
	"time"
)

func TestRunQuickProfileCoversProductionScenarios(t *testing.T) {
	report, err := RunQuickProfile(context.Background(), t.TempDir(), Options{
		WorkspaceFiles: 12,
		ActivityLines:  30,
		DataRows:       40,
		ArtifactCount:  8,
		SearchResults:  10,
	})
	if err != nil {
		t.Fatalf("RunQuickProfile returned error: %v", err)
	}
	if len(report.Scenarios) != 5 {
		t.Fatalf("expected five scenarios, got %#v", report.Scenarios)
	}
	if report.FixtureRoot != "" {
		t.Fatalf("expected fixture cleanup by default, got %q", report.FixtureRoot)
	}
	if report.Duration < 0 || report.CompletedAt.Before(report.StartedAt) {
		t.Fatalf("unexpected report timing: %#v", report)
	}
	seen := map[string]ScenarioResult{}
	for _, result := range report.Scenarios {
		seen[result.Name] = result
		if result.Items <= 0 {
			t.Fatalf("expected scenario %s to process items: %#v", result.Name, result)
		}
		if result.Budget <= 0 || result.Duration < 0 || result.Detail == "" {
			t.Fatalf("expected timing/budget/detail for %s: %#v", result.Name, result)
		}
	}
	for _, expected := range []string{
		ScenarioShellRedraw,
		ScenarioActivityLog,
		ScenarioDataGrid,
		ScenarioLargeSearch,
		ScenarioLargeArtifacts,
	} {
		if _, ok := seen[expected]; !ok {
			t.Fatalf("missing scenario %s in %#v", expected, report.Scenarios)
		}
	}
	if seen[ScenarioLargeSearch].Items > 10 {
		t.Fatalf("search scenario ignored result cap: %#v", seen[ScenarioLargeSearch])
	}
}

func TestRunQuickProfileKeepsFixtureWhenRequested(t *testing.T) {
	report, err := RunQuickProfile(context.Background(), t.TempDir(), Options{
		WorkspaceFiles: 2,
		ActivityLines:  2,
		DataRows:       2,
		ArtifactCount:  1,
		SearchResults:  2,
		KeepFixture:    true,
	})
	if err != nil {
		t.Fatalf("RunQuickProfile returned error: %v", err)
	}
	if report.FixtureRoot == "" {
		t.Fatalf("expected fixture path when KeepFixture is true")
	}
}

func TestRunQuickProfileHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunQuickProfile(ctx, t.TempDir(), Options{}); err == nil {
		t.Fatal("expected canceled profile to return an error")
	}
}

func TestRunQuickProfileRequiresScratchParent(t *testing.T) {
	if _, err := RunQuickProfile(context.Background(), "", Options{}); err == nil {
		t.Fatal("expected scratch parent requirement")
	}
}

func TestScenarioBudgetFlag(t *testing.T) {
	result := scenario("slow", 1, time.Now().Add(-time.Second), time.Millisecond, "detail")
	if result.WithinBudget {
		t.Fatalf("expected over-budget scenario: %#v", result)
	}
}
