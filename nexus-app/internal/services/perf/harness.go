// Package perf owns deterministic performance smoke profiles for release validation.
package perf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	datasetsSvc "nexusdesk/internal/services/datasets"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const (
	ScenarioShellRedraw    = "shell-redraw-model"
	ScenarioActivityLog    = "activity-log-model"
	ScenarioDataGrid       = "data-grid-model"
	ScenarioLargeSearch    = "large-search"
	ScenarioLargeArtifacts = "large-artifacts"
)

type Options struct {
	WorkspaceFiles int
	ActivityLines  int
	DataRows       int
	ArtifactCount  int
	SearchResults  int
	KeepFixture    bool
}

type Report struct {
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	FixtureRoot string
	Scenarios   []ScenarioResult
	Passed      bool
}

type ScenarioResult struct {
	Name         string
	Items        int
	Duration     time.Duration
	Budget       time.Duration
	WithinBudget bool
	Detail       string
}

func QuickOptions() Options {
	return Options{
		WorkspaceFiles: 120,
		ActivityLines:  500,
		DataRows:       750,
		ArtifactCount:  80,
		SearchResults:  80,
	}
}

func RunQuickProfile(ctx context.Context, scratchParent string, options Options) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	scratchParent = strings.TrimSpace(scratchParent)
	if scratchParent == "" {
		return Report{}, errors.New("scratch parent is required")
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	options = normalizeOptions(options)
	started := time.Now().UTC()
	fixtureRoot, err := os.MkdirTemp(scratchParent, "nexusdesk-perf-*")
	if err != nil {
		return Report{}, err
	}
	if !options.KeepFixture {
		defer os.RemoveAll(fixtureRoot)
	}

	report := Report{
		StartedAt:   started,
		FixtureRoot: fixtureRoot,
		Passed:      true,
	}
	runners := []func(context.Context, string, Options) (ScenarioResult, error){
		runShellRedrawModel,
		runActivityLogModel,
		runDataGridModel,
		runLargeSearch,
		runLargeArtifacts,
	}
	for _, runner := range runners {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		result, err := runner(ctx, fixtureRoot, options)
		if err != nil {
			return Report{}, err
		}
		report.Scenarios = append(report.Scenarios, result)
		if !result.WithinBudget {
			report.Passed = false
		}
	}
	report.CompletedAt = time.Now().UTC()
	report.Duration = report.CompletedAt.Sub(report.StartedAt)
	if !options.KeepFixture {
		report.FixtureRoot = ""
	}
	return report, nil
}

func normalizeOptions(options Options) Options {
	defaults := QuickOptions()
	if options.WorkspaceFiles <= 0 {
		options.WorkspaceFiles = defaults.WorkspaceFiles
	}
	if options.ActivityLines <= 0 {
		options.ActivityLines = defaults.ActivityLines
	}
	if options.DataRows <= 0 {
		options.DataRows = defaults.DataRows
	}
	if options.ArtifactCount <= 0 {
		options.ArtifactCount = defaults.ArtifactCount
	}
	if options.SearchResults <= 0 {
		options.SearchResults = defaults.SearchResults
	}
	return options
}

func runShellRedrawModel(ctx context.Context, root string, options Options) (ScenarioResult, error) {
	started := time.Now()
	labels := make([]string, 0, options.WorkspaceFiles)
	for index := 0; index < options.WorkspaceFiles; index++ {
		if index%128 == 0 {
			if err := ctx.Err(); err != nil {
				return ScenarioResult{}, err
			}
		}
		labels = append(labels, fmt.Sprintf("tab=%03d status=ready branch=main jobs=%d diagnostics=ok", index, index%7))
	}
	var total int
	for _, label := range labels {
		total += len(label)
	}
	return scenario(ScenarioShellRedraw, options.WorkspaceFiles, started, 150*time.Millisecond, fmt.Sprintf("materialized %d shell labels (%d bytes)", len(labels), total)), nil
}

