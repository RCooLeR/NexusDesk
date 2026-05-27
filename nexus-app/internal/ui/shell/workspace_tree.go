package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type treeStore struct {
	root     string
	service  *workspaceSvc.Service
	roots    []string
	children map[string][]string
	loaded   map[string]bool
	nodes    map[string]domain.WorkspaceNode
}

func newTreeStore(workspace domain.Workspace, service *workspaceSvc.Service) *treeStore {
	store := &treeStore{
		root:     workspace.Root,
		service:  service,
		roots:    []string{},
		children: map[string][]string{},
		loaded:   map[string]bool{"": true},
		nodes:    map[string]domain.WorkspaceNode{},
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
	listing, err := s.service.ListChildren(s.root, parentID)
	if err != nil {
		return err
	}
	s.setChildren(parentID, listing.Nodes)
	s.loaded[parentID] = true
	return nil
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

func newWorkspaceTree(
	state *State,
	service *workspaceSvc.Service,
	onSelected func(domain.WorkspaceNode),
	onContext func(domain.WorkspaceNode, *fyne.PointEvent),
) *widget.Tree {
	store := newTreeStore(state.Workspace(), service)
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
			row.setNode(node)
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
		tree.Refresh()
	}
	return tree
}

type workspaceTreeRow struct {
	widget.BaseWidget

	icon        *widget.Icon
	label       *widget.Label
	node        domain.WorkspaceNode
	onSecondary func(domain.WorkspaceNode, *fyne.PointEvent)
}

func newWorkspaceTreeRow(onSecondary func(domain.WorkspaceNode, *fyne.PointEvent)) *workspaceTreeRow {
	row := &workspaceTreeRow{
		icon:        widget.NewIcon(nil),
		label:       widget.NewLabel(""),
		onSecondary: onSecondary,
	}
	row.label.Truncation = fyne.TextTruncateEllipsis
	row.ExtendBaseWidget(row)
	return row
}

func (r *workspaceTreeRow) setNode(node domain.WorkspaceNode) {
	r.node = node
	if node.Kind == domain.NodeDirectory {
		r.icon.SetResource(theme.FolderIcon())
	} else {
		r.icon.SetResource(theme.FileIcon())
	}
	r.label.SetText(node.Name)
	r.Refresh()
}

func (r *workspaceTreeRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, r.icon, nil, r.label))
}

func (r *workspaceTreeRow) TappedSecondary(event *fyne.PointEvent) {
	if r.onSecondary == nil || r.node.ID == "" {
		return
	}
	r.onSecondary(r.node, event)
}
