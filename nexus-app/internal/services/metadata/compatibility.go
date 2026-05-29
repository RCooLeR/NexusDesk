package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultCompatibilityArtifactLimit = 300
const compatibilityAgentRunID = "compatibility-json-tool-runs"

type CompatibilityImportOptions struct {
	ChatHistoryPath string
	ArtifactLimit   int
	Force           bool
}

type CompatibilityImportReport struct {
	Chats               int
	Approvals           int
	Artifacts           int
	ToolRuns            int
	SQLRuns             int
	DatasetDependencies int
	Skipped             int
	Message             string
}

func (s *Store) ImportCompatibilityData(options CompatibilityImportOptions) (CompatibilityImportReport, error) {
	return s.ImportCompatibilityDataContext(context.Background(), options)
}

func (s *Store) ImportCompatibilityDataContext(ctx context.Context, options CompatibilityImportOptions) (CompatibilityImportReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := compatibilityContextErr(ctx); err != nil {
		return CompatibilityImportReport{}, err
	}
	report := CompatibilityImportReport{}
	if !options.Force {
		if alreadyImported, err := s.compatibilityImportAlreadyCompleted(); err == nil && alreadyImported {
			report.Message = "Compatibility metadata import already completed for this workspace."
			return report, nil
		}
	}
	sqlRuns, dependencies, skipped, err := s.importCompatibilitySQLiteDatasets(ctx)
	if err != nil {
		return report, err
	}
	report.SQLRuns += sqlRuns
	report.DatasetDependencies += dependencies
	report.Skipped += skipped
	if err := compatibilityContextErr(ctx); err != nil {
		return report, err
	}
	if _, err := s.Ensure(); err != nil {
		return CompatibilityImportReport{}, err
	}
	chatCount, skipped, err := s.importCompatibilityChats(ctx, options.ChatHistoryPath)
	if err != nil {
		return report, err
	}
	report.Chats += chatCount
	report.Skipped += skipped
	if err := compatibilityContextErr(ctx); err != nil {
		return report, err
	}
	count, skipped, err := s.importCompatibilityApprovals(ctx)
	if err != nil {
		return report, err
	}
	report.Approvals += count
	report.Skipped += skipped
	if err := compatibilityContextErr(ctx); err != nil {
		return report, err
	}
	count, skipped, err = s.importCompatibilityArtifacts(ctx, options.ArtifactLimit)
	if err != nil {
		return report, err
	}
	report.Artifacts += count
	report.Skipped += skipped
	if err := compatibilityContextErr(ctx); err != nil {
		return report, err
	}
	count, skipped, err = s.importCompatibilityToolRuns(ctx)
	if err != nil {
		return report, err
	}
	report.ToolRuns += count
	report.Skipped += skipped
	report.Message = report.compatibilityMessage()
	if err := s.markCompatibilityImportCompleted(report); err != nil {
		report.Message += " (compatibility import marker write failed: " + err.Error() + ")"
	}
	return report, nil
}

func compatibilityContextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type compatibilityImportStamp struct {
	ImportedAt string                    `json:"importedAt"`
	Report     CompatibilityImportReport `json:"report"`
}

func (s *Store) CompatibilityImportPending() (bool, error) {
	done, err := s.compatibilityImportAlreadyCompleted()
	if err != nil {
		return true, err
	}
	return !done, nil
}

func (s *Store) compatibilityImportStampPath() string {
	return filepath.Join(filepath.Dir(s.path), "compatibility-import.json")
}

func (s *Store) compatibilityImportAlreadyCompleted() (bool, error) {
	path := s.compatibilityImportStampPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return false, nil
	}
	stamp := compatibilityImportStamp{}
	if err := json.Unmarshal(data, &stamp); err != nil {
		if quarantineErr := s.quarantineCompatibilityImportStamp(path); quarantineErr != nil {
			return false, quarantineErr
		}
		return false, nil
	}
	return true, nil
}

