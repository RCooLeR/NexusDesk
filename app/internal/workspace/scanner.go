package workspace

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultMaxDepth   = 4
	defaultMaxEntries = 400
)

type ScanOptions struct {
	MaxDepth   int
	MaxEntries int
}

type FileNode struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	RelPath  string `json:"relPath"`
	Kind     string `json:"kind"`
	FileType string `json:"fileType"`
	Depth    int    `json:"depth"`
	Meta     string `json:"meta"`
}

type WorkspaceSnapshot struct {
	Root      string     `json:"root"`
	Name      string     `json:"name"`
	Nodes     []FileNode `json:"nodes"`
	Truncated bool       `json:"truncated"`
}

func Scan(root string, options ScanOptions) (WorkspaceSnapshot, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

	maxDepth := options.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	maxEntries := options.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}

	snapshot := WorkspaceSnapshot{
		Root: absRoot,
		Name: filepath.Base(absRoot),
	}

	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if path == absRoot {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath = filepath.ToSlash(relPath)
		depth := strings.Count(relPath, "/") + 1

		if shouldIgnore(relPath, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if depth > maxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if len(snapshot.Nodes) >= maxEntries {
			snapshot.Truncated = true
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}

		kind := "file"
		if entry.IsDir() {
			kind = "directory"
		}

		snapshot.Nodes = append(snapshot.Nodes, FileNode{
			Name:     entry.Name(),
			Path:     path,
			RelPath:  relPath,
			Kind:     kind,
			FileType: detectFileType(entry),
			Depth:    depth,
			Meta:     describeEntry(entry, info.Size()),
		})

		return nil
	})
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

	sort.SliceStable(snapshot.Nodes, func(i, j int) bool {
		left := snapshot.Nodes[i]
		right := snapshot.Nodes[j]
		if left.Depth != right.Depth {
			return left.Depth < right.Depth
		}
		if left.Kind != right.Kind {
			return left.Kind == "directory"
		}
		return strings.ToLower(left.RelPath) < strings.ToLower(right.RelPath)
	})

	return snapshot, nil
}

func shouldIgnore(relPath string, entry fs.DirEntry) bool {
	name := entry.Name()
	lowerName := strings.ToLower(name)

	if entry.Type()&fs.ModeSymlink != 0 {
		return true
	}

	if ignoredNames[lowerName] {
		return true
	}

	for _, part := range strings.Split(strings.ToLower(relPath), "/") {
		if ignoredNames[part] {
			return true
		}
	}

	return false
}

func detectFileType(entry fs.DirEntry) string {
	return detectFileTypeName(entry.Name(), entry.IsDir())
}

func detectFileTypeName(name string, isDir bool) string {
	if isDir {
		return "folder"
	}

	switch strings.ToLower(filepath.Ext(name)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".css", ".html", ".json", ".yaml", ".yml", ".md", ".sql":
		return "code"
	case ".csv", ".xlsx", ".xls", ".parquet":
		return "data"
	case ".pdf", ".doc", ".docx", ".txt", ".rtf":
		return "document"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico":
		return "image"
	default:
		return "file"
	}
}

func describeEntry(entry fs.DirEntry, size int64) string {
	if entry.IsDir() {
		return "Folder"
	}
	return formatBytes(size)
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return "File"
	}

	value := float64(size)
	units := []string{"KB", "MB", "GB"}
	for _, suffix := range units {
		value = value / unit
		if value < unit {
			return trimFloat(value) + " " + suffix
		}
	}
	return trimFloat(value) + " TB"
}

func trimFloat(value float64) string {
	text := strconvFormatFloat(value)
	text = strings.TrimRight(text, "0")
	return strings.TrimRight(text, ".")
}

func strconvFormatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}

var ignoredNames = map[string]bool{
	".git":            true,
	".idea":           true,
	".vscode":         true,
	"node_modules":    true,
	"dist":            true,
	"build":           true,
	".svelte-kit":     true,
	".vite":           true,
	"coverage":        true,
	"logs":            true,
	"workspace-cache": true,
}
