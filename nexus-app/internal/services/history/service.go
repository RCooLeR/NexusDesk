package history

import (
	"fmt"
	"sort"
	"strings"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

const defaultLimit = 80

type Service struct {
	metadata  *metadataSvc.Store
	artifacts *artifactsSvc.Store
}

func New(metadata *metadataSvc.Store, artifacts *artifactsSvc.Store) *Service {
	return &Service{metadata: metadata, artifacts: artifacts}
}

func (s *Service) List(options Options) ([]Item, error) {
	limit := normalizedLimit(options.Limit)
	items := []Item{}
	if s.metadata != nil {
		if wants(options.Kind, KindChat) {
			chatItems, err := s.chatItems(options.Query, limit)
			if err != nil {
				return nil, err
			}
			items = append(items, chatItems...)
		}
		if wants(options.Kind, KindJob) {
			jobItems, err := s.jobItems(options.Query)
			if err != nil {
				return nil, err
			}
			items = append(items, jobItems...)
		}
		if wants(options.Kind, KindAgent) {
			agentItems, err := s.agentItems(options.Query, limit)
			if err != nil {
				return nil, err
			}
			items = append(items, agentItems...)
		}
		if wants(options.Kind, KindData) {
			dataItems, err := s.dataItems(options.Query, limit)
			if err != nil {
				return nil, err
			}
			items = append(items, dataItems...)
		}
		if wants(options.Kind, KindArtifact) {
			artifactItems, err := s.metadataArtifactItems(options.Query, limit)
			if err != nil {
				return nil, err
			}
			if len(artifactItems) == 0 && s.artifacts != nil {
				artifactItems, err = s.artifactItems(options.Query)
				if err != nil {
					return nil, err
				}
			}
			items = append(items, artifactItems...)
		}
	}
	if s.artifacts != nil && s.metadata == nil && wants(options.Kind, KindArtifact) {
		artifactItems, err := s.artifactItems(options.Query)
		if err != nil {
			return nil, err
		}
		items = append(items, artifactItems...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].When.Equal(items[j].When) {
			return items[i].Ref > items[j].Ref
		}
		return items[i].When.After(items[j].When)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Service) dataItems(query string, limit int) ([]Item, error) {
	runs, err := s.metadata.ListSQLRuns(limit)
	if err != nil {
		return nil, err
	}
	dependencies, err := s.metadata.ListDatasetDependencies("", limit)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(runs)+len(dependencies))
	for _, run := range runs {
		item := sqlRunItem(run)
		if matches(item, query) {
			items = append(items, item)
		}
	}
	for _, dependency := range dependencies {
		item := datasetDependencyItem(dependency)
		if matches(item, query) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Service) metadataArtifactItems(query string, limit int) ([]Item, error) {
	records, err := s.metadata.ListArtifacts(query, true, limit)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(records))
	for _, record := range records {
		items = append(items, artifactItem(artifactFromRecord(record)))
	}
	return items, nil
}

func (s *Service) chatItems(query string, limit int) ([]Item, error) {
	records, err := s.metadata.SearchChatMessages(query, limit)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(records))
	for _, record := range records {
		item := Item{
			Kind:        KindChat,
			Ref:         record.ID,
			Title:       "Chat " + strings.ToLower(record.Role),
			Summary:     compact(record.Content, 180),
			Detail:      chatDetail(record),
			When:        record.CreatedAt,
			SourcePaths: append([]string{}, record.SourcePaths...),
		}
		if record.Model != "" {
			item.Title += " - " + record.Model
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) jobItems(query string) ([]Item, error) {
	jobs, err := s.metadata.ListJobs()
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(jobs))
	for _, job := range jobs {
		item := jobItem(job)
		if matches(item, query) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Service) agentItems(query string, limit int) ([]Item, error) {
	runs, err := s.metadata.ListAgentRuns(limit)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(runs))
	for _, run := range runs {
		item := Item{
			Kind:        KindAgent,
			Ref:         run.ID,
			Title:       "Agent run - " + firstNonEmpty(run.Status, "unknown"),
			Summary:     compact(firstNonEmpty(run.Message, run.Prompt), 180),
			Detail:      agentDetail(run),
			When:        firstTime(run.CompletedAt, run.StartedAt),
			SourcePaths: append([]string{}, run.SourcePaths...),
		}
		if matches(item, query) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Service) artifactItems(query string) ([]Item, error) {
	artifacts, err := s.artifacts.ListArtifacts(artifactsSvc.ListOptions{Query: query})
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(artifacts))
	for _, artifact := range artifacts {
		items = append(items, artifactItem(artifact))
	}
	return items, nil
}

func artifactItem(artifact artifactsSvc.Artifact) Item {
	return Item{
		Kind:        KindArtifact,
		Ref:         artifact.RelPath,
		Title:       firstNonEmpty(artifact.Title, artifact.RelPath),
		Summary:     artifactSummary(artifact),
		Detail:      artifactDetail(artifact),
		When:        firstTime(artifact.GeneratedAt, artifact.CreatedAt),
		SourcePaths: append([]string{}, artifact.SourcePaths...),
	}
}

func artifactFromRecord(record metadataSvc.ArtifactRecord) artifactsSvc.Artifact {
	return artifactsSvc.Artifact{
		Kind:         record.Kind,
		Title:        record.Title,
		RelPath:      record.RelPath,
		MetadataPath: record.MetadataPath,
		Size:         record.Size,
		JobID:        record.JobID,
		TaskID:       record.TaskID,
		Source:       record.Source,
		SourcePaths:  append([]string{}, record.SourcePaths...),
		Archived:     record.Archived,
		CreatedAt:    record.CreatedAt,
		GeneratedAt:  record.GeneratedAt,
	}
}

func jobItem(job jobsSvc.Job) Item {
	return Item{
		Kind:    KindJob,
		Ref:     job.ID,
		Title:   job.Label,
		Summary: strings.TrimSpace(strings.Join([]string{job.Kind, string(job.Status), firstNonEmpty(job.Message, job.Error)}, " - ")),
		Detail:  jobDetail(job),
		When:    firstTime(job.CompletedAt, job.StartedAt),
	}
}

func sqlRunItem(record metadataSvc.SQLRunRecord) Item {
	status := firstNonEmpty(record.Status, "unknown")
	return Item{
		Kind:        KindData,
		Ref:         record.ID,
		Title:       "SQL run - " + status,
		Summary:     fmt.Sprintf("%s - %s - %d/%d rows shown", record.RelPath, firstNonEmpty(record.Engine, "native-dataset-sql"), record.ShownRows, record.MatchedRows),
		Detail:      sqlRunDetail(record),
		When:        firstTime(record.CompletedAt, record.StartedAt),
		SourcePaths: []string{record.RelPath},
	}
}

func datasetDependencyItem(record metadataSvc.DatasetDependencyRecord) Item {
	return Item{
		Kind:        KindData,
		Ref:         record.ID,
		Title:       "Dataset dependency - " + firstNonEmpty(record.DependentKind, "unknown"),
		Summary:     fmt.Sprintf("%s %s %s:%s", record.SourcePath, firstNonEmpty(record.Relation, "links"), record.DependentKind, record.DependentRef),
		Detail:      datasetDependencyDetail(record),
		When:        firstTime(record.UpdatedAt, record.CreatedAt),
		SourcePaths: []string{record.SourcePath},
	}
}

func chatDetail(record metadataSvc.ChatMessageRecord) string {
	var builder strings.Builder
	writeLine(&builder, "Kind", "chat")
	writeLine(&builder, "Role", record.Role)
	writeLine(&builder, "Model", record.Model)
	writeLine(&builder, "Created", formatTime(record.CreatedAt))
	if len(record.SourcePaths) > 0 {
		writeLine(&builder, "Sources", strings.Join(record.SourcePaths, ", "))
	}
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(record.Content))
	return builder.String()
}

