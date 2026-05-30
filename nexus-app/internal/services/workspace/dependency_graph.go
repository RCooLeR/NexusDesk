package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	defaultDependencyGraphMaxFiles = 160
	defaultDependencyGraphMaxEdges = 400
)

var (
	jsImportPattern        = regexp.MustCompile(`^\s*(?:import|export)\s+(?:.+?\s+from\s+)?["']([^"']+)["']`)
	jsRequirePattern       = regexp.MustCompile(`\brequire\s*\(\s*["']([^"']+)["']\s*\)`)
	jsDynamicImportPattern = regexp.MustCompile(`\bimport\s*\(\s*["']([^"']+)["']\s*\)`)
	pythonFromPattern      = regexp.MustCompile(`^\s*from\s+([A-Za-z0-9_\.]+)\s+import\s+`)
	pythonImportPattern    = regexp.MustCompile(`^\s*import\s+(.+)$`)
	cssImportPattern       = regexp.MustCompile(`^\s*@import\s+(?:url\(\s*)?["']?([^"')\s]+)`)
	goModulePattern        = regexp.MustCompile(`(?m)^\s*module\s+(\S+)\s*$`)
)

type DependencyGraphOptions struct {
	RelPath  string
	MaxFiles int
	MaxEdges int
}

type SourceFileOptions struct {
	RelPath  string
	MaxFiles int
}

type SourceFileList struct {
	RootRelPath  string
	Files        []string
	FilesSkipped int
	Truncated    bool
	Message      string
}

type DependencyGraph struct {
	RootRelPath  string
	FilesScanned int
	FilesSkipped int
	Edges        []DependencyEdge
	Truncated    bool
	Message      string
}

type DependencyEdge struct {
	From     string
	To       string
	Spec     string
	Kind     string
	Line     int
	Resolved bool
}

func (s *Service) DependencyGraph(root string, options DependencyGraphOptions) (DependencyGraph, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return DependencyGraph{}, err
	}
	cleanRelPath, err := cleanRel(options.RelPath)
	if err != nil {
		return DependencyGraph{}, err
	}
	maxFiles := options.MaxFiles
	if maxFiles <= 0 || maxFiles > defaultDependencyGraphMaxFiles {
		maxFiles = defaultDependencyGraphMaxFiles
	}
	maxEdges := options.MaxEdges
	if maxEdges <= 0 || maxEdges > defaultDependencyGraphMaxEdges {
		maxEdges = defaultDependencyGraphMaxEdges
	}
	modulePath := readGoModulePath(absRoot)
	graph := DependencyGraph{RootRelPath: firstNonEmptyDependencyGraphString(cleanRelPath, ".")}
	relPaths, skipped, truncated, err := s.dependencyGraphFiles(absRoot, cleanRelPath, maxFiles)
	if err != nil {
		return DependencyGraph{}, err
	}
	graph.FilesSkipped += skipped
	graph.Truncated = truncated
	for _, relPath := range relPaths {
		read, err := s.ReadTextFile(absRoot, relPath)
		if err != nil {
			graph.FilesSkipped++
			continue
		}
		graph.FilesScanned++
		edges := parseDependencyEdges(absRoot, modulePath, read.RelPath, read.Content)
		for _, edge := range edges {
			graph.Edges = append(graph.Edges, edge)
			if len(graph.Edges) >= maxEdges {
				graph.Truncated = true
				break
			}
		}
		if len(graph.Edges) >= maxEdges {
			break
		}
	}
	sort.SliceStable(graph.Edges, func(left int, right int) bool {
		if graph.Edges[left].From == graph.Edges[right].From {
			if graph.Edges[left].Line == graph.Edges[right].Line {
				return graph.Edges[left].Spec < graph.Edges[right].Spec
			}
			return graph.Edges[left].Line < graph.Edges[right].Line
		}
		return compareSearchPaths(graph.Edges[left].From, graph.Edges[right].From)
	})
	graph.Message = fmt.Sprintf("Dependency graph scanned %d file(s), found %d edge(s), skipped %d item(s).", graph.FilesScanned, len(graph.Edges), graph.FilesSkipped)
	if graph.Truncated {
		graph.Message += " Results truncated by safety caps."
	}
	return graph, nil
}

func (s *Service) SourceFiles(root string, options SourceFileOptions) (SourceFileList, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return SourceFileList{}, err
	}
	cleanRelPath, err := cleanRel(options.RelPath)
	if err != nil {
		return SourceFileList{}, err
	}
	maxFiles := options.MaxFiles
	if maxFiles <= 0 || maxFiles > defaultDependencyGraphMaxFiles {
		maxFiles = defaultDependencyGraphMaxFiles
	}
	relPaths, skipped, truncated, err := s.dependencyGraphFiles(absRoot, cleanRelPath, maxFiles)
	if err != nil {
		return SourceFileList{}, err
	}
	list := SourceFileList{
		RootRelPath:  firstNonEmptyDependencyGraphString(cleanRelPath, "."),
		Files:        relPaths,
		FilesSkipped: skipped,
		Truncated:    truncated,
	}
	list.Message = fmt.Sprintf("Source file scan found %d file(s), skipped %d item(s).", len(list.Files), list.FilesSkipped)
	if list.Truncated {
		list.Message += " Results truncated by safety caps."
	}
	return list, nil
}

