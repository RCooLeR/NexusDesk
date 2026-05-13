package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultContextMaxFiles       = 32
	defaultContextMaxScanEntries = 1200
)

type ContextCollectOptions struct {
	MaxFiles   int
	MaxEntries int
	MaxDepth   int
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

func CollectContextFiles(root string, relPaths []string, options ContextCollectOptions) (ContextCollection, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ContextCollection{}, err
	}

	if len(relPaths) == 0 {
		return ContextCollection{}, errors.New("context paths are required")
	}

	maxFiles := options.MaxFiles
	if maxFiles <= 0 {
		maxFiles = defaultContextMaxFiles
	}

	maxEntries := options.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultContextMaxScanEntries
	}

	snapshot, err := Scan(absRoot, ScanOptions{MaxDepth: options.MaxDepth, MaxEntries: maxEntries})
	if err != nil {
		return ContextCollection{}, err
	}

	collection := ContextCollection{}
	seenFiles := map[string]bool{}
	seenRoots := map[string]bool{}

	for _, relPath := range relPaths {
		cleanRel, err := cleanContextRelPath(relPath)
		if err != nil {
			return ContextCollection{}, err
		}
		if !seenRoots[cleanRel] {
			collection.Roots = append(collection.Roots, cleanRel)
			seenRoots[cleanRel] = true
		}

		isDir, err := contextPathIsDirectory(absRoot, cleanRel)
		if err != nil {
			return ContextCollection{}, err
		}
		if !isDir {
			if isContextCandidate(cleanRel) {
				collection.Truncated = appendContextFile(&collection, seenFiles, ContextFile{RelPath: cleanRel, Required: true}, maxFiles)
			} else {
				collection.Truncated = true
			}
			continue
		}

		for _, node := range snapshot.Nodes {
			if node.Kind != "file" || !pathIsInsideContextRoot(node.RelPath, cleanRel) || !isContextCandidate(node.RelPath) {
				continue
			}
			if appendContextFile(&collection, seenFiles, ContextFile{RelPath: node.RelPath}, maxFiles) {
				collection.Truncated = true
				break
			}
		}
		if collection.Truncated {
			break
		}
	}

	if snapshot.Truncated {
		collection.Truncated = true
	}
	if len(collection.Files) == 0 {
		return ContextCollection{}, errors.New("context paths did not contain previewable text files")
	}

	return collection, nil
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

func cleanContextRelPath(relPath string) (string, error) {
	trimmed := strings.TrimSpace(relPath)
	if trimmed == "" || trimmed == "." || trimmed == "/" {
		return ".", nil
	}

	cleanRel := filepath.Clean(filepath.FromSlash(trimmed))
	if cleanRel == "." || filepath.IsAbs(cleanRel) {
		return "", errors.New("workspace context path must be relative")
	}

	parts := strings.Split(cleanRel, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", errors.New("workspace context path must stay inside the workspace")
		}
	}

	return filepath.ToSlash(cleanRel), nil
}

func contextPathIsDirectory(absRoot string, relPath string) (bool, error) {
	target := absRoot
	if relPath != "." {
		target = filepath.Join(absRoot, filepath.FromSlash(relPath))
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false, err
	}
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return false, err
	}

	info, err := os.Lstat(absTarget)
	if err != nil {
		return false, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return false, errors.New("workspace context cannot follow symlinks")
	}

	evalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return false, err
	}
	evalTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return false, err
	}
	if err := ensureInsideRoot(evalRoot, evalTarget); err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

func pathIsInsideContextRoot(relPath string, rootRelPath string) bool {
	if rootRelPath == "." {
		return true
	}
	return relPath == rootRelPath || strings.HasPrefix(relPath, rootRelPath+"/")
}

func isContextCandidate(relPath string) bool {
	ext := strings.ToLower(filepath.Ext(relPath))
	if ext == "" {
		return true
	}

	switch ext {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".css", ".html", ".json", ".yaml", ".yml", ".md", ".sql",
		".txt", ".rtf", ".csv", ".pdf", ".docx", ".toml", ".xml", ".py", ".java", ".cs", ".cpp", ".c",
		".h", ".hpp", ".rs", ".php", ".rb", ".sh", ".ps1", ".env", ".ini", ".conf":
		return true
	default:
		return false
	}
}