func sqlRunDetail(record metadataSvc.SQLRunRecord) string {
	var builder strings.Builder
	writeLine(&builder, "Kind", "data/sql-run")
	writeLine(&builder, "ID", record.ID)
	writeLine(&builder, "Dataset", record.RelPath)
	writeLine(&builder, "Engine", record.Engine)
	writeLine(&builder, "Status", record.Status)
	writeLine(&builder, "Rows", fmt.Sprintf("loaded %d, matched %d, shown %d", record.RowCount, record.MatchedRows, record.ShownRows))
	writeLine(&builder, "Duration", fmt.Sprintf("%d ms", record.DurationMs))
	writeLine(&builder, "Started", formatTime(record.StartedAt))
	writeLine(&builder, "Completed", formatTime(record.CompletedAt))
	writeLine(&builder, "Message", firstNonEmpty(record.Message, record.Error))
	writeLine(&builder, "Artifact", record.ArtifactPath)
	builder.WriteString("\nSQL\n")
	builder.WriteString(strings.TrimSpace(record.SQL))
	return builder.String()
}

func datasetDependencyDetail(record metadataSvc.DatasetDependencyRecord) string {
	var builder strings.Builder
	writeLine(&builder, "Kind", "data/dependency")
	writeLine(&builder, "ID", record.ID)
	writeLine(&builder, "Source", record.SourcePath)
	writeLine(&builder, "Dependent kind", record.DependentKind)
	writeLine(&builder, "Dependent ref", record.DependentRef)
	writeLine(&builder, "Relation", record.Relation)
	writeLine(&builder, "Created", formatTime(record.CreatedAt))
	writeLine(&builder, "Updated", formatTime(record.UpdatedAt))
	if len(record.Metadata) > 0 {
		builder.WriteString("\nMetadata\n")
		keys := make([]string, 0, len(record.Metadata))
		for key := range record.Metadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			writeLine(&builder, key, record.Metadata[key])
		}
	}
	return builder.String()
}

