package editor

import "strconv"

type Session struct {
	tabs     []Tab
	activeID string
	nextID   int
}

func NewSession() *Session {
	return &Session{}
}

func (s *Session) Tabs() []Tab {
	tabs := make([]Tab, len(s.tabs))
	copy(tabs, s.tabs)
	return tabs
}

func (s *Session) ActiveID() string {
	return s.activeID
}

func (s *Session) Tab(id string) (Tab, bool) {
	index := s.find(id)
	if index < 0 {
		return Tab{}, false
	}
	return s.tabs[index], true
}

func (s *Session) OpenWelcome(title string) Tab {
	return s.open(Tab{Title: title, Kind: KindWelcome})
}

func (s *Session) OpenPlaceholder(title string) Tab {
	return s.open(Tab{Title: title, Kind: KindPlaceholder})
}

func (s *Session) OpenFile(relPath, title string) Tab {
	return s.OpenFileWithSource(relPath, title, "")
}

func (s *Session) OpenFileWithSource(relPath, title, source string) Tab {
	for index := range s.tabs {
		if s.tabs[index].Kind == KindFile && s.tabs[index].RelPath == relPath {
			s.tabs[index].Title = title
			if !s.tabs[index].Dirty {
				s.tabs[index].SourceText = source
				s.tabs[index].DraftText = source
			}
			s.activeID = s.tabs[index].ID
			return s.tabs[index]
		}
	}
	return s.open(Tab{Title: title, RelPath: relPath, Kind: KindFile, SourceText: source, DraftText: source})
}

func (s *Session) MarkDirty(id string, dirty bool) bool {
	index := s.find(id)
	if index < 0 {
		return false
	}
	s.tabs[index].Dirty = dirty
	return true
}

func (s *Session) UpdateDraft(id, draft string) bool {
	index := s.find(id)
	if index < 0 {
		return false
	}
	s.tabs[index].DraftText = draft
	s.tabs[index].Dirty = draft != s.tabs[index].SourceText
	return true
}

func (s *Session) RevertDraft(id string) (Tab, bool) {
	index := s.find(id)
	if index < 0 {
		return Tab{}, false
	}
	s.tabs[index].DraftText = s.tabs[index].SourceText
	s.tabs[index].Dirty = false
	return s.tabs[index], true
}

func (s *Session) MarkDraftSaved(id string) (Tab, bool) {
	index := s.find(id)
	if index < 0 {
		return Tab{}, false
	}
	s.tabs[index].SourceText = s.tabs[index].DraftText
	s.tabs[index].Dirty = false
	return s.tabs[index], true
}

func (s *Session) TogglePinned(id string) (Tab, bool) {
	index := s.find(id)
	if index < 0 {
		return Tab{}, false
	}
	s.tabs[index].Pinned = !s.tabs[index].Pinned
	tab := s.tabs[index]
	s.reorderPinned()
	return tab, true
}

func (s *Session) Close(id string, force bool) (Tab, bool) {
	index := s.find(id)
	if index < 0 || (s.tabs[index].Dirty && !force) {
		return Tab{}, false
	}
	tab := s.tabs[index]
	s.tabs = append(s.tabs[:index], s.tabs[index+1:]...)
	if s.activeID == id {
		s.activeID = ""
		if len(s.tabs) > 0 {
			next := index
			if next >= len(s.tabs) {
				next = len(s.tabs) - 1
			}
			s.activeID = s.tabs[next].ID
		}
	}
	return tab, true
}

func (s *Session) open(tab Tab) Tab {
	s.nextID++
	tab.ID = "tab-" + strconv.Itoa(s.nextID)
	s.tabs = append(s.tabs, tab)
	s.reorderPinned()
	s.activeID = tab.ID
	return tab
}

func (s *Session) find(id string) int {
	for index := range s.tabs {
		if s.tabs[index].ID == id {
			return index
		}
	}
	return -1
}

func (s *Session) reorderPinned() {
	ordered := make([]Tab, 0, len(s.tabs))
	for _, tab := range s.tabs {
		if tab.Pinned {
			ordered = append(ordered, tab)
		}
	}
	for _, tab := range s.tabs {
		if !tab.Pinned {
			ordered = append(ordered, tab)
		}
	}
	s.tabs = ordered
}
