// Package issuereport builds redacted support bundles for user-triggered
// diagnostics exports without depending on the Fyne UI.
package issuereport

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"nexusdesk/internal/buildinfo"
)

const (
	maxIncludedWorkspaceFileBytes = 64 * 1024
)

type Options struct {
	WorkspaceRoot            string
	OutputDir                string
	DiagnosticsReport        string
	ActivityTail             []string
	IncludeWorkspaceContents bool
	WorkspaceContentRelPaths []string
	Now                      time.Time
}

type Result struct {
	Path      string
	Files     []string
	SizeBytes int64
	CreatedAt time.Time
}

type manifest struct {
	CreatedAt                 string         `json:"createdAt"`
	App                       buildinfo.Info `json:"app"`
	GoVersion                 string         `json:"goVersion"`
	OS                        string         `json:"os"`
	Arch                      string         `json:"arch"`
	WorkspaceName             string         `json:"workspaceName,omitempty"`
	WorkspacePath             string         `json:"workspacePath"`
	WorkspacePathHash         string         `json:"workspacePathHash,omitempty"`
	IncludesWorkspaceContents bool           `json:"includesWorkspaceContents"`
	Safety                    []string       `json:"safety"`
	Files                     []string       `json:"files"`
}

type workspaceSummary struct {
	WorkspaceName     string              `json:"workspaceName,omitempty"`
	WorkspacePath     string              `json:"workspacePath"`
	WorkspacePathHash string              `json:"workspacePathHash,omitempty"`
	StateFiles        []workspaceFileInfo `json:"stateFiles,omitempty"`
}

type workspaceFileInfo struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
}

