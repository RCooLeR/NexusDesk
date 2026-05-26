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

func newWorkspaceTree(state *State, service *workspaceSvc.Service, onSelected func(domain.WorkspaceNode)) *widget.Tree {
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
			icon := widget.NewIcon(nil)
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, icon, nil, label)
		},
		func(uid widget.TreeNodeID, branch bool, object fyne.CanvasObject) {
			node, ok := store.node(uid)
			if !ok {
				return
			}
			row := object.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			label := row.Objects[1].(*widget.Label)
			if node.Kind == domain.NodeDirectory {
				icon.SetResource(theme.FolderIcon())
			} else {
				icon.SetResource(theme.FileIcon())
			}
			label.SetText(node.Name)
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