func runActivityLogModel(ctx context.Context, root string, options Options) (ScenarioResult, error) {
	started := time.Now()
	const capLines = 400
	lines := make([]string, 0, capLines)
	for index := 0; index < options.ActivityLines; index++ {
		if index%128 == 0 {
			if err := ctx.Err(); err != nil {
				return ScenarioResult{}, err
			}
		}
		lines = append(lines, fmt.Sprintf("activity %04d workspace event with bounded redraw text", index))
		if len(lines) > capLines {
			copy(lines, lines[len(lines)-capLines:])
			lines = lines[:capLines]
		}
	}
	return scenario(ScenarioActivityLog, len(lines), started, 150*time.Millisecond, fmt.Sprintf("retained %d of %d activity lines", len(lines), options.ActivityLines)), nil
}

func runDataGridModel(ctx context.Context, root string, options Options) (ScenarioResult, error) {
	started := time.Now()
	csvPath := filepath.Join(root, "data", "large-grid.csv")
	if err := os.MkdirAll(filepath.Dir(csvPath), 0o755); err != nil {
		return ScenarioResult{}, err
	}
	var builder strings.Builder
	builder.WriteString("id,channel,spend,status\n")
	for index := 0; index < options.DataRows; index++ {
		if index%256 == 0 {
			if err := ctx.Err(); err != nil {
				return ScenarioResult{}, err
			}
		}
		builder.WriteString(fmt.Sprintf("%d,channel-%02d,%d,%s\n", index+1, index%12, 100+index%900, gridStatus(index)))
	}
	if err := os.WriteFile(csvPath, []byte(builder.String()), 0o644); err != nil {
		return ScenarioResult{}, err
	}
	result, err := datasetsSvc.New(nil).QueryContext(ctx, root, "data/large-grid.csv", "limit 50")
	if err != nil {
		return ScenarioResult{}, err
	}
	return scenario(ScenarioDataGrid, result.TotalRows, started, 300*time.Millisecond, fmt.Sprintf("queried %d row(s), rendered %d capped row(s)", result.TotalRows, len(result.Rows))), nil
}

func runLargeSearch(ctx context.Context, root string, options Options) (ScenarioResult, error) {
	started := time.Now()
	for index := 0; index < options.WorkspaceFiles; index++ {
		if index%128 == 0 {
			if err := ctx.Err(); err != nil {
				return ScenarioResult{}, err
			}
		}
		rel := filepath.Join("src", fmt.Sprintf("module-%03d.txt", index))
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return ScenarioResult{}, err
		}
		content := fmt.Sprintf("module %03d\nneedle performance fixture line %03d\n", index, index)
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			return ScenarioResult{}, err
		}
	}
	results, err := workspaceSvc.New().Search(root, "needle", workspaceSvc.SearchOptions{MaxResults: options.SearchResults})
	if err != nil {
		return ScenarioResult{}, err
	}
	return scenario(ScenarioLargeSearch, len(results), started, 600*time.Millisecond, fmt.Sprintf("searched %d file(s), returned %d result(s)", options.WorkspaceFiles, len(results))), nil
}

func runLargeArtifacts(ctx context.Context, root string, options Options) (ScenarioResult, error) {
	started := time.Now()
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return ScenarioResult{}, err
	}
	for index := 0; index < options.ArtifactCount; index++ {
		if index%64 == 0 {
			if err := ctx.Err(); err != nil {
				return ScenarioResult{}, err
			}
		}
		_, err := store.WriteTaskRunReport(artifactsSvc.TaskRunReport{
			ID:        fmt.Sprintf("perf-task-%03d", index),
			Label:     fmt.Sprintf("Perf task %03d", index),
			Command:   "go test ./...",
			Cwd:       ".",
			Status:    "success",
			Stdout:    "ok",
			StartedAt: time.Now().UTC(),
		})
		if err != nil {
			return ScenarioResult{}, err
		}
	}
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{})
	if err != nil {
		return ScenarioResult{}, err
	}
	return scenario(ScenarioLargeArtifacts, len(artifacts), started, 900*time.Millisecond, fmt.Sprintf("listed %d generated artifact(s)", len(artifacts))), nil
}

func scenario(name string, items int, started time.Time, budget time.Duration, detail string) ScenarioResult {
	duration := time.Since(started)
	return ScenarioResult{
		Name:         name,
		Items:        items,
		Duration:     duration,
		Budget:       budget,
		WithinBudget: duration <= budget,
		Detail:       detail,
	}
}

func gridStatus(index int) string {
	if index%5 == 0 {
		return "review"
	}
	return "ok"
}
