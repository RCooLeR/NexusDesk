package workspace

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	defaultSearchMaxResults = 100
	defaultSearchMaxDepth   = 20
	defaultSearchPerFileMax = 10
	searchPreviewMaxBytes   = 64 * 1024
	searchSnippetMaxRunes   = 160
)

type SearchOptions struct {
	MaxResults int
	Regex      bool
}

type SearchResult struct {
	RelPath   string
	Name      string
	Kind      string
	MediaType string
	MatchType string
	Line      int
	Snippet   string
}

func (s *Service) Search(root string, query string, options SearchOptions) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	maxResults := options.MaxResults
	if maxResults <= 0 {
		maxResults = defaultSearchMaxResults
	}
	matcher, err := newSearchMatcher(query, options.Regex)
	if err != nil {
		return nil, err
	}

	searchService := *s
	searchService.previewByteLimit = searchPreviewMaxBytes
	results := []SearchResult{}
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
		if shouldSkipSearchPath(relPath, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depthOf(relPath) > defaultSearchMaxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if matcher.matches(relPath) {
			results = append(results, SearchResult{
				RelPath:   relPath,
				Name:      entry.Name(),
				Kind:      searchEntryKind(entry),
				MediaType: mediaType(relPath),
				MatchType: matcher.matchType("path"),
				Snippet:   relPath,
			})
		}
		if len(results) >= maxResults {
			return nil
		}
		if !entry.IsDir() {
			remaining := maxResults - len(results)
			results = append(results, searchService.searchFileContent(absRoot, relPath, matcher, remaining)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(left int, right int) bool {
		if results[left].RelPath == results[right].RelPath {
			return results[left].MatchType < results[right].MatchType
		}
		return compareSearchPaths(results[left].RelPath, results[right].RelPath)
	})
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

func (s *Service) searchFileContent(root string, relPath string, matcher searchMatcher, maxResults int) []SearchResult {
	if maxResults <= 0 {
		return nil
	}
	preview, err := s.PreviewFile(root, relPath)
	if err != nil || strings.TrimSpace(preview.Text) == "" {
		return nil
	}
	lines := strings.Split(preview.Text, "\n")
	results := make([]SearchResult, 0, min(maxResults, defaultSearchPerFileMax))
	for index, line := range lines {
		matchStart, isMatch := matcher.match(line)
		if !isMatch {
			continue
		}
		results = append(results, SearchResult{
			RelPath:   preview.RelPath,
			Name:      preview.Name,
			Kind:      string(preview.Kind),
			MediaType: preview.MediaType,
			MatchType: matcher.matchType("content"),
			Line:      index + 1,
			Snippet:   trimSearchSnippet(line, matchStart),
		})
		if len(results) >= maxResults || len(results) >= defaultSearchPerFileMax {
			break
		}
	}
	return results
}

type searchMatcher struct {
	lowerQuery string
	pattern    *regexp.Regexp
	regex      bool
}

func newSearchMatcher(query string, regexMode bool) (searchMatcher, error) {
	if !regexMode {
		return searchMatcher{lowerQuery: strings.ToLower(query)}, nil
	}
	pattern, err := regexp.Compile("(?i)" + query)
	if err != nil {
		return searchMatcher{}, err
	}
	return searchMatcher{pattern: pattern, regex: true}, nil
}

func (m searchMatcher) matches(value string) bool {
	_, ok := m.match(value)
	return ok
}

func (m searchMatcher) match(value string) (int, bool) {
	if m.regex {
		index := m.pattern.FindStringIndex(value)
		if index == nil {
			return 0, false
		}
		return index[0], true
	}
	index := strings.Index(strings.ToLower(value), m.lowerQuery)
	return index, index >= 0
}

func (m searchMatcher) matchType(base string) string {
	if m.regex {
		return base + "-regex"
	}
	return base
}

func trimSearchSnippet(value string, matchStart int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= searchSnippetMaxRunes {
		return value
	}
	if matchStart < 0 || matchStart >= len(value) {
		matchStart = len(value) / 2
	}
	matchRune := 0
	for byteIndex := range value {
		if byteIndex < matchStart {
			matchRune++
		} else {
			break
		}
	}
	windowHalf := (searchSnippetMaxRunes - 3) / 2
	start := matchRune - windowHalf
	if start < 0 {
		start = 0
	}
	end := start + (searchSnippetMaxRunes - 3)
	if end > len(runes) {
		end = len(runes)
	}
	if end-start < (searchSnippetMaxRunes - 3) {
		start = end - (searchSnippetMaxRunes - 3)
		if start < 0 {
			start = 0
		}
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func shouldSkipSearchPath(relPath string, entry fs.DirEntry) bool {
	if isIgnoredName(entry.Name()) {
		return true
	}
	if info, err := entry.Info(); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		return true
	}
	return isInternalMetadataPath(relPath)
}

func searchEntryKind(entry fs.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	return "file"
}

func depthOf(relPath string) int {
	if relPath == "" {
		return 0
	}
	return strings.Count(relPath, "/") + 1
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
