package metadata

import (
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
	report := CompatibilityImportReport{}
	sqlRuns, dependencies, skipped, err := s.importCompatibilitySQLiteDatasets()
	if err != nil {
		return report, err
	}
	report.SQLRuns += sqlRuns
	report.DatasetDependencies += dependencies
	report.Skipped += skipped
	if _, err := s.Ensure(); err != nil {
		return CompatibilityImportReport{}, err
	}
	chatCount, skipped, err := s.importCompatibilityChats(options.ChatHistoryPath)
	if err != nil {
		return report, err
	}
	report.Chats += chatCount
	report.Skipped += skipped
	count, skipped, err := s.importCompatibilityApprovals()
	if err != nil {
		return report, err
	}
	report.Approvals += count
	report.Skipped += skipped
	count, skipped, err = s.importCompatibilityArtifacts(options.ArtifactLimit)
	if err != nil {
		return report, err
	}
	report.Artifacts += count
	report.Skipped += skipped
	count, skipped, err = s.importCompatibilityToolRuns()
	if err != nil {
		return report, err
	}
	report.ToolRuns += count
	report.Skipped += skipped
	report.Message = report.compatibilityMessage()
	return report, nil
}

func (r CompatibilityImportReport) compatibilityMessage() string {
	total := r.Chats + r.Approvals + r.Artifacts + r.ToolRuns + r.SQLRuns + r.DatasetDependencies
	if total == 0 {
		if r.Skipped > 0 {
			return fmt.Sprintf("No Wails-era metadata imported; %d malformed or unsupported item(s) skipped.", r.Skipped)
		}
		return "No Wails-era metadata found to import."
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
	message := "Imported Wails-era metadata: " + strings.Join(parts, ", ") + "."
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

func (s *Store) importCompatibilityChats(path string) (int, int, error) {
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
	for _, message := range messages {
		sourcePaths := cleanCompatibilityPaths(append(message.SourcePaths, message.ContextRelPath))
		record := ChatMessageRecord{
			Role:        message.Role,
			Content:     message.Content,
			SourcePaths: sourcePaths,
			CreatedAt:   parseCompatibilityTime(message.CreatedAt),
		}
		if err := s.SaveChatMessage(record); err != nil {
			skipped++
			continue
		}
		imported++
	}
	return imported, skipped, nil
}

func (s *Store) importCompatibilityApprovals() (int, int, error) {
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
	for _, item := range items {
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

func (s *Store) importCompatibilityArtifacts(limit int) (int, int, error) {
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
		return 0, 0, err
	}
	sort.Strings(sidecars)
	imported := 0
	skipped := 0
	for index, sidecar := range sidecars {
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

func (s *Store) importCompatibilityToolRuns() (int, int, error) {
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
		Prompt:      "Imported Wails-era tool-run log",
		Status:      "imported",
		Message:     fmt.Sprintf("Imported %d Wails-era tool run record(s).", len(items)),
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
