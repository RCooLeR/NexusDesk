package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const recentWorkspaceLimit = 10

type RecentWorkspace struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	LastOpened string `json:"lastOpened"`
}

type RecentWorkspaceStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultRecentWorkspaceStore() *RecentWorkspaceStore {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}

	return NewRecentWorkspaceStore(filepath.Join(configDir, "NexusDesk", "recent-workspaces.json"))
}

func NewRecentWorkspaceStore(path string) *RecentWorkspaceStore {
	return &RecentWorkspaceStore{path: path}
}

func (s *RecentWorkspaceStore) List() ([]RecentWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.read()
}

func (s *RecentWorkspaceStore) Add(path string) ([]RecentWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	items, err := s.read()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	next := []RecentWorkspace{{
		Name:       filepath.Base(absPath),
		Path:       absPath,
		LastOpened: now,
	}}

	for _, item := range items {
		if samePath(item.Path, absPath) {
			continue
		}
		next = append(next, item)
	}

	sortRecent(next)
	if len(next) > recentWorkspaceLimit {
		next = next[:recentWorkspaceLimit]
	}

	if err := s.write(next); err != nil {
		return nil, err
	}

	return next, nil
}

func (s *RecentWorkspaceStore) Remove(path string) ([]RecentWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	items, err := s.read()
	if err != nil {
		return nil, err
	}

	next := make([]RecentWorkspace, 0, len(items))
	for _, item := range items {
		if samePath(item.Path, absPath) {
			continue
		}
		next = append(next, item)
	}

	if err := s.write(next); err != nil {
		return nil, err
	}

	return next, nil
}

func (s *RecentWorkspaceStore) Clear() ([]RecentWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := []RecentWorkspace{}
	if err := s.write(items); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *RecentWorkspaceStore) read() ([]RecentWorkspace, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []RecentWorkspace{}, nil
	}
	if err != nil {
		return nil, err
	}

	var items []RecentWorkspace
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	sortRecent(items)
	if len(items) > recentWorkspaceLimit {
		items = items[:recentWorkspaceLimit]
	}

	return items, nil
}

func (s *RecentWorkspaceStore) write(items []RecentWorkspace) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func sortRecent(items []RecentWorkspace) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].LastOpened > items[j].LastOpened
	})
}

func samePath(left string, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}
