package shell

import (
	"sync"

	"nexusdesk/internal/domain"
)

type State struct {
	mu        sync.RWMutex
	workspace domain.Workspace
	selected  string
}

func NewState() *State {
	return &State{}
}

func (s *State) Workspace() domain.Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspace
}

func (s *State) SetWorkspace(workspace domain.Workspace) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = workspace
	s.selected = ""
}

func (s *State) SelectedPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selected
}

func (s *State) SetSelectedPath(relPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selected = relPath
}
