package recentworkspaces

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const Limit = 10

type Workspace struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	LastOpened string `json:"lastOpened"`
	Exists     bool   `json:"-"`
	Missing    bool   `json:"-"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewFileStore(filepath.Join(dir, "NexusDesk", "recent-workspaces.json")), nil
}

func NewFileStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) List() ([]Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	return withPathStatus(items), nil
}

func (s *Store) Add(path string) ([]Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	next := []Workspace{{
		Name:       filepath.Base(absPath),
		Path:       absPath,
		LastOpened: time.Now().UTC().Format(time.RFC3339),
	}}
	for _, item := range items {
		if samePath(item.Path, absPath) {
			continue
		}
		next = append(next, item)
	}
	sortRecent(next)
	if len(next) > Limit {
		next = next[:Limit]
	}
	if err := s.write(next); err != nil {
		return nil, err
	}
	return next, nil
}

func (s *Store) Remove(path string) ([]Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	next := make([]Workspace, 0, len(items))
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

func (s *Store) Clear() ([]Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []Workspace{}
	if err := s.write(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) RemoveMissing() ([]Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	next := make([]Workspace, 0, len(items))
	for _, item := range items {
		if workspaceMissing(item.Path) {
			continue
		}
		next = append(next, item)
	}
	if err := s.write(next); err != nil {
		return nil, err
	}
	return withPathStatus(next), nil
}

func (s *Store) read() ([]Workspace, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Workspace{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []Workspace
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	sortRecent(items)
	if len(items) > Limit {
		items = items[:Limit]
	}
	return items, nil
}

func (s *Store) write(items []Workspace) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func sortRecent(items []Workspace) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].LastOpened > items[j].LastOpened
	})
}

func samePath(left string, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func withPathStatus(items []Workspace) []Workspace {
	next := make([]Workspace, 0, len(items))
	for _, item := range items {
		item.Exists = false
		item.Missing = false
		if info, err := os.Stat(item.Path); err == nil {
			item.Exists = info.IsDir()
		} else if os.IsNotExist(err) {
			item.Missing = true
		}
		next = append(next, item)
	}
	return next
}

func workspaceMissing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