// Export writes a redacted support bundle. Workspace file contents are excluded
// unless IncludeWorkspaceContents is true and explicit relative paths are passed.
func Export(options Options) (Result, error) {
	root := strings.TrimSpace(options.WorkspaceRoot)
	if root == "" {
		return Result{}, errors.New("issue report requires a workspace root")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	if info, err := os.Stat(absRoot); err != nil {
		return Result{}, err
	} else if !info.IsDir() {
		return Result{}, fmt.Errorf("issue report workspace root is not a directory: %s", absRoot)
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = filepath.Join(absRoot, ".nexusdesk", "issue-reports")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, err
	}
	reportPath := filepath.Join(outputDir, fmt.Sprintf("issue-report-%s.zip", now.Format("20060102-150405")))
	file, err := os.OpenFile(reportPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return Result{}, err
	}
	removeOnError := true
	defer func() {
		_ = file.Close()
		if removeOnError {
			_ = os.Remove(reportPath)
		}
	}()

	writer := zip.NewWriter(file)
	files := []string{}
	redactor := newRedactor(absRoot)

	summary, err := collectWorkspaceSummary(absRoot)
	if err != nil {
		_ = writer.Close()
		return Result{}, err
	}
	summary.WorkspacePath = redactor.redact(summary.WorkspacePath)
	if err := addJSON(writer, "workspace-summary.json", summary); err != nil {
		_ = writer.Close()
		return Result{}, err
	}
	files = append(files, "workspace-summary.json")

	diagnostics := strings.TrimSpace(options.DiagnosticsReport)
	if diagnostics == "" {
		diagnostics = "No diagnostics report text was available when the issue report was exported."
	}
	if err := addText(writer, "diagnostics.md", redactor.redact(diagnostics)+"\n"); err != nil {
		_ = writer.Close()
		return Result{}, err
	}
	files = append(files, "diagnostics.md")

	activity := strings.TrimSpace(strings.Join(options.ActivityTail, "\n"))
	if activity == "" {
		activity = "No activity tail was available when the issue report was exported."
	}
	if err := addText(writer, "activity-tail.txt", redactor.redact(activity)+"\n"); err != nil {
		_ = writer.Close()
		return Result{}, err
	}
	files = append(files, "activity-tail.txt")

	env := map[string]string{
		"goVersion": runtime.Version(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"version":   buildinfo.Current().Version,
		"commit":    buildinfo.Current().Commit,
		"buildDate": buildinfo.Current().BuildDate,
	}
	if err := addJSON(writer, "environment.json", env); err != nil {
		_ = writer.Close()
		return Result{}, err
	}
	files = append(files, "environment.json")

	if options.IncludeWorkspaceContents {
		included, err := addExplicitWorkspaceFiles(writer, absRoot, redactor, options.WorkspaceContentRelPaths)
		if err != nil {
			_ = writer.Close()
			return Result{}, err
		}
		files = append(files, included...)
	}

	m := manifest{
		CreatedAt:                 now.Format(time.RFC3339),
		App:                       buildinfo.Current(),
		GoVersion:                 runtime.Version(),
		OS:                        runtime.GOOS,
		Arch:                      runtime.GOARCH,
		WorkspaceName:             filepath.Base(absRoot),
		WorkspacePath:             "[workspace-root]",
		WorkspacePathHash:         hashString(absRoot),
		IncludesWorkspaceContents: options.IncludeWorkspaceContents && len(options.WorkspaceContentRelPaths) > 0,
		Safety: []string{
			"Secrets and common credential patterns are redacted from text entries.",
			"Workspace file contents are excluded unless explicit relative paths are provided.",
			"Default bundle entries contain diagnostics, environment metadata, activity tail, and workspace-state file names only.",
		},
		Files: append([]string{}, files...),
	}
	if err := addJSON(writer, "issue-report.json", m); err != nil {
		_ = writer.Close()
		return Result{}, err
	}
	files = append(files, "issue-report.json")

	if err := writer.Close(); err != nil {
		return Result{}, err
	}
	if err := file.Close(); err != nil {
		return Result{}, err
	}
	info, err := os.Stat(reportPath)
	if err != nil {
		return Result{}, err
	}
	removeOnError = false
	sort.Strings(files)
	return Result{Path: reportPath, Files: files, SizeBytes: info.Size(), CreatedAt: now}, nil
}

func collectWorkspaceSummary(absRoot string) (workspaceSummary, error) {
	stateRoot := filepath.Join(absRoot, ".nexusdesk")
	summary := workspaceSummary{
		WorkspaceName:     filepath.Base(absRoot),
		WorkspacePath:     absRoot,
		WorkspacePathHash: hashString(absRoot),
	}
	info, err := os.Stat(stateRoot)
	if os.IsNotExist(err) {
		return summary, nil
	}
	if err != nil {
		return workspaceSummary{}, err
	}
	if !info.IsDir() {
		return summary, nil
	}
	err = filepath.WalkDir(stateRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := strings.ToLower(entry.Name())
			if name == "backups" || name == "issue-reports" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		summary.StateFiles = append(summary.StateFiles, workspaceFileInfo{
			Path:      filepath.ToSlash(rel),
			SizeBytes: info.Size(),
		})
		return nil
	})
	if err != nil {
		return workspaceSummary{}, err
	}
	sort.Slice(summary.StateFiles, func(i, j int) bool {
		return summary.StateFiles[i].Path < summary.StateFiles[j].Path
	})
	return summary, nil
}

func addExplicitWorkspaceFiles(writer *zip.Writer, absRoot string, redactor redactor, relPaths []string) ([]string, error) {
	included := []string{}
	seen := map[string]bool{}
	for _, relPath := range relPaths {
		cleanRel, err := cleanExplicitRelPath(relPath)
		if err != nil {
			return nil, err
		}
		if cleanRel == "" || seen[cleanRel] {
			continue
		}
		seen[cleanRel] = true
		absPath := filepath.Join(absRoot, filepath.FromSlash(cleanRel))
		if !isInside(absRoot, absPath) {
			return nil, fmt.Errorf("explicit issue-report path escapes workspace: %s", relPath)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		if info.Size() > maxIncludedWorkspaceFileBytes {
			return nil, fmt.Errorf("explicit issue-report file %s exceeds %d bytes", cleanRel, maxIncludedWorkspaceFileBytes)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		zipPath := filepath.ToSlash(filepath.Join("explicit-workspace-files", cleanRel))
		if err := addText(writer, zipPath, redactor.redact(string(data))); err != nil {
			return nil, err
		}
		included = append(included, zipPath)
	}
	return included, nil
}

func cleanExplicitRelPath(relPath string) (string, error) {
	value := strings.TrimSpace(filepath.ToSlash(relPath))
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return "", fmt.Errorf("explicit issue-report path must be workspace-relative: %s", relPath)
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." {
		return "", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("explicit issue-report path escapes workspace: %s", relPath)
	}
	if strings.HasPrefix(cleaned, ".nexusdesk/") || cleaned == ".nexusdesk" {
		return "", errors.New("explicit issue-report workspace contents cannot include .nexusdesk state")
	}
	return cleaned, nil
}

func addText(writer *zip.Writer, name string, text string) error {
	header := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate}
	header.SetMode(0o600)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(entry, text)
	return err
}

func addJSON(writer *zip.Writer, name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return addText(writer, name, string(data)+"\n")
}

func isInside(root string, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(value)))
	return hex.EncodeToString(sum[:])
}

type redactor struct {
	replacements []string
}

func newRedactor(workspaceRoot string) redactor {
	replacements := []string{}
	if cleaned := strings.TrimSpace(filepath.Clean(workspaceRoot)); cleaned != "" {
		replacements = append(replacements, cleaned, filepath.ToSlash(cleaned))
	}
	if home, err := os.UserHomeDir(); err == nil {
		if cleaned := strings.TrimSpace(filepath.Clean(home)); cleaned != "" {
			replacements = append(replacements, cleaned, filepath.ToSlash(cleaned))
		}
	}
	return redactor{replacements: replacements}
}

func (r redactor) redact(value string) string {
	result := strings.TrimSpace(value)
	for _, replacement := range r.replacements {
		if replacement == "" || replacement == "." {
			continue
		}
		result = strings.ReplaceAll(result, replacement, "[redacted-path]")
	}
	for _, pattern := range secretPatterns {
		result = pattern.re.ReplaceAllString(result, pattern.replacement)
	}
	return result
}

var secretPatterns = []struct {
	re          *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`), "Bearer [redacted]"},
	{regexp.MustCompile(`sk-[A-Za-z0-9._-]+`), "sk-[redacted]"},
	{regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(token\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(password\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(secret\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(Authorization\s*[:=]\s*)[^\n\r]+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(postgres|mysql|sqlserver)://([^:\s/@]+):([^@\s]+)@`), `${1}://${2}:[redacted]@`},
}