func (s *Service) dependencyGraphFiles(absRoot string, cleanRelPath string, maxFiles int) ([]string, int, bool, error) {
	target := absRoot
	if cleanRelPath != "" {
		target = filepath.Join(absRoot, filepath.FromSlash(cleanRelPath))
	}
	info, err := os.Lstat(target)
	if err != nil {
		return nil, 0, false, err
	}
	resolvedTarget, err := ensureResolvedReadPathInsideRoot(absRoot, target)
	if err != nil {
		return nil, 0, false, err
	}
	walkRelRoot := absRoot
	if resolvedRoot, err := filepath.EvalSymlinks(absRoot); err == nil {
		walkRelRoot = resolvedRoot
	}
	if !info.IsDir() {
		if !isDependencyGraphFile(cleanRelPath) {
			return nil, 1, false, nil
		}
		return []string{cleanRelPath}, 0, false, nil
	}
	relPaths := []string{}
	skipped := 0
	truncated := false
	err = filepath.WalkDir(resolvedTarget, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			skipped++
			return nil
		}
		if path == resolvedTarget {
			return nil
		}
		relPath, err := filepath.Rel(walkRelRoot, path)
		if err != nil {
			skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if shouldSkipSearchPath(relPath, entry) {
			skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depthOf(relPath) > defaultSearchMaxDepth {
			skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !isDependencyGraphFile(relPath) {
			return nil
		}
		relPaths = append(relPaths, relPath)
		if len(relPaths) >= maxFiles {
			truncated = true
			return fs.SkipAll
		}
		return nil
	})
	sort.SliceStable(relPaths, func(left int, right int) bool {
		return compareSearchPaths(relPaths[left], relPaths[right])
	})
	return relPaths, skipped, truncated, err
}

func parseDependencyEdges(absRoot string, modulePath string, relPath string, content string) []DependencyEdge {
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(relPath)), ".")
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"), "\n")
	edges := []DependencyEdge{}
	inGoImportBlock := false
	for index, line := range lines {
		lineNumber := index + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		switch extension {
		case "go":
			if strings.HasPrefix(trimmed, "import (") {
				inGoImportBlock = true
				continue
			}
			if inGoImportBlock {
				if strings.HasPrefix(trimmed, ")") {
					inGoImportBlock = false
					continue
				}
				addDependencyEdge(&edges, absRoot, modulePath, relPath, extractGoImportSpec(trimmed), "go-import", lineNumber)
				continue
			}
			if strings.HasPrefix(trimmed, "import ") {
				addDependencyEdge(&edges, absRoot, modulePath, relPath, extractGoImportSpec(strings.TrimPrefix(trimmed, "import ")), "go-import", lineNumber)
			}
		case "js", "jsx", "ts", "tsx", "mjs", "cjs":
			if match := jsImportPattern.FindStringSubmatch(line); match != nil {
				addDependencyEdge(&edges, absRoot, modulePath, relPath, match[1], "js-import", lineNumber)
			}
			for _, match := range jsRequirePattern.FindAllStringSubmatch(line, -1) {
				addDependencyEdge(&edges, absRoot, modulePath, relPath, match[1], "js-require", lineNumber)
			}
			for _, match := range jsDynamicImportPattern.FindAllStringSubmatch(line, -1) {
				addDependencyEdge(&edges, absRoot, modulePath, relPath, match[1], "js-dynamic-import", lineNumber)
			}
		case "py":
			if match := pythonFromPattern.FindStringSubmatch(line); match != nil {
				addDependencyEdge(&edges, absRoot, modulePath, relPath, match[1], "python-from", lineNumber)
				continue
			}
			if match := pythonImportPattern.FindStringSubmatch(line); match != nil {
				for _, spec := range strings.Split(match[1], ",") {
					spec = strings.TrimSpace(strings.Split(spec, " as ")[0])
					addDependencyEdge(&edges, absRoot, modulePath, relPath, spec, "python-import", lineNumber)
				}
			}
		case "css", "scss", "sass":
			if match := cssImportPattern.FindStringSubmatch(line); match != nil {
				addDependencyEdge(&edges, absRoot, modulePath, relPath, match[1], "css-import", lineNumber)
			}
		}
	}
	return edges
}

