package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	searchMetadataVersion        = 1
	searchMetadataRelPath        = ".nexusdesk/search/index-metadata.json"
	searchMetadataRecoveryDir    = ".nexusdesk/search/recovery"
	searchMetadataMaxStoredItems = 200
)

var nowUTC = func() time.Time {
	return time.Now().UTC()
}

type SearchMetadata struct {
	Version                 int                    `json:"version"`
	WorkspaceName           string                 `json:"workspaceName,omitempty"`
	Query                   string                 `json:"query"`
	Regex                   bool                   `json:"regex"`
	MaxResults              int                    `json:"maxResults,omitempty"`
	ResultCount             int                    `json:"resultCount"`
	PathMatches             int                    `json:"pathMatches"`
	ContentMatches          int                    `json:"contentMatches"`
	FilesScanned            int                    `json:"filesScanned"`
	FilesWithContentMatches int                    `json:"filesWithContentMatches"`
	DirectoriesSkipped      int                    `json:"directoriesSkipped"`
	Truncated               bool                   `json:"truncated"`
	GeneratedAt             time.Time              `json:"generatedAt"`
	Results                 []SearchMetadataResult `json:"results,omitempty"`
}

type SearchMetadataResult struct {
	RelPath   string `json:"relPath"`
	Kind      string `json:"kind"`
	MediaType string `json:"mediaType"`
	MatchType string `json:"matchType"`
	Line      int    `json:"line,omitempty"`
}

type SearchMetadataExport struct {
	RelPath          string `json:"relPath"`
	AbsPath          string `json:"absPath"`
	Recovered        bool   `json:"recovered"`
	RecoveredRelPath string `json:"recoveredRelPath,omitempty"`
	RecoveredAbsPath string `json:"recoveredAbsPath,omitempty"`
}

type searchStats struct {
	PathMatches             int
	ContentMatches          int
	FilesScanned            int
	FilesWithContentMatches int
	DirectoriesSkipped      int
	Truncated               bool
}

func (m SearchMetadata) withResults(results []SearchResult, stats searchStats) SearchMetadata {
	m.ResultCount = len(results)
	m.PathMatches = stats.PathMatches
	m.ContentMatches = stats.ContentMatches
	m.FilesScanned = stats.FilesScanned
	m.FilesWithContentMatches = stats.FilesWithContentMatches
	m.DirectoriesSkipped = stats.DirectoriesSkipped
	m.Truncated = stats.Truncated
	m.Results = searchMetadataResults(results)
	return m
}

func searchMetadataResults(results []SearchResult) []SearchMetadataResult {
	count := len(results)
	if count > searchMetadataMaxStoredItems {
		count = searchMetadataMaxStoredItems
	}
	metadataResults := make([]SearchMetadataResult, 0, count)
	for _, result := range results[:count] {
		metadataResults = append(metadataResults, SearchMetadataResult{
			RelPath:   result.RelPath,
			Kind:      result.Kind,
			MediaType: result.MediaType,
			MatchType: result.MatchType,
			Line:      result.Line,
		})
	}
	return metadataResults
}

func (s *Service) SearchMetadataPath(root string) (string, string, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", err
	}
	absPath := filepath.Join(absRoot, filepath.FromSlash(searchMetadataRelPath))
	if !isInside(absRoot, absPath) {
		return "", "", errors.New("search metadata path must stay inside the workspace")
	}
	return searchMetadataRelPath, absPath, nil
}

func (s *Service) WriteSearchMetadata(root string, metadata SearchMetadata) (SearchMetadataExport, error) {
	recovery, err := s.RecoverSearchMetadata(root)
	if err != nil {
		return SearchMetadataExport{}, err
	}
	relPath, absPath, err := s.SearchMetadataPath(root)
	if err != nil {
		return SearchMetadataExport{}, err
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return SearchMetadataExport{}, err
	}
	if info, err := os.Lstat(absPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return SearchMetadataExport{}, errors.New("search metadata file cannot be a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return SearchMetadataExport{}, err
	}
	if err := ensureWriteParentInsideRoot(absRoot, absPath); err != nil {
		return SearchMetadataExport{}, err
	}
	if metadata.Version == 0 {
		metadata.Version = searchMetadataVersion
	}
	if metadata.GeneratedAt.IsZero() {
		metadata.GeneratedAt = nowUTC()
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return SearchMetadataExport{}, err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return SearchMetadataExport{}, err
	}
	tempPath := absPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return SearchMetadataExport{}, err
	}
	if err := os.Rename(tempPath, absPath); err != nil {
		_ = os.Remove(tempPath)
		return SearchMetadataExport{}, err
	}
	return SearchMetadataExport{
		RelPath:          relPath,
		AbsPath:          absPath,
		Recovered:        recovery.Recovered,
		RecoveredRelPath: recovery.RecoveredRelPath,
		RecoveredAbsPath: recovery.RecoveredAbsPath,
	}, nil
}

func (s *Service) ReadSearchMetadata(root string) (SearchMetadata, error) {
	_, absPath, err := s.SearchMetadataPath(root)
	if err != nil {
		return SearchMetadata{}, err
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		return SearchMetadata{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return SearchMetadata{}, errors.New("search metadata file cannot be a symlink")
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return SearchMetadata{}, err
	}
	var metadata SearchMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return SearchMetadata{}, err
	}
	if metadata.Version == 0 {
		return SearchMetadata{}, errors.New("search metadata version is missing")
	}
	return metadata, nil
}

func (s *Service) RecoverSearchMetadata(root string) (SearchMetadataExport, error) {
	relPath, absPath, err := s.SearchMetadataPath(root)
	if err != nil {
		return SearchMetadataExport{}, err
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return SearchMetadataExport{RelPath: relPath, AbsPath: absPath}, nil
		}
		return SearchMetadataExport{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return SearchMetadataExport{}, errors.New("search metadata file cannot be a symlink")
	}
	if _, err := s.ReadSearchMetadata(root); err == nil {
		return SearchMetadataExport{RelPath: relPath, AbsPath: absPath}, nil
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return SearchMetadataExport{}, err
	}
	recoveryRelPath := filepath.ToSlash(filepath.Join(
		searchMetadataRecoveryDir,
		fmt.Sprintf("index-metadata-%s-%d.corrupt.json", nowUTC().Format("20060102T150405Z"), nowUTC().UnixNano()),
	))
	recoveryAbsPath := filepath.Join(absRoot, filepath.FromSlash(recoveryRelPath))
	if err := ensureWriteParentInsideRoot(absRoot, recoveryAbsPath); err != nil {
		return SearchMetadataExport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(recoveryAbsPath), 0o755); err != nil {
		return SearchMetadataExport{}, err
	}
	if err := os.Rename(absPath, recoveryAbsPath); err != nil {
		return SearchMetadataExport{}, err
	}
	return SearchMetadataExport{
		RelPath:          relPath,
		AbsPath:          absPath,
		Recovered:        true,
		RecoveredRelPath: recoveryRelPath,
		RecoveredAbsPath: recoveryAbsPath,
	}, nil
}
