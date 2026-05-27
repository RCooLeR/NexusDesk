package datasets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	queryStoreName            = "queries.json"
	maxSavedQueriesPerSource  = 12
	maxSavedQueryLabelLength  = 120
	maxSavedQueryContentBytes = 20000
)

func (s *Service) SaveQuery(root string, relPath string, query string, label string, kind string) (SavedQuery, error) {
	return SaveQuery(root, relPath, query, label, kind)
}

func (s *Service) ListSavedQueries(root string, relPath string, kind string) ([]SavedQuery, error) {
	return ListSavedQueries(root, relPath, kind)
}

func SaveQuery(root string, relPath string, query string, label string, kind string) (SavedQuery, error) {
	_, cleanRel, _, err := resolveDatasetFile(root, relPath)
	if err != nil {
		return SavedQuery{}, err
	}
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return SavedQuery{}, err
	}
	now := time.Now().UTC()
	query = cleanSavedQuery(query)
	label = cleanSavedQueryLabel(label, query)
	kind = cleanSavedQueryKind(kind)
	items, err := readSavedQueries(absRoot)
	if err != nil {
		return SavedQuery{}, err
	}
	saved := SavedQuery{
		RelPath:   filepath.ToSlash(cleanRel),
		Query:     query,
		Label:     label,
		Kind:      kind,
		UpdatedAt: now,
	}
	items[savedQueryKey(saved.RelPath, saved.Query, saved.Kind)] = saved
	trimSavedQueries(items, saved.RelPath, saved.Kind)
	if err := writeSavedQueries(absRoot, items); err != nil {
		return SavedQuery{}, err
	}
	return saved, nil
}

func ListSavedQueries(root string, relPath string, kind string) ([]SavedQuery, error) {
	_, cleanRel, _, err := resolveDatasetFile(root, relPath)
	if err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, err
	}
	items, err := readSavedQueries(absRoot)
	if err != nil {
		return nil, err
	}
	kind = cleanSavedQueryKind(kind)
	queries := []SavedQuery{}
	for _, item := range items {
		if filepath.ToSlash(item.RelPath) == cleanRel && cleanSavedQueryKind(item.Kind) == kind {
			queries = append(queries, item)
		}
	}
	sort.SliceStable(queries, func(left int, right int) bool {
		return queries[left].UpdatedAt.After(queries[right].UpdatedAt)
	})
	return queries, nil
}

func readSavedQueries(absRoot string) (map[string]SavedQuery, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(notebookStoreRelDir), queryStoreName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]SavedQuery{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]SavedQuery{}
	if strings.TrimSpace(string(data)) == "" {
		return items, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func writeSavedQueries(absRoot string, items map[string]SavedQuery) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(notebookStoreRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, queryStoreName), append(data, '\n'), 0o644)
}

func trimSavedQueries(items map[string]SavedQuery, relPath string, kind string) {
	queries := []SavedQuery{}
	for _, item := range items {
		if filepath.ToSlash(item.RelPath) == relPath && cleanSavedQueryKind(item.Kind) == kind {
			queries = append(queries, item)
		}
	}
	sort.SliceStable(queries, func(left int, right int) bool {
		return queries[left].UpdatedAt.After(queries[right].UpdatedAt)
	})
	for index, query := range queries {
		if index < maxSavedQueriesPerSource {
			continue
		}
		delete(items, savedQueryKey(query.RelPath, query.Query, query.Kind))
	}
}

func cleanSavedQuery(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxSavedQueryContentBytes {
		return value[:maxSavedQueryContentBytes]
	}
	return value
}

func cleanSavedQueryLabel(label string, query string) string {
	label = strings.Join(strings.Fields(label), " ")
	if label == "" {
		label = strings.Join(strings.Fields(query), " ")
	}
	if label == "" {
		label = "Saved query"
	}
	if len(label) > maxSavedQueryLabelLength {
		return label[:maxSavedQueryLabelLength]
	}
	return label
}

func savedQueryKey(relPath string, query string, kind string) string {
	return cleanSavedQueryKind(kind) + "\x00" + strings.ToLower(filepath.ToSlash(relPath)) + "\x00" + strings.ToLower(strings.TrimSpace(query))
}

func cleanSavedQueryKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "sql", "sqlite-sql", "filter":
		return kind
	default:
		return "filter"
	}
}