func jobDetail(job jobsSvc.Job) string {
	var builder strings.Builder
	writeLine(&builder, "Kind", "job")
	writeLine(&builder, "ID", job.ID)
	writeLine(&builder, "Type", job.Kind)
	writeLine(&builder, "Status", string(job.Status))
	writeLine(&builder, "Started", formatTime(job.StartedAt))
	writeLine(&builder, "Completed", formatTime(job.CompletedAt))
	writeLine(&builder, "Message", firstNonEmpty(job.Message, job.Error))
	if len(job.LogTail) > 0 {
		builder.WriteString("\nLog tail\n")
		for _, line := range job.LogTail {
			builder.WriteString("- ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func agentDetail(run metadataSvc.AgentRunRecord) string {
	var builder strings.Builder
	writeLine(&builder, "Kind", "agent")
	writeLine(&builder, "ID", run.ID)
	writeLine(&builder, "Job", run.JobID)
	writeLine(&builder, "Status", run.Status)
	writeLine(&builder, "Stop reason", run.StopReason)
	writeLine(&builder, "Iterations", fmt.Sprintf("%d", run.Iterations))
	writeLine(&builder, "Started", formatTime(run.StartedAt))
	writeLine(&builder, "Completed", formatTime(run.CompletedAt))
	if len(run.SourcePaths) > 0 {
		writeLine(&builder, "Sources", strings.Join(run.SourcePaths, ", "))
	}
	if len(run.Plan) > 0 {
		builder.WriteString("\nPlan\n")
		for _, step := range run.Plan {
			builder.WriteString("- ")
			builder.WriteString(step.Status)
			builder.WriteString(": ")
			builder.WriteString(step.Step)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nPrompt\n")
	builder.WriteString(strings.TrimSpace(run.Prompt))
	builder.WriteString("\n\nMessage\n")
	builder.WriteString(strings.TrimSpace(run.Message))
	return builder.String()
}

func artifactDetail(artifact artifactsSvc.Artifact) string {
	var builder strings.Builder
	writeLine(&builder, "Kind", "artifact")
	writeLine(&builder, "Path", artifact.RelPath)
	writeLine(&builder, "Type", artifact.Kind)
	writeLine(&builder, "Generated", formatTime(firstTime(artifact.GeneratedAt, artifact.CreatedAt)))
	writeLine(&builder, "Size", fmt.Sprintf("%d bytes", artifact.Size))
	writeLine(&builder, "Job", artifact.JobID)
	writeLine(&builder, "Task", artifact.TaskID)
	writeLine(&builder, "Source", artifact.Source)
	if len(artifact.SourcePaths) > 0 {
		writeLine(&builder, "Source paths", strings.Join(artifact.SourcePaths, ", "))
	}
	return builder.String()
}

func artifactSummary(artifact artifactsSvc.Artifact) string {
	parts := []string{artifact.Kind, fmt.Sprintf("%d bytes", artifact.Size)}
	if artifact.Source != "" {
		parts = append(parts, "source "+artifact.Source)
	}
	if artifact.Archived {
		parts = append(parts, "archived")
	}
	return strings.Join(nonEmpty(parts), " - ")
}

func wants(selected Kind, candidate Kind) bool {
	return selected == "" || selected == "all" || selected == candidate
}

func matches(item Item, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		string(item.Kind),
		item.Ref,
		item.Title,
		item.Summary,
		item.Detail,
		strings.Join(item.SourcePaths, " "),
	}, " "))
	return strings.Contains(haystack, query)
}

func normalizedLimit(limit int) int {
	if limit <= 0 || limit > 200 {
		return defaultLimit
	}
	return limit
}

func compact(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func firstTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func writeLine(builder *strings.Builder, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(value)
	builder.WriteString("\n")
}
