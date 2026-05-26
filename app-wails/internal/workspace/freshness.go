package workspace

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

type FileFingerprint struct {
	RelPath    string `json:"relPath"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

type FileChange struct {
	RelPath string `json:"relPath"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type FreshnessStatus struct {
	Changed        []FileChange `json:"changed"`
	StaleArtifacts []string     `json:"staleArtifacts"`
	StaleDatasets  []string     `json:"staleDatasets"`
	Message        string       `json:"message"`
}

func SnapshotFingerprints(root string) (map[string]FileFingerprint, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	items := map[string]FileFingerprint{}
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if shouldIgnoreFreshnessPath(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		items[rel] = FileFingerprint{
			RelPath:    rel,
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339Nano),
		}
		return nil
	})
	return items, err
}

func CompareFingerprints(previous map[string]FileFingerprint, current map[string]FileFingerprint) []FileChange {
	changes := []FileChange{}
	for relPath, currentItem := range current {
		previousItem, ok := previous[relPath]
		if !ok {
			changes = append(changes, FileChange{RelPath: relPath, Kind: "created", Message: relPath + " was created."})
			continue
		}
		if previousItem.Size != currentItem.Size || previousItem.ModifiedAt != currentItem.ModifiedAt {
			changes = append(changes, FileChange{RelPath: relPath, Kind: "modified", Message: relPath + " changed on disk."})
		}
	}
	for relPath := range previous {
		if _, ok := current[relPath]; !ok {
			changes = append(changes, FileChange{RelPath: relPath, Kind: "deleted", Message: relPath + " was deleted."})
		}
	}
	return changes
}

func shouldIgnoreFreshnessPath(relPath string) bool {
	normalized := strings.ToLower(filepath.ToSlash(relPath))
	return normalized == ".git" ||
		strings.HasPrefix(normalized, ".git/") ||
		strings.HasPrefix(normalized, ".nexusdesk/tool-runs/") ||
		strings.HasPrefix(normalized, ".nexusdesk/metadata/")
}
