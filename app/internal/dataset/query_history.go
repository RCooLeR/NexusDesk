package dataset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const queryStoreName = "queries.json"
const maxSavedQueriesPerDataset = 12

type SavedQuery struct {
	RelPath   string `json:"relPath"`
	Query     string `json:"query"`
	Label     string `json:"label"`
	UpdatedAt string `json:"updatedAt"`
}

func SaveQuery(root string, relPath string, query string, label string) (SavedQuery, error) {
	absRoot, _, cleanRel, err := resolveDatasetPath(root, relPath)
	if err != nil {
		return SavedQuery{}, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		query = ""
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = query
	}
	if label == "" {
		label = "First rows"
	}

	items, err := readSavedQueries(absRoot)
	if err != nil {
		return SavedQuery{}, err
	}

	saved := SavedQuery{
		RelPath:   filepath.ToSlash(cleanRel),
		Query:     query,
		Label:     label,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	key := savedQueryKey(saved.RelPath, saved.Query)
	items[key] = saved
	items = trimSavedQueries(items, saved.RelPath)

	if err := writeSavedQueries(absRoot, items); err != nil {
		return SavedQuery{}, err
	}

	return saved, nil
}

func ListSavedQueries(root string, relPath string) ([]SavedQuery, error) {
	absRoot, _, cleanRel, err := resolveDatasetPath(root, relPath)
	if err != nil {
		return nil, err
	}
	items, err := readSavedQueries(absRoot)
	if err != nil {
		return nil, err
	}

	cleanRelPath := filepath.ToSlash(cleanRel)
	queries := []SavedQuery{}
	for _, item := range items {
		if item.RelPath == cleanRelPath {
			queries = append(queries, item)
		}
	}
	sort.SliceStable(queries, func(i, j int) bool {
		return queries[i].UpdatedAt > queries[j].UpdatedAt
	})
	return queries, nil
}

func readSavedQueries(absRoot string) (map[string]SavedQuery, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(profileDirRelPath), queryStoreName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]SavedQuery{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]SavedQuery{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func writeSavedQueries(absRoot string, items map[string]SavedQuery) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(profileDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, queryStoreName), append(data, '\n'), 0o644)
}

func trimSavedQueries(items map[string]SavedQuery, relPath string) map[string]SavedQuery {
	queries := []SavedQuery{}
	for _, item := range items {
		if item.RelPath == relPath {
			queries = append(queries, item)
		}
	}
	sort.SliceStable(queries, func(i, j int) bool {
		return queries[i].UpdatedAt > queries[j].UpdatedAt
	})
	for index, query := range queries {
		if index < maxSavedQueriesPerDataset {
			continue
		}
		delete(items, savedQueryKey(query.RelPath, query.Query))
	}
	return items
}

func savedQueryKey(relPath string, query string) string {
	return strings.ToLower(relPath) + "\x00" + strings.ToLower(strings.TrimSpace(query))
}
