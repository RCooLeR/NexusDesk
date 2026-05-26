package main

import (
	"fmt"
	"strings"

	"NexusAugenticStudio/internal/analytics"
	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/dbconnector"
	"NexusAugenticStudio/internal/webfetch"
	"NexusAugenticStudio/internal/workspace"
)

func unifiedPatchProposalSummary(proposal workspace.UnifiedPatchProposal, includeDiff bool) string {
	lines := []string{proposal.Message}
	for _, file := range proposal.Files {
		lines = append(lines, fmt.Sprintf("- %s %s", file.Action, file.RelPath))
		if includeDiff && strings.TrimSpace(file.Diff) != "" {
			lines = append(lines, file.Diff)
		}
	}
	return strings.Join(lines, "\n")
}

func unifiedPatchRollbackPaths(proposal workspace.UnifiedPatchProposal) []string {
	paths := make([]string, 0, len(proposal.Files))
	for _, file := range proposal.Files {
		if strings.TrimSpace(file.RelPath) != "" {
			paths = append(paths, file.RelPath)
		}
	}
	return paths
}

func formatRollbackListObservation(items []workspace.RollbackRecord) string {
	if len(items) == 0 {
		return "No rollback snapshots are available."
	}
	lines := []string{fmt.Sprintf("Rollback snapshots: %d", len(items))}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s [%s] %s target=%s paths=%d created=%s", item.ID, item.Status, item.Action, item.Target, len(item.Entries), item.CreatedAt))
		if item.Message != "" {
			lines = append(lines, "  "+item.Message)
		}
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatRollbackApplyObservation(result workspace.RollbackApplyResult) string {
	lines := []string{result.Message}
	if len(result.Restored) > 0 {
		lines = append(lines, "Restored: "+strings.Join(result.Restored, ", "))
	}
	if len(result.Removed) > 0 {
		lines = append(lines, "Removed: "+strings.Join(result.Removed, ", "))
	}
	return strings.Join(lines, "\n")
}

func formatGitStatusObservation(status GitStatus) string {
	if !status.Available {
		return status.Message
	}
	lines := []string{
		fmt.Sprintf("Git: %s @ %s", status.Branch, status.Head),
		status.Message,
		fmt.Sprintf("Changed files: %d (%d staged, %d unstaged)", len(status.ChangedFiles), len(status.StagedFiles), len(status.UnstagedFiles)),
	}
	for _, change := range status.ChangedFiles {
		lines = append(lines, fmt.Sprintf("- %s %s [%s%s]", change.Summary, change.Path, change.Index, change.Worktree))
	}
	if strings.TrimSpace(status.StagedDiff) != "" {
		lines = append(lines, "\nStaged diff:\n"+status.StagedDiff)
	}
	if strings.TrimSpace(status.UnstagedDiff) != "" {
		lines = append(lines, "\nUnstaged diff:\n"+status.UnstagedDiff)
	}
	if status.StagedDiffTruncated || status.UnstagedDiffTruncated {
		lines = append(lines, "\nDiff output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func formatGitFileDiffObservation(diff GitFileDiff) string {
	lines := []string{diff.Message}
	if strings.TrimSpace(diff.StagedDiff) != "" {
		lines = append(lines, "\nStaged diff:\n"+diff.StagedDiff)
	}
	if strings.TrimSpace(diff.UnstagedDiff) != "" {
		lines = append(lines, "\nUnstaged diff:\n"+diff.UnstagedDiff)
	}
	if diff.StagedDiffTruncated || diff.UnstagedDiffTruncated {
		lines = append(lines, "\nDiff output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func formatGitHistoryObservation(result GitHistoryResult) string {
	if !result.Available {
		return result.Message
	}
	label := "repository"
	if result.Path != "" {
		label = result.Path
	}
	lines := []string{
		result.Message,
		fmt.Sprintf("History target: %s limit=%d", label, result.Limit),
	}
	for _, entry := range result.Entries {
		lines = append(lines, fmt.Sprintf("- %s %s %s <%s> %s", entry.ShortHash, entry.Date, entry.Author, entry.Email, entry.Subject))
	}
	if result.Truncated {
		lines = append(lines, "History output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func formatGitBlameObservation(result GitBlameResult) string {
	if !result.Available {
		return result.Message
	}
	lines := []string{result.Message}
	if result.StartLine > 0 {
		lines = append(lines, fmt.Sprintf("Requested lines: %d-%d", result.StartLine, result.EndLine))
	}
	for _, line := range result.Lines {
		lines = append(lines, fmt.Sprintf("%d %s %s %s | %s", line.Line, line.ShortHash, line.Author, line.Date, line.Content))
		if line.Summary != "" {
			lines = append(lines, "  summary: "+line.Summary)
		}
	}
	if result.Truncated {
		lines = append(lines, "Blame output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func formatProblemSummaryObservation(summary workspace.ProblemSummary) string {
	lines := []string{summary.Message}
	for _, problem := range summary.Problems {
		lines = append(lines, fmt.Sprintf("- [%s/%s] %s:%d %s", problem.Severity, problem.Source, problem.RelPath, problem.Line, problem.Message))
	}
	if summary.Truncated {
		lines = append(lines, "Problem output was truncated.")
	}
	return strings.Join(lines, "\n")
}

func formatTaskSummaryObservation(summary WorkspaceTaskSummary) string {
	lines := []string{summary.Message}
	for _, task := range summary.Tasks {
		lines = append(lines, fmt.Sprintf("- %s | %s | cwd=%s | id=%s | source=%s", task.Kind, task.Command, task.Cwd, task.ID, task.Source))
	}
	return strings.Join(lines, "\n")
}

func formatTaskRunObservation(result WorkspaceTaskRunResult) string {
	lines := []string{
		result.Message,
		fmt.Sprintf("Task: %s", result.Task.Label),
		fmt.Sprintf("Status: %s exit=%d duration=%dms", result.Status, result.ExitCode, result.DurationMs),
		fmt.Sprintf("Artifact: %s", result.ArtifactRelPath),
	}
	if strings.TrimSpace(result.Stdout) != "" {
		lines = append(lines, "\nStdout:\n"+result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		lines = append(lines, "\nStderr:\n"+result.Stderr)
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatArtifactListObservation(items []artifact.WorkspaceArtifact) string {
	if len(items) == 0 {
		return "No artifacts found."
	}
	lines := []string{fmt.Sprintf("%d artifact(s) found.", len(items))}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s | %s", item.Kind, item.RelPath, item.Summary, item.Source, item.Model))
	}
	return strings.Join(lines, "\n")
}

func formatArtifactObservation(relPath string, metadata artifact.ArtifactMetadata, preview workspace.FilePreview) string {
	lines := []string{
		"Artifact: " + relPath,
		"Kind: " + metadata.Kind,
		"Title: " + metadata.Title,
		"Source: " + metadata.Source,
		"Source paths: " + strings.Join(metadata.SourcePaths, ", "),
		"Context: " + metadata.ContextRelPath,
	}
	content := firstNonEmpty(preview.Text, preview.Content)
	if strings.TrimSpace(content) != "" {
		lines = append(lines, "\nContent:\n"+content)
	} else {
		lines = append(lines, "Content is not previewable as text.")
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatArtifactLineageObservation(lineage ArtifactLineage) string {
	lines := []string{
		lineage.Message,
		fmt.Sprintf("Nodes: %d", len(lineage.Nodes)),
		fmt.Sprintf("Relationships: %d", len(lineage.Edges)),
	}
	if len(lineage.RelationshipCounts) > 0 {
		lines = append(lines, "\nRelationship counts:")
		for label, count := range lineage.RelationshipCounts {
			lines = append(lines, fmt.Sprintf("- %s: %d", label, count))
		}
	}
	if len(lineage.Nodes) > 0 {
		lines = append(lines, "\nNodes:")
		for index, node := range lineage.Nodes {
			if index >= 80 {
				lines = append(lines, fmt.Sprintf("Skipped %d additional node(s).", len(lineage.Nodes)-index))
				break
			}
			lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s", node.Kind, node.ID, node.Label, node.RelPath))
		}
	}
	if len(lineage.Edges) > 0 {
		lines = append(lines, "\nEdges:")
		for index, edge := range lineage.Edges {
			if index >= 120 {
				lines = append(lines, fmt.Sprintf("Skipped %d additional edge(s).", len(lineage.Edges)-index))
				break
			}
			lines = append(lines, fmt.Sprintf("- %s --%s--> %s", edge.From, edge.Label, edge.To))
		}
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatWebFetchObservation(result webfetch.Result) string {
	lines := []string{
		result.Message,
		"URL: " + result.URL,
		"Final URL: " + result.FinalURL,
		fmt.Sprintf("Status: %d", result.Status),
		"Content-Type: " + result.ContentType,
		fmt.Sprintf("Redirects: %d", result.Redirects),
		fmt.Sprintf("Truncated: %t", result.Truncated),
	}
	if result.Title != "" {
		lines = append(lines, "Title: "+result.Title)
	}
	if strings.TrimSpace(result.Text) != "" {
		lines = append(lines, "\nContent:\n"+result.Text)
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatDatasetProfilesObservation(profiles []dataset.Profile) string {
	if len(profiles) == 0 {
		return "No persisted dataset profiles found. Call profile_dataset with a CSV, TSV, JSON, NDJSON, XLSX, Parquet, or log path to create one."
	}
	lines := []string{fmt.Sprintf("%d persisted dataset profile(s).", len(profiles))}
	for _, profile := range profiles {
		lines = append(lines, fmt.Sprintf("- %s | %s | rows=%d columns=%d updated=%s", profile.RelPath, profile.Kind, profile.Rows, profile.Columns, profile.UpdatedAt))
	}
	return strings.Join(lines, "\n")
}

func formatDatasetProfileObservation(profile dataset.Profile) string {
	lines := []string{
		"Dataset: " + profile.RelPath,
		"Kind: " + profile.Kind,
		fmt.Sprintf("Rows: %d", profile.Rows),
		fmt.Sprintf("Columns: %d", profile.Columns),
		"Updated: " + profile.UpdatedAt,
		"Message: " + profile.Message,
	}
	if len(profile.Profiles) > 0 {
		lines = append(lines, "\nColumn profiles:")
		for _, column := range profile.Profiles {
			bounds := strings.TrimSpace(strings.Join([]string{column.Min, column.Max}, " .. "))
			if bounds != ".." && bounds != "" {
				bounds = " | range=" + bounds
			} else {
				bounds = ""
			}
			lines = append(lines, fmt.Sprintf("- %s | type=%s | missing=%d | distinct=%d%s", column.Name, column.Type, column.Missing, column.Distinct, bounds))
		}
	}
	if len(profile.Workbook.Sheets) > 0 {
		lines = append(lines, "\nWorkbook sheets:")
		for _, sheet := range profile.Workbook.Sheets {
			lines = append(lines, fmt.Sprintf("- %s | %s | rows=%d columns=%d formulas=%d tables=%d", sheet.Name, sheet.Dimension, sheet.Rows, sheet.Columns, sheet.FormulaCount, sheet.TableCount))
		}
	}
	if profile.Parquet.Message != "" {
		lines = append(lines, fmt.Sprintf("\nParquet: size=%d metadata=%d data=%d message=%s", profile.Parquet.FileSize, profile.Parquet.FooterMetadataBytes, profile.Parquet.DataBytes, profile.Parquet.Message))
	}
	if profile.Log.Message != "" {
		lines = append(lines, fmt.Sprintf("\nLog: sampled=%d total=%d truncated=%t levels=%v stack-trace-lines=%d", profile.Log.SampledLines, profile.Log.TotalLines, profile.Log.Truncated, profile.Log.LevelCounts, profile.Log.StackTraceLines))
		for _, pattern := range profile.Log.TopPatterns {
			lines = append(lines, fmt.Sprintf("- pattern x%d: %s", pattern.Count, pattern.Pattern))
		}
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatDatasetQueryObservation(result workspace.DatasetQueryResult) string {
	lines := []string{
		result.Message,
		"Dataset: " + result.RelPath,
		"Query: " + fallbackInput(result.Query, "first rows"),
		fmt.Sprintf("Rows: %d matched of %d", result.MatchedRows, result.TotalRows),
	}
	lines = append(lines, formatRowsAsMarkdown(result.Columns, result.Rows)...)
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatSQLQueryObservation(result analytics.SQLQueryResult) string {
	lines := []string{
		result.Message,
		"Dataset: " + result.RelPath,
		"Engine: " + result.Engine,
		"SQL: " + result.SQL,
		fmt.Sprintf("Rows: %d matched of %d", result.MatchedRows, result.TotalRows),
	}
	if len(result.Plan) > 0 {
		lines = append(lines, "\nPlan:")
		for _, step := range result.Plan {
			lines = append(lines, "- "+step)
		}
	}
	lines = append(lines, formatRowsAsMarkdown(result.Columns, result.Rows)...)
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatSQLiteMetadataObservation(metadata dbconnector.ConnectorMetadata) string {
	lines := []string{
		metadata.Message,
		"Database: " + metadata.RelPath,
		"Engine: " + metadata.Engine,
		fmt.Sprintf("Read-only: %t", metadata.ReadOnly),
	}
	if len(metadata.Tables) > 0 {
		lines = append(lines, "\nTables:")
		for _, table := range metadata.Tables {
			lines = append(lines, formatSQLiteTableLine(table))
		}
	}
	if len(metadata.Views) > 0 {
		lines = append(lines, "\nViews:")
		for _, view := range metadata.Views {
			lines = append(lines, formatSQLiteTableLine(view))
		}
	}
	if len(metadata.Relationships) > 0 {
		lines = append(lines, "\nRelationships:")
		for _, rel := range metadata.Relationships {
			lines = append(lines, fmt.Sprintf("- %s.%s -> %s.%s (%s, %s)", rel.FromTable, rel.FromColumn, rel.ToTable, rel.ToColumn, rel.Kind, rel.Confidence))
		}
	}
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatSQLiteTableLine(table dbconnector.ConnectorTable) string {
	columnParts := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		traits := []string{column.Type}
		if column.PrimaryKey {
			traits = append(traits, "pk")
		}
		if !column.Nullable {
			traits = append(traits, "not-null")
		}
		columnParts = append(columnParts, fmt.Sprintf("%s(%s)", column.Name, strings.Join(traits, ",")))
	}
	return fmt.Sprintf("- %s | rows=%d | columns=%s", table.Name, table.RowCount, strings.Join(columnParts, ", "))
}

func formatSQLiteQueryObservation(result dbconnector.SQLiteQueryResult) string {
	lines := []string{
		result.Message,
		"Database: " + result.RelPath,
		"Engine: " + result.Engine,
		"SQL: " + result.SQL,
		fmt.Sprintf("Rows: %d limit=%d truncated=%t timeout=%ds", result.TotalRows, result.ResultLimit, result.Truncated, result.TimeoutSeconds),
	}
	lines = append(lines, formatRowsAsMarkdown(result.Columns, result.Rows)...)
	return limitAgentOutput(strings.Join(lines, "\n"), maxAgentContextOutputSize)
}

func formatRowsAsMarkdown(columns []string, rows [][]string) []string {
	if len(columns) == 0 {
		return []string{"\nNo columns returned."}
	}
	lines := []string{
		"\nRows:",
		"| " + strings.Join(columns, " | ") + " |",
		"| " + strings.Join(repeatString("---", len(columns)), " | ") + " |",
	}
	for _, row := range rows {
		cells := make([]string, len(columns))
		for index := range columns {
			if index < len(row) {
				cells[index] = sanitizeMarkdownCell(row[index])
			}
		}
		lines = append(lines, "| "+strings.Join(cells, " | ")+" |")
	}
	if len(rows) == 0 {
		lines = append(lines, "| "+strings.Join(repeatString("", len(columns)), " | ")+" |")
	}
	return lines
}

func sanitizeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.TrimSpace(value)
	if len(value) > 160 {
		return value[:160] + "..."
	}
	return value
}

func repeatString(value string, count int) []string {
	items := make([]string, count)
	for index := range items {
		items[index] = value
	}
	return items
}
