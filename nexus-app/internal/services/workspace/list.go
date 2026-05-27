package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"nexusdesk/internal/domain"
)

var ignoredNames = map[string]struct{}{
	".git":         {},
	".idea":        {},
	".nexusdesk":   {},
	"build":        {},
	"dist":         {},
	"node_modules": {},
}

type ListResult struct {
	Nodes   []domain.WorkspaceNode
	Summary domain.ScanSummary
}

type ListOptions struct {
	IncludeIgnored bool
}

func (s *Service) ListChildren(root string, relPath string) (ListResult, error) {
	return s.ListChildrenWithOptions(root, relPath, ListOptions{})
}

func (s *Service) ListChildrenWithOptions(root string, relPath string, options ListOptions) (ListResult, error) {
	target, cleanRelPath, err := resolveDirectory(root, relPath)
	if err != nil {
		return ListResult{}, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return ListResult{Summary: domain.ScanSummary{Unreadable: 1}}, nil
	}
	sortEntries(entries)
	return s.nodesFromEntries(cleanRelPath, entries, options), nil
}

func (s *Service) nodesFromEntries(parentID string, entries []os.DirEntry, options ListOptions) ListResult {
	result := ListResult{Nodes: []domain.WorkspaceNode{}}
	for _, entry := range entries {
		if result.Summary.Included >= s.entryLimit {
			result.Summary.EntryCap++
			break
		}
		node, ok := nodeFromEntry(parentID, entry, options, &result.Summary)
		if ok {
			result.Nodes = append(result.Nodes, node)
			result.Summary.Included++
		}
	}
	return result
}

func resolveDirectory(root string, relPath string) (string, string, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return "", "", err
	}
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(absRoot, filepath.FromSlash(cleanRelPath))
	if !isInside(absRoot, target) {
		return "", "", errors.New("workspace path must stay inside the root")
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", errors.New("workspace path must be a directory")
	}
	return target, cleanRelPath, nil
}

func nodeFromEntry(parentID string, entry os.DirEntry, options ListOptions, summary *domain.ScanSummary) (domain.WorkspaceNode, bool) {
	name := entry.Name()
	ignored := isIgnoredName(name)
	if ignored {
		summary.Ignored++
		if !options.IncludeIgnored || isInternalMetadataPath(filepath.ToSlash(filepath.Join(parentID, name))) {
			return domain.WorkspaceNode{}, false
		}
	}
	info, err := entry.Info()
	if err != nil {
		summary.Unreadable++
		return domain.WorkspaceNode{}, false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		summary.Ignored++
		return domain.WorkspaceNode{}, false
	}
	childRelPath := filepath.ToSlash(filepath.Join(parentID, name))
	childRelPath = strings.TrimPrefix(childRelPath, "./")
	kind := domain.NodeFile
	if info.IsDir() {
		kind = domain.NodeDirectory
	}
	return domain.WorkspaceNode{
		ID:       childRelPath,
		ParentID: parentID,
		Name:     name,
		RelPath:  childRelPath,
		Kind:     kind,
		Size:     info.Size(),
		Ignored:  ignored,
	}, true
}

func isIgnoredName(name string) bool {
	_, ignored := ignoredNames[name]
	return ignored
}

func sortEntries(entries []os.DirEntry) {
	sort.Slice(entries, func(left int, right int) bool {
		leftInfo, _ := entries[left].Info()
		rightInfo, _ := entries[right].Info()
		if leftInfo != nil && rightInfo != nil && leftInfo.IsDir() != rightInfo.IsDir() {
			return leftInfo.IsDir()
		}
		return strings.ToLower(entries[left].Name()) < strings.ToLower(entries[right].Name())
	})
}
