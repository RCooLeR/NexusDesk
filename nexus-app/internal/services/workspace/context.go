package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"nexusdesk/internal/domain"
)

const (
	defaultContextMaxFiles   = 32
	defaultContextMaxEntries = 1200
	defaultContextMaxDepth   = 8
	defaultContextMaxBytes   = 96 * 1024
)

type ContextCollectOptions struct {
	MaxFiles   int
	MaxEntries int
	MaxDepth   int
}

type ContextPackOptions struct {
	ContextCollectOptions
	MaxBytes int
}

type ContextFile struct {
	RelPath  string
	Required bool
}

type ContextCollection struct {
	Files     []ContextFile
	Roots     []string
	Truncated bool
}

type ContextPreviewFile struct {
	RelPath  string
	Required bool
}

type ContextPreview struct {
	Roots     []string
	Files     []ContextPreviewFile
	FileCount int
	Truncated bool
	Message   string
}

type ContextPack struct {
	Label       string
	Content     string
	SourcePaths []string
	Files       []ContextPreviewFile
	Truncated   bool
	Message     string
}

func (s *Service) CollectContextFiles(root string, relPaths []string, options ContextCollectOptions) (ContextCollection, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return ContextCollection{}, err
	}
	if len(relPaths) == 0 {
		return ContextCollection{}, errors.New("context paths are required")
	}

	maxFiles := normalizedPositive(options.MaxFiles, defaultContextMaxFiles)
	maxEntries := normalizedPositive(options.MaxEntries, defaultContextMaxEntries)
	maxDepth := normalizedPositive(options.MaxDepth, defaultContextMaxDepth)

	collection := ContextCollection{}
	seenFiles := map[string]bool{}
	seenRoots := map[string]bool{}

	for _, relPath := range relPaths {
		cleanRelPath, err := cleanContextRelPath(relPath)
		if err != nil {
			return ContextCollection{}, err
		}
		if !seenRoots[cleanRelPath] {
			collection.Roots = append(collection.Roots, cleanRelPath)
			seenRoots[cleanRelPath] = true
		}

		isDir, err := contextPathIsDirectory(absRoot, cleanRelPath)
		if err != nil {
			return ContextCollection{}, err
		}
		if !isDir {
			if isContextCandidate(cleanRelPath) {
				if appendContextFile(&collection, seenFiles, ContextFile{RelPath: cleanRelPath, Required: true}, maxFiles) {
					collection.Truncated = true
				}
			} else {
				collection.Truncated = true
			}
			continue
		}
		if s.collectDirectoryContext(absRoot, cleanRelPath, maxFiles, maxEntries, maxDepth, &collection, seenFiles) {
			break
		}
	}
	if len(collection.Files) == 0 {
		return ContextCollection{}, errors.New("context paths did not contain previewable text files")
	}
	return collection, nil
}

func (s *Service) PreviewContextPack(root string, relPaths []string, options ContextCollectOptions) (ContextPreview, error) {
	collection, err := s.CollectContextFiles(root, relPaths, options)
	if err != nil {
		return ContextPreview{}, err
	}
	files := contextPreviewFiles(collection.Files)
	message := "Context pack will include " + pluralizeFileCount(len(files)) + "."
	if collection.Truncated {
		message += " Some matching files were skipped by safety or size limits."
	}
	return ContextPreview{
		Roots:     append([]string{}, collection.Roots...),
		Files:     files,
		FileCount: len(files),
		Truncated: collection.Truncated,
		Message:   message,
	}, nil
}

func (s *Service) BuildContextPack(root string, relPaths []string, options ContextPackOptions) (ContextPack, error) {
	collection, err := s.CollectContextFiles(root, relPaths, options.ContextCollectOptions)
	if err != nil {
		return ContextPack{}, err
	}
	maxBytes := normalizedPositive(options.MaxBytes, defaultContextMaxBytes)
	var builder strings.Builder
	builder.WriteString("Workspace context pack\n")
	builder.WriteString("Requested roots: ")
	builder.WriteString(strings.Join(collection.Roots, ", "))
	builder.WriteString("\n")
	builder.WriteString("Included files:\n")

	sourcePaths := make([]string, 0, len(collection.Files))
	files := make([]ContextPreviewFile, 0, len(collection.Files))
	truncated := collection.Truncated
	for _, file := range collection.Files {
		builder.WriteString("- ")
		builder.WriteString(file.RelPath)
		if file.Required {
			builder.WriteString(" (selected)")
		}
		builder.WriteString("\n")
	}
	for _, file := range collection.Files {
		preview, err := s.PreviewFile(root, file.RelPath)
		if err != nil {
			truncated = true
			continue
		}
		content := previewContextText(preview)
		if strings.TrimSpace(content) == "" {
			truncated = true
			continue
		}
		section := "\n---\nWorkspace context: " + file.RelPath + "\n\n" + content + "\n"
		if builder.Len()+len(section) > maxBytes {
			remaining := maxBytes - builder.Len()
			if remaining > len("\n[context pack truncated]\n") {
				builder.WriteString(truncateUTF8(section, remaining-len("\n[context pack truncated]\n")))
				sourcePaths = append(sourcePaths, file.RelPath)
				files = append(files, ContextPreviewFile{RelPath: file.RelPath, Required: file.Required})
			}
			builder.WriteString("\n[context pack truncated]\n")
			truncated = true
			break
		}
		builder.WriteString(section)
		sourcePaths = append(sourcePaths, file.RelPath)
		files = append(files, ContextPreviewFile{RelPath: file.RelPath, Required: file.Required})
	}
	if len(sourcePaths) == 0 {
		return ContextPack{}, errors.New("context pack did not include usable text")
	}
	return ContextPack{
		Label:       contextPackLabel(collection.Roots),
		Content:     builder.String(),
		SourcePaths: sourcePaths,
		Files:       files,
		Truncated:   truncated,
		Message:     contextPackMessage(len(sourcePaths), truncated),
	}, nil
}

