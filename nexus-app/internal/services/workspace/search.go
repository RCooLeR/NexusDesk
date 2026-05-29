package workspace

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	defaultSearchMaxResults = 100
	defaultSearchMaxDepth   = 20
	defaultSearchPerFileMax = 10
	defaultSearchMaxTime    = 2 * time.Second
	searchContentMaxBytes   = 8 * 1024 * 1024
	searchBinarySampleBytes = 8192
	searchMaxLineBytes      = 256 * 1024
	searchSnippetMaxRunes   = 160
)

type SearchOptions struct {
	MaxResults     int
	Regex          bool
	MaxDuration    time.Duration
	ResultCallback func([]SearchResult)
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
	results, _, err := s.SearchWithMetadata(root, query, options)
	return results, err
}

func (s *Service) SearchWithMetadata(root string, query string, options SearchOptions) ([]SearchResult, SearchMetadata, error) {
	return s.SearchWithMetadataContext(context.Background(), root, query, options)
}

func (s *Service) SearchWithMetadataContext(ctx context.Context, root string, query string, options SearchOptions) ([]SearchResult, SearchMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query = strings.TrimSpace(query)
	started := nowUTC()
	metadata := SearchMetadata{
		Version:     searchMetadataVersion,
		Query:       query,
		Regex:       options.Regex,
		GeneratedAt: started,
	}
	if query == "" {
		return []SearchResult{}, metadata, nil
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return nil, metadata, err
	}
	maxResults := options.MaxResults
	if maxResults <= 0 {
		maxResults = defaultSearchMaxResults
	}
	maxDuration := options.MaxDuration
	if maxDuration <= 0 {
		maxDuration = defaultSearchMaxTime
	}
	deadline := started.Add(maxDuration)
	metadata.WorkspaceName = filepath.Base(absRoot)
	metadata.MaxResults = maxResults
	matcher, err := newSearchMatcher(query, options.Regex)
	if err != nil {
		return nil, metadata, err
	}

	stats := searchStats{}
	results := []SearchResult{}
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if searchDeadlineExceeded(deadline) {
			stats.TimedOut = true
			return errSearchDeadlineExceeded
		}
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
				stats.DirectoriesSkipped++
				return filepath.SkipDir
			}
			return nil
		}
		if depthOf(relPath) > defaultSearchMaxDepth {
			if entry.IsDir() {
				stats.DirectoriesSkipped++
				return filepath.SkipDir
			}
			return nil
		}

		if matcher.matches(relPath) {
			stats.PathMatches++
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
			stats.FilesScanned++
			if err := ctx.Err(); err != nil {
				return err
			}
			if searchDeadlineExceeded(deadline) {
				stats.TimedOut = true
				return errSearchDeadlineExceeded
			}
			matches := searchFileContentFast(absRoot, relPath, matcher, remaining)
			if len(matches) > 0 {
				stats.FilesWithContentMatches++
				stats.ContentMatches += len(matches)
			}
			results = append(results, matches...)
			if len(matches) > 0 && options.ResultCallback != nil {
				options.ResultCallback(append([]SearchResult(nil), results...))
			}
		}
		return nil
	})
	if errors.Is(err, errSearchDeadlineExceeded) {
		err = nil
	} else if err != nil {
		return nil, metadata, err
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
	stats.DurationMs = nowUTC().Sub(started).Milliseconds()
	stats.Truncated = len(results) >= maxResults || stats.TimedOut
	metadata = metadata.withResults(results, stats)
	return results, metadata, nil
}

var errSearchDeadlineExceeded = errors.New("search deadline exceeded")

func searchDeadlineExceeded(deadline time.Time) bool {
	return !deadline.IsZero() && nowUTC().After(deadline)
}

