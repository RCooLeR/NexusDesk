package workspace

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultSearchMaxResults = 100
	searchPreviewMaxBytes   = 64 * 1024
)

type SearchOptions struct {
	MaxResults int
}

type SearchResult struct {
	RelPath   string `json:"relPath"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FileType  string `json:"fileType"`
	MatchType string `json:"matchType"`
	Line      int    `json:"line"`
	Snippet   string `json:"snippet"`
}

func Search(root string, query string, options SearchOptions) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	maxResults := options.MaxResults
	if maxResults <= 0 {
		maxResults = defaultSearchMaxResults
	}

	results := []SearchResult{}
	lowerQuery := strings.ToLower(query)
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || path == absRoot {
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
		if shouldIgnore(relPath, entry) || depth > defaultMaxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		node := FileNode{
			Name:     entry.Name(),
			RelPath:  relPath,
			Kind:     "file",
			FileType: detectFileType(entry),
		}
		if entry.IsDir() {
			node.Kind = "directory"
		}

		if strings.Contains(strings.ToLower(relPath), lowerQuery) {
			results = append(results, SearchResult{
				RelPath:   relPath,
				Name:      entry.Name(),
				Kind:      node.Kind,
				FileType:  node.FileType,
				MatchType: "path",
				Snippet:   relPath,
			})
		}

		if !entry.IsDir() && len(results) < maxResults {
			results = append(results, searchFileContent(absRoot, relPath, lowerQuery)...)
		}

		if len(results) >= maxResults {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].RelPath == results[j].RelPath {
			return results[i].MatchType < results[j].MatchType
		}
		return compareSearchPaths(results[i].RelPath, results[j].RelPath)
	})
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

func searchFileContent(root string, relPath string, lowerQuery string) []SearchResult {
	preview, err := Preview(root, relPath, PreviewOptions{MaxBytes: searchPreviewMaxBytes})
	if err != nil {
		return nil
	}

	content := preview.Content
	if preview.Kind == "pdf" {
		content = preview.Text
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	for index, line := range lines {
		if !strings.Contains(strings.ToLower(line), lowerQuery) {
			continue
		}
		return []SearchResult{{
			RelPath:   preview.RelPath,
			Name:      preview.Name,
			Kind:      preview.Kind,
			FileType:  preview.FileType,
			MatchType: "content",
			Line:      index + 1,
			Snippet:   trimSearchSnippet(line),
		}}
	}
	return nil
}

func trimSearchSnippet(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 160 {
		return value
	}
	return value[:157] + "..."
}

func compareSearchPaths(left string, right string) bool {
	leftParts := strings.Split(strings.ToLower(left), "/")
	rightParts := strings.Split(strings.ToLower(right), "/")
	for index := 0; index < len(leftParts) && index < len(rightParts); index++ {
		if leftParts[index] == rightParts[index] {
			continue
		}
		return leftParts[index] < rightParts[index]
	}
	return len(leftParts) < len(rightParts)
}