func (s *Service) collectDirectoryContext(absRoot string, rootRelPath string, maxFiles int, maxEntries int, maxDepth int, collection *ContextCollection, seenFiles map[string]bool) bool {
	start := absRoot
	if rootRelPath != "." {
		start = filepath.Join(absRoot, filepath.FromSlash(rootRelPath))
	}
	scanned := 0
	walkErr := filepath.WalkDir(start, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			collection.Truncated = true
			return filepath.SkipDir
		}
		if path == start {
			return nil
		}
		scanned++
		if scanned > maxEntries {
			collection.Truncated = true
			return errors.New("context entry cap reached")
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			collection.Truncated = true
			return nil
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if isIgnoredName(entry.Name()) {
				return filepath.SkipDir
			}
			if contextDepth(rel) > maxDepth {
				collection.Truncated = true
				return filepath.SkipDir
			}
			info, err := entry.Info()
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				collection.Truncated = true
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			collection.Truncated = true
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			collection.Truncated = true
			return nil
		}
		if !isContextCandidate(rel) {
			return nil
		}
		if appendContextFile(collection, seenFiles, ContextFile{RelPath: rel}, maxFiles) {
			collection.Truncated = true
			return errors.New("context file cap reached")
		}
		return nil
	})
	return walkErr != nil && collection.Truncated
}

func cleanContextRelPath(relPath string) (string, error) {
	trimmed := strings.Trim(strings.TrimSpace(relPath), `"'`)
	if trimmed == "" || trimmed == "." || trimmed == "/" {
		return ".", nil
	}
	cleanRelPath, err := cleanRel(trimmed)
	if err != nil {
		return "", err
	}
	if cleanRelPath == "" {
		return ".", nil
	}
	return cleanRelPath, nil
}

func contextPathIsDirectory(absRoot string, relPath string) (bool, error) {
	target := absRoot
	if relPath != "." {
		target = filepath.Join(absRoot, filepath.FromSlash(relPath))
	}
	if !isInside(absRoot, target) {
		return false, errors.New("workspace context path must stay inside the workspace")
	}
	info, err := os.Lstat(target)
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("workspace context cannot follow symlinks")
	}
	return info.IsDir(), nil
}

func appendContextFile(collection *ContextCollection, seen map[string]bool, file ContextFile, maxFiles int) bool {
	if seen[file.RelPath] {
		return false
	}
	if len(collection.Files) >= maxFiles {
		return true
	}
	collection.Files = append(collection.Files, file)
	seen[file.RelPath] = true
	return false
}

func isContextCandidate(relPath string) bool {
	ext := strings.ToLower(filepath.Ext(relPath))
	if ext == "" {
		return true
	}
	switch ext {
	case ".c", ".conf", ".cpp", ".cs", ".css", ".csv", ".docx", ".env", ".go", ".h", ".hpp",
		".html", ".ini", ".java", ".js", ".json", ".jsonl", ".jsx", ".log", ".md", ".ndjson",
		".pdf", ".php", ".ps1", ".py", ".rb", ".rs", ".rtf", ".sh", ".sql", ".toml", ".ts",
		".tsx", ".tsv", ".txt", ".xml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func previewContextText(preview domain.FilePreview) string {
	switch preview.Kind {
	case domain.PreviewText, domain.PreviewTable, domain.PreviewDoc, domain.PreviewPDF:
		return preview.Text
	default:
		return ""
	}
}

func contextPreviewFiles(files []ContextFile) []ContextPreviewFile {
	previewFiles := make([]ContextPreviewFile, 0, len(files))
	for _, file := range files {
		previewFiles = append(previewFiles, ContextPreviewFile{RelPath: file.RelPath, Required: file.Required})
	}
	return previewFiles
}

func contextPackLabel(roots []string) string {
	if len(roots) == 1 && roots[0] == "." {
		return "project: ."
	}
	if len(roots) == 1 {
		return "context: " + roots[0]
	}
	return fmt.Sprintf("context: %d roots", len(roots))
}

func contextPackMessage(fileCount int, truncated bool) string {
	message := "Built context pack with " + pluralizeFileCount(fileCount) + "."
	if truncated {
		message += " Some content was skipped or capped."
	}
	return message
}

func pluralizeFileCount(count int) string {
	if count == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", count)
}

func normalizedPositive(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func contextDepth(relPath string) int {
	if relPath == "" || relPath == "." {
		return 0
	}
	return strings.Count(relPath, "/") + 1
}

func truncateUTF8(content string, byteLimit int) string {
	if byteLimit <= 0 {
		return ""
	}
	if len(content) <= byteLimit {
		return content
	}
	truncated := content[:byteLimit]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
