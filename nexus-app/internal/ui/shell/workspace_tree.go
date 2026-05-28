package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type treeStore struct {
	root           string
	service        *workspaceSvc.Service
	includeIgnored bool
	roots          []string
	children       map[string][]string
	loaded         map[string]bool
	nodes          map[string]domain.WorkspaceNode
	summaries      map[string]domain.ScanSummary
	badges         map[string]string
}

func newTreeStore(workspace domain.Workspace, service *workspaceSvc.Service, badges map[string]string) *treeStore {
	store := &treeStore{
		root:     workspace.Root,
		service:  service,
		roots:    []string{},
		children: map[string][]string{},
		loaded:   map[string]bool{"": true},
		nodes:    map[string]domain.WorkspaceNode{},
		summaries: map[string]domain.ScanSummary{
			"": workspace.Summary,
		},
		badges: cloneStringMap(badges),
	}
	store.setChildren("", workspace.Tree)
	return store
}

func (s *treeStore) childIDs(parentID string) []string {
	if !s.loaded[parentID] {
		_ = s.load(parentID)
	}
	return s.children[parentID]
}

func (s *treeStore) load(parentID string) error {
	listing, err := s.service.ListChildrenWithOptions(s.root, parentID, workspaceSvc.ListOptions{IncludeIgnored: s.includeIgnored})
	if err != nil {
		return err
	}
	s.setChildren(parentID, listing.Nodes)
	s.summaries[parentID] = listing.Summary
	s.loaded[parentID] = true
	return nil
}

func (s *treeStore) setIncludeIgnored(include bool) error {
	s.includeIgnored = include
	s.roots = []string{}
	s.children = map[string][]string{}
	s.loaded = map[string]bool{}
	s.nodes = map[string]domain.WorkspaceNode{}
	s.summaries = map[string]domain.ScanSummary{}
	return s.load("")
}

func (s *treeStore) setChildren(parentID string, nodes []domain.WorkspaceNode) {
	ids := []string{}
	for _, node := range nodes {
		ids = append(ids, node.ID)
		s.nodes[node.ID] = node
	}
	if parentID == "" {
		s.roots = ids
	}
	s.children[parentID] = ids
}

func (s *treeStore) node(uid widget.TreeNodeID) (domain.WorkspaceNode, bool) {
	node, ok := s.nodes[uid]
	return node, ok
}

func (s *treeStore) summary(parentID string) domain.ScanSummary {
	return s.summaries[parentID]
}

func (s *treeStore) visibleSummary() domain.ScanSummary {
	summary := s.summary("")
	summary.EntryCap = 0
	for _, loaded := range s.summaries {
		summary.EntryCap += loaded.EntryCap
	}
	return summary
}

func (s *treeStore) badge(relPath string) string {
	return s.badges[relPath]
}

func (s *treeStore) branchPathForSelection(selected string) []string {
	selected = strings.Trim(selected, "/")
	if selected == "" {
		return []string{}
	}
	if node, ok := s.nodes[selected]; ok && node.Kind != domain.NodeDirectory {
		selected = node.ParentID
	}
	parts := strings.Split(selected, "/")
	branches := []string{}
	for index := range parts {
		branch := strings.Join(parts[:index+1], "/")
		if node, ok := s.nodes[branch]; ok && node.Kind == domain.NodeDirectory {
			branches = append(branches, branch)
		}
	}
	return branches
}

func newWorkspaceTree(
	state *State,
	service *workspaceSvc.Service,
	badges map[string]string,
	onSelected func(domain.WorkspaceNode),
	onContext func(domain.WorkspaceNode, *fyne.PointEvent),
	onLoaded func(string, domain.ScanSummary),
) (*widget.Tree, *treeStore) {
	store := newTreeStore(state.Workspace(), service, badges)
	tree := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				return store.roots
			}
			return store.childIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			node, ok := store.node(uid)
			return ok && node.Kind == domain.NodeDirectory
		},
		func(branch bool) fyne.CanvasObject {
			return newWorkspaceTreeRow(onContext)
		},
		func(uid widget.TreeNodeID, branch bool, object fyne.CanvasObject) {
			node, ok := store.node(uid)
			if !ok {
				return
			}
			row := object.(*workspaceTreeRow)
			row.setNode(node, store.badge(node.RelPath))
		},
	)
	tree.OnSelected = func(uid widget.TreeNodeID) {
		node, ok := store.node(uid)
		if !ok {
			return
		}
		state.SetSelectedPath(node.RelPath)
		onSelected(node)
	}
	tree.OnBranchOpened = func(uid widget.TreeNodeID) {
		if err := store.load(uid); err != nil {
			return
		}
		if onLoaded != nil {
			onLoaded(uid, store.summary(uid))
		}
		tree.Refresh()
	}
	return tree, store
}

type workspaceTreeRow struct {
	widget.BaseWidget

	icon        *widget.Icon
	label       *widget.Label
	badge       *widget.Label
	node        domain.WorkspaceNode
	onSecondary func(domain.WorkspaceNode, *fyne.PointEvent)
}

func newWorkspaceTreeRow(onSecondary func(domain.WorkspaceNode, *fyne.PointEvent)) *workspaceTreeRow {
	row := &workspaceTreeRow{
		icon:        widget.NewIcon(nil),
		label:       widget.NewLabel(""),
		badge:       widget.NewLabel(""),
		onSecondary: onSecondary,
	}
	row.label.Truncation = fyne.TextTruncateEllipsis
	row.badge.TextStyle = fyne.TextStyle{Italic: true}
	row.ExtendBaseWidget(row)
	return row
}

func (r *workspaceTreeRow) setNode(node domain.WorkspaceNode, badge string) {
	r.node = node
	if node.Kind == domain.NodeDirectory {
		r.icon.SetResource(theme.FolderIcon())
	} else {
		r.icon.SetResource(theme.FileIcon())
	}
	r.label.SetText(node.Name)
	if node.Ignored {
		r.badge.SetText("ignored")
		r.badge.Show()
	} else if strings.TrimSpace(badge) != "" {
		r.badge.SetText(badge)
		r.badge.Show()
	} else {
		r.badge.SetText("")
		r.badge.Hide()
	}
	r.Refresh()
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func (r *workspaceTreeRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, r.icon, r.badge, r.label))
}

func (r *workspaceTreeRow) TappedSecondary(event *fyne.PointEvent) {
	if r.onSecondary == nil || r.node.ID == "" {
		return
	}
	r.onSecondary(r.node, event)
}
