package shell

import (
	"strings"
	"sync"

	"nexusdesk/internal/domain"
)

type State struct {
	mu                    sync.RWMutex
	workspace             domain.Workspace
	selected              string
	assistantContextPaths []string
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
	s.assistantContextPaths = nil
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

func (s *State) AssistantContextPaths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string{}, s.assistantContextPaths...)
}

func (s *State) AddAssistantContextPath(relPath string) bool {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.assistantContextPaths {
		if existing == relPath {
			return false
		}
	}
	s.assistantContextPaths = append(s.assistantContextPaths, relPath)
	return true
}

func (s *State) RemoveAssistantContextPath(relPath string) bool {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, existing := range s.assistantContextPaths {
		if existing != relPath {
			continue
		}
		s.assistantContextPaths = append(s.assistantContextPaths[:index], s.assistantContextPaths[index+1:]...)
		return true
	}
	return false
}

func (s *State) ClearAssistantContextPaths() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assistantContextPaths = nil
}
