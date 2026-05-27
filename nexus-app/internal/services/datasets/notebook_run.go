package datasets

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Service) RunNotebook(root string, notebook Notebook) (NotebookRunResult, error) {
	if strings.TrimSpace(notebook.RelPath) == "" {
		return NotebookRunResult{}, errors.New("notebook source dataset is required")
	}
	started := time.Now().UTC()
	result := NotebookRunResult{
		RelPath:    notebook.RelPath,
		NotebookID: notebook.ID,
		Label:      notebook.Label,
		Cells:      []NotebookCellRun{},
		StartedAt:  started,
	}
	executed := 0
	failed := 0
	for _, cell := range notebook.Cells {
		if cell.Kind != "sql" && cell.Kind != "chart" {
			continue
		}
		run := s.runNotebookCell(root, notebook.RelPath, cell)
		if run.SQL != "" || run.Kind == "chart" {
			executed++
		}
		if run.Error != "" {
			failed++
		}
		result.Cells = append(result.Cells, run)
	}
	completed := time.Now().UTC()
	result.CompletedAt = completed
	result.DurationMs = completed.Sub(started).Milliseconds()
	result.Message = fmt.Sprintf("Ran %d notebook cell(s) from %s.", executed, notebook.Label)
	if failed > 0 {
		result.Message = fmt.Sprintf("Ran %d notebook cell(s) from %s with %d failure(s).", executed, notebook.Label, failed)
	}
	return result, nil
}

func (s *Service) runNotebookCell(root string, relPath string, cell NotebookCell) NotebookCellRun {
	started := time.Now().UTC()
	run := NotebookCellRun{
		CellID:    cell.ID,
		Label:     cell.Label,
		Kind:      cell.Kind,
		SQL:       strings.TrimSpace(cell.SQL),
		StartedAt: started,
	}
	if run.SQL == "" {
		run.Error = "notebook cell has no SQL text"
		run.CompletedAt = time.Now().UTC()
		run.DurationMs = run.CompletedAt.Sub(started).Milliseconds()
		return run
	}
	sqlResult, err := s.QuerySQL(root, relPath, run.SQL)
	if err != nil {
		run.Error = err.Error()
		run.CompletedAt = time.Now().UTC()
		run.DurationMs = run.CompletedAt.Sub(started).Milliseconds()
		return run
	}
	run.SQLResult = sqlResult
	if cell.Kind == "chart" {
		chart, err := BuildChart(sqlResult.QueryResult)
		if err != nil {
			run.Error = err.Error()
		} else {
			run.ChartResult = chart
		}
	}
	run.CompletedAt = time.Now().UTC()
	run.DurationMs = run.CompletedAt.Sub(started).Milliseconds()
	return run
}