func searchFileContentFast(root string, relPath string, matcher searchMatcher, maxResults int) []SearchResult {
	if maxResults <= 0 {
		return nil
	}
	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	info, err := os.Lstat(absPath)
	if err != nil || info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return nil
	}
	if isKnownBinarySearchPath(relPath) {
		return nil
	}
	if info.Size() > searchContentMaxBytes {
		return nil
	}
	return searchFileContentStreaming(absPath, relPath, info.Size(), matcher, maxResults)
}

func searchFileContentStreaming(absPath string, relPath string, size int64, matcher searchMatcher, maxResults int) []SearchResult {
	file, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer file.Close()
	sampleLimit := searchBinarySampleBytes
	if size < int64(sampleLimit) {
		sampleLimit = int(size)
	}
	sample := make([]byte, sampleLimit)
	read, err := io.ReadFull(file, sample)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil
	}
	sample = sample[:read]
	if len(sample) == 0 || !isSearchableContent(relPath, sample) {
		return nil
	}
	if looksLikeUTF16LE(sample) || looksLikeUTF16BE(sample) {
		if size > writeContentMaxBytes {
			return nil
		}
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil
		}
		text, _, err := decodeText(content)
		if err != nil || strings.TrimSpace(text) == "" {
			return nil
		}
		return searchTextLines(relPath, text, matcher, maxResults)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil
	}
	results := make([]SearchResult, 0, min(maxResults, defaultSearchPerFileMax))
	reader := bufio.NewReaderSize(file, 32*1024)
	lineNumber := 0
	for len(results) < maxResults && len(results) < defaultSearchPerFileMax {
		line, err := reader.ReadString('\n')
		if line != "" {
			lineNumber++
			if len(line) > searchMaxLineBytes {
				line = line[:searchMaxLineBytes]
			}
			if result, ok := searchLineResult(relPath, line, lineNumber, matcher); ok {
				results = append(results, result)
			}
		}
		if err != nil {
			break
		}
	}
	return results
}

func searchTextLines(relPath string, text string, matcher searchMatcher, maxResults int) []SearchResult {
	lines := strings.Split(text, "\n")
	results := make([]SearchResult, 0, min(maxResults, defaultSearchPerFileMax))
	for index, line := range lines {
		if result, ok := searchLineResult(relPath, line, index+1, matcher); ok {
			results = append(results, result)
		}
		if len(results) >= maxResults || len(results) >= defaultSearchPerFileMax {
			break
		}
	}
	return results
}

func searchLineResult(relPath string, line string, lineNumber int, matcher searchMatcher) (SearchResult, bool) {
	line = strings.TrimRight(line, "\r\n")
	matchStart, isMatch := matcher.match(line)
	if !isMatch {
		return SearchResult{}, false
	}
	return SearchResult{
		RelPath:   relPath,
		Name:      filepath.Base(filepath.FromSlash(relPath)),
		Kind:      "file",
		MediaType: mediaType(relPath),
		MatchType: matcher.matchType("content"),
		Line:      lineNumber,
		Snippet:   trimSearchSnippet(line, matchStart),
	}, true
}

func isSearchableContent(relPath string, content []byte) bool {
	if isKnownBinarySearchPath(relPath) {
		return false
	}
	extension := strings.ToLower(filepath.Ext(relPath))
	if looksLikeUTF16LE(content) || looksLikeUTF16BE(content) {
		return true
	}
	if looksBinary(content) {
		return false
	}
	if isTextLikePath(relPath) || extension == ".csv" || extension == ".tsv" {
		return true
	}
	return true
}

func isKnownBinarySearchPath(relPath string) bool {
	extension := strings.ToLower(filepath.Ext(relPath))
	if isImageExtension(extension) || isPDFExtension(extension) || isDocumentExtension(extension) || extension == ".xlsx" {
		return true
	}
	switch extension {
	case ".zip", ".gz", ".tgz", ".rar", ".7z", ".tar", ".exe", ".dll", ".so", ".dylib", ".bin", ".wasm", ".class", ".jar":
		return true
	default:
		return false
	}
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