func addDependencyEdge(edges *[]DependencyEdge, absRoot string, modulePath string, from string, spec string, kind string, line int) {
	spec = strings.TrimSpace(strings.Trim(spec, `"'`))
	if spec == "" {
		return
	}
	target, resolved := resolveDependencyTarget(absRoot, modulePath, from, spec)
	*edges = append(*edges, DependencyEdge{
		From:     filepath.ToSlash(from),
		To:       target,
		Spec:     spec,
		Kind:     kind,
		Line:     line,
		Resolved: resolved,
	})
}

func extractGoImportSpec(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "_ ") || strings.HasPrefix(value, ". ") {
		value = strings.TrimSpace(value[1:])
	}
	fields := strings.Fields(value)
	if len(fields) > 1 {
		value = fields[len(fields)-1]
	}
	return strings.Trim(value, "`\"")
}

func resolveDependencyTarget(absRoot string, modulePath string, from string, spec string) (string, bool) {
	if strings.HasPrefix(spec, ".") {
		if strings.EqualFold(filepath.Ext(from), ".py") {
			return resolveLocalDependencyTarget(absRoot, pythonRelativeImportBase(from, spec), dependencyExtensionsFor(from))
		}
		return resolveLocalDependencyTarget(absRoot, filepath.ToSlash(filepath.Join(filepath.Dir(from), spec)), dependencyExtensionsFor(from))
	}
	if modulePath != "" && (spec == modulePath || strings.HasPrefix(spec, modulePath+"/")) {
		rel := strings.TrimPrefix(strings.TrimPrefix(spec, modulePath), "/")
		if rel == "" {
			return ".", true
		}
		if target, ok := resolveLocalDependencyTarget(absRoot, filepath.ToSlash(rel), []string{".go"}); ok {
			return target, true
		}
		return filepath.ToSlash(rel), false
	}
	return "external:" + spec, false
}

func pythonRelativeImportBase(from string, spec string) string {
	dots := 0
	for dots < len(spec) && spec[dots] == '.' {
		dots++
	}
	rest := strings.Trim(spec[dots:], ".")
	baseDir := filepath.Dir(from)
	for index := 1; index < dots; index++ {
		baseDir = filepath.Dir(baseDir)
	}
	if rest == "" {
		return filepath.ToSlash(baseDir)
	}
	return filepath.ToSlash(filepath.Join(baseDir, strings.ReplaceAll(rest, ".", "/")))
}

func resolveLocalDependencyTarget(absRoot string, base string, extensions []string) (string, bool) {
	base = strings.Trim(filepath.ToSlash(filepath.Clean(base)), "/")
	cleanBase, err := cleanRel(base)
	if err != nil {
		return base, false
	}
	base = cleanBase
	if base == "." || base == "" {
		base = ""
	}
	candidates := []string{base}
	if filepath.Ext(base) == "" {
		for _, extension := range extensions {
			candidates = append(candidates, base+extension)
		}
		for _, extension := range extensions {
			candidates = append(candidates, filepath.ToSlash(filepath.Join(base, "index"+extension)))
			candidates = append(candidates, filepath.ToSlash(filepath.Join(base, "__init__"+extension)))
		}
	}
	for _, candidate := range candidates {
		candidate = strings.Trim(filepath.ToSlash(candidate), "/")
		if candidate == "" {
			continue
		}
		target := filepath.Join(absRoot, filepath.FromSlash(candidate))
		info, err := os.Stat(target)
		if err == nil {
			if info.IsDir() {
				for _, extension := range extensions {
					indexCandidate := filepath.ToSlash(filepath.Join(candidate, "index"+extension))
					if _, err := os.Stat(filepath.Join(absRoot, filepath.FromSlash(indexCandidate))); err == nil {
						return indexCandidate, true
					}
				}
				return candidate, true
			}
			return candidate, true
		}
	}
	return base, false
}

func dependencyExtensionsFor(relPath string) []string {
	switch strings.TrimPrefix(strings.ToLower(filepath.Ext(relPath)), ".") {
	case "go":
		return []string{".go"}
	case "py":
		return []string{".py"}
	case "css":
		return []string{".css"}
	case "scss":
		return []string{".scss", ".css"}
	case "sass":
		return []string{".sass", ".scss", ".css"}
	default:
		return []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".json"}
	}
}

func isDependencyGraphFile(relPath string) bool {
	switch strings.TrimPrefix(strings.ToLower(filepath.Ext(relPath)), ".") {
	case "go", "js", "jsx", "ts", "tsx", "mjs", "cjs", "py", "css", "scss", "sass":
		return true
	default:
		return false
	}
}

func readGoModulePath(absRoot string) string {
	content, err := os.ReadFile(filepath.Join(absRoot, "go.mod"))
	if err != nil {
		return ""
	}
	match := goModulePattern.FindStringSubmatch(string(content))
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func firstNonEmptyDependencyGraphString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