func (s *Store) markCompatibilityImportCompleted(report CompatibilityImportReport) error {
	path := s.compatibilityImportStampPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := compatibilityImportStamp{
		ImportedAt: formatTime(time.Now().UTC()),
		Report:     report,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tempPath := path + ".tmp." + fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func (s *Store) quarantineCompatibilityImportStamp(path string) error {
	target := path + ".corrupt." + fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	if err := os.Rename(path, target); err != nil {
		return err
	}
	return nil
}

func (r CompatibilityImportReport) compatibilityMessage() string {
	total := r.Chats + r.Approvals + r.Artifacts + r.ToolRuns + r.SQLRuns + r.DatasetDependencies
	if total == 0 {
		if r.Skipped > 0 {
			return fmt.Sprintf("No legacy metadata imported; %d malformed or unsupported item(s) skipped.", r.Skipped)
		}
		return "No legacy metadata found to import."
	}
	parts := []string{}
	if r.Chats > 0 {
		parts = append(parts, fmt.Sprintf("%d chat", r.Chats))
	}
	if r.Approvals > 0 {
		parts = append(parts, fmt.Sprintf("%d approval", r.Approvals))
	}
	if r.Artifacts > 0 {
		parts = append(parts, fmt.Sprintf("%d artifact", r.Artifacts))
	}
	if r.ToolRuns > 0 {
		parts = append(parts, fmt.Sprintf("%d tool run", r.ToolRuns))
	}
	if r.SQLRuns > 0 {
		parts = append(parts, fmt.Sprintf("%d SQL run", r.SQLRuns))
	}
	if r.DatasetDependencies > 0 {
		parts = append(parts, fmt.Sprintf("%d dataset dependency", r.DatasetDependencies))
	}
	message := "Imported legacy metadata: " + strings.Join(parts, ", ") + "."
	if r.Skipped > 0 {
		message += fmt.Sprintf(" Skipped %d malformed or unsupported item(s).", r.Skipped)
	}
	return message
}

type compatibilityChatMessage struct {
	Role           string   `json:"role"`
	Content        string   `json:"content"`
	ContextRelPath string   `json:"contextRelPath"`
	SourcePaths    []string `json:"sourcePaths"`
	CreatedAt      string   `json:"createdAt"`
}

type compatibilityApprovalRecord struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Risk      string `json:"risk"`
	Decision  string `json:"decision"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type compatibilityArtifactMetadata struct {
	Kind           string   `json:"kind"`
	Title          string   `json:"title"`
	Source         string   `json:"source"`
	SourcePaths    []string `json:"sourcePaths"`
	ContextRelPath string   `json:"contextRelPath"`
	Prompt         string   `json:"prompt"`
	Model          string   `json:"model"`
	CreatedAt      string   `json:"createdAt"`
}

type compatibilityToolRunRecord struct {
	ID            string            `json:"id"`
	ToolName      string            `json:"toolName"`
	Target        string            `json:"target"`
	Risk          string            `json:"risk"`
	Status        string            `json:"status"`
	Mode          string            `json:"mode"`
	Inputs        map[string]string `json:"inputs"`
	OutputSummary string            `json:"outputSummary"`
	Error         string            `json:"error"`
	ApprovalID    string            `json:"approvalId"`
	StartedAt     string            `json:"startedAt"`
	CompletedAt   string            `json:"completedAt"`
	DurationMs    int64             `json:"durationMs"`
}

func (s *Store) importCompatibilityChats(ctx context.Context, path string) (int, int, error) {
	if err := compatibilityContextErr(ctx); err != nil {
		return 0, 0, err
	}
	path = firstNonEmptyString(path, defaultCompatibilityChatHistoryPath())
	if strings.TrimSpace(path) == "" {
		return 0, 0, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	records := map[string][]compatibilityChatMessage{}
	if err := json.Unmarshal(data, &records); err != nil {
		return 0, 0, err
	}
	messages := records[compatibilityWorkspaceHistoryKey(s.root)]
	imported := 0
	skipped := 0
	for index, message := range messages {
		if index%64 == 0 {
			if err := compatibilityContextErr(ctx); err != nil {
				return imported, skipped, err
			}
		}
		sourcePaths := cleanCompatibilityPaths(message.SourcePaths)
		record := ChatMessageRecord{
			Role:           message.Role,
			Content:        message.Content,
			ContextRelPath: filepath.ToSlash(strings.TrimSpace(message.ContextRelPath)),
			SourcePaths:    sourcePaths,
			CreatedAt:      parseCompatibilityTime(message.CreatedAt),
		}
		if err := s.SaveChatMessage(record); err != nil {
			skipped++
			continue
		}
		imported++
	}
	return imported, skipped, nil
}

func (s *Store) importCompatibilityApprovals(ctx context.Context) (int, int, error) {
	if err := compatibilityContextErr(ctx); err != nil {
		return 0, 0, err
	}
	path := filepath.Join(s.root, ".nexusdesk", "approvals", "log.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	items := []compatibilityApprovalRecord{}
	if err := json.Unmarshal(data, &items); err != nil {
		return 0, 0, err
	}
	imported := 0
	skipped := 0
	for index, item := range items {
		if index%64 == 0 {
			if err := compatibilityContextErr(ctx); err != nil {
				return imported, skipped, err
			}
		}
		record := ApprovalRecord{
			ID:        strings.TrimSpace(item.ID),
			Action:    item.Action,
			Target:    item.Target,
			Risk:      firstNonEmptyString(item.Risk, "medium"),
			Decision:  firstNonEmptyString(item.Decision, "applied"),
			Message:   item.Message,
			CreatedAt: parseCompatibilityTime(item.CreatedAt),
		}
		if err := s.SaveApprovalRecord(record); err != nil {
			skipped++
			continue
		}
		imported++
	}
	return imported, skipped, nil
}

func (s *Store) importCompatibilityArtifacts(ctx context.Context, limit int) (int, int, error) {
	if err := compatibilityContextErr(ctx); err != nil {
		return 0, 0, err
	}
	if limit <= 0 || limit > defaultCompatibilityArtifactLimit {
		limit = defaultCompatibilityArtifactLimit
	}
	artifactRoot := filepath.Join(s.root, ".nexusdesk", "artifacts")
	if info, err := os.Stat(artifactRoot); os.IsNotExist(err) {
		return 0, 0, nil
	} else if err != nil {
		return 0, 0, err
	} else if !info.IsDir() {
		return 0, 0, nil
	}
	sidecars := []string{}
	err := filepath.WalkDir(artifactRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if err := compatibilityContextErr(ctx); err != nil {
			return err
		}
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".meta.json") {
			sidecars = append(sidecars, path)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, 0, err
		}
		return 0, 0, err
	}
	sort.Strings(sidecars)
	imported := 0
	skipped := 0
	for index, sidecar := range sidecars {
		if index%64 == 0 {
			if err := compatibilityContextErr(ctx); err != nil {
				return imported, skipped, err
			}
		}
		if imported >= limit {
			skipped += len(sidecars) - index
			break
		}
		record, err := s.compatibilityArtifactRecord(sidecar)
		if err != nil {
			skipped++
			continue
		}
		if err := s.SaveArtifact(record); err != nil {
			skipped++
			continue
		}
		imported++
	}
	return imported, skipped, nil
}

func (s *Store) compatibilityArtifactRecord(sidecarPath string) (ArtifactRecord, error) {
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return ArtifactRecord{}, err
	}
	var metadata compatibilityArtifactMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return ArtifactRecord{}, err
	}
	artifactPath, err := findCompatibilityArtifactFile(sidecarPath)
	if err != nil {
		return ArtifactRecord{}, err
	}
	relPath, err := filepath.Rel(s.root, artifactPath)
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadataRelPath, err := filepath.Rel(s.root, sidecarPath)
	if err != nil {
		return ArtifactRecord{}, err
	}
	info, err := os.Stat(artifactPath)
	if err != nil {
		return ArtifactRecord{}, err
	}
	sourcePaths := cleanCompatibilityPaths(append(metadata.SourcePaths, metadata.ContextRelPath))
	createdAt := parseCompatibilityTime(metadata.CreatedAt)
	if createdAt.IsZero() {
		createdAt = info.ModTime().UTC()
	}
	return ArtifactRecord{
		Kind:         firstNonEmptyString(metadata.Kind, compatibilityArtifactKind(artifactPath)),
		Title:        firstNonEmptyString(metadata.Title, filepath.Base(artifactPath)),
		RelPath:      filepath.ToSlash(relPath),
		MetadataPath: filepath.ToSlash(metadataRelPath),
		Size:         info.Size(),
		Source:       metadata.Source,
		SourcePaths:  sourcePaths,
		Archived:     strings.Contains(filepath.ToSlash(relPath), "/archive/"),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		UpdatedAt:    info.ModTime().UTC(),
	}, nil
}

func findCompatibilityArtifactFile(sidecarPath string) (string, error) {
	if !strings.HasSuffix(strings.ToLower(sidecarPath), ".meta.json") {
		return "", errors.New("artifact sidecar must end with .meta.json")
	}
	prefix := strings.TrimSuffix(sidecarPath, ".meta.json")
	matches, err := filepath.Glob(prefix + ".*")
	if err != nil {
		return "", err
	}
	preferred := []string{".md", ".csv", ".svg", ".json", ".txt", ".html", ".xml"}
	for _, extension := range preferred {
		for _, match := range matches {
			if strings.EqualFold(filepath.Ext(match), extension) && !strings.HasSuffix(strings.ToLower(match), ".meta.json") {
				return match, nil
			}
		}
	}
	for _, match := range matches {
		if !strings.HasSuffix(strings.ToLower(match), ".meta.json") {
			return match, nil
		}
	}
	return "", errors.New("artifact file for sidecar not found")
}

func (s *Store) importCompatibilityToolRuns(ctx context.Context) (int, int, error) {
	if err := compatibilityContextErr(ctx); err != nil {
		return 0, 0, err
	}
	path := filepath.Join(s.root, ".nexusdesk", "tool-runs", "log.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	items := []compatibilityToolRunRecord{}
	if err := json.Unmarshal(data, &items); err != nil {
		return 0, 0, err
	}
	if len(items) == 0 {
		return 0, 0, nil
	}
	agentRun := AgentRunRecord{
		ID:          compatibilityAgentRunID,
		Prompt:      "Imported legacy tool-run log",
		Status:      "imported",
		Message:     fmt.Sprintf("Imported %d legacy tool run record(s).", len(items)),
		StopReason:  "compatibility-json",
		StartedAt:   compatibilityToolRunStart(items),
		CompletedAt: compatibilityToolRunCompleted(items),
	}
	if err := s.SaveAgentRun(agentRun); err != nil {
		return 0, 0, err
	}
	imported := 0
	skipped := 0
	for index, item := range items {
		if index%64 == 0 {
			if err := compatibilityContextErr(ctx); err != nil {
				return imported, skipped, err
			}
		}
		record := ToolRunRecord{
			ID:         strings.TrimSpace(item.ID),
			AgentRunID: compatibilityAgentRunID,
			Sequence:   index + 1,
			ToolName:   item.ToolName,
			Risk:       firstNonEmptyString(item.Risk, "low"),
			Args:       item.Inputs,
			Observation: firstNonEmptyString(
				item.OutputSummary,
				strings.TrimSpace(strings.Join([]string{item.Mode, item.Status, item.Target}, " ")),
			),
			Error:       item.Error,
			StartedAt:   parseCompatibilityTime(item.StartedAt),
			CompletedAt: parseCompatibilityTime(item.CompletedAt),
		}
		if record.ToolName == "" {
			skipped++
			continue
		}
		if err := s.SaveToolRun(record); err != nil {
			skipped++
			continue
		}
		imported++
	}
	return imported, skipped, nil
}

func compatibilityToolRunStart(items []compatibilityToolRunRecord) time.Time {
	var first time.Time
	for _, item := range items {
		started := parseCompatibilityTime(item.StartedAt)
		if started.IsZero() {
			continue
		}
		if first.IsZero() || started.Before(first) {
			first = started
		}
	}
	if first.IsZero() {
		return time.Now().UTC()
	}
	return first
}

func compatibilityToolRunCompleted(items []compatibilityToolRunRecord) time.Time {
	var last time.Time
	for _, item := range items {
		completed := parseCompatibilityTime(item.CompletedAt)
		if completed.IsZero() {
			completed = parseCompatibilityTime(item.StartedAt)
		}
		if completed.After(last) {
			last = completed
		}
	}
	if last.IsZero() {
		return compatibilityToolRunStart(items)
	}
	return last
}

func defaultCompatibilityChatHistoryPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		return ""
	}
	return filepath.Join(configDir, "NexusAugenticStudio", "chat-history.json")
}

func compatibilityWorkspaceHistoryKey(root string) string {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(absRoot))))
	return hex.EncodeToString(sum[:])
}

func parseCompatibilityTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func cleanCompatibilityPaths(paths []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" || path == "." || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

func compatibilityArtifactKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		return "markdown"
	case ".csv":
		return "csv"
	case ".svg":
		return "chart"
	case ".json":
		return "json"
	default:
		return "artifact"
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
