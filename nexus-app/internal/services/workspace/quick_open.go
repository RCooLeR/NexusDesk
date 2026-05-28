package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultQuickOpenLimit      = 50
	defaultQuickOpenMaxEntries = 5000
)

type QuickOpenFile struct {
	RelPath string
	Name    string
}

func (s *Service) QuickOpenFiles(root string, query string, limit int) ([]QuickOpenFile, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultQuickOpenLimit
	}
	query = strings.ToLower(strings.Join(strings.Fields(query), " "))
	candidates := []quickOpenCandidate{}
	scanned := 0
	walkErr := filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == absRoot {
			return nil
		}
		scanned++
		if scanned > defaultQuickOpenMaxEntries {
			return errors.New("quick open entry cap reached")
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if isIgnoredName(entry.Name()) || isInternalMetadataPath(rel) {
				return filepath.SkipDir
			}
			info, err := entry.Info()
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		score, ok := quickOpenScore(rel, query)
		if !ok {
			return nil
		}
		candidates = append(candidates, quickOpenCandidate{file: QuickOpenFile{RelPath: rel, Name: entry.Name()}, score: score})
		return nil
	})
	if walkErr != nil && len(candidates) == 0 {
		return nil, walkErr
	}
	sort.Slice(candidates, func(left int, right int) bool {
		if candidates[left].score != candidates[right].score {
			return candidates[left].score < candidates[right].score
		}
		return strings.ToLower(candidates[left].file.RelPath) < strings.ToLower(candidates[right].file.RelPath)
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	files := make([]QuickOpenFile, 0, len(candidates))
	for _, candidate := range candidates {
		files = append(files, candidate.file)
	}
	return files, nil
}

type quickOpenCandidate struct {
	file  QuickOpenFile
	score int
}

func quickOpenScore(relPath string, query string) (int, bool) {
	normalizedPath := strings.ToLower(filepath.ToSlash(relPath))
	name := strings.ToLower(filepath.Base(normalizedPath))
	if query == "" {
		return 4, true
	}
	switch {
	case name == query:
		return 0, true
	case strings.HasPrefix(name, query):
		return 1, true
	case strings.Contains(name, query):
		return 2, true
	case strings.Contains(normalizedPath, query):
		return 3, true
	default:
		return 0, false
	}
}
